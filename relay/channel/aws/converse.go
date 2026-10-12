package aws

import (
	"encoding/base64"
	"fmt"
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

// Bedrock model ids may carry an inference profile prefix before the provider,
// as in eu.mistral.pixtral-large-2502-v1:0 or global.anthropic.claude-sonnet-4-6.
var awsInferenceProfilePrefixes = map[string]bool{
	"us": true, "us-gov": true, "eu": true, "apac": true, "jp": true, "au": true, "ca": true, "global": true,
}

// isConverseModel reports whether a Bedrock model is served through the
// Converse API. InvokeModel takes each provider's native body, and this
// adaptor only builds Anthropic's and Nova's, so the models of every other
// provider (Mistral, Meta, DeepSeek, Qwen and others) go through Converse,
// which takes one body for all of them. An id this cannot read a provider from,
// such as an inference profile ARN, keeps the Anthropic path.
func isConverseModel(modelId string) bool {
	if isNovaModel(modelId) {
		return false
	}
	id := modelId
	if prefix, rest, ok := strings.Cut(id, "."); ok && awsInferenceProfilePrefixes[prefix] {
		id = rest
	}
	provider, _, ok := strings.Cut(id, ".")
	if !ok || provider == "" || provider == "anthropic" {
		return false
	}
	for _, r := range provider {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// converseRequest is a Converse request in the API's JSON shape. It is the body
// this adaptor builds, so channel parameter overrides apply to Converse fields.
type converseRequest struct {
	System          []converseText           `json:"system,omitempty"`
	Messages        []converseMessage        `json:"messages"`
	InferenceConfig *converseInferenceConfig `json:"inferenceConfig,omitempty"`
}

type converseText struct {
	Text string `json:"text"`
}

type converseMessage struct {
	Role    string                 `json:"role"`
	Content []converseContentBlock `json:"content"`
}

type converseContentBlock struct {
	Text  *string        `json:"text,omitempty"`
	Image *converseImage `json:"image,omitempty"`
}

type converseImage struct {
	Format string              `json:"format"`
	Source converseImageSource `json:"source"`
}

type converseImageSource struct {
	// Bytes is base64 encoded, as in the API's JSON form.
	Bytes string `json:"bytes"`
}

type converseInferenceConfig struct {
	MaxTokens     *int     `json:"maxTokens,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	TopP          *float64 `json:"topP,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

func newConverseUnsupportedError(path string, message string) error {
	return &types.ConversionLossError{Diagnostics: []types.ConversionDiagnostic{{
		Code:     "aws_converse_unsupported",
		Path:     path,
		Message:  message,
		Severity: types.ConversionDiagnosticError,
		From:     types.RelayFormatOpenAI,
	}}}
}

// convertToConverseRequest converts a chat completion request to a Converse
// request: system and developer messages become the system prompt, user and
// assistant turns become messages (consecutive turns of one role are joined,
// as Converse requires the roles to alternate), and image parts become image
// blocks. Tool calls are refused rather than dropped.
func convertToConverseRequest(c *gin.Context, request *dto.GeneralOpenAIRequest) (*converseRequest, error) {
	if len(request.Tools) > 0 {
		return nil, newConverseUnsupportedError("tools", "tool calling is not supported for this model on AWS Bedrock")
	}
	if len(request.Functions) > 0 && string(request.Functions) != "null" && string(request.Functions) != "[]" {
		return nil, newConverseUnsupportedError("functions", "tool calling is not supported for this model on AWS Bedrock")
	}
	converseReq := &converseRequest{Messages: make([]converseMessage, 0, len(request.Messages))}
	for i, message := range request.Messages {
		var role string
		switch message.Role {
		case "system", "developer":
			if text := message.StringContent(); strings.TrimSpace(text) != "" {
				converseReq.System = append(converseReq.System, converseText{Text: text})
			}
			continue
		case "user":
			role = string(bedrockruntimeTypes.ConversationRoleUser)
		case "assistant":
			role = string(bedrockruntimeTypes.ConversationRoleAssistant)
		default:
			return nil, newConverseUnsupportedError(fmt.Sprintf("messages[%d].role", i), fmt.Sprintf("role %q is not supported for this model on AWS Bedrock", message.Role))
		}
		if len(message.ToolCalls) > 0 && string(message.ToolCalls) != "null" && string(message.ToolCalls) != "[]" {
			return nil, newConverseUnsupportedError(fmt.Sprintf("messages[%d].tool_calls", i), "tool calling is not supported for this model on AWS Bedrock")
		}

		var blocks []converseContentBlock
		if message.IsStringContent() {
			if text := message.StringContent(); strings.TrimSpace(text) != "" {
				blocks = append(blocks, converseContentBlock{Text: &text})
			}
		} else {
			for _, part := range message.ParseContent() {
				switch part.Type {
				case dto.ContentTypeText:
					if strings.TrimSpace(part.Text) != "" {
						text := part.Text
						blocks = append(blocks, converseContentBlock{Text: &text})
					}
				case dto.ContentTypeImageURL:
					source := part.ToFileSource()
					if source == nil {
						continue
					}
					base64Data, mimeType, err := service.GetBase64Data(c, source, "formatting image for Bedrock Converse")
					if err != nil {
						return nil, fmt.Errorf("get image for Bedrock Converse failed: %w", err)
					}
					format, _ := strings.CutPrefix(strings.ToLower(mimeType), "image/")
					if format == "jpg" {
						format = "jpeg"
					}
					switch bedrockruntimeTypes.ImageFormat(format) {
					case bedrockruntimeTypes.ImageFormatPng, bedrockruntimeTypes.ImageFormatJpeg, bedrockruntimeTypes.ImageFormatGif, bedrockruntimeTypes.ImageFormatWebp:
					default:
						return nil, newConverseUnsupportedError(fmt.Sprintf("messages[%d].content", i), fmt.Sprintf("image type %q is not supported by AWS Bedrock (png, jpeg, gif or webp)", mimeType))
					}
					blocks = append(blocks, converseContentBlock{Image: &converseImage{Format: format, Source: converseImageSource{Bytes: base64Data}}})
				default:
					return nil, newConverseUnsupportedError(fmt.Sprintf("messages[%d].content", i), fmt.Sprintf("content type %q is not supported for this model on AWS Bedrock", part.Type))
				}
			}
		}
		if len(blocks) == 0 {
			continue
		}
		if last := len(converseReq.Messages) - 1; last >= 0 && converseReq.Messages[last].Role == role {
			converseReq.Messages[last].Content = append(converseReq.Messages[last].Content, blocks...)
			continue
		}
		converseReq.Messages = append(converseReq.Messages, converseMessage{Role: role, Content: blocks})
	}

	inference := &converseInferenceConfig{
		Temperature:   request.Temperature,
		TopP:          request.TopP,
		StopSequences: parseStopSequences(request.Stop),
	}
	if maxTokens := request.MaxCompletionTokens; maxTokens != nil && *maxTokens > 0 {
		inference.MaxTokens = common.GetPointer(int(min(*maxTokens, uint(math.MaxInt32))))
	} else if maxTokens := request.MaxTokens; maxTokens != nil && *maxTokens > 0 {
		inference.MaxTokens = common.GetPointer(int(min(*maxTokens, uint(math.MaxInt32))))
	}
	if inference.MaxTokens != nil || inference.Temperature != nil || inference.TopP != nil || len(inference.StopSequences) > 0 {
		converseReq.InferenceConfig = inference
	}
	return converseReq, nil
}

// newConverseInput builds the SDK's Converse or ConverseStream input from the
// request body.
func newConverseInput(modelId string, converseReq *converseRequest, isStream bool) (any, error) {
	system := make([]bedrockruntimeTypes.SystemContentBlock, 0, len(converseReq.System))
	for _, block := range converseReq.System {
		system = append(system, &bedrockruntimeTypes.SystemContentBlockMemberText{Value: block.Text})
	}
	messages := make([]bedrockruntimeTypes.Message, 0, len(converseReq.Messages))
	for _, message := range converseReq.Messages {
		content := make([]bedrockruntimeTypes.ContentBlock, 0, len(message.Content))
		for _, block := range message.Content {
			switch {
			case block.Text != nil:
				content = append(content, &bedrockruntimeTypes.ContentBlockMemberText{Value: *block.Text})
			case block.Image != nil:
				data, err := base64.StdEncoding.DecodeString(block.Image.Source.Bytes)
				if err != nil {
					return nil, errors.Wrap(err, "decode converse image")
				}
				content = append(content, &bedrockruntimeTypes.ContentBlockMemberImage{Value: bedrockruntimeTypes.ImageBlock{
					Format: bedrockruntimeTypes.ImageFormat(block.Image.Format),
					Source: &bedrockruntimeTypes.ImageSourceMemberBytes{Value: data},
				}})
			}
		}
		messages = append(messages, bedrockruntimeTypes.Message{Role: bedrockruntimeTypes.ConversationRole(message.Role), Content: content})
	}

	var inference *bedrockruntimeTypes.InferenceConfiguration
	if config := converseReq.InferenceConfig; config != nil {
		inference = &bedrockruntimeTypes.InferenceConfiguration{StopSequences: config.StopSequences}
		if config.MaxTokens != nil {
			inference.MaxTokens = aws.Int32(int32(min(max(*config.MaxTokens, 0), math.MaxInt32)))
		}
		if config.Temperature != nil {
			inference.Temperature = aws.Float32(float32(*config.Temperature))
		}
		if config.TopP != nil {
			inference.TopP = aws.Float32(float32(*config.TopP))
		}
	}

	if isStream {
		return &bedrockruntime.ConverseStreamInput{
			ModelId:         aws.String(modelId),
			System:          system,
			Messages:        messages,
			InferenceConfig: inference,
		}, nil
	}
	return &bedrockruntime.ConverseInput{
		ModelId:         aws.String(modelId),
		System:          system,
		Messages:        messages,
		InferenceConfig: inference,
	}, nil
}

// converseFinishReason maps a Converse stop reason to an OpenAI finish reason.
func converseFinishReason(stopReason bedrockruntimeTypes.StopReason) string {
	switch stopReason {
	case bedrockruntimeTypes.StopReasonEndTurn, bedrockruntimeTypes.StopReasonStopSequence:
		return types.FinishReasonStop
	case bedrockruntimeTypes.StopReasonMaxTokens, bedrockruntimeTypes.StopReasonModelContextWindowExceeded:
		return types.FinishReasonLength
	case bedrockruntimeTypes.StopReasonToolUse:
		return types.FinishReasonToolCalls
	case bedrockruntimeTypes.StopReasonContentFiltered, bedrockruntimeTypes.StopReasonGuardrailIntervened:
		return types.FinishReasonContentFilter
	}
	return string(stopReason)
}

// converseUsage converts Converse token usage. Bedrock counts cache reads and
// writes apart from the input tokens; OpenAI's prompt tokens include them.
func converseUsage(tokenUsage *bedrockruntimeTypes.TokenUsage) *dto.Usage {
	usage := &dto.Usage{}
	if tokenUsage == nil {
		return usage
	}
	cacheRead := int(aws.ToInt32(tokenUsage.CacheReadInputTokens))
	cacheWrite := int(aws.ToInt32(tokenUsage.CacheWriteInputTokens))
	usage.PromptTokens = int(aws.ToInt32(tokenUsage.InputTokens)) + cacheRead + cacheWrite
	usage.CompletionTokens = int(aws.ToInt32(tokenUsage.OutputTokens))
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	usage.PromptTokensDetails.CachedTokens = cacheRead
	usage.PromptTokensDetails.CachedCreationTokens = cacheWrite
	return usage
}

func converseHandler(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {
	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	output, err := a.AwsClient.Converse(ctx, a.AwsReq.(*bedrockruntime.ConverseInput))
	if err != nil {
		return newAwsInvokeError(requestContext, err, "Converse"), nil
	}

	text := strings.Builder{}
	reasoning := strings.Builder{}
	if message, ok := output.Output.(*bedrockruntimeTypes.ConverseOutputMemberMessage); ok {
		for _, block := range message.Value.Content {
			switch block := block.(type) {
			case *bedrockruntimeTypes.ContentBlockMemberText:
				text.WriteString(block.Value)
			case *bedrockruntimeTypes.ContentBlockMemberReasoningContent:
				if reasoningText, ok := block.Value.(*bedrockruntimeTypes.ReasoningContentBlockMemberReasoningText); ok {
					reasoning.WriteString(aws.ToString(reasoningText.Value.Text))
				}
			}
		}
	}
	usage := converseUsage(output.Usage)
	if usage.TotalTokens == 0 {
		usage = service.ResponseText2Usage(c, reasoning.String()+text.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	}
	message := dto.Message{Role: "assistant", Content: text.String()}
	if reasoning.Len() > 0 {
		message.ReasoningContent = common.GetPointer(reasoning.String())
	}

	response := dto.OpenAITextResponse{
		Id:      helper.GetResponseID(c),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   info.UpstreamModelName,
		Choices: []dto.OpenAITextResponseChoice{{
			Index:        0,
			Message:      message,
			FinishReason: converseFinishReason(output.StopReason),
		}},
		Usage: *usage,
	}
	c.JSON(http.StatusOK, response)
	return nil, usage
}

// converseStreamHandler relays a ConverseStream answer as OpenAI chat
// completion chunks: each text or reasoning delta (the first one with the
// role), the finish reason, then the usage of the metadata event.
func converseStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {
	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	output, err := a.AwsClient.ConverseStream(ctx, a.AwsReq.(*bedrockruntime.ConverseStreamInput))
	if err != nil {
		return newAwsInvokeError(requestContext, err, "ConverseStream"), nil
	}
	stream := output.GetStream()
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
			info.SetFirstResponseTime()

			var responses []*dto.ChatCompletionsStreamResponse
			switch v := event.(type) {
			case *bedrockruntimeTypes.ConverseStreamOutputMemberContentBlockDelta:
				response := &dto.ChatCompletionsStreamResponse{
					Id:      id,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   model,
					Choices: []dto.ChatCompletionsStreamResponseChoice{{}},
				}
				switch delta := v.Value.Delta.(type) {
				case *bedrockruntimeTypes.ContentBlockDeltaMemberText:
					if delta.Value == "" {
						continue
					}
					responseText.WriteString(delta.Value)
					response.Choices[0].Delta.SetContentString(delta.Value)
				case *bedrockruntimeTypes.ContentBlockDeltaMemberReasoningContent:
					reasoning, ok := delta.Value.(*bedrockruntimeTypes.ReasoningContentBlockDeltaMemberText)
					if !ok || reasoning.Value == "" {
						continue
					}
					responseText.WriteString(reasoning.Value)
					response.Choices[0].Delta.SetReasoningContent(reasoning.Value)
				default:
					continue
				}
				// The role goes with the first chunk rather than on messageStart, so
				// an exception right after messageStart still fails the request
				// before anything reaches the client.
				if info.SendResponseCount == 0 {
					response.Choices[0].Delta.Role = "assistant"
				}
				responses = append(responses, response)
			case *bedrockruntimeTypes.ConverseStreamOutputMemberMessageStop:
				info.StreamStatus.MarkCompleted()
				if info.SendResponseCount == 0 {
					responses = append(responses, helper.GenerateStartEmptyResponse(id, created, model, nil))
				}
				responses = append(responses, helper.GenerateStopResponse(id, created, model, converseFinishReason(v.Value.StopReason)))
			case *bedrockruntimeTypes.ConverseStreamOutputMemberMetadata:
				usage = converseUsage(v.Value.Usage)
			}
			for _, response := range responses {
				data, err := common.Marshal(response)
				if err != nil {
					return types.NewError(errors.Wrap(err, "marshal converse stream chunk"), types.ErrorCodeJsonMarshalFailed), nil
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
		return newAwsInvokeError(requestContext, streamErr, "ConverseStream"), nil
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
	openai.HandleFinalResponse(c, info, lastStreamData, id, created, model, "", usage, false)
	return nil, usage
}
