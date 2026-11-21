package openai

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
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	eofFrameRole    = `{"id":"chatcmpl-eof","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`
	eofFrameContent = `{"id":"chatcmpl-eof","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hello streaming world"}}]}`
	eofFrameTool    = `{"id":"chatcmpl-eof","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]}}]}`
	eofFrameFinish  = `{"id":"chatcmpl-eof","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	eofFrameUsage   = `{"id":"chatcmpl-eof","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
)

// runOaiStream relays upstream frames to an OpenAI-format chat completions
// client. The upstream body ends with [DONE] when done is set, else at EOF.
func runOaiStream(t *testing.T, done bool, frames ...string) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo, *dto.Usage, *types.NewAPIError) {
	t.Helper()
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	var body strings.Builder
	for _, frame := range frames {
		body.WriteString("data: " + frame + "\n\n")
	}
	if done {
		body.WriteString("data: [DONE]\n\n")
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-test"},
		RelayFormat: types.RelayFormatOpenAI,
		RelayMode:   relayconstant.RelayModeChatCompletions,
	}
	usage, apiErr := OaiStreamHandler(c, info, &http.Response{Body: io.NopCloser(strings.NewReader(body.String()))})
	return c, recorder, info, usage, apiErr
}

// Upstream issue 7957: an upstream that answers 200, sends only a role frame
// and a usage frame, then closes the body generated nothing. The attempt must
// fail so the controller can retry it instead of billing the prompt.
func TestOaiStreamHandlerMetadataOnlyEOFFails(t *testing.T) {
	c, recorder, info, _, apiErr := runOaiStream(t, false,
		`{"id":"chatcmpl-empty","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-empty","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":481801,"completion_tokens":0,"total_tokens":481801}}`,
	)

	require.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.EndReason)
	require.NotNil(t, apiErr, "a stream that generated nothing must not end as a success")
	assert.Equal(t, "retry", service.DecideRelayRetry(c, apiErr, 1).Action)
	assert.NotContains(t, recorder.Body.String(), "[DONE]", "the failed attempt must not end the client's stream")
}

func TestOaiStreamHandlerEOFWithOutputSucceeds(t *testing.T) {
	tests := []struct {
		name           string
		done           bool
		frames         []string
		wantPrompt     int
		wantCompletion int
	}{
		{
			name:           "content deltas then EOF without [DONE]",
			frames:         []string{eofFrameRole, eofFrameContent},
			wantCompletion: 3,
		},
		{
			name:           "tool call deltas only then EOF",
			frames:         []string{eofFrameRole, eofFrameTool},
			wantCompletion: 11,
		},
		{
			name:           "finish_reason with empty content then [DONE]",
			done:           true,
			frames:         []string{eofFrameRole, eofFrameFinish},
			wantCompletion: 0,
		},
		{
			name:           "usage frame after content keeps the upstream usage",
			frames:         []string{eofFrameRole, eofFrameContent, eofFrameUsage},
			wantPrompt:     10,
			wantCompletion: 5,
		},
		{
			name: "usage reporting completion tokens without a text delta",
			frames: []string{eofFrameRole,
				`{"id":"chatcmpl-eof","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`},
			wantPrompt:     10,
			wantCompletion: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, usage, apiErr := runOaiStream(t, tt.done, tt.frames...)

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tt.wantPrompt, usage.PromptTokens)
			assert.Equal(t, tt.wantCompletion, usage.CompletionTokens)
		})
	}
}
