package vertex

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLWithAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		region    string
		stream    bool
		want      string
	}{
		{
			name:      "project ID keeps the global region",
			projectID: "my-project",
			region:    `{"default":"global"}`,
			want:      "https://aiplatform.googleapis.com/v1/projects/my-project/locations/global/publishers/google/models/gemini-4-argon:generateContent?key=test-key",
		},
		{
			name:      "project ID keeps a regional endpoint when streaming",
			projectID: "my-project",
			region:    `{"default":"us-central1"}`,
			stream:    true,
			want:      "https://us-central1-aiplatform.googleapis.com/v1/projects/my-project/locations/us-central1/publishers/google/models/gemini-4-argon:streamGenerateContent?alt=sse&key=test-key",
		},
		{
			name:   "no project ID keeps the express-mode URL",
			region: `{"default":"global"}`,
			want:   "https://aiplatform.googleapis.com/v1/publishers/google/models/gemini-4-argon:generateContent?key=test-key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				IsStream:        tt.stream,
				OriginModelName: "gemini-4-argon",
				ChannelMeta: &relaycommon.ChannelMeta{
					ApiVersion:        tt.region,
					ApiKey:            "test-key",
					UpstreamModelName: "gemini-4-argon",
					ChannelOtherSettings: dto.ChannelOtherSettings{
						VertexKeyType:   dto.VertexKeyTypeAPIKey,
						VertexProjectID: tt.projectID,
					},
				},
			}
			adaptor := &Adaptor{}
			adaptor.Init(info)

			got, err := adaptor.GetRequestURL(info)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// A Vertex AI Gemini answer can report promptTokenCount 0 with a totalTokenCount that holds the
// prompt; the chat path bills the remainder, not zero.
func TestVertexGeminiChatBillsPromptFromTotalWhenPromptCountMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 300
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})
	const frame = `{"candidates":[{"content":{"role":"model","parts":[{"text":"summary"}]},"finishReason":"STOP"}],` +
		`"usageMetadata":{"promptTokenCount":0,"candidatesTokenCount":50,"totalTokenCount":5050}}`

	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "non-stream", true: "stream"}[stream], func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				RelayMode:       relayconstant.RelayModeChatCompletions,
				OriginModelName: "gemini-2.5-flash",
				IsStream:        stream,
				ChannelMeta: &relaycommon.ChannelMeta{
					UpstreamModelName:    "gemini-2.5-flash",
					ChannelOtherSettings: dto.ChannelOtherSettings{VertexKeyType: dto.VertexKeyTypeAPIKey},
				},
			}
			info.SetEstimatePromptTokens(8)
			body := frame
			if stream {
				body = "data: " + frame + "\ndata: [DONE]\n"
			}
			adaptor := &Adaptor{}
			adaptor.Init(info)
			require.Equal(t, RequestModeGemini, adaptor.RequestMode)

			result, apiErr := adaptor.DoResponse(c, &http.Response{Body: io.NopCloser(strings.NewReader(body))}, info)
			require.Nil(t, apiErr)
			usage, ok := result.(*dto.Usage)
			require.True(t, ok)
			billed, ok := usage.BillingUsage.CanonicalUsage()
			require.True(t, ok)
			assert.Equal(t, 5000, usage.PromptTokens, "usage prompt tokens")
			assert.Equal(t, 5000, billed.PromptTokens, "billed prompt tokens")
			assert.Equal(t, 50, billed.CompletionTokens, "billed completion tokens")
		})
	}
}

// Mistral's rawPredict URL needs the project of a service account; an API key channel is refused before
// any request goes out, as it is for the open-source endpoint.
func TestGetRequestURLRefusesMistralWithAPIKey(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "mistral-medium-3",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiVersion:        `{"default":"europe-west4"}`,
			ApiKey:            "test-key",
			UpstreamModelName: "mistral-medium-3",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				VertexKeyType:   dto.VertexKeyTypeAPIKey,
				VertexProjectID: "my-project",
			},
		},
	}
	adaptor := &Adaptor{}
	adaptor.Init(info)

	_, err := adaptor.GetRequestURL(info)
	require.EqualError(t, err, "vertex mistral models need a service account key, not an API key")
}
