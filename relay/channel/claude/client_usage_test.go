package claude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeAnswersLeaveOutServerAccounting relays one Claude upstream answer,
// buffered and streamed, to Chat Completions and Responses clients. The client
// never receives usage_semantic, usage_source or billing_usage, while the usage
// handed to settlement keeps them and bills the same amount.
func TestClaudeAnswersLeaveOutServerAccounting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = originalTimeout })

	const upstreamUsage = `{"input_tokens":10000,"output_tokens":5000,"cache_read_input_tokens":7000,"cache_creation_input_tokens":30000,"cache_creation":{"ephemeral_5m_input_tokens":10000,"ephemeral_1h_input_tokens":20000}}`
	buffered := `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-4-8","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","usage":` + upstreamUsage + `}`
	streamed := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-opus-4-8\",\"content\":[],\"usage\":{\"input_tokens\":10000,\"output_tokens\":1,\"cache_read_input_tokens\":7000,\"cache_creation_input_tokens\":30000,\"cache_creation\":{\"ephemeral_5m_input_tokens\":10000,\"ephemeral_1h_input_tokens\":20000}}}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":5000}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	// Claude list prices per million tokens: input 3, output 15, cache read 0.3,
	// 5-minute cache write 3.75, 1-hour cache write 6.
	const expr = `tier("base", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6)`
	const wantQuota = 132_300 // (10000*3 + 5000*15 + 7000*0.3 + 10000*3.75 + 20000*6) / 1e6 * 500000

	for _, tc := range []struct {
		name   string
		format types.RelayFormat
		stream bool
		usage  func(t *testing.T, answer map[string]any) map[string]any
	}{
		{name: "buffered chat", format: types.RelayFormatOpenAI, usage: answerUsage("usage")},
		{name: "streamed chat", format: types.RelayFormatOpenAI, stream: true, usage: answerUsage("usage")},
		{name: "buffered responses", format: types.RelayFormatOpenAIResponses, usage: answerUsage("usage")},
		{name: "streamed responses", format: types.RelayFormatOpenAIResponses, stream: true, usage: answerUsage("response", "usage")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:        tc.format,
				IsStream:           tc.stream,
				ShouldIncludeUsage: true,
				ChannelMeta:        &relaycommon.ChannelMeta{UpstreamModelName: "claude-opus-4-8"},
				TieredBillingSnapshot: &billingexpr.BillingSnapshot{
					BillingMode:  "tiered_expr",
					ExprString:   expr,
					ExprHash:     billingexpr.ExprHashString(expr),
					GroupRatio:   1,
					QuotaPerUnit: 500_000,
				},
			}

			var usage *dto.Usage
			var apiErr *types.NewAPIError
			switch {
			case tc.stream && tc.format == types.RelayFormatOpenAIResponses:
				usage, apiErr = ClaudeResponsesStreamHandler(c, sseResponse(streamed), info)
			case tc.stream:
				usage, apiErr = ClaudeStreamHandler(c, sseResponse(streamed), info)
			default:
				usage, apiErr = ClaudeHandler(c, jsonResponse(buffered), info)
			}
			require.Nil(t, apiErr)

			answers := clientAnswers(t, recorder.Body.String(), tc.stream)
			usageSeen := false
			for _, answer := range answers {
				clientUsage := tc.usage(t, answer)
				if clientUsage == nil {
					continue
				}
				usageSeen = true
				for _, internal := range []string{"usage_semantic", "usage_source", "billing_usage"} {
					assert.NotContains(t, clientUsage, internal)
				}
				if tc.format == types.RelayFormatOpenAI {
					assert.Equal(t, clientUsage["prompt_tokens"], clientUsage["input_tokens"])
					assert.Equal(t, clientUsage["completion_tokens"], clientUsage["output_tokens"])
				}
			}
			require.True(t, usageSeen, "the client answer must report usage")

			require.NotNil(t, usage)
			assert.Equal(t, "anthropic", usage.UsageSemantic)
			require.NotNil(t, usage.BillingUsage, "settlement keeps the upstream's own usage")
			settled, ok := usage.BillingUsage.CanonicalUsage()
			require.True(t, ok)
			params := service.BuildTieredTokenParams(settled, settled.UsageSemantic == "anthropic", billingexpr.UsedVarsByHash(expr, billingexpr.ExprHashString(expr)))
			billed, quota, _ := service.TryTieredSettle(info, params)
			require.True(t, billed)
			assert.Equal(t, wantQuota, quota)
		})
	}
}

func answerUsage(path ...string) func(t *testing.T, answer map[string]any) map[string]any {
	return func(t *testing.T, answer map[string]any) map[string]any {
		var node any = answer
		for _, key := range path {
			object, ok := node.(map[string]any)
			if !ok {
				return nil
			}
			node = object[key]
		}
		usage, _ := node.(map[string]any)
		return usage
	}
}

func clientAnswers(t *testing.T, body string, stream bool) []map[string]any {
	t.Helper()
	if !stream {
		var answer map[string]any
		require.NoError(t, common.UnmarshalJsonStr(body, &answer))
		return []map[string]any{answer}
	}
	var answers []map[string]any
	for line := range strings.SplitSeq(body, "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "[DONE]" {
			continue
		}
		var answer map[string]any
		require.NoError(t, common.UnmarshalJsonStr(data, &answer), data)
		answers = append(answers, answer)
	}
	return answers
}

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func sseResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
}
