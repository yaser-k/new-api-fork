package aws

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockruntimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// pegasusModelPrefix names TwelveLabs Pegasus, the Bedrock model that takes
// a video, after any inference profile prefix.
const pegasusModelPrefix = "twelvelabs.pegasus"

// pegasusMaxInlineVideoBytes is Bedrock's limit for a video sent inline as
// mediaSource.base64String; a larger one needs an S3 location.
const pegasusMaxInlineVideoBytes = 25 * 1024 * 1024

// Pegasus's range for maxOutputTokens.
const (
	pegasusMinOutputTokens = 512
	pegasusMaxOutputTokens = 98304
)

// awsLoadFileSource is the gateway's bounded file loader; tests replace it to
// serve a video URL without a download.
var awsLoadFileSource = service.LoadFileSource

// pegasusRequest is Pegasus's InvokeModel body. It is the body this adaptor
// builds, so channel parameter overrides apply to its fields.
type pegasusRequest struct {
	InputPrompt     string             `json:"inputPrompt"`
	MediaSource     pegasusMediaSource `json:"mediaSource"`
	MaxOutputTokens *int               `json:"maxOutputTokens,omitempty"`
	Temperature     *float64           `json:"temperature,omitempty"`
	IncludeMetering bool               `json:"includeMetering"`
}

type pegasusMediaSource struct {
	Base64String string `json:"base64String"`
}

// pegasusResponse is a Pegasus InvokeModel answer, and with Message as a
// delta, one chunk of its stream; metering and invocation metrics come on the
// last chunk only.
type pegasusResponse struct {
	Message           string  `json:"message"`
	StopReason        string  `json:"stopReason"`
	InputTokens       int     `json:"input_tokens"`
	OutputTokens      int     `json:"output_tokens"`
	OutputTokenCount  int     `json:"outputTokenCount"`
	InputVideoSecond  float64 `json:"inputVideoSecond"`
	InvocationMetrics *struct {
		OutputTokenCount int `json:"outputTokenCount"`
	} `json:"amazon-bedrock-invocationMetrics"`
}

func isPegasusModel(modelId string) bool {
	_, unprefixed, _ := strings.Cut(modelId, ".")
	return strings.HasPrefix(modelId, pegasusModelPrefix) || strings.HasPrefix(unprefixed, pegasusModelPrefix)
}

// awsContentParts returns a message's content parts as the client sent them.
// ParseContent keeps a video_url part only in its string form and drops the
// {"url": ...} form OpenAI-style clients send, so the parts are read here,
// with VideoUrl left as sent.
func awsContentParts(message dto.Message) []dto.MediaContent {
	items, ok := message.Content.([]any)
	if !ok {
		return message.ParseContent()
	}
	parts := make([]dto.MediaContent, 0, len(items))
	for _, item := range items {
		switch item := item.(type) {
		case dto.MediaContent:
			parts = append(parts, item)
		case map[string]any:
			partType, _ := item["type"].(string)
			text, _ := item["text"].(string)
			parts = append(parts, dto.MediaContent{Type: partType, Text: text, VideoUrl: item["video_url"]})
		}
	}
	return parts
}

func newAwsVideoUnsupportedError(path string, message string) error {
	return &types.ConversionLossError{Diagnostics: []types.ConversionDiagnostic{{
		Code:     "aws_video_unsupported",
		Path:     path,
		Message:  message,
		Severity: types.ConversionDiagnosticError,
		From:     types.RelayFormatOpenAI,
	}}}
}

