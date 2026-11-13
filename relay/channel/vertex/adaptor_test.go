package vertex

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An Anthropic Messages request (/v1/messages, as Claude Code sends it) reaches Vertex in the format of the endpoint
// the model is served on: Gemini's generateContent for Gemini models, the OpenAI-compatible chat completions endpoint
// for partner open models, and Anthropic's rawPredict body only for Claude.
func TestConvertClaudeRequestUsesTheModelsEndpointFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		upstream   string
		wantMode   int
		wantFields []string
		noFields   []string
	}{
		{
			name:       "gemini model",
			upstream:   "gemini-2.5-flash",
			wantMode:   RequestModeGemini,
			wantFields: []string{"contents", "systemInstruction", "tools", "generationConfig"},
			noFields:   []string{"anthropic_version", "messages", "max_tokens", "system"},
		},
		{
			name:       "partner open model",
			upstream:   "qwen/qwen3-coder-480b-a35b-instruct-maas",
			wantMode:   RequestModeOpenSource,
			wantFields: []string{"model", "messages", "tools", "max_tokens"},
			noFields:   []string{"anthropic_version", "system", "contents"},
		},
		{
			name:       "claude model",
			upstream:   "claude-sonnet-4-5@20250929",
			wantMode:   RequestModeClaude,
			wantFields: []string{"anthropic_version", "messages", "max_tokens", "system", "tools"},
			noFields:   []string{"model", "contents"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:     types.RelayFormatClaude,
				OriginModelName: "client-model",
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: tt.upstream},
			}
			request := &dto.ClaudeRequest{
				Model:     tt.upstream,
				MaxTokens: common.GetPointer[uint](64),
				System:    "You are terse.",
				Messages:  []dto.ClaudeMessage{{Role: "user", Content: "What is the weather in Tehran?"}},
				Tools: []any{map[string]any{
					"name":         "get_weather",
					"description":  "Weather for a city",
					"input_schema": map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}},
				}},
			}

			adaptor := &Adaptor{}
			adaptor.Init(info)
			require.Equal(t, tt.wantMode, adaptor.RequestMode)

			converted, err := adaptor.ConvertClaudeRequest(c, info, request)
			require.NoError(t, err)
			body, err := common.Marshal(converted)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, common.Unmarshal(body, &fields))

			for _, field := range tt.wantFields {
				assert.Contains(t, fields, field)
			}
			for _, field := range tt.noFields {
				assert.NotContains(t, fields, field)
			}
			if tt.wantMode == RequestModeOpenSource {
				assert.Equal(t, tt.upstream, fields["model"])
				tools, ok := fields["tools"].([]any)
				require.True(t, ok)
				require.Len(t, tools, 1)
				assert.Equal(t, "function", tools[0].(map[string]any)["type"])
			}
		})
	}
}
