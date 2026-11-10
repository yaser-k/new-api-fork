package dto

import (
	"encoding/json"
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A content block parsed into ClaudeMediaMessage and marshalled again must be
// written back as the client sent it, so channels that forward the Claude body
// after reading its blocks do not change them.
func TestClaudeMessageContentBlocksSurviveParseAndMarshal(t *testing.T) {
	tests := []struct {
		name  string
		block string
	}{
		{
			name:  "search_result with a string source",
			block: `{"type":"search_result","source":"https://example.com/a","title":"A","content":[{"type":"text","text":"The first result."}],"citations":{"enabled":true}}`,
		},
		{
			name:  "image with a base64 source",
			block: `{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}}`,
		},
		{
			name:  "image with a url source",
			block: `{"type":"image","source":{"type":"url","url":"https://example.com/a.png"}}`,
		},
		{
			name:  "document with a base64 PDF source",
			block: `{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"JVBERi0x"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request ClaudeRequest
			require.NoError(t, kitutil.UnmarshalJsonStr(`{"model":"claude-test","max_tokens":16,"messages":[{"role":"user","content":[{"type":"text","text":"Answer from the result."},`+tt.block+`]}]}`, &request))

			blocks, err := request.Messages[0].ParseContent()
			require.NoError(t, err)
			require.Len(t, blocks, 2)
			request.Messages[0].SetContent(blocks)

			encoded, err := kitutil.Marshal(request)
			require.NoError(t, err)
			var forwarded struct {
				Messages []struct {
					Content []json.RawMessage `json:"content"`
				} `json:"messages"`
			}
			require.NoError(t, kitutil.Unmarshal(encoded, &forwarded))
			require.Len(t, forwarded.Messages, 1)
			require.Len(t, forwarded.Messages[0].Content, 2)
			assert.JSONEq(t, tt.block, string(forwarded.Messages[0].Content[1]))
		})
	}
}
