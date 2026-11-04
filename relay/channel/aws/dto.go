package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type AwsClaudeRequest struct {
	// AnthropicVersion should be "bedrock-2023-05-31"
	AnthropicVersion  string              `json:"anthropic_version"`
	AnthropicBeta     json.RawMessage     `json:"anthropic_beta,omitempty"`
	System            any                 `json:"system,omitempty"`
	Messages          []dto.ClaudeMessage `json:"messages"`
	MaxTokens         *uint               `json:"max_tokens,omitempty"`
	Temperature       *float64            `json:"temperature,omitempty"`
	TopP              *float64            `json:"top_p,omitempty"`
	TopK              *int                `json:"top_k,omitempty"`
	StopSequences     []string            `json:"stop_sequences,omitempty"`
	Tools             any                 `json:"tools,omitempty"`
	ToolChoice        any                 `json:"tool_choice,omitempty"`
	Safeguards        json.RawMessage     `json:"safeguards,omitempty"`
	ContextManagement json.RawMessage     `json:"context_management,omitempty"`
	Thinking          *dto.Thinking       `json:"thinking,omitempty"`
	OutputConfig      json.RawMessage     `json:"output_config,omitempty"`
	//Metadata         json.RawMessage     `json:"metadata,omitempty"`
}

func formatRequest(requestBody io.Reader, requestHeader http.Header) (*AwsClaudeRequest, error) {
	var awsClaudeRequest AwsClaudeRequest
	err := common.DecodeJson(requestBody, &awsClaudeRequest)
	if err != nil {
		return nil, err
	}
	awsClaudeRequest.AnthropicVersion = "bedrock-2023-05-31"

	// check header anthropic-beta
	var betas []string
	for beta := range strings.SplitSeq(requestHeader.Get("anthropic-beta"), ",") {
		if beta = strings.TrimSpace(beta); beta != "" {
			betas = append(betas, beta)
		}
	}
	if len(betas) > 0 {
		betaJson, err := common.Marshal(betas)
		if err != nil {
			return nil, err
		}
		awsClaudeRequest.AnthropicBeta = betaJson
	}
	logger.LogJson(context.Background(), "json", awsClaudeRequest)
	return &awsClaudeRequest, nil
}

// NovaMessage Nova模型使用messages-v1格式
type NovaMessage struct {
	Role    string        `json:"role"`
	Content []NovaContent `json:"content"`
}

// NovaContent is one content block: text or an image.
type NovaContent struct {
	Text  *string    `json:"text,omitempty"`
	Image *NovaImage `json:"image,omitempty"`
}

type NovaImage struct {
	Format string          `json:"format"`
	Source NovaImageSource `json:"source"`
}

type NovaImageSource struct {
	// Bytes is base64 encoded, as the Invoke API takes it.
	Bytes string `json:"bytes"`
}

type NovaSystemContent struct {
	Text string `json:"text"`
}

type NovaRequest struct {
	SchemaVersion   string               `json:"schemaVersion"`             // 请求版本，例如 "1.0"
	System          []NovaSystemContent  `json:"system,omitempty"`          // 系统提示
	Messages        []NovaMessage        `json:"messages"`                  // 对话消息列表
	InferenceConfig *NovaInferenceConfig `json:"inferenceConfig,omitempty"` // 推理配置，可选
}

type NovaInferenceConfig struct {
	MaxTokens     *int     `json:"maxTokens,omitempty"`     // 最大生成的 token 数
	Temperature   *float64 `json:"temperature,omitempty"`   // 随机性 (默认 0.7, 范围 0.00001-1)
	TopP          *float64 `json:"topP,omitempty"`          // nucleus sampling (默认 0.9, 范围 0-1)
	TopK          *int     `json:"topK,omitempty"`          // 限制候选 token 数 (范围 0-128)
	StopSequences []string `json:"stopSequences,omitempty"` // 停止生成的序列
}

// novaMinTemperature is the lowest temperature Nova accepts; an explicit 0
// (greedy decoding) is sent as this rather than dropped.
const novaMinTemperature = 0.00001

func newNovaUnsupportedError(path string, message string) error {
	return &types.ConversionLossError{Diagnostics: []types.ConversionDiagnostic{{
		Code:     "aws_nova_unsupported",
		Path:     path,
		Message:  message,
		Severity: types.ConversionDiagnosticError,
		From:     types.RelayFormatOpenAI,
	}}}
}

