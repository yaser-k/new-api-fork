package aws

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	relaytypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	novaTestModel = "eu.amazon.nova-lite-v1:0"
	novaTestPNG   = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGP4z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC"
)

func convertNovaTestRequest(t *testing.T, body string) (any, *Adaptor, error) {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	var request dto.GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalJsonStr(body, &request))
	adaptor := &Adaptor{}
	converted, err := adaptor.ConvertOpenAIRequest(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: novaTestModel}}, &request)
	return converted, adaptor, err
}

func TestConvertOpenAIRequestBuildsNovaBody(t *testing.T) {
	converted, adaptor, err := convertNovaTestRequest(t, `{
		"model": "`+novaTestModel+`",
		"messages": [
			{"role": "system", "content": "Be brief."},
			{"role": "developer", "content": "Answer in French."},
			{"role": "user", "content": "What is this?"},
			{"role": "user", "content": [
				{"type": "text", "text": "Look:"},
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,`+novaTestPNG+`"}}
			]},
			{"role": "assistant", "content": "A pixel."},
			{"role": "assistant", "content": " "},
			{"role": "user", "content": "Which colour?"}
		],
		"temperature": 0,
		"top_p": 0,
		"max_tokens": 100,
		"max_completion_tokens": 300,
		"stop": ["END", "STOP"]
	}`)

	require.NoError(t, err)
	assert.True(t, adaptor.IsNova)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"schemaVersion": "messages-v1",
		"system": [{"text": "Be brief."}, {"text": "Answer in French."}],
		"messages": [
			{"role": "user", "content": [
				{"text": "What is this?"},
				{"text": "Look:"},
				{"image": {"format": "png", "source": {"bytes": "`+novaTestPNG+`"}}}
			]},
			{"role": "assistant", "content": [{"text": "A pixel."}]},
			{"role": "user", "content": [{"text": "Which colour?"}]}
		],
		"inferenceConfig": {"maxTokens": 300, "temperature": 0.00001, "topP": 0, "stopSequences": ["END", "STOP"]}
	}`, string(body))
}

func TestConvertOpenAIRequestRefusesWhatNovaCannotCarry(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "tools", body: `{"model":"` + novaTestModel + `","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}]}`},
		{name: "functions", body: `{"model":"` + novaTestModel + `","messages":[{"role":"user","content":"hi"}],"functions":[{"name":"f","parameters":{"type":"object"}}]}`},
		{name: "tool result", body: `{"model":"` + novaTestModel + `","messages":[{"role":"user","content":"hi"},{"role":"tool","tool_call_id":"call_1","content":"42"}]}`},
		{name: "assistant tool call", body: `{"model":"` + novaTestModel + `","messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"f","arguments":"{}"}}]}]}`},
		{name: "audio", body: `{"model":"` + novaTestModel + `","messages":[{"role":"user","content":[{"type":"input_audio","input_audio":{"data":"AAAA","format":"wav"}}]}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := convertNovaTestRequest(t, test.body)

			var lossErr *relaytypes.ConversionLossError
			require.ErrorAs(t, err, &lossErr)
		})
	}
}

func TestHandleNovaRequestJoinsTextBlocks(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "two blocks", content: `[{"text":"Un pixel "},{"text":"vert."}]`, want: "Un pixel vert."},
		{name: "no blocks", content: `[]`, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{"output":{"message":{"role":"assistant","content":` + test.content +
						`}},"stopReason":"end_turn","usage":{"inputTokens":12,"outputTokens":4,"totalTokens":16}}`)),
					Request: request,
				}, nil
			}))
			adaptor := &Adaptor{AwsClient: client, AwsReq: newAwsInvokeModelInput(), IsNova: true}
			recorder := httptest.NewRecorder()
			c := newAwsTestContext(recorder, context.Background())
			info := newAwsTestRelayInfo()
			info.IsStream = false

			handlerErr, usage := handleNovaRequest(c, info, adaptor)

			require.Nil(t, handlerErr)
			require.NotNil(t, usage)
			assert.Equal(t, 16, usage.TotalTokens)
			var response dto.OpenAITextResponse
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.Len(t, response.Choices, 1)
			assert.Equal(t, test.want, response.Choices[0].Message.StringContent())
		})
	}
}
