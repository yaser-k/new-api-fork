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
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runOaiStreamUntilEOF relays upstream frames that end with EOF and no
// [DONE] to an OpenAI-format chat completions client.
func runOaiStreamUntilEOF(t *testing.T, frames ...string) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo, *types.NewAPIError) {
	t.Helper()
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	var body strings.Builder
	for _, frame := range frames {
		body.WriteString("data: " + frame + "\n\n")
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-test"},
		RelayFormat: types.RelayFormatOpenAI,
		RelayMode:   relayconstant.RelayModeChatCompletions,
	}
	_, apiErr := OaiStreamHandler(c, info, &http.Response{Body: io.NopCloser(strings.NewReader(body.String()))})
	return c, recorder, info, apiErr
}

// Upstream issue 7957: an upstream that answers 200, sends only a role frame
// and a usage frame, then closes the body generated nothing. The attempt must
// fail so the controller can retry it instead of billing the prompt.
func TestOaiStreamHandlerMetadataOnlyEOFFails(t *testing.T) {
	c, _, info, apiErr := runOaiStreamUntilEOF(t,
		`{"id":"chatcmpl-empty","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-empty","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":481801,"completion_tokens":0,"total_tokens":481801}}`,
	)

	require.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.EndReason)
	require.NotNil(t, apiErr, "a stream that generated nothing must not end as a success")
	assert.Equal(t, "retry", service.DecideRelayRetry(c, apiErr, 1).Action)
}
