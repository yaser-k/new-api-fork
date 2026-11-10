package gemini

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"
)

// ConvertImageGenerationRequest turns an OpenAI image generation request into the generateContent
// request of a Gemini image model: the prompt as the user turn, image output only, the size as an
// aspect ratio, and n as the candidate count.
func ConvertImageGenerationRequest(request dto.ImageRequest) (*dto.GeminiChatRequest, error) {
	if lo.FromPtrOr(request.Stream, false) {
		return nil, types.NewErrorWithStatusCode(errors.New("gemini image models do not stream image generation"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	imageConfig, err := common.Marshal(map[string]string{"aspectRatio": imageAspectRatio(request.Size)})
	if err != nil {
		return nil, err
	}
	geminiRequest := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{{
			Role:  "user",
			Parts: []dto.GeminiPart{{Text: request.Prompt}},
		}},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ResponseModalities: []string{"IMAGE"},
			ImageConfig:        imageConfig,
		},
	}
	if n := lo.FromPtrOr(request.N, uint(1)); n > 1 {
		geminiRequest.GenerationConfig.CandidateCount = common.GetPointer(int(n))
	}
	return geminiRequest, nil
}

// GeminiImageGenerationHandler answers a Gemini image model's generateContent response as an OpenAI
// image generation response, one data[].b64_json per inline image. Nothing else of a part is copied,
// so the thoughtSignature that comes with each image never reaches the client. An answer without an
// image is an error the client sees, and the usage it reports is still billed, as for a blocked chat
// answer.
func GeminiImageGenerationHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
	var geminiResponse dto.GeminiChatResponse
	if err := common.Unmarshal(responseBody, &geminiResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	info.ObserveResponseModel(gjson.GetBytes(responseBody, "modelVersion").Str)

	images := make([]dto.ImageData, 0, 1)
	for _, candidate := range geminiResponse.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && strings.HasPrefix(part.InlineData.MimeType, "image/") && part.InlineData.Data != "" {
				images = append(images, dto.ImageData{B64Json: part.InlineData.Data})
			}
		}
	}

	usage := buildUsageFromGeminiResponse(c, info, &geminiResponse)
	// Without a completion count from upstream, each returned image is estimated as on the chat path,
	// and the estimate counts as image output so that an image output price applies to it.
	if len(images) > 0 && (usage.BillingUsage == nil || usage.BillingUsage.Estimated) {
		if usage.CompletionTokens == 0 {
			usage.CompletionTokens = len(images) * geminiEstimatedImageOutputTokens
			common.SetContextKey(c, constant.ContextKeyLocalCountTokens, true)
		}
		usage.CompletionTokenDetails.ImageTokens = max(usage.CompletionTokens-usage.CompletionTokenDetails.ReasoningTokens, 0)
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		usage.BillingUsage = dto.NewEstimatedGeminiChatBillingUsage(&usage)
	}

	if len(images) == 0 {
		var newAPIError *types.NewAPIError
		finishReason := ""
		if len(geminiResponse.Candidates) > 0 && geminiResponse.Candidates[0].FinishReason != nil {
			finishReason = *geminiResponse.Candidates[0].FinishReason
		}
		reason := "finish reason " + cmp.Or(finishReason, "none")
		if finishMessage := gjson.GetBytes(responseBody, "candidates.0.finishMessage").Str; finishMessage != "" {
			reason += ": " + finishMessage
		}
		switch {
		case geminiResponse.PromptFeedback != nil && geminiResponse.PromptFeedback.BlockReason != nil:
			info.PerformanceBusinessRejection = true
			common.SetContextKey(c, constant.ContextKeyAdminRejectReason, fmt.Sprintf("gemini_block_reason=%s", *geminiResponse.PromptFeedback.BlockReason))
			newAPIError = types.NewOpenAIError(errors.New("request blocked by Gemini API: "+*geminiResponse.PromptFeedback.BlockReason), types.ErrorCodePromptBlocked, http.StatusBadRequest)
		case slices.Contains(geminiBlockingFinishReasons, finishReason):
			info.PerformanceBusinessRejection = true
			common.SetContextKey(c, constant.ContextKeyAdminRejectReason, fmt.Sprintf("gemini_finish_reason=%s", finishReason))
			newAPIError = types.NewOpenAIError(errors.New("no image generated, "+reason), types.ErrorCodePromptBlocked, http.StatusBadRequest)
		default:
			common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "gemini_no_image")
			newAPIError = types.NewOpenAIError(errors.New("no image in the Gemini response, "+reason), types.ErrorCodeEmptyResponse, http.StatusInternalServerError)
		}
		logger.LogWarn(c, common.LogText("gemini image generation returned no image: finish reason %s", finishReason))
		service.ResetStatusCode(newAPIError, c.GetString("status_code_mapping"))
		c.JSON(newAPIError.StatusCode, gin.H{
			"error": newAPIError.ToOpenAIError(common.UpstreamAddresses(c)...),
		})
		return &usage, nil
	}

	info.UpdateImageCount(int64(len(images)))
	body, err := common.Marshal(dto.ImageResponse{
		Created: common.GetTimestamp(),
		Data:    images,
	})
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, body)
	return &usage, nil
}
