package vertex

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shape of Vertex's answers for gemini-3.1-flash-image, with the image's base64 and its
// thoughtSignature (1.4 million characters upstream) cut down.
const (
	geminiImageAnswer = `{"candidates":[{"content":{"role":"model","parts":[{"inlineData":{"mimeType":"image/png","data":"iVBORw0KGgoAAAANSUhEUg=="},"thoughtSignature":"Q2lJQmpaVDY0LXNpZ25hdHVyZQ=="}]},"finishReason":"STOP","index":0}],` +
		`"usageMetadata":{"promptTokenCount":21,"candidatesTokenCount":1120,"totalTokenCount":1141,"trafficType":"ON_DEMAND","promptTokensDetails":[{"modality":"TEXT","tokenCount":21}],"candidatesTokensDetails":[{"modality":"IMAGE","tokenCount":1120}]},` +
		`"modelVersion":"gemini-3.1-flash-image"}`
	geminiImageRecitationAnswer = `{"candidates":[{"content":{"role":"model"},"finishReason":"IMAGE_RECITATION","finishMessage":"Unable to show the generated image. The model could not generate the image.","index":0}],` +
		`"usageMetadata":{"promptTokenCount":15,"totalTokenCount":15,"trafficType":"ON_DEMAND","promptTokensDetails":[{"modality":"TEXT","tokenCount":15}]},` +
		`"modelVersion":"gemini-3.1-flash-image"}`
	geminiImageAnswerWithoutUsage = `{"candidates":[{"content":{"role":"model","parts":[{"inlineData":{"mimeType":"image/png","data":"iVBORw0KGgoAAAANSUhEUg=="}}]},"finishReason":"STOP","index":0}]}`
)

func vertexImageInfo(relayMode int, model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode:       relayMode,
		RelayFormat:     types.RelayFormatOpenAIImage,
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiVersion:        "global",
			ApiKey:            "test-key",
			UpstreamModelName: model,
			ChannelOtherSettings: dto.ChannelOtherSettings{
				VertexKeyType: dto.VertexKeyTypeAPIKey,
			},
		},
	}
}

func TestConvertImageRequestForGeminiImageModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		model    string
		request  dto.ImageRequest
		wantBody string
		wantURL  string
	}{
		{
			name:     "gemini image model",
			model:    "gemini-3.1-flash-image",
			request:  dto.ImageRequest{Prompt: "A lighthouse at dusk", Size: "1024x1024"},
			wantBody: `{"contents":[{"role":"user","parts":[{"text":"A lighthouse at dusk"}]}],"generationConfig":{"responseModalities":["IMAGE"],"imageConfig":{"aspectRatio":"1:1"}}}`,
			wantURL:  "https://aiplatform.googleapis.com/v1/publishers/google/models/gemini-3.1-flash-image:generateContent?key=test-key",
		},
		{
			name:     "aspect ratio and n",
			model:    "gemini-nano-banana-2.1",
			request:  dto.ImageRequest{Prompt: "A lighthouse at dusk", Size: "16:9", N: common.GetPointer(uint(2))},
			wantBody: `{"contents":[{"role":"user","parts":[{"text":"A lighthouse at dusk"}]}],"generationConfig":{"candidateCount":2,"responseModalities":["IMAGE"],"imageConfig":{"aspectRatio":"16:9"}}}`,
			wantURL:  "https://aiplatform.googleapis.com/v1/publishers/google/models/gemini-nano-banana-2.1:generateContent?key=test-key",
		},
		{
			name:     "imagen keeps predict",
			model:    "imagen-4.0-generate-001",
			request:  dto.ImageRequest{Prompt: "A lighthouse at dusk", Size: "1792x1024", N: common.GetPointer(uint(2))},
			wantBody: `{"instances":[{"prompt":"A lighthouse at dusk"}],"parameters":{"sampleCount":2,"aspectRatio":"16:9","personGeneration":"allow_adult"}}`,
			wantURL:  "https://aiplatform.googleapis.com/v1/publishers/google/models/imagen-4.0-generate-001:predict?key=test-key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := vertexImageInfo(relayconstant.RelayModeImagesGenerations, tt.model)
			adaptor := &Adaptor{}
			adaptor.Init(info)

			converted, err := adaptor.ConvertImageRequest(c, info, tt.request)
			require.NoError(t, err)
			body, err := common.Marshal(converted)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantBody, string(body))
			url, err := adaptor.GetRequestURL(info)
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

func TestConvertImageRequestRefusesWhatGeminiImageModelsCannotServe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name      string
		relayMode int
		request   dto.ImageRequest
		wantError string
	}{
		{"stream", relayconstant.RelayModeImagesGenerations, dto.ImageRequest{Prompt: "x", Stream: common.GetPointer(true)}, "gemini image models do not stream image generation"},
		{"edits", relayconstant.RelayModeImagesEdits, dto.ImageRequest{Prompt: "x"}, "gemini image models take image generation requests only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := vertexImageInfo(tt.relayMode, "gemini-3.1-flash-image")
			adaptor := &Adaptor{}
			adaptor.Init(info)

			_, err := adaptor.ConvertImageRequest(c, info, tt.request)
			var apiErr *types.NewAPIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.Equal(t, tt.wantError, apiErr.Error())
		})
	}
}

