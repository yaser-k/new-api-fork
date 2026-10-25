package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockruntimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go/auth/bearer"
)

// getAwsErrorStatusCode extracts HTTP status code from AWS SDK error
func getAwsErrorStatusCode(err error) int {
	// Check for HTTP response error which contains status code
	var httpErr interface{ HTTPStatusCode() int }
	if errors.As(err, &httpErr) {
		return httpErr.HTTPStatusCode()
	}
	// Default to 500 if we can't determine the status code
	return http.StatusInternalServerError
}

func newAwsInvokeContext(parent context.Context) (context.Context, context.CancelFunc) {
	if common.RelayTimeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, time.Duration(common.RelayTimeout)*time.Second)
}

func newAwsInvokeError(requestContext context.Context, err error, operation string) *types.NewAPIError {
	options := make([]types.NewAPIErrorOptions, 0, 1)
	if requestContext.Err() != nil {
		options = append(options, types.ErrOptionWithSkipRetry())
	}
	return types.NewOpenAIError(
		errors.Wrap(err, operation),
		types.ErrorCodeAwsInvokeError,
		getAwsErrorStatusCode(err),
		options...,
	)
}

func newAwsClient(c *gin.Context, info *relaycommon.RelayInfo) (*bedrockruntime.Client, error) {
	httpClient, err := service.GetHttpClientWithProxySettings(info.ChannelSetting.Proxy, info.ChannelSetting)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}

	awsSecret := strings.Split(info.ApiKey, "|")
	var client *bedrockruntime.Client
	switch len(awsSecret) {
	case 2:
		apiKey := awsSecret[0]
		region := awsSecret[1]
		client = bedrockruntime.New(bedrockruntime.Options{
			Region:                  region,
			BearerAuthTokenProvider: bearer.StaticTokenProvider{Token: bearer.Token{Value: apiKey}},
			HTTPClient:              httpClient,
		})
	case 3:
		ak := awsSecret[0]
		sk := awsSecret[1]
		region := awsSecret[2]
		client = bedrockruntime.New(bedrockruntime.Options{
			Region:      region,
			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(ak, sk, "")),
			HTTPClient:  httpClient,
		})
	default:
		return nil, errors.New("invalid aws secret key")
	}

	return client, nil
}

func doAwsClientRequest(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor, requestBody io.Reader) (any, error) {
	awsCli, err := newAwsClient(c, info)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeChannelAwsClientError)
	}
	a.AwsClient = awsCli

	// 获取对应的AWS模型ID
	awsModelId := getAwsModelID(info.UpstreamModelName)

	awsRegionPrefix := getAwsRegionPrefix(awsCli.Options().Region)
	canCrossRegion := awsModelCanCrossRegion(awsModelId, awsRegionPrefix)
	if canCrossRegion {
		awsModelId = awsModelCrossRegion(awsModelId, awsRegionPrefix)
	}

	// init empty request.header
	requestHeader := http.Header{}
	a.SetupRequestHeader(c, &requestHeader, info)
	headerOverride, err := channel.ResolveHeaderOverride(info, c)
	if err != nil {
		return nil, err
	}
	for key, value := range headerOverride {
		requestHeader.Set(key, value)
	}

	if isNovaModel(awsModelId) {
		// Only ConvertOpenAIRequest builds a Nova body. Any other request (the
		// Messages API, or a passed-through body) would be sent as one and its
		// answer read as Anthropic's, billed as zero.
		if !a.IsNova {
			return nil, types.NewErrorWithStatusCode(errors.New("this request is not supported for Nova models on AWS Bedrock; use chat completions without pass-through"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		var novaReq *NovaRequest
		err = common.DecodeJson(requestBody, &novaReq)
		if err != nil {
			return nil, types.NewError(errors.Wrap(err, "decode nova request fail"), types.ErrorCodeBadRequestBody)
		}

		// 使用InvokeModel API，但使用Nova格式的请求体
		awsReq := &bedrockruntime.InvokeModelInput{
			ModelId:     aws.String(awsModelId),
			Accept:      aws.String("application/json"),
			ContentType: aws.String("application/json"),
		}

		reqBody, err := common.Marshal(novaReq)
		if err != nil {
			return nil, types.NewError(errors.Wrap(err, "marshal nova request"), types.ErrorCodeBadResponseBody)
		}
		if info.IsStream {
			a.AwsReq = &bedrockruntime.InvokeModelWithResponseStreamInput{
				ModelId:     awsReq.ModelId,
				Accept:      awsReq.Accept,
				ContentType: awsReq.ContentType,
				Body:        reqBody,
			}
			return nil, nil
		}
		awsReq.Body = reqBody
		a.AwsReq = awsReq
		return nil, nil
	} else {
		awsClaudeReq, err := formatRequest(requestBody, requestHeader)
		if err != nil {
			return nil, types.NewError(errors.Wrap(err, "format aws request fail"), types.ErrorCodeBadRequestBody)
		}

		if info.IsStream {
			awsReq := &bedrockruntime.InvokeModelWithResponseStreamInput{
				ModelId:     aws.String(awsModelId),
				Accept:      aws.String("application/json"),
				ContentType: aws.String("application/json"),
			}
			awsReq.Body, err = buildAwsRequestBody(c, info, awsClaudeReq)
			if err != nil {
				return nil, types.NewError(errors.Wrap(err, "marshal aws request fail"), types.ErrorCodeBadRequestBody)
			}
			a.AwsReq = awsReq
			return nil, nil
		} else {
			awsReq := &bedrockruntime.InvokeModelInput{
				ModelId:     aws.String(awsModelId),
				Accept:      aws.String("application/json"),
				ContentType: aws.String("application/json"),
			}
			awsReq.Body, err = buildAwsRequestBody(c, info, awsClaudeReq)
			if err != nil {
				return nil, types.NewError(errors.Wrap(err, "marshal aws request fail"), types.ErrorCodeBadRequestBody)
			}
			a.AwsReq = awsReq
			return nil, nil
		}
	}
}

// buildAwsRequestBody prepares the payload for AWS requests, applying passthrough rules when enabled.
func buildAwsRequestBody(c *gin.Context, info *relaycommon.RelayInfo, awsClaudeReq any) ([]byte, error) {
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return nil, errors.Wrap(err, "get request body for pass-through fail")
		}
		body, err := storage.Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "get request body bytes fail")
		}
		var data map[string]any
		if err := common.Unmarshal(body, &data); err != nil {
			return nil, errors.Wrap(err, "pass-through unmarshal request body fail")
		}
		delete(data, "model")
		delete(data, "stream")
		return common.Marshal(data)
	}
	return common.Marshal(awsClaudeReq)
}

