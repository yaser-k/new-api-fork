package geminichat

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gemini splits generated text across parts at arbitrary points, and its own
// clients join text parts by plain concatenation (upstream issue 8135). Media
// and code parts keep the line-separated rendering the converter already uses.
func TestGeminiChat2OpenAIJoinsTextPartsVerbatim(t *testing.T) {
	type rendered struct {
		content   string
		reasoning string
	}
	cases := []struct {
		name     string
		parts    []dto.GeminiPart
		wantJSON bool
		full     rendered
		stream   rendered
	}{
		{
			name:     "json object split across text parts",
			parts:    []dto.GeminiPart{{Text: `{"reply`}, {Text: `":"hi","reactions":[]}`}},
			wantJSON: true,
			full:     rendered{content: `{"reply":"hi","reactions":[]}`},
			stream:   rendered{content: `{"reply":"hi","reactions":[]}`},
		},
		{
			name:   "newline-only text parts are kept",
			parts:  []dto.GeminiPart{{Text: "a"}, {Text: "\n"}, {Text: "\n"}, {Text: "b"}},
			full:   rendered{content: "a\n\nb"},
			stream: rendered{content: "a\n\nb"},
		},
		{
			name:   "single text part",
			parts:  []dto.GeminiPart{{Text: "hello world"}},
			full:   rendered{content: "hello world"},
			stream: rendered{content: "hello world"},
		},
		{
			name: "inline image between text parts",
			parts: []dto.GeminiPart{
				{Text: "Here:"},
				{InlineData: &dto.GeminiInlineData{MimeType: "image/png", Data: "aGk="}},
				{Text: "done"},
			},
			full:   rendered{content: "Here:\n![image](data:image/png;base64,aGk=)\ndone"},
			stream: rendered{content: "Here:\n![image](data:image/png;base64,aGk=)\ndone"},
		},
		{
			name: "code execution after text part",
			parts: []dto.GeminiPart{
				{Text: "Run:"},
				{ExecutableCode: &dto.GeminiPartExecutableCode{Language: "PYTHON", Code: "print(1)"}},
				{CodeExecutionResult: &dto.GeminiPartCodeExecutionResult{Outcome: "OUTCOME_OK", Output: "1"}},
				{Text: "Done."},
			},
			full:   rendered{content: "Run:\n```PYTHON\nprint(1)\n```\n```output\n1\n```\nDone."},
			stream: rendered{content: "Run:\n```PYTHON\nprint(1)\n```\n\n```output\n1\n```\n\nDone."},
		},
		{
			name: "thought part followed by text parts",
			parts: []dto.GeminiPart{
				{Text: "Thinking.", Thought: true},
				{Text: "Answer "},
				{Text: "here."},
			},
			full: rendered{content: "Answer here.", reasoning: "Thinking."},
			// The streaming converter reports every text-like part of an event
			// that carries a thought as reasoning.
			stream: rendered{reasoning: "Thinking.Answer here."},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := &dto.GeminiChatResponse{
				Candidates: []dto.GeminiChatCandidate{{Content: dto.GeminiChatContent{Role: "model", Parts: tc.parts}}},
			}

			t.Run("non-streaming", func(t *testing.T) {
				converted := ResponseGeminiChat2OpenAI("chatcmpl-test", 1, response)
				require.Len(t, converted.Choices, 1)
				message := converted.Choices[0].Message
				assert.Equal(t, tc.full.content, message.StringContent())
				assert.Equal(t, tc.full.reasoning, message.GetReasoningContent())
				if tc.wantJSON {
					assert.JSONEq(t, tc.full.content, message.StringContent())
				}
			})

			t.Run("streaming", func(t *testing.T) {
				converted, _ := StreamResponseGeminiChat2OpenAI(response)
				require.NotNil(t, converted)
				require.Len(t, converted.Choices, 1)
				delta := converted.Choices[0].Delta
				assert.Equal(t, tc.stream.content, delta.GetContentString())
				assert.Equal(t, tc.stream.reasoning, delta.GetReasoningContent())
				if tc.wantJSON {
					assert.JSONEq(t, tc.stream.content, delta.GetContentString())
				}
			})
		})
	}
}
