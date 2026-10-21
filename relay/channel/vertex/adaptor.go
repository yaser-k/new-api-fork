package vertex

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/gemini"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const (
	RequestModeClaude     = 1
	RequestModeGemini     = 2
	RequestModeOpenSource = 3
	RequestModeMistral    = 4
)

// Mistral's chat models on Vertex AI take an OpenAI chat body through rawPredict on the
// mistralai publisher. Mistral's OCR model takes a document body and is not one of them.
var mistralChatModelPrefixes = []string{"mistral-", "codestral-"}

// mistralChatModelID reports whether an upstream model name is one of Mistral's chat models
// and returns its bare id, without a "mistralai/" publisher prefix.
func mistralChatModelID(upstreamModelName string) (string, bool) {
	id := strings.TrimPrefix(upstreamModelName, PublisherMistral+"/")
	if strings.HasPrefix(id, "mistral-ocr") {
		return "", false
	}
	for _, prefix := range mistralChatModelPrefixes {
		if strings.HasPrefix(id, prefix) {
			return id, true
		}
	}
	return "", false
}

var claudeModelMap = map[string]string{
	"claude-3-sonnet-20240229":   "claude-3-sonnet@20240229",
	"claude-3-opus-20240229":     "claude-3-opus@20240229",
	"claude-3-haiku-20240307":    "claude-3-haiku@20240307",
	"claude-3-5-sonnet-20240620": "claude-3-5-sonnet@20240620",
	"claude-3-5-sonnet-20241022": "claude-3-5-sonnet-v2@20241022",
	"claude-3-7-sonnet-20250219": "claude-3-7-sonnet@20250219",
	"claude-sonnet-4-20250514":   "claude-sonnet-4@20250514",
	"claude-opus-4-20250514":     "claude-opus-4@20250514",
	"claude-opus-4-1-20250805":   "claude-opus-4-1@20250805",
	"claude-sonnet-4-5-20250929": "claude-sonnet-4-5@20250929",
	"claude-haiku-4-5-20251001":  "claude-haiku-4-5@20251001",
	"claude-opus-4-5-20251101":   "claude-opus-4-5@20251101",
	"claude-opus-4-6":            "claude-opus-4-6",
	"claude-opus-4-7":            "claude-opus-4-7",
	"claude-opus-4-8":            "claude-opus-4-8",
}

const anthropicVersion = "vertex-2023-10-16"

type Adaptor struct {
	// transcription is set by ConvertAudioRequest for a Gemini transcription model; its answer follows it.
	transcription *gemini.TranscriptionRequest

	RequestMode        int
	AccountCredentials Credentials
	// embeddingEncodingFormat is the encoding_format of an embeddings request, applied to its answer.
	embeddingEncodingFormat string
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	// Vertex AI's generateContent schema does not expose the Gemini API's
	// function-call identity fields. Strip both sides at this provider boundary.
	if model_setting.GetGeminiSettings().RemoveFunctionResponseIdEnabled {
		removeFunctionCallIDs(request)
	}
	geminiAdaptor := gemini.Adaptor{}
	return geminiAdaptor.ConvertGeminiRequest(c, info, request)
}

func removeFunctionCallIDs(request *dto.GeminiChatRequest) {
	if request == nil {
		return
	}

	if len(request.Contents) > 0 {
		for i := range request.Contents {
			if len(request.Contents[i].Parts) == 0 {
				continue
			}
			for j := range request.Contents[i].Parts {
				part := &request.Contents[i].Parts[j]
				if part.FunctionCall != nil {
					part.FunctionCall.ID = ""
				}
				if part.FunctionResponse != nil && len(part.FunctionResponse.ID) > 0 {
					part.FunctionResponse.ID = nil
				}
			}
		}
	}

	if len(request.Requests) > 0 {
		for i := range request.Requests {
			removeFunctionCallIDs(&request.Requests[i])
		}
	}
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	// The URL follows the model (Init), so the body must too: only Claude models take
	// Anthropic's rawPredict body.
	switch a.RequestMode {
	case RequestModeGemini:
		result, err := service.ConvertRequest(c, info, types.RelayFormatGemini, request)
		if err != nil {
			return nil, err
		}
		geminiRequest, ok := result.Value.(*dto.GeminiChatRequest)
		if !ok {
			return nil, fmt.Errorf("expected Gemini generateContent request, got %T", result.Value)
		}
		if model_setting.GetGeminiSettings().RemoveFunctionResponseIdEnabled {
			removeFunctionCallIDs(geminiRequest)
		}
		c.Set("request_model", request.Model)
		return geminiRequest, nil
	case RequestModeOpenSource:
		result, err := service.ConvertRequest(c, info, types.RelayFormatOpenAI, request)
		if err != nil {
			return nil, err
		}
		openAIRequest, ok := result.Value.(*dto.GeneralOpenAIRequest)
		if !ok {
			return nil, fmt.Errorf("expected OpenAI chat completions request, got %T", result.Value)
		}
		return openAIRequest, nil
	}
	claudeAdaptor := claude.Adaptor{}
	if _, err := claudeAdaptor.ConvertClaudeRequest(c, info, request); err != nil {
		return nil, err
	}
	if v, ok := claudeModelMap[info.UpstreamModelName]; ok {
		c.Set("request_model", v)
	} else {
		c.Set("request_model", request.Model)
	}
	vertexClaudeReq := copyRequest(request, anthropicVersion)
	return vertexClaudeReq, nil
}