// Both routes report a completion's image tokens as image output, which an image output price
// (img_o in a model's billing expression) settles apart from text.
func TestGeminiImageAnswers(t *testing.T) {
	tests := []struct {
		name           string
		relayMode      int
		answer         string
		wantStatus     int
		wantBody       []string
		wantPrompt     int
		wantCompletion int
		wantImage      int
		wantImageCount *int
	}{
		{
			name:           "image route",
			relayMode:      relayconstant.RelayModeImagesGenerations,
			answer:         geminiImageAnswer,
			wantStatus:     http.StatusOK,
			wantBody:       []string{`"b64_json":"iVBORw0KGgoAAAANSUhEUg=="`},
			wantPrompt:     21,
			wantCompletion: 1120,
			wantImage:      1120,
			wantImageCount: common.GetPointer(1),
		},
		{
			name:       "image route, no image",
			relayMode:  relayconstant.RelayModeImagesGenerations,
			answer:     geminiImageRecitationAnswer,
			wantStatus: http.StatusBadRequest,
			wantBody:   []string{"no image generated, finish reason IMAGE_RECITATION: Unable to show the generated image.", `"code":"prompt_blocked"`},
			wantPrompt: 15,
		},
		{
			name:           "image route, no counts",
			relayMode:      relayconstant.RelayModeImagesGenerations,
			answer:         geminiImageAnswerWithoutUsage,
			wantStatus:     http.StatusOK,
			wantBody:       []string{`"b64_json":"iVBORw0KGgoAAAANSUhEUg=="`},
			wantPrompt:     9,
			wantCompletion: 1400,
			wantImage:      1400,
			wantImageCount: common.GetPointer(1),
		},
		{
			name:           "chat route keeps the image in the message",
			relayMode:      relayconstant.RelayModeChatCompletions,
			answer:         geminiImageAnswer,
			wantStatus:     http.StatusOK,
			wantBody:       []string{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUg=="},
			wantPrompt:     21,
			wantCompletion: 1120,
			wantImage:      1120,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			info := vertexImageInfo(tt.relayMode, "gemini-3.1-flash-image")
			if tt.relayMode == relayconstant.RelayModeChatCompletions {
				info.RelayFormat = types.RelayFormatOpenAI
			}
			info.SetEstimatePromptTokens(9)
			// An image-quantity expression lets the answer settle the count of images it returned.
			info.TieredBillingSnapshot = &billingexpr.BillingSnapshot{EstimatedImageCount: common.GetPointer(1)}
			adaptor := &Adaptor{}
			adaptor.Init(info)

			usage, apiErr := adaptor.DoResponse(c, &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(tt.answer)),
			}, info)
			require.Nil(t, apiErr)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			for _, want := range tt.wantBody {
				assert.Contains(t, recorder.Body.String(), want)
			}
			assert.NotContains(t, recorder.Body.String(), "thoughtSignature")
			assert.NotContains(t, recorder.Body.String(), "Q2lJQmpaVDY0LXNpZ25hdHVyZQ==")
			if tt.wantStatus != http.StatusOK {
				assert.NotContains(t, recorder.Body.String(), `"data"`)
			}
			assert.Equal(t, tt.wantImageCount, info.BillingImageCount)

			// Settlement reads the usage the upstream dialect reports.
			got, ok := usage.(*dto.Usage).BillingUsage.CanonicalUsage()
			require.True(t, ok)
			assert.Equal(t, tt.wantPrompt, got.PromptTokens)
			assert.Equal(t, tt.wantCompletion, got.CompletionTokens)
			assert.Equal(t, tt.wantImage, got.CompletionTokenDetails.ImageTokens)
		})
	}
}
