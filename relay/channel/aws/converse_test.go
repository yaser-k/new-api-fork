package aws

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	relaytypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream/eventstreamapi"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockruntimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	converseTestModel = "eu.mistral.pixtral-large-2502-v1:0"
	converseTestPNG   = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGP4z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC"
)

func newConverseTestMessages() []bedrockruntimeTypes.Message {
	return []bedrockruntimeTypes.Message{{
		Role:    bedrockruntimeTypes.ConversationRoleUser,
		Content: []bedrockruntimeTypes.ContentBlock{&bedrockruntimeTypes.ContentBlockMemberText{Value: "hello"}},
	}}
}

func newConverseTestInput() *bedrockruntime.ConverseInput {
	return &bedrockruntime.ConverseInput{ModelId: aws.String(converseTestModel), Messages: newConverseTestMessages()}
}

func newConverseTestStreamInput() *bedrockruntime.ConverseStreamInput {
	return &bedrockruntime.ConverseStreamInput{ModelId: aws.String(converseTestModel), Messages: newConverseTestMessages()}
}

func writeConverseStreamEvent(writer io.Writer, eventType string, payload string) error {
	return eventstream.NewEncoder().Encode(writer, eventstream.Message{
		Headers: eventstream.Headers{
			{Name: eventstreamapi.MessageTypeHeader, Value: eventstream.StringValue(eventstreamapi.EventMessageType)},
			{Name: eventstreamapi.EventTypeHeader, Value: eventstream.StringValue(eventType)},
			{Name: eventstreamapi.ContentTypeHeader, Value: eventstream.StringValue("application/json")},
		},
		Payload: []byte(payload),
	})
}

func TestIsConverseModel(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{modelID: "eu.mistral.pixtral-large-2502-v1:0", want: true},
		{modelID: "mistral.mistral-large-2407-v1:0", want: true},
		{modelID: "us.meta.llama3-3-70b-instruct-v1:0", want: true},
		{modelID: "deepseek.v3-v1:0", want: true},
		{modelID: "global.anthropic.claude-sonnet-4-6", want: false},
		{modelID: "anthropic.claude-3-5-sonnet-20240620-v1:0", want: false},
		{modelID: "eu.amazon.nova-micro-v1:0", want: false},
		{modelID: "arn:aws:bedrock:eu-north-1:123456789012:inference-profile/eu.anthropic.claude-sonnet-4-6", want: false},
		{modelID: "arn:aws:bedrock:us-east-1:123456789012:application-inference-profile/a1b2c3", want: false},
		{modelID: "claude-3-5-sonnet-20240620", want: false},
	}
	for _, test := range tests {
		t.Run(test.modelID, func(t *testing.T) {
			assert.Equal(t, test.want, isConverseModel(test.modelID))
		})
	}
}

