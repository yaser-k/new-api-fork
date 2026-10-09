package openai

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
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runChatToClaudeStream relays Chat Completions SSE frames through
// OaiStreamHandler to a Claude Messages client and returns the event names
// and data payloads the client received, in order.
func runChatToClaudeStream(t *testing.T, frames ...string) ([]string, []string) {
	t.Helper()
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	var body strings.Builder
	for _, frame := range frames {
		body.WriteString("data: " + frame + "\n\n")
	}
	body.WriteString("data: [DONE]\n\n")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Set(common.RequestIdKey, "chat-to-claude-test")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body.String())),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta:       &relaycommon.ChannelMeta{UpstreamModelName: "gpt-test"},
		IsStream:          true,
		RelayFormat:       types.RelayFormatClaude,
		RelayMode:         relayconstant.RelayModeChatCompletions,
		DisablePing:       true,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{LastMessagesType: relaycommon.LastMessageTypeNone},
	}

	_, apiErr := OaiStreamHandler(c, info, resp)
	require.Nil(t, apiErr)

	var events, payloads []string
	for frame := range strings.SplitSeq(recorder.Body.String(), "\n\n") {
		event, data, ok := strings.Cut(frame, "\n")
		if !ok {
			continue
		}
		events = append(events, strings.TrimPrefix(event, "event: "))
		payloads = append(payloads, strings.TrimPrefix(data, "data: "))
	}
	return events, payloads
}

func TestOaiStreamHandlerClaudeSendsOneMessageStart(t *testing.T) {
	tests := []struct {
		name   string
		frames []string
	}{
		{
			name: "single frame",
			frames: []string{
				`{"id":"chatcmpl_1","model":"gpt-test","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}`,
			},
		},
		{
			name: "two frames",
			frames: []string{
				`{"id":"chatcmpl_1","model":"gpt-test","choices":[{"index":0,"delta":{"content":"hi"}}]}`,
				`{"id":"chatcmpl_1","model":"gpt-test","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, _ := runChatToClaudeStream(t, tt.frames...)

			assert.Equal(t, []string{
				"message_start",
				"content_block_start", "content_block_delta", "content_block_stop",
				"message_delta", "message_stop",
			}, events)
		})
	}
}