// isGeminiTranscriptionModel reports whether a model is one of Gemini's transcription models, which
// transcribe audio through generateContent.
func isGeminiTranscriptionModel(upstreamModelName string) bool {
	return strings.HasPrefix(upstreamModelName, "gemini-") && strings.Contains(upstreamModelName, "transcribe")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	if info.RelayMode != constant.RelayModeAudioTranscription || a.RequestMode != RequestModeGemini || !isGeminiTranscriptionModel(info.UpstreamModelName) {
		return nil, errors.New("not implemented")
	}
	transcription, body, err := gemini.ConvertTranscriptionRequest(c, info)
	if err != nil {
		return nil, err
	}
	a.transcription = transcription
	return body, nil
}

// isGeminiImageModel reports whether a model is one of Gemini's image models, which generate images
// through generateContent, not through Imagen's predict.
func isGeminiImageModel(upstreamModelName string) bool {
	return strings.HasPrefix(upstreamModelName, "gemini-") &&
		(strings.Contains(upstreamModelName, "-image") || strings.Contains(upstreamModelName, "nano-banana"))
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if a.RequestMode == RequestModeGemini && isGeminiImageModel(info.UpstreamModelName) {
		if info.RelayMode != constant.RelayModeImagesGenerations {
			return nil, types.NewErrorWithStatusCode(errors.New("gemini image models take image generation requests only"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		geminiRequest, err := gemini.ConvertImageGenerationRequest(request)
		if err != nil {
			return nil, err
		}
		return geminiRequest, nil
	}
	geminiAdaptor := gemini.Adaptor{}
	return geminiAdaptor.ConvertImageRequest(c, info, request)
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	if strings.HasPrefix(info.UpstreamModelName, "claude") {
		a.RequestMode = RequestModeClaude
	} else if strings.Contains(info.UpstreamModelName, "llama") ||
		// open source models
		strings.Contains(info.UpstreamModelName, "-maas") {
		a.RequestMode = RequestModeOpenSource
	} else if _, ok := mistralChatModelID(info.UpstreamModelName); ok {
		a.RequestMode = RequestModeMistral
	} else {
		a.RequestMode = RequestModeGemini
	}
}

func (a *Adaptor) getRequestUrl(info *relaycommon.RelayInfo, modelName, suffix string) (string, error) {
	region := GetModelRegion(info.ApiVersion, info.OriginModelName)
	if info.ChannelOtherSettings.VertexKeyType != dto.VertexKeyTypeAPIKey {
		adc := &Credentials{}
		if err := common.Unmarshal([]byte(info.ApiKey), adc); err != nil {
			return "", fmt.Errorf("failed to decode credentials file: %w", err)
		}
		a.AccountCredentials = *adc

		if a.RequestMode == RequestModeGemini {
			return BuildGoogleModelURL(info.ChannelBaseUrl, DefaultAPIVersion, adc.ProjectID, region, modelName, suffix), nil
		} else if a.RequestMode == RequestModeClaude {
			return BuildAnthropicModelURL(info.ChannelBaseUrl, DefaultAPIVersion, adc.ProjectID, region, modelName, suffix), nil
		} else if a.RequestMode == RequestModeOpenSource {
			return BuildOpenSourceChatCompletionsURL(info.ChannelBaseUrl, adc.ProjectID, region), nil
		} else if a.RequestMode == RequestModeMistral {
			return BuildMistralModelURL(info.ChannelBaseUrl, DefaultAPIVersion, adc.ProjectID, region, modelName, suffix), nil
		}
	} else {
		if a.RequestMode == RequestModeMistral {
			return "", errors.New("vertex mistral models need a service account key, not an API key")
		}
		var keyPrefix string
		if strings.HasSuffix(suffix, "?alt=sse") {
			keyPrefix = "&"
		} else {
			keyPrefix = "?"
		}
		// An API key carries no project. Without a configured project ID the URL
		// keeps the express-mode form, which has no region in its path.
		projectID := strings.TrimSpace(info.ChannelOtherSettings.VertexProjectID)
		if a.RequestMode == RequestModeGemini {
			return fmt.Sprintf(
				"%s%skey=%s",
				BuildGoogleModelURL(info.ChannelBaseUrl, DefaultAPIVersion, projectID, region, modelName, suffix),
				keyPrefix,
				info.ApiKey,
			), nil
		} else if a.RequestMode == RequestModeClaude {
			return fmt.Sprintf(
				"%s%skey=%s",
				BuildAnthropicModelURL(info.ChannelBaseUrl, DefaultAPIVersion, projectID, region, modelName, suffix),
				keyPrefix,
				info.ApiKey,
			), nil
		}
	}
	return "", errors.New("unsupported request mode")
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	suffix := ""
	if a.RequestMode == RequestModeGemini {
		if info.IsStream {
			suffix = "streamGenerateContent?alt=sse"
		} else {
			suffix = "generateContent"
		}

		if strings.HasPrefix(info.UpstreamModelName, "imagen") {
			suffix = "predict"
		}
		if info.RelayMode == constant.RelayModeEmbeddings {
			suffix = "embedContent"
		}
		return a.getRequestUrl(info, info.UpstreamModelName, suffix)
	} else if a.RequestMode == RequestModeClaude {
		if info.IsStream {
			suffix = "streamRawPredict?alt=sse"
		} else {
			suffix = "rawPredict"
		}
		model := info.UpstreamModelName
		if v, ok := claudeModelMap[info.UpstreamModelName]; ok {
			model = v
		}
		return a.getRequestUrl(info, model, suffix)
	} else if a.RequestMode == RequestModeOpenSource {
		return a.getRequestUrl(info, "", "")
	} else if a.RequestMode == RequestModeMistral {
		if info.IsStream {
			suffix = "streamRawPredict"
		} else {
			suffix = "rawPredict"
		}
		modelID, _ := mistralChatModelID(info.UpstreamModelName)
		return a.getRequestUrl(info, modelID, suffix)
	}
	return "", errors.New("unsupported request mode")
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	if a.transcription != nil {
		// The transcription body is JSON, not the client's multipart upload.
		req.Set("Content-Type", "application/json")
	}
	if info.ChannelOtherSettings.VertexKeyType != dto.VertexKeyTypeAPIKey {
		accessToken, err := getAccessToken(a, info)
		if err != nil {
			return err
		}
		req.Set("Authorization", "Bearer "+accessToken)
	}
	if a.AccountCredentials.ProjectID != "" {
		req.Set("x-goog-user-project", a.AccountCredentials.ProjectID)
	}
	if strings.Contains(info.UpstreamModelName, "claude") {
		claude.CommonClaudeHeadersOperation(c, req, info)
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if a.RequestMode == RequestModeGemini && strings.HasPrefix(info.UpstreamModelName, "imagen") {
		prompt := ""
		for _, m := range request.Messages {
			if m.Role == "user" {
				prompt = m.StringContent()
				if prompt != "" {
					break
				}
			}
		}
		if prompt == "" {
			if p, ok := request.Prompt.(string); ok {
				prompt = p
			}
		}
		if prompt == "" {
			return nil, errors.New("prompt is required for image generation")
		}

		imgReq := dto.ImageRequest{
			Model:  request.Model,
			Prompt: prompt,
			N:      lo.ToPtr(uint(1)),
			Size:   "1024x1024",
		}
		if request.N != nil && *request.N > 0 {
			imgReq.N = lo.ToPtr(uint(*request.N))
		}
		if request.Size != "" {
			imgReq.Size = request.Size
		}
		if len(request.ExtraBody) > 0 {
			var extra map[string]any
			if err := common.Unmarshal(request.ExtraBody, &extra); err == nil {
				if n, ok := extra["n"].(float64); ok && n > 0 {
					imgReq.N = lo.ToPtr(uint(n))
				}
				if size, ok := extra["size"].(string); ok {
					imgReq.Size = size
				}
				// accept aspectRatio in extra body (top-level or under parameters)
				if ar, ok := extra["aspectRatio"].(string); ok && ar != "" {
					imgReq.Size = ar
				}
				if params, ok := extra["parameters"].(map[string]any); ok {
					if ar, ok := params["aspectRatio"].(string); ok && ar != "" {
						imgReq.Size = ar
					}
				}
			}
		}
		c.Set("request_model", request.Model)
		return a.ConvertImageRequest(c, info, imgReq)
	}
	if a.RequestMode == RequestModeClaude {
		result, err := service.ConvertRequest(c, info, types.RelayFormatClaude, request)
		if err != nil {
			return nil, err
		}
		claudeReq, ok := result.Value.(*dto.ClaudeRequest)
		if !ok {
			return nil, fmt.Errorf("expected Anthropic Messages request, got %T", result.Value)
		}
		vertexClaudeReq := copyRequest(claudeReq, anthropicVersion)
		c.Set("request_model", claudeReq.Model)
		info.UpstreamModelName = claudeReq.Model
		return vertexClaudeReq, nil
	} else if a.RequestMode == RequestModeGemini {
		result, err := service.ConvertRequest(c, info, types.RelayFormatGemini, request)
		if err != nil {
			return nil, err
		}
		geminiRequest, ok := result.Value.(*dto.GeminiChatRequest)
		if !ok {
			return nil, fmt.Errorf("expected Gemini generateContent request, got %T", result.Value)
		}
		if model_setting.GetGeminiSettings().RemoveFunctionResponseIdEnabled {
			removeFunctionCallIDs(geminiRequest)
		}
		c.Set("request_model", request.Model)
		return geminiRequest, nil
	} else if a.RequestMode == RequestModeOpenSource {
		return request, nil
	} else if a.RequestMode == RequestModeMistral {
		// rawPredict takes Mistral's own chat body, which names the model by its bare id.
		request.Model, _ = mistralChatModelID(info.UpstreamModelName)
		return request, nil
	}
	return nil, errors.New("unsupported request mode")
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	// Gemini embedding models are the ones that answer embedContent here.
	if a.RequestMode != RequestModeGemini || !strings.HasPrefix(info.UpstreamModelName, "gemini-embedding") {
		return nil, errors.New("not implemented")
	}

	var inputs []string
	switch input := request.Input.(type) {
	case string:
		inputs = []string{input}
	case []any:
		for _, item := range input {
			text, ok := item.(string)
			if !ok {
				return nil, types.NewErrorWithStatusCode(errors.New("vertex embeddings take text inputs, not token arrays"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			inputs = append(inputs, text)
		}
	}
	if len(inputs) == 0 {
		return nil, types.NewErrorWithStatusCode(errors.New("input is empty"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if len(inputs) > maxEmbedContentInputs {
		return nil, types.NewErrorWithStatusCode(fmt.Errorf("vertex embeddings take at most %d inputs in one request, got %d", maxEmbedContentInputs, len(inputs)), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	batch := embedContentBatch{Requests: make([]embedContentRequest, 0, len(inputs))}
	for _, text := range inputs {
		batch.Requests = append(batch.Requests, embedContentRequest{
			Content:              dto.GeminiChatContent{Parts: []dto.GeminiPart{{Text: text}}},
			OutputDimensionality: request.Dimensions,
		})
	}
	a.embeddingEncodingFormat = request.EncodingFormat
	return batch, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	if info.RelayMode == constant.RelayModeEmbeddings {
		return a.doEmbedContentCalls(c, info, requestBody)
	}
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.RelayMode == constant.RelayModeEmbeddings {
		return embedContentHandler(c, info, resp, a.embeddingEncodingFormat)
	}
	claudeAdaptor := claude.Adaptor{}
	if info.IsStream {
		switch a.RequestMode {
		case RequestModeClaude:
			return claudeAdaptor.DoResponse(c, resp, info)
		case RequestModeGemini:
			if info.RelayMode == constant.RelayModeGemini {
				return gemini.GeminiTextGenerationStreamHandler(c, info, resp)
			} else {
				return gemini.GeminiChatStreamHandler(c, info, resp)
			}
		case RequestModeOpenSource, RequestModeMistral:
			return openai.OaiStreamHandler(c, info, resp)
		}
	} else {
		switch a.RequestMode {
		case RequestModeClaude:
			return claudeAdaptor.DoResponse(c, resp, info)
		case RequestModeGemini:
			if a.transcription != nil {
				return gemini.GeminiTranscriptionHandler(c, info, resp, a.transcription)
			}
			if info.RelayMode == constant.RelayModeGemini {
				return gemini.GeminiTextGenerationHandler(c, info, resp)
			} else {
				if info.RelayMode == constant.RelayModeImagesGenerations && isGeminiImageModel(info.UpstreamModelName) {
					return gemini.GeminiImageGenerationHandler(c, info, resp)
				}
				if strings.HasPrefix(info.UpstreamModelName, "imagen") {
					return gemini.GeminiImageHandler(c, info, resp)
				}
				return gemini.GeminiChatHandler(c, info, resp)
			}
		case RequestModeOpenSource, RequestModeMistral:
			return openai.OpenaiHandler(c, info, resp)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	var modelList []string
	for i, s := range ModelList {
		modelList = append(modelList, s)
		ModelList[i] = s
	}
	for i, s := range claude.ModelList {
		modelList = append(modelList, s)
		claude.ModelList[i] = s
	}
	for i, s := range gemini.ModelList {
		modelList = append(modelList, s)
		gemini.ModelList[i] = s
	}
	return modelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