func TestConvertOpenAIRequestUsesConverseForNonAnthropicModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	var request dto.GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalJsonStr(`{
		"model": "`+converseTestModel+`",
		"messages": [
			{"role": "system", "content": "Be brief."},
			{"role": "developer", "content": "Answer in French."},
			{"role": "user", "content": "What is this?"},
			{"role": "user", "content": [
				{"type": "text", "text": "Look:"},
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,`+converseTestPNG+`"}}
			]},
			{"role": "assistant", "content": "A pixel."},
			{"role": "user", "content": "Which colour?"}
		],
		"temperature": 0,
		"max_tokens": 256,
		"stop": "END"
	}`, &request))
	adaptor := &Adaptor{}

	converted, err := adaptor.ConvertOpenAIRequest(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: converseTestModel}}, &request)

	require.NoError(t, err)
	assert.True(t, adaptor.IsConverse)
	assert.False(t, adaptor.IsNova)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"system": [{"text": "Be brief."}, {"text": "Answer in French."}],
		"messages": [
			{"role": "user", "content": [
				{"text": "What is this?"},
				{"text": "Look:"},
				{"image": {"format": "png", "source": {"bytes": "`+converseTestPNG+`"}}}
			]},
			{"role": "assistant", "content": [{"text": "A pixel."}]},
			{"role": "user", "content": [{"text": "Which colour?"}]}
		],
		"inferenceConfig": {"maxTokens": 256, "temperature": 0, "stopSequences": ["END"]}
	}`, string(body))
}

func TestConvertOpenAIRequestRefusesWhatConverseCannotCarry(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "tools", body: `{"model":"` + converseTestModel + `","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}]}`},
		{name: "tool result", body: `{"model":"` + converseTestModel + `","messages":[{"role":"user","content":"hi"},{"role":"tool","tool_call_id":"call_1","content":"42"}]}`},
		{name: "functions", body: `{"model":"` + converseTestModel + `","messages":[{"role":"user","content":"hi"}],"functions":[{"name":"f","parameters":{"type":"object"}}]}`},
		{name: "assistant tool call", body: `{"model":"` + converseTestModel + `","messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"f","arguments":"{}"}}]}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			var request dto.GeneralOpenAIRequest
			require.NoError(t, common.UnmarshalJsonStr(test.body, &request))

			_, err := (&Adaptor{}).ConvertOpenAIRequest(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: converseTestModel}}, &request)

			var lossErr *relaytypes.ConversionLossError
			require.ErrorAs(t, err, &lossErr)
		})
	}
}

func TestDoAwsClientRequestBuildsConverseInput(t *testing.T) {
	tests := []struct {
		name     string
		isStream bool
	}{
		{name: "stream", isStream: true},
		{name: "non-stream", isStream: false},
	}
	body := `{"system":[{"text":"Be brief."}],"messages":[{"role":"user","content":[{"text":"Look:"},{"image":{"format":"png","source":{"bytes":"` + converseTestPNG + `"}}}]}],"inferenceConfig":{"maxTokens":256,"temperature":0,"topP":0.5,"stopSequences":["END"]}}`

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				IsStream: test.isStream,
				ChannelMeta: &relaycommon.ChannelMeta{
					ApiKey:            "bedrock-api-key|eu-north-1",
					UpstreamModelName: converseTestModel,
				},
			}
			adaptor := &Adaptor{IsConverse: true}

			_, err := doAwsClientRequest(c, info, adaptor, bytes.NewBufferString(body))
			require.NoError(t, err)

			var modelID *string
			var system []bedrockruntimeTypes.SystemContentBlock
			var messages []bedrockruntimeTypes.Message
			var inference *bedrockruntimeTypes.InferenceConfiguration
			if test.isStream {
				input, ok := adaptor.AwsReq.(*bedrockruntime.ConverseStreamInput)
				require.True(t, ok, "got %T", adaptor.AwsReq)
				modelID, system, messages, inference = input.ModelId, input.System, input.Messages, input.InferenceConfig
			} else {
				input, ok := adaptor.AwsReq.(*bedrockruntime.ConverseInput)
				require.True(t, ok, "got %T", adaptor.AwsReq)
				modelID, system, messages, inference = input.ModelId, input.System, input.Messages, input.InferenceConfig
			}
			assert.Equal(t, converseTestModel, aws.ToString(modelID))
			assert.Equal(t, []bedrockruntimeTypes.SystemContentBlock{&bedrockruntimeTypes.SystemContentBlockMemberText{Value: "Be brief."}}, system)
			require.Len(t, messages, 1)
			assert.Equal(t, bedrockruntimeTypes.ConversationRoleUser, messages[0].Role)
			require.Len(t, messages[0].Content, 2)
			assert.Equal(t, &bedrockruntimeTypes.ContentBlockMemberText{Value: "Look:"}, messages[0].Content[0])
			image, ok := messages[0].Content[1].(*bedrockruntimeTypes.ContentBlockMemberImage)
			require.True(t, ok)
			assert.Equal(t, bedrockruntimeTypes.ImageFormatPng, image.Value.Format)
			imageBytes, ok := image.Value.Source.(*bedrockruntimeTypes.ImageSourceMemberBytes)
			require.True(t, ok)
			assert.Equal(t, []byte("\x89PNG\r\n\x1a\n"), imageBytes.Value[:8])
			require.NotNil(t, inference)
			assert.Equal(t, int32(256), aws.ToInt32(inference.MaxTokens))
			require.NotNil(t, inference.Temperature)
			assert.Equal(t, float32(0), *inference.Temperature)
			assert.Equal(t, float32(0.5), aws.ToFloat32(inference.TopP))
			assert.Equal(t, []string{"END"}, inference.StopSequences)
		})
	}
}

func TestConverseAnswersOpenAIChatCompletion(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	var upstreamPath string
	var upstreamBody []byte
	client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		upstreamPath = request.URL.Path
		upstreamBody, _ = io.ReadAll(request.Body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"output":{"message":{"role":"assistant","content":[{"text":"Un pixel "},{"text":"vert."}]}},` +
				`"stopReason":"end_turn","usage":{"inputTokens":20,"outputTokens":3,"totalTokens":31,"cacheReadInputTokens":8},"metrics":{"latencyMs":120}}`)),
			Request: request,
		}, nil
	}))
	adaptor := &Adaptor{AwsClient: client, AwsReq: newConverseTestInput(), IsConverse: true}
	recorder := httptest.NewRecorder()
	c := newAwsTestContext(recorder, context.Background())
	info := newAwsTestRelayInfo()
	info.IsStream = false

	usage, handlerErr := adaptor.DoResponse(c, nil, info)

	require.Nil(t, handlerErr)
	assert.True(t, strings.HasSuffix(upstreamPath, "/converse"), upstreamPath)
	assert.NotContains(t, string(upstreamBody), "anthropic_version")
	require.IsType(t, &dto.Usage{}, usage)
	assert.Equal(t, 28, usage.(*dto.Usage).PromptTokens)
	assert.Equal(t, 8, usage.(*dto.Usage).PromptTokensDetails.CachedTokens)
	assert.Equal(t, 3, usage.(*dto.Usage).CompletionTokens)
	assert.Equal(t, 31, usage.(*dto.Usage).TotalTokens)

	var response dto.OpenAITextResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "chat.completion", response.Object)
	require.Len(t, response.Choices, 1)
	assert.Equal(t, "Un pixel vert.", response.Choices[0].Message.StringContent())
	assert.Equal(t, "stop", response.Choices[0].FinishReason)
	assert.Equal(t, 31, response.Usage.TotalTokens)
}