// convertToPegasusRequest converts a chat completion request with one video
// part to Pegasus's body. Pegasus takes a single prompt, so the system and
// developer messages lead it, then the user's text; when more than one turn
// has text, each user and assistant turn follows as a "User:" or "Assistant:"
// line, in order. The
// video, from a data: URL or an http(s) URL fetched by the bounded loader, is
// sent inline; tools, other media and a second video are refused.
func convertToPegasusRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (*pegasusRequest, error) {
	if info.ChannelOtherSettings.AwsApiType == dto.AwsApiTypeMantle {
		return nil, newAwsVideoUnsupportedError("model", "TwelveLabs Pegasus is not available through the AWS Mantle API; use the InvokeModel API type")
	}
	if len(request.Tools) > 0 || (len(request.Functions) > 0 && string(request.Functions) != "null" && string(request.Functions) != "[]") {
		return nil, newAwsVideoUnsupportedError("tools", "tool calling is not supported for TwelveLabs Pegasus on AWS Bedrock")
	}

	var system []string
	var turns []string
	var video *dto.MediaContent
	videoPath := ""
	for i, message := range request.Messages {
		var texts []string
		for j, part := range awsContentParts(message) {
			path := fmt.Sprintf("messages[%d].content[%d]", i, j)
			switch part.Type {
			case dto.ContentTypeText:
				if strings.TrimSpace(part.Text) != "" {
					texts = append(texts, part.Text)
				}
			case dto.ContentTypeVideoUrl:
				if message.Role != "user" {
					return nil, newAwsVideoUnsupportedError(path, "a video must be in a user message")
				}
				if video != nil {
					return nil, newAwsVideoUnsupportedError(path, fmt.Sprintf("TwelveLabs Pegasus takes one video per request; %s already carries one", videoPath))
				}
				video, videoPath = &part, path
			default:
				return nil, newAwsVideoUnsupportedError(path, fmt.Sprintf("content type %q is not supported for TwelveLabs Pegasus on AWS Bedrock", part.Type))
			}
		}
		text := strings.Join(texts, "\n")
		switch message.Role {
		case "system", "developer":
			if text != "" {
				system = append(system, text)
			}
		case "user":
			if text != "" {
				turns = append(turns, "User: "+text)
			}
		case "assistant":
			if len(message.ToolCalls) > 0 && string(message.ToolCalls) != "null" && string(message.ToolCalls) != "[]" {
				return nil, newAwsVideoUnsupportedError(fmt.Sprintf("messages[%d].tool_calls", i), "tool calling is not supported for TwelveLabs Pegasus on AWS Bedrock")
			}
			if text != "" {
				turns = append(turns, "Assistant: "+text)
			}
		default:
			return nil, newAwsVideoUnsupportedError(fmt.Sprintf("messages[%d].role", i), fmt.Sprintf("role %q is not supported for TwelveLabs Pegasus on AWS Bedrock", message.Role))
		}
	}
	if video == nil {
		return nil, newAwsVideoUnsupportedError("messages", "TwelveLabs Pegasus needs one video_url part in a user message")
	}
	if len(turns) == 1 {
		turns[0] = strings.TrimPrefix(turns[0], "User: ")
	}

	videoUrl, _ := video.VideoUrl.(string)
	if parsed := video.GetVideoUrl(); parsed != nil {
		videoUrl = parsed.Url
	}
	if !strings.HasPrefix(videoUrl, "data:") && !strings.HasPrefix(videoUrl, "http://") && !strings.HasPrefix(videoUrl, "https://") {
		return nil, newAwsVideoUnsupportedError(videoPath, "video_url must be a data: URL or an http(s) URL")
	}
	videoData, err := awsLoadFileSource(c, types.NewFileSourceFromData(videoUrl, ""), "formatting video for TwelveLabs Pegasus")
	if err != nil {
		return nil, fmt.Errorf("get video for TwelveLabs Pegasus failed: %w", err)
	}
	if videoData.Size > pegasusMaxInlineVideoBytes {
		return nil, newAwsVideoUnsupportedError(videoPath, fmt.Sprintf("the video is %d bytes; TwelveLabs Pegasus on AWS Bedrock takes at most %d bytes inline", videoData.Size, pegasusMaxInlineVideoBytes))
	}
	base64Video, err := videoData.GetBase64Data()
	if err != nil {
		return nil, fmt.Errorf("get video for TwelveLabs Pegasus failed: %w", err)
	}

	pegasusReq := &pegasusRequest{
		InputPrompt:     strings.Join(append(system, turns...), "\n\n"),
		MediaSource:     pegasusMediaSource{Base64String: base64Video},
		IncludeMetering: true,
	}
	maxTokens := request.MaxCompletionTokens
	if maxTokens == nil || *maxTokens == 0 {
		maxTokens = request.MaxTokens
	}
	if maxTokens != nil && *maxTokens > 0 {
		pegasusReq.MaxOutputTokens = common.GetPointer(int(min(max(*maxTokens, pegasusMinOutputTokens), pegasusMaxOutputTokens)))
	}
	if request.Temperature != nil {
		pegasusReq.Temperature = common.GetPointer(min(max(*request.Temperature, 0), 1))
	}
	return pegasusReq, nil
}