func getAwsRegionPrefix(awsRegionId string) string {
	parts := strings.Split(awsRegionId, "-")
	regionPrefix := ""
	if len(parts) > 0 {
		regionPrefix = parts[0]
	}
	return regionPrefix
}

func awsModelCanCrossRegion(awsModelId, awsRegionPrefix string) bool {
	regionSet, exists := awsModelCanCrossRegionMap[awsModelId]
	return exists && regionSet[awsRegionPrefix]
}

func awsModelCrossRegion(awsModelId, awsRegionPrefix string) string {
	modelPrefix, find := awsRegionCrossModelPrefixMap[awsRegionPrefix]
	if !find {
		return awsModelId
	}
	return modelPrefix + "." + awsModelId
}

func getAwsModelID(requestModel string) string {
	if awsModelIDName, ok := awsModelIDMap[requestModel]; ok {
		return awsModelIDName
	}
	return requestModel
}

func awsHandler(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {

	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	awsResp, err := a.AwsClient.InvokeModel(ctx, a.AwsReq.(*bedrockruntime.InvokeModelInput))
	if err != nil {
		return newAwsInvokeError(requestContext, err, "InvokeModel"), nil
	}

	claudeInfo := &claude.ClaudeResponseInfo{
		ResponseId:   helper.GetResponseID(c),
		Created:      common.GetTimestamp(),
		Model:        info.UpstreamModelName,
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}

	// 复制上游 Content-Type 到客户端响应头
	if awsResp.ContentType != nil && *awsResp.ContentType != "" {
		c.Writer.Header().Set("Content-Type", *awsResp.ContentType)
	}

	handlerErr := claude.HandleClaudeResponseData(c, info, claudeInfo, nil, awsResp.Body)
	if handlerErr != nil {
		return handlerErr, nil
	}
	return nil, claudeInfo.Usage
}

func awsStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {
	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	awsResp, err := a.AwsClient.InvokeModelWithResponseStream(ctx, a.AwsReq.(*bedrockruntime.InvokeModelWithResponseStreamInput))
	if err != nil {
		return newAwsInvokeError(requestContext, err, "InvokeModelWithResponseStream"), nil
	}
	stream := awsResp.GetStream()
	defer stream.Close()
	info.StreamStatus = relaycommon.NewStreamStatus()

	claudeInfo := &claude.ClaudeResponseInfo{
		ResponseId:   helper.GetResponseID(c),
		Created:      common.GetTimestamp(),
		Model:        info.UpstreamModelName,
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}

	events := stream.Events()
streamLoop:
	for {
		select {
		case <-ctx.Done():
			break streamLoop
		case event, ok := <-events:
			if !ok {
				break streamLoop
			}
			if ctx.Err() != nil {
				break streamLoop
			}

			switch v := event.(type) {
			case *bedrockruntimeTypes.ResponseStreamMemberChunk:
				info.SetFirstResponseTime()
				respErr := claude.HandleStreamResponseData(c, info, claudeInfo, string(v.Value.Bytes))
				if respErr != nil {
					return respErr, nil
				}
			case *bedrockruntimeTypes.UnknownUnionMember:
				fmt.Println(common.LogText("unknown tag: %v", v.Tag))
				return types.NewError(errors.New("unknown response type"), types.ErrorCodeInvalidRequest), nil
			default:
				fmt.Println(common.LogText("union is nil or unknown type"))
				return types.NewError(errors.New("nil or unknown response type"), types.ErrorCodeInvalidRequest), nil
			}
		}
	}

	switch {
	case requestContext.Err() != nil:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, requestContext.Err())
	case ctx.Err() != nil:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, ctx.Err())
	case stream.Err() != nil:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, stream.Err())
	default:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
	}
	_ = stream.Close()
	claude.HandleStreamFinalResponse(c, info, claudeInfo)
	return nil, claudeInfo.Usage
}