func TestConverseStreamRelaysOpenAIChunks(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	events := [][2]string{
		{"messageStart", `{"role":"assistant"}`},
		{"contentBlockDelta", `{"contentBlockIndex":0,"delta":{"reasoningContent":{"text":"Looking."}}}`},
		{"contentBlockStop", `{"contentBlockIndex":0}`},
		{"contentBlockDelta", `{"contentBlockIndex":1,"delta":{"text":"Un pixel"}}`},
		{"contentBlockDelta", `{"contentBlockIndex":1,"delta":{"text":" vert."}}`},
		{"contentBlockStop", `{"contentBlockIndex":1}`},
		{"messageStop", `{"stopReason":"max_tokens"}`},
		{"metadata", `{"usage":{"inputTokens":20,"outputTokens":3,"totalTokens":23},"metrics":{"latencyMs":120}}`},
	}
	tests := []struct {
		name         string
		includeUsage bool
	}{
		{name: "with usage chunk", includeUsage: true},
		{name: "without usage chunk", includeUsage: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var upstreamPath string
			client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
				upstreamPath = request.URL.Path
				var body bytes.Buffer
				for _, event := range events {
					if err := writeConverseStreamEvent(&body, event[0], event[1]); err != nil {
						return nil, err
					}
				}
				return newConverseStreamResponse(request, body.Bytes()), nil
			}))
			adaptor := &Adaptor{AwsClient: client, AwsReq: newConverseTestStreamInput(), IsConverse: true}
			recorder := httptest.NewRecorder()
			c := newAwsTestContext(recorder, context.Background())
			info := newAwsTestRelayInfo()
			info.ShouldIncludeUsage = test.includeUsage

			usage, handlerErr := adaptor.DoResponse(c, nil, info)

			require.Nil(t, handlerErr)
			assert.True(t, strings.HasSuffix(upstreamPath, "/converse-stream"), upstreamPath)
			require.IsType(t, &dto.Usage{}, usage)
			assert.Equal(t, 20, usage.(*dto.Usage).PromptTokens)
			assert.Equal(t, 3, usage.(*dto.Usage).CompletionTokens)
			assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))

			var chunks []dto.ChatCompletionsStreamResponse
			var lines []string
			for line := range strings.SplitSeq(recorder.Body.String(), "\n") {
				data, ok := strings.CutPrefix(line, "data: ")
				if !ok {
					continue
				}
				lines = append(lines, data)
				if data == "[DONE]" {
					continue
				}
				var chunk dto.ChatCompletionsStreamResponse
				require.NoError(t, common.UnmarshalJsonStr(data, &chunk))
				chunks = append(chunks, chunk)
			}
			require.NotEmpty(t, chunks)
			require.NotEmpty(t, chunks[0].Choices)
			assert.Equal(t, "assistant", chunks[0].Choices[0].Delta.Role)
			assert.Equal(t, "Looking.", chunks[0].Choices[0].Delta.GetReasoningContent())

			var content strings.Builder
			var finishReasons []string
			var streamUsage *dto.Usage
			for _, chunk := range chunks {
				if chunk.Usage != nil {
					streamUsage = chunk.Usage
				}
				for _, choice := range chunk.Choices {
					content.WriteString(choice.Delta.GetContentString())
					if choice.FinishReason != nil {
						finishReasons = append(finishReasons, *choice.FinishReason)
					}
				}
			}
			assert.Equal(t, "Un pixel vert.", content.String())
			assert.Equal(t, []string{"length"}, finishReasons)
			if test.includeUsage {
				require.NotNil(t, streamUsage)
				assert.Equal(t, 23, streamUsage.TotalTokens)
			} else {
				assert.Nil(t, streamUsage)
			}
			assert.Equal(t, "[DONE]", lines[len(lines)-1])
			assert.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.EndReason)
			assert.Equal(t, string(relaycommon.ResponseOutcomeCompleted), info.StreamStatus.ResponseOutcome())
		})
	}
}