// convertToNovaRequest converts a chat completion request to Nova's
// messages-v1 body: system and developer messages become the top-level
// system prompt (Nova's messages take only the user and assistant roles),
// consecutive turns of one role are joined, image parts become image blocks,
// and explicit zero sampling values are kept. Tool calls are refused rather
// than dropped.
func convertToNovaRequest(c *gin.Context, req *dto.GeneralOpenAIRequest) (*NovaRequest, error) {
	if len(req.Tools) > 0 {
		return nil, newNovaUnsupportedError("tools", "tool calling is not supported for Nova models on AWS Bedrock")
	}
	if len(req.Functions) > 0 && string(req.Functions) != "null" && string(req.Functions) != "[]" {
		return nil, newNovaUnsupportedError("functions", "tool calling is not supported for Nova models on AWS Bedrock")
	}

	novaReq := &NovaRequest{
		SchemaVersion: "messages-v1",
		Messages:      make([]NovaMessage, 0, len(req.Messages)),
	}
	for i, msg := range req.Messages {
		switch msg.Role {
		case "system", "developer":
			if text := msg.StringContent(); strings.TrimSpace(text) != "" {
				novaReq.System = append(novaReq.System, NovaSystemContent{Text: text})
			}
			continue
		case "user", "assistant":
		default:
			return nil, newNovaUnsupportedError(fmt.Sprintf("messages[%d].role", i), fmt.Sprintf("role %q is not supported for Nova models on AWS Bedrock", msg.Role))
		}
		if len(msg.ToolCalls) > 0 && string(msg.ToolCalls) != "null" && string(msg.ToolCalls) != "[]" {
			return nil, newNovaUnsupportedError(fmt.Sprintf("messages[%d].tool_calls", i), "tool calling is not supported for Nova models on AWS Bedrock")
		}

		var content []NovaContent
		if msg.IsStringContent() {
			if text := msg.StringContent(); strings.TrimSpace(text) != "" {
				content = append(content, NovaContent{Text: &text})
			}
		} else {
			for _, part := range msg.ParseContent() {
				switch part.Type {
				case dto.ContentTypeText:
					if strings.TrimSpace(part.Text) != "" {
						text := part.Text
						content = append(content, NovaContent{Text: &text})
					}
				case dto.ContentTypeImageURL:
					source := part.ToFileSource()
					if source == nil {
						continue
					}
					base64Data, mimeType, err := service.GetBase64Data(c, source, "formatting image for Nova")
					if err != nil {
						return nil, fmt.Errorf("get image for Nova failed: %w", err)
					}
					format, _ := strings.CutPrefix(strings.ToLower(mimeType), "image/")
					if format == "jpg" {
						format = "jpeg"
					}
					switch format {
					case "png", "jpeg", "gif", "webp":
					default:
						return nil, newNovaUnsupportedError(fmt.Sprintf("messages[%d].content", i), fmt.Sprintf("image type %q is not supported by Nova (png, jpeg, gif or webp)", mimeType))
					}
					content = append(content, NovaContent{Image: &NovaImage{Format: format, Source: NovaImageSource{Bytes: base64Data}}})
				default:
					return nil, newNovaUnsupportedError(fmt.Sprintf("messages[%d].content", i), fmt.Sprintf("content type %q is not supported for Nova models on AWS Bedrock", part.Type))
				}
			}
		}
		if len(content) == 0 {
			continue
		}
		if last := len(novaReq.Messages) - 1; last >= 0 && novaReq.Messages[last].Role == msg.Role {
			novaReq.Messages[last].Content = append(novaReq.Messages[last].Content, content...)
			continue
		}
		novaReq.Messages = append(novaReq.Messages, NovaMessage{Role: msg.Role, Content: content})
	}

	inference := &NovaInferenceConfig{
		TopP:          req.TopP,
		TopK:          req.TopK,
		StopSequences: parseStopSequences(req.Stop),
	}
	if req.Temperature != nil {
		inference.Temperature = common.GetPointer(max(*req.Temperature, novaMinTemperature))
	}
	if maxTokens := req.MaxCompletionTokens; maxTokens != nil && *maxTokens > 0 {
		inference.MaxTokens = common.GetPointer(int(min(*maxTokens, uint(math.MaxInt32))))
	} else if maxTokens := req.MaxTokens; maxTokens != nil && *maxTokens > 0 {
		inference.MaxTokens = common.GetPointer(int(min(*maxTokens, uint(math.MaxInt32))))
	}
	if inference.MaxTokens != nil || inference.Temperature != nil || inference.TopP != nil || inference.TopK != nil || len(inference.StopSequences) > 0 {
		novaReq.InferenceConfig = inference
	}
	return novaReq, nil
}

// parseStopSequences 解析停止序列，支持字符串或字符串数组
func parseStopSequences(stop any) []string {
	if stop == nil {
		return nil
	}

	switch v := stop.(type) {
	case string:
		if v != "" {
			return []string{v}
		}
	case []string:
		return v
	case []any:
		var sequences []string
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				sequences = append(sequences, str)
			}
		}
		return sequences
	}
	return nil
}