// newPegasusInput builds the InvokeModel input, or the stream input, from the
// Pegasus body after parameter overrides, with includeMetering always on: the
// answer then reports its output tokens and input video seconds.
func newPegasusInput(modelId string, requestBody io.Reader, isStream bool) (any, error) {
	var body map[string]json.RawMessage
	if err := common.DecodeJson(requestBody, &body); err != nil {
		return nil, types.NewErrorWithStatusCode(errors.Wrap(err, "decode pegasus request fail"), types.ErrorCodeBadRequestBody, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	body["includeMetering"] = json.RawMessage("true")
	payload, err := common.Marshal(body)
	if err != nil {
		return nil, types.NewError(errors.Wrap(err, "marshal pegasus request"), types.ErrorCodeJsonMarshalFailed)
	}
	if isStream {
		return &bedrockruntime.InvokeModelWithResponseStreamInput{
			ModelId:     aws.String(modelId),
			Accept:      aws.String("application/json"),
			ContentType: aws.String("application/json"),
			Body:        payload,
		}, nil
	}
	return &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelId),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
		Body:        payload,
	}, nil
}

// pegasusUsage builds the usage of a Pegasus answer from the counts Bedrock
// reported, falling back to the local estimate for a count it did not report
// (a stream reports no input count), and carries its input video seconds.
func pegasusUsage(c *gin.Context, info *relaycommon.RelayInfo, inputTokens int, outputTokens int, responseText string, videoSeconds float64) *dto.Usage {
	usage := &dto.Usage{PromptTokens: inputTokens, CompletionTokens: outputTokens}
	if inputTokens <= 0 || outputTokens <= 0 {
		estimate := service.ResponseText2Usage(c, responseText, info.UpstreamModelName, info.GetEstimatePromptTokens())
		if inputTokens <= 0 {
			usage.PromptTokens = estimate.PromptTokens
		}
		if outputTokens <= 0 {
			usage.CompletionTokens = estimate.CompletionTokens
		}
	}
	usage.PromptTokens = min(usage.PromptTokens, math.MaxInt32)
	usage.CompletionTokens = min(usage.CompletionTokens, math.MaxInt32)
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	if videoSeconds > 0 && videoSeconds <= math.MaxInt32 {
		usage.InputVideoSeconds = videoSeconds
	}
	return usage
}

func pegasusFinishReason(stopReason string) string {
	if stopReason == "length" {
		return types.FinishReasonLength
	}
	return types.FinishReasonStop
}

// pegasusHandler relays a Pegasus InvokeModel answer as a chat completion.
func pegasusHandler(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {
	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	awsResp, err := a.AwsClient.InvokeModel(ctx, a.AwsReq.(*bedrockruntime.InvokeModelInput))
	if err != nil {
		return newAwsInvokeError(requestContext, err, "InvokeModel"), nil
	}
	var pegasusResp pegasusResponse
	if err := common.Unmarshal(awsResp.Body, &pegasusResp); err != nil {
		return types.NewOpenAIError(errors.Wrap(err, "unmarshal pegasus response"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError), nil
	}

	outputTokens := pegasusResp.OutputTokens
	if outputTokens <= 0 {
		outputTokens = pegasusResp.OutputTokenCount
	}
	usage := pegasusUsage(c, info, pegasusResp.InputTokens, outputTokens, pegasusResp.Message, pegasusResp.InputVideoSecond)
	response := dto.OpenAITextResponse{
		Id:      helper.GetResponseID(c),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   info.UpstreamModelName,
		Choices: []dto.OpenAITextResponseChoice{{
			Index:        0,
			Message:      dto.Message{Role: "assistant", Content: pegasusResp.Message},
			FinishReason: pegasusFinishReason(pegasusResp.StopReason),
		}},
		Usage: *usage,
	}
	responseBody, err := common.Marshal(response)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError), nil
	}
	c.Data(http.StatusOK, "application/json", responseBody)
	return nil, usage
}