func newConverseStreamResponse(request *http.Request, body []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/vnd.amazon.eventstream"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    request,
	}
}

func writeConverseStreamException(writer io.Writer, exceptionType string, message string) error {
	return eventstream.NewEncoder().Encode(writer, eventstream.Message{
		Headers: eventstream.Headers{
			{Name: eventstreamapi.MessageTypeHeader, Value: eventstream.StringValue(eventstreamapi.ExceptionMessageType)},
			{Name: eventstreamapi.ExceptionTypeHeader, Value: eventstream.StringValue(exceptionType)},
			{Name: eventstreamapi.ContentTypeHeader, Value: eventstream.StringValue("application/json")},
		},
		Payload: []byte(`{"message":"` + message + `"}`),
	})
}

// An exception before the first content delta reaches the client as an
// ordinary error with Bedrock's status, so it can be retried on another
// channel; the stream has not started.
func TestConverseStreamFailsCleanlyBeforeFirstChunk(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		var body bytes.Buffer
		if err := writeConverseStreamEvent(&body, "messageStart", `{"role":"assistant"}`); err != nil {
			return nil, err
		}
		if err := writeConverseStreamException(&body, "throttlingException", "Too many requests"); err != nil {
			return nil, err
		}
		return newConverseStreamResponse(request, body.Bytes()), nil
	}))
	adaptor := &Adaptor{AwsClient: client, AwsReq: newConverseTestStreamInput(), IsConverse: true}
	recorder := httptest.NewRecorder()
	c := newAwsTestContext(recorder, context.Background())

	handlerErr, usage := converseStreamHandler(c, newAwsTestRelayInfo(), adaptor)

	require.NotNil(t, handlerErr)
	assert.Contains(t, handlerErr.Error(), "Too many requests")
	assert.Equal(t, http.StatusTooManyRequests, handlerErr.StatusCode)
	assert.False(t, relaytypes.IsSkipRetryError(handlerErr))
	assert.Nil(t, usage)
	assert.Empty(t, recorder.Body.String())
	assert.NotEqual(t, "text/event-stream", recorder.Header().Get("Content-Type"))
}

