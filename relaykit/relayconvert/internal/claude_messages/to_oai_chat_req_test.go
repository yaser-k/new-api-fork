package claudemessages

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeMessagesRequestToOpenAIChatToolResultContent(t *testing.T) {
	const imageData = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	const dataURL = "data:image/png;base64," + imageData
	imageBlock := dto.ClaudeMediaMessage{
		Type:   "image",
		Source: &dto.ClaudeMessageSource{Type: "base64", MediaType: "image/png", Data: imageData},
	}
	textBlock := func(text string) dto.ClaudeMediaMessage {
		return dto.ClaudeMediaMessage{Type: "text", Text: lo.ToPtr(text)}
	}

	tests := []struct {
		name        string
		content     any
		wantContent string
		wantImages  []string
	}{
		{
			name:        "string passes through",
			content:     "done",
			wantContent: "done",
		},
		{
			name:        "text and image keep text on tool message",
			content:     []dto.ClaudeMediaMessage{textBlock("screenshot taken"), imageBlock},
			wantContent: "screenshot taken",
			wantImages:  []string{dataURL},
		},
		{
			name:        "image only uses placeholder",
			content:     []dto.ClaudeMediaMessage{imageBlock},
			wantContent: "[image]",
			wantImages:  []string{dataURL},
		},
		{
			name: "url image source is forwarded as is",
			content: []dto.ClaudeMediaMessage{{
				Type:   "image",
				Source: &dto.ClaudeMessageSource{Type: "url", Url: "https://example.com/shot.png"},
			}},
			wantContent: "[image]",
			wantImages:  []string{"https://example.com/shot.png"},
		},
		{
			name:        "text blocks join with newline",
			content:     []dto.ClaudeMediaMessage{textBlock("first"), textBlock(""), textBlock("second")},
			wantContent: "first\nsecond",
		},
		{
			name:        "unknown block keeps whole array",
			content:     []dto.ClaudeMediaMessage{textBlock("see doc"), {Type: "document", Data: "opaque"}},
			wantContent: `[{"type":"text","text":"see doc"},{"type":"document","data":"opaque"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClaudeMessagesRequestToOpenAIChat(context.Background(), dto.ClaudeRequest{
				Model:     "claude-test",
				MaxTokens: lo.ToPtr(uint(64)),
				Messages: []dto.ClaudeMessage{
					{Role: "user", Content: "take a screenshot"},
					{Role: "assistant", Content: []dto.ClaudeMediaMessage{
						{Type: "tool_use", Id: "toolu_1", Name: "screenshot", Input: map[string]any{}},
					}},
					{Role: "user", Content: []dto.ClaudeMediaMessage{
						{Type: "tool_result", ToolUseId: "toolu_1", Content: tt.content},
						textBlock("what do you see?"),
					}},
				},
			}, &convmeta.Values{})
			require.NoError(t, err)

			require.Len(t, got.Messages, 4)
			tool := got.Messages[2]
			assert.Equal(t, "tool", tool.Role)
			assert.Equal(t, "toolu_1", tool.ToolCallId)
			if tt.wantImages == nil && tt.wantContent[0] == '[' {
				assert.JSONEq(t, tt.wantContent, tool.StringContent())
			} else {
				assert.Equal(t, tt.wantContent, tool.StringContent())
			}

			user := got.Messages[3]
			assert.Equal(t, "user", user.Role)
			parts := user.ParseContent()
			require.Len(t, parts, len(tt.wantImages)+1)
			for i, wantURL := range tt.wantImages {
				assert.Equal(t, dto.ContentTypeImageURL, parts[i].Type)
				require.NotNil(t, parts[i].GetImageMedia())
				assert.Equal(t, wantURL, parts[i].GetImageMedia().Url)
			}
			assert.Equal(t, dto.ContentTypeText, parts[len(parts)-1].Type)
			assert.Equal(t, "what do you see?", parts[len(parts)-1].Text)
		})
	}
}

func TestClaudeMessagesRequestToOpenAIChatKeepsParallelToolResultsContiguous(t *testing.T) {
	const dataURL = "data:image/png;base64,iVBORw0KGgo="
	got, err := ClaudeMessagesRequestToOpenAIChat(context.Background(), dto.ClaudeRequest{
		Model:     "claude-test",
		MaxTokens: lo.ToPtr(uint(64)),
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "screenshot and read the note"},
			{Role: "assistant", Content: []dto.ClaudeMediaMessage{
				{Type: "text", Text: lo.ToPtr("Let me look.")},
				{Type: "tool_use", Id: "toolu_shot", Name: "screenshot", Input: map[string]any{}},
				{Type: "tool_use", Id: "toolu_read", Name: "read_file", Input: map[string]any{"path": "note.txt"}},
			}},
			{Role: "user", Content: []dto.ClaudeMediaMessage{
				{Type: "tool_result", ToolUseId: "toolu_shot", Content: []dto.ClaudeMediaMessage{{
					Type:   "image",
					Source: &dto.ClaudeMessageSource{Type: "base64", MediaType: "image/png", Data: "iVBORw0KGgo="},
				}}},
				{Type: "tool_result", ToolUseId: "toolu_read", Content: "note contents"},
			}},
		},
	}, &convmeta.Values{})
	require.NoError(t, err)

	roles := make([]string, 0, len(got.Messages))
	for _, message := range got.Messages {
		roles = append(roles, message.Role)
	}
	assert.Equal(t, []string{"user", "assistant", "tool", "tool", "user"}, roles)
	assert.Len(t, got.Messages[1].ParseToolCalls(), 2)
	assert.Equal(t, "[image]", got.Messages[2].StringContent())
	assert.Equal(t, "toolu_shot", got.Messages[2].ToolCallId)
	assert.Equal(t, "note contents", got.Messages[3].StringContent())
	assert.NotContains(t, got.Messages[2].StringContent()+got.Messages[3].StringContent(), "base64")

	parts := got.Messages[4].ParseContent()
	require.Len(t, parts, 1)
	assert.Equal(t, dto.ContentTypeImageURL, parts[0].Type)
	require.NotNil(t, parts[0].GetImageMedia())
	assert.Equal(t, dataURL, parts[0].GetImageMedia().Url)
}

func TestClaudeMessagesStopSequencesPreservesArray(t *testing.T) {
	maxTokens := uint(16)
	cases := []struct {
		name string
		stop []string
		want any
	}{
		{name: "single element", stop: []string{"</block>"}, want: []string{"</block>"}},
		{name: "multiple elements", stop: []string{"A", "B"}, want: []string{"A", "B"}},
		{name: "empty", stop: nil, want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			openAIRequest, err := ClaudeMessagesRequestToOpenAIChat(context.Background(), dto.ClaudeRequest{
				Model:         "claude-test",
				MaxTokens:     &maxTokens,
				StopSequences: tc.stop,
			}, &convmeta.Values{})
			require.NoError(t, err)
			require.NotNil(t, openAIRequest)
			assert.Equal(t, tc.want, openAIRequest.Stop)
		})
	}
}
