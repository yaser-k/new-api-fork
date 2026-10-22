package vertex

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitPicksTheRequestModeByModelFamily(t *testing.T) {
	tests := []struct {
		upstream string
		want     int
	}{
		{"claude-sonnet-4-5@20250929", RequestModeClaude},
		{"claude-3-5-sonnet-20240620", RequestModeClaude},
		{"claude-opus-4-7", RequestModeClaude},
		{"meta/llama3-405b-instruct-maas", RequestModeOpenSource},
		{"meta/llama-4-maverick-17b-128e-instruct-maas", RequestModeOpenSource},
		{"qwen/qwen3-coder-480b-a35b-instruct-maas", RequestModeOpenSource},
		{"gemini-2.5-flash", RequestModeGemini},
		{"gemini-4-argon", RequestModeGemini},
		{"imagen-3.0-generate-002", RequestModeGemini},
		{"mistral-medium-3", RequestModeMistral},
		{"mistral-small-2503", RequestModeMistral},
		{"codestral-2", RequestModeMistral},
		{"mistralai/mistral-medium-3", RequestModeMistral},
		// The OCR model takes a document body, not a chat one, and keeps the mode it had.
		{"mistral-ocr-2505", RequestModeGemini},
	}
	for _, tt := range tests {
		t.Run(tt.upstream, func(t *testing.T) {
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tt.upstream}}
			adaptor := &Adaptor{}
			adaptor.Init(info)
			assert.Equal(t, tt.want, adaptor.RequestMode)
		})
	}
}

// Mistral's models answer rawPredict only in a regional location, so the region comes from the
// channel's region map by model name while its default stays global.
func TestGetRequestURLSendsMistralToRawPredict(t *testing.T) {
	tests := []struct {
		name     string
		upstream string
		stream   bool
		want     string
	}{
		{
			name:     "rawPredict",
			upstream: "mistral-medium-3",
			want:     "https://europe-west4-aiplatform.googleapis.com/v1/projects/my-project/locations/europe-west4/publishers/mistralai/models/mistral-medium-3:rawPredict",
		},
		{
			name:     "streamRawPredict",
			upstream: "mistral-medium-3",
			stream:   true,
			want:     "https://europe-west4-aiplatform.googleapis.com/v1/projects/my-project/locations/europe-west4/publishers/mistralai/models/mistral-medium-3:streamRawPredict",
		},
		{
			name:     "publisher prefix dropped from the path",
			upstream: "mistralai/mistral-medium-3",
			want:     "https://europe-west4-aiplatform.googleapis.com/v1/projects/my-project/locations/europe-west4/publishers/mistralai/models/mistral-medium-3:rawPredict",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				IsStream:        tt.stream,
				OriginModelName: "mistral-medium-3",
				ChannelMeta: &relaycommon.ChannelMeta{
					ApiVersion:        `{"default":"global","mistral-medium-3":"europe-west4"}`,
					ApiKey:            `{"project_id":"my-project"}`,
					UpstreamModelName: tt.upstream,
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

func TestConvertOpenAIRequestNamesTheBareMistralModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatOpenAI,
		OriginModelName: "mistral-medium-3",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "mistralai/mistral-medium-3"},
	}
	request := &dto.GeneralOpenAIRequest{
		Model:     "mistralai/mistral-medium-3",
		Messages:  []dto.Message{{Role: "user", Content: "Say hello."}},
		MaxTokens: common.GetPointer[uint](4),
		Stream:    common.GetPointer(true),
	}

	adaptor := &Adaptor{}
	adaptor.Init(info)
	converted, err := adaptor.ConvertOpenAIRequest(c, info, request)
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)

	assert.JSONEq(t, `{"model":"mistral-medium-3","messages":[{"role":"user","content":"Say hello."}],"max_tokens":4,"stream":true}`, string(body))
}

func TestDoResponseRelaysMistralChat(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	const completion = `{"id":"cmpl-1","object":"chat.completion","created":1760950080,"model":"mistral-medium-3",` +
		`"choices":[{"index":0,"message":{"role":"assistant","content":"Hello there"},"finish_reason":"length"}],` +
		`"usage":{"prompt_tokens":7,"total_tokens":11,"completion_tokens":4}}`
	// Mistral's chat stream: chunks of the OpenAI shape, usage on the last chunk with the finish reason, then [DONE].
	const stream = `data: {"id":"cmpl-2","object":"chat.completion.chunk","created":1760950080,"model":"mistral-medium-3","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"cmpl-2","object":"chat.completion.chunk","created":1760950080,"model":"mistral-medium-3","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"cmpl-2","object":"chat.completion.chunk","created":1760950080,"model":"mistral-medium-3","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"total_tokens":9,"completion_tokens":2}}

data: [DONE]

`
	tests := []struct {
		name           string
		stream         bool
		upstream       string
		contentType    string
		wantCompletion int
		wantTotal      int
		wantBody       []string
	}{
		{
			name:           "chat.completion",
			upstream:       completion,
			contentType:    "application/json",
			wantCompletion: 4,
			wantTotal:      11,
			wantBody:       []string{`"object":"chat.completion"`, `"content":"Hello there"`, `"prompt_tokens":7`},
		},
		{
			name:           "chat.completion.chunk stream",
			stream:         true,
			upstream:       stream,
			contentType:    "text/event-stream",
			wantCompletion: 2,
			wantTotal:      9,
			wantBody:       []string{`"content":"Hello"`, `"content":" there"`, `"completion_tokens":2`, "data: [DONE]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:        types.RelayFormatOpenAI,
				RelayMode:          relayconstant.RelayModeChatCompletions,
				IsStream:           tt.stream,
				DisablePing:        true,
				ShouldIncludeUsage: true,
				OriginModelName:    "mistral-medium-3",
				ChannelMeta:        &relaycommon.ChannelMeta{UpstreamModelName: "mistral-medium-3"},
			}
			adaptor := &Adaptor{}
			adaptor.Init(info)

			usage, apiErr := adaptor.DoResponse(c, &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{tt.contentType}},
				Body:       io.NopCloser(strings.NewReader(tt.upstream)),
			}, info)
			require.Nil(t, apiErr)
			require.IsType(t, &dto.Usage{}, usage)
			got := usage.(*dto.Usage)
			assert.Equal(t, 7, got.PromptTokens)
			assert.Equal(t, tt.wantCompletion, got.CompletionTokens)
			assert.Equal(t, tt.wantTotal, got.TotalTokens)
			for _, want := range tt.wantBody {
				assert.Contains(t, recorder.Body.String(), want)
			}
		})
	}
}