func TestConverseStreamStopsAtClientCancellation(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	requestContext, cancelRequest := context.WithCancel(context.Background())
	t.Cleanup(cancelRequest)
	releaseRest := make(chan struct{})
	t.Cleanup(func() { close(releaseRest) })

	client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		reader, writer := io.Pipe()
		go func() {
			defer writer.Close()
			for _, event := range [][2]string{
				{"messageStart", `{"role":"assistant"}`},
				{"contentBlockDelta", `{"contentBlockIndex":0,"delta":{"text":"partial answer"}}`},
			} {
				if err := writeConverseStreamEvent(writer, event[0], event[1]); err != nil {
					return
				}
			}
			<-releaseRest
			_ = writeConverseStreamEvent(writer, "messageStop", `{"stopReason":"end_turn"}`)
		}()
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/vnd.amazon.eventstream"}},
			Body:       reader,
			Request:    request,
		}, nil
	}))
	responseWriter := newAwsNotifyingResponseWriter("partial answer")
	c := newAwsTestContext(responseWriter, requestContext)
	adaptor := &Adaptor{AwsClient: client, AwsReq: newConverseTestStreamInput(), IsConverse: true}
	info := newAwsTestRelayInfo()

	type handlerResult struct {
		err   *relaytypes.NewAPIError
		usage *dto.Usage
	}
	results := make(chan handlerResult, 1)
	go func() {
		err, usage := converseStreamHandler(c, info, adaptor)
		results <- handlerResult{err: err, usage: usage}
	}()

	var bodyLengthBeforeCancel int
	select {
	case bodyLengthBeforeCancel = <-responseWriter.notified:
	case <-time.After(5 * time.Second):
		t.Fatal("partial response was not written")
	}
	cancelRequest()

	var result handlerResult
	select {
	case result = <-results:
	case <-time.After(5 * time.Second):
		t.Fatal("stream handler did not stop after client cancellation")
	}

	require.Nil(t, result.err)
	require.NotNil(t, result.usage)
	assert.Positive(t, result.usage.CompletionTokens)
	assert.Equal(t, bodyLengthBeforeCancel, responseWriter.Body.Len())
	assert.NotContains(t, responseWriter.Body.String(), "[DONE]")
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
}

func TestConvertClaudeRequestRefusesConverseModels(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	_, err := (&Adaptor{}).ConvertClaudeRequest(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: converseTestModel}}, &dto.ClaudeRequest{Model: converseTestModel})

	var lossErr *relaytypes.ConversionLossError
	require.ErrorAs(t, err, &lossErr)
}

func TestGetAwsErrorStatusCodeMapsStreamExceptions(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "throttling", err: &bedrockruntimeTypes.ThrottlingException{Message: aws.String("slow down")}, want: http.StatusTooManyRequests},
		{name: "validation", err: &bedrockruntimeTypes.ValidationException{Message: aws.String("bad")}, want: http.StatusBadRequest},
		{name: "service unavailable", err: &bedrockruntimeTypes.ServiceUnavailableException{Message: aws.String("down")}, want: http.StatusServiceUnavailable},
		{name: "model stream error", err: &bedrockruntimeTypes.ModelStreamErrorException{Message: aws.String("broken")}, want: http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, getAwsErrorStatusCode(test.err))
		})
	}
}

// A stream that ends before any content fails the request so it is refunded
// and retried, rather than answering 200 with nothing and billing the prompt.
func TestConverseStreamWithoutContentFailsTheRequest(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	tests := []struct {
		name   string
		events [][2]string
	}{
		{name: "no events"},
		{name: "only messageStart", events: [][2]string{{"messageStart", `{"role":"assistant"}`}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
				var body bytes.Buffer
				for _, event := range test.events {
					if err := writeConverseStreamEvent(&body, event[0], event[1]); err != nil {
						return nil, err
					}
				}
				return newConverseStreamResponse(request, body.Bytes()), nil
			}))
			adaptor := &Adaptor{AwsClient: client, AwsReq: newConverseTestStreamInput(), IsConverse: true}
			recorder := httptest.NewRecorder()
			c := newAwsTestContext(recorder, context.Background())

			handlerErr, usage := converseStreamHandler(c, newAwsTestRelayInfo(), adaptor)

			require.NotNil(t, handlerErr)
			assert.False(t, relaytypes.IsSkipRetryError(handlerErr))
			assert.Nil(t, usage)
			assert.Empty(t, recorder.Body.String())
		})
	}
}

// A body this adaptor did not build for Converse, such as a passed-through
// request, is refused rather than sent to a non-Anthropic model.
func TestDoAwsClientRequestRefusesUnconvertedBodyForConverseModels(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-api-key|eu-north-1",
			UpstreamModelName: converseTestModel,
		},
	}

	_, err := doAwsClientRequest(c, info, &Adaptor{}, bytes.NewBufferString(`{"model":"`+converseTestModel+`","max_tokens":100,"messages":[{"role":"user","content":"hi"}]}`))

	var apiErr *relaytypes.NewAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.True(t, relaytypes.IsSkipRetryError(apiErr))
}