// Nova模型处理函数
func handleNovaRequest(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {

	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	awsResp, err := a.AwsClient.InvokeModel(ctx, a.AwsReq.(*bedrockruntime.InvokeModelInput))
	if err != nil {
		return newAwsInvokeError(requestContext, err, "InvokeModel"), nil
	}

	// 解析Nova响应
	var novaResp struct {
		Output struct {
			Message struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"inputTokens"`
			OutputTokens int `json:"outputTokens"`
			TotalTokens  int `json:"totalTokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(awsResp.Body, &novaResp); err != nil {
		return types.NewError(errors.Wrap(err, "unmarshal nova response"), types.ErrorCodeBadResponseBody), nil
	}
	text := strings.Builder{}
	for _, block := range novaResp.Output.Message.Content {
		text.WriteString(block.Text)
	}

	// 构造OpenAI格式响应
	response := dto.OpenAITextResponse{
		Id:      helper.GetResponseID(c),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   info.UpstreamModelName,
		Choices: []dto.OpenAITextResponseChoice{{
			Index: 0,
			Message: dto.Message{
				Role:    "assistant",
				Content: text.String(),
			},
			FinishReason: "stop",
		}},
		Usage: dto.Usage{
			PromptTokens:     novaResp.Usage.InputTokens,
			CompletionTokens: novaResp.Usage.OutputTokens,
			TotalTokens:      novaResp.Usage.TotalTokens,
		},
	}

	c.JSON(http.StatusOK, response)
	return nil, &response.Usage
}

// novaStreamEvent is one chunk of a Nova InvokeModelWithResponseStream answer:
// a single event in the Converse stream shape, plus the invocation metrics
// Bedrock adds to the last chunk.
type novaStreamEvent struct {
	MessageStart *struct {
		Role string `json:"role"`
	} `json:"messageStart"`
	ContentBlockDelta *struct {
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"contentBlockDelta"`
	MessageStop *struct {
		StopReason string `json:"stopReason"`
	} `json:"messageStop"`
	Metadata *struct {
		Usage struct {
			InputTokens  int `json:"inputTokens"`
			OutputTokens int `json:"outputTokens"`
		} `json:"usage"`
	} `json:"metadata"`
	InvocationMetrics *struct {
		InputTokenCount  int `json:"inputTokenCount"`
		OutputTokenCount int `json:"outputTokenCount"`
	} `json:"amazon-bedrock-invocationMetrics"`
}

// novaFinishReason maps a Bedrock stop reason to an OpenAI finish reason.
func novaFinishReason(stopReason string) string {
	switch stopReason {
	case "content_filtered", "guardrail_intervened":
		return types.FinishReasonContentFilter
	case "model_context_window_exceeded":
		return types.FinishReasonLength
	}
	return relayconvert.StopReasonClaudeToOpenAI(stopReason)
}

// handleNovaStreamRequest relays a Nova event stream as OpenAI chat completion
// chunks: the role, each text delta, the finish reason, then the usage Bedrock
// reports for the stream.
func handleNovaStreamRequest(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {
	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	awsResp, err := a.AwsClient.InvokeModelWithResponseStream(ctx, a.AwsReq.(*bedrockruntime.InvokeModelWithResponseStreamInput))
	if err != nil {
		return newAwsInvokeError(requestContext, err, "InvokeModelWithResponseStream"), nil
	}
	stream := awsResp.GetStream()
	defer stream.Close()
	info.StreamStatus = relaycommon.NewStreamStatus()
	info.StreamStatus.RequireTerminal()

	id := helper.GetResponseID(c)
	created := common.GetTimestamp()
	model := info.UpstreamModelName
	usage := &dto.Usage{}
	responseText := strings.Builder{}
	lastStreamData := ""

	events := stream.Events()
streamLoop:
	for {
		select {
		case <-ctx.Done():
			break streamLoop
		case event, ok := <-events:
			if !ok || ctx.Err() != nil {
				break streamLoop
			}
			var eventErr error
			var novaEvent novaStreamEvent
			if chunk, ok := event.(*bedrockruntimeTypes.ResponseStreamMemberChunk); !ok {
				eventErr = errors.New("unknown response type")
			} else if err := common.Unmarshal(chunk.Value.Bytes, &novaEvent); err != nil {
				eventErr = errors.Wrap(err, "unmarshal nova stream event")
			}
			if eventErr != nil {
				// Once the client has part of the answer, the stream ends as cut
				// short and what was delivered is billed; before that, the request
				// fails and can be retried.
				if info.SendResponseCount == 0 {
					return newAwsInvokeError(requestContext, eventErr, "InvokeModelWithResponseStream"), nil
				}
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, eventErr)
				break streamLoop
			}
			info.SetFirstResponseTime()

			if novaEvent.Metadata != nil {
				usage.PromptTokens = novaEvent.Metadata.Usage.InputTokens
				usage.CompletionTokens = novaEvent.Metadata.Usage.OutputTokens
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			} else if novaEvent.InvocationMetrics != nil && usage.TotalTokens == 0 {
				usage.PromptTokens = novaEvent.InvocationMetrics.InputTokenCount
				usage.CompletionTokens = novaEvent.InvocationMetrics.OutputTokenCount
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}

			var responses []*dto.ChatCompletionsStreamResponse
			switch {
			case novaEvent.ContentBlockDelta != nil && novaEvent.ContentBlockDelta.Delta.Text != "":
				responseText.WriteString(novaEvent.ContentBlockDelta.Delta.Text)
				response := &dto.ChatCompletionsStreamResponse{
					Id:      id,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   model,
					Choices: []dto.ChatCompletionsStreamResponseChoice{{}},
				}
				response.Choices[0].Delta.SetContentString(novaEvent.ContentBlockDelta.Delta.Text)
				// The role goes with the first chunk rather than on messageStart, so
				// an exception right after messageStart still fails the request
				// before anything reaches the client.
				if info.SendResponseCount == 0 {
					response.Choices[0].Delta.Role = "assistant"
				}
				responses = append(responses, response)
			case novaEvent.MessageStop != nil:
				info.StreamStatus.MarkCompleted()
				if info.SendResponseCount == 0 {
					responses = append(responses, helper.GenerateStartEmptyResponse(id, created, model, nil))
				}
				responses = append(responses, helper.GenerateStopResponse(id, created, model, novaFinishReason(novaEvent.MessageStop.StopReason)))
			}
			for _, response := range responses {
				data, err := common.Marshal(response)
				if err != nil {
					return types.NewError(errors.Wrap(err, "marshal nova stream chunk"), types.ErrorCodeJsonMarshalFailed), nil
				}
				lastStreamData = string(data)
				helper.SetEventStreamHeaders(c)
				if err := openai.HandleStreamFormat(c, info, lastStreamData, info.ChannelSetting.ForceFormat, info.ChannelSetting.ThinkingToContent); err != nil {
					break streamLoop
				}
			}
		}
	}

	// Nothing has reached the client yet, so the request fails cleanly, is
	// refunded and can be retried instead of answering an empty stream.
	if info.SendResponseCount == 0 && requestContext.Err() == nil {
		streamErr := stream.Err()
		if streamErr == nil {
			streamErr = ctx.Err()
		}
		if streamErr == nil {
			streamErr = errors.New("upstream stream ended before any content")
		}
		return newAwsInvokeError(requestContext, streamErr, "InvokeModelWithResponseStream"), nil
	}
	switch {
	case requestContext.Err() != nil:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, requestContext.Err())
	case ctx.Err() != nil:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, ctx.Err())
	case stream.Err() != nil:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, stream.Err())
	default:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
	}
	_ = stream.Close()

	if usage.TotalTokens == 0 {
		usage = service.ResponseText2Usage(c, responseText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	}
	helper.SetEventStreamHeaders(c)
	if helper.UpstreamStreamInterrupted(c, info) {
		helper.SendStreamInterrupted(c, info)
		return nil, usage
	}
	openai.HandleFinalResponse(c, info, lastStreamData, id, created, model, "", usage, false)
	return nil, usage
}