// pegasusStreamHandler relays a Pegasus event stream as chat completion
// chunks: each text delta, the finish reason, then the usage from the last
// chunk's metering.
func pegasusStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {
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
	responseText := strings.Builder{}
	lastStreamData := ""
	outputTokens := 0
	videoSeconds := 0.0

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
			var chunk pegasusResponse
			if member, ok := event.(*bedrockruntimeTypes.ResponseStreamMemberChunk); !ok {
				eventErr = errors.New("unknown response type")
			} else if err := common.Unmarshal(member.Value.Bytes, &chunk); err != nil {
				eventErr = errors.Wrap(err, "unmarshal pegasus stream chunk")
			}
			if eventErr != nil {
				// Before anything reaches the client the request fails and can be
				// retried; after, the stream ends cut short and what was sent bills.
				if info.SendResponseCount == 0 {
					return newAwsInvokeError(requestContext, eventErr, "InvokeModelWithResponseStream"), nil
				}
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, eventErr)
				break streamLoop
			}
			info.SetFirstResponseTime()

			if chunk.OutputTokenCount > 0 {
				outputTokens = chunk.OutputTokenCount
			} else if chunk.InvocationMetrics != nil && chunk.InvocationMetrics.OutputTokenCount > 0 && outputTokens == 0 {
				outputTokens = chunk.InvocationMetrics.OutputTokenCount
			}
			if chunk.InputVideoSecond > 0 {
				videoSeconds = chunk.InputVideoSecond
			}

			var responses []*dto.ChatCompletionsStreamResponse
			if chunk.Message != "" {
				responseText.WriteString(chunk.Message)
				response := &dto.ChatCompletionsStreamResponse{
					Id:      id,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   model,
					Choices: []dto.ChatCompletionsStreamResponseChoice{{}},
				}
				response.Choices[0].Delta.SetContentString(chunk.Message)
				if info.SendResponseCount == 0 {
					response.Choices[0].Delta.Role = "assistant"
				}
				responses = append(responses, response)
			}
			if chunk.StopReason != "" {
				info.StreamStatus.MarkCompleted()
				if info.SendResponseCount == 0 && len(responses) == 0 {
					responses = append(responses, helper.GenerateStartEmptyResponse(id, created, model, nil))
				}
				responses = append(responses, helper.GenerateStopResponse(id, created, model, pegasusFinishReason(chunk.StopReason)))
			}
			for _, response := range responses {
				data, err := common.Marshal(response)
				if err != nil {
					return types.NewError(errors.Wrap(err, "marshal pegasus stream chunk"), types.ErrorCodeJsonMarshalFailed), nil
				}
				lastStreamData = string(data)
				helper.SetEventStreamHeaders(c)
				if err := openai.HandleStreamFormat(c, info, lastStreamData, info.ChannelSetting.ForceFormat, info.ChannelSetting.ThinkingToContent); err != nil {
					break streamLoop
				}
			}
		}
	}

	// Nothing has reached the client, so the request fails cleanly and can be
	// retried instead of answering an empty stream.
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

	usage := pegasusUsage(c, info, 0, outputTokens, responseText.String(), videoSeconds)
	helper.SetEventStreamHeaders(c)
	openai.HandleFinalResponse(c, info, lastStreamData, id, created, model, "", usage, false)
	return nil, usage
}
