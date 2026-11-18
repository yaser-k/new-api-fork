package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	relaytypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pegasusTestModel = "global.twelvelabs.pegasus-1-5-v1:0"

// pegasusTestVideo stands in for the 2-second MP4 the supervisor measured.
var pegasusTestVideo = []byte("\x00\x00\x00\x18ftypmp42 two seconds of blue")

var pegasusTestVideoBase64 = base64.StdEncoding.EncodeToString(pegasusTestVideo)

func newPegasusTestInfo(model string, stream bool) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		StartTime:          time.Now(),
		IsStream:           stream,
		OriginModelName:    model,
		RelayFormat:        relaytypes.RelayFormatOpenAI,
		ShouldIncludeUsage: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "bedrock-api-key|us-east-1",
			UpstreamModelName: model,
		},
	}
}

func newPegasusTestRequest(t *testing.T, requestJSON string) *dto.GeneralOpenAIRequest {
	t.Helper()
	var request dto.GeneralOpenAIRequest
	require.NoError(t, common.Unmarshal([]byte(requestJSON), &request))
	return &request
}

// pegasusDataVideoRequest is a chat request for model with a system message
// and one user message carrying text and the test video as a data: URL.
func pegasusDataVideoRequest(model string, extra string) string {
	return `{"model":"` + model + `",` + extra + `"messages":[` +
		`{"role":"system","content":"Answer briefly."},` +
		`{"role":"user","content":[{"type":"text","text":"Describe this video in one short sentence."},` +
		`{"type":"video_url","video_url":{"url":"data:video/mp4;base64,` + pegasusTestVideoBase64 + `"}}]}]}`
}

// runPegasus relays a chat request through the adaptor against a fake Bedrock
// and returns the model id and body it received.
func runPegasus(t *testing.T, info *relaycommon.RelayInfo, requestJSON string, answer func(*http.Request) *http.Response) (string, string, *httptest.ResponseRecorder, *dto.Usage, *relaytypes.NewAPIError) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c := newAwsTestContext(recorder, context.Background())
	a := &Adaptor{}
	a.Init(info)

	converted, err := a.ConvertOpenAIRequest(c, info, newPegasusTestRequest(t, requestJSON))
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	response, err := a.DoRequest(c, info, bytes.NewReader(body))
	require.NoError(t, err)
	require.Nil(t, response)

	var sentModel, sentBody string
	a.AwsClient = newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		requestBody, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		path := strings.TrimPrefix(request.URL.EscapedPath(), "/model/")
		path, _, _ = strings.Cut(path, "/")
		sentModel, err = url.PathUnescape(path)
		require.NoError(t, err)
		sentBody = string(requestBody)
		return answer(request), nil
	}))
	usage, apiErr := a.DoResponse(c, nil, info)
	if apiErr != nil {
		return sentModel, sentBody, recorder, nil, apiErr
	}
	return sentModel, sentBody, recorder, usage.(*dto.Usage), nil
}

func TestPegasusRequestBodyFromDataAndURLVideos(t *testing.T) {
	originalLoader := awsLoadFileSource
	t.Cleanup(func() { awsLoadFileSource = originalLoader })
	awsLoadFileSource = func(c *gin.Context, source relaytypes.FileSource, reason ...string) (*relaytypes.CachedFileData, error) {
		if !source.IsURL() {
			return originalLoader(c, source, reason...)
		}
		require.Equal(t, "https://videos.example/blue.mp4", source.(*relaytypes.URLSource).URL)
		return relaytypes.NewMemoryCachedData(pegasusTestVideoBase64, "video/mp4", int64(len(pegasusTestVideo))), nil
	}

	for _, test := range []struct {
		name    string
		request string
		want    string
	}{
		{"data video, max_tokens raised to 512, temperature lowered to 1",
			pegasusDataVideoRequest(pegasusTestModel, `"max_tokens":100,"temperature":1.5,`),
			`{"inputPrompt":"Answer briefly.\n\nDescribe this video in one short sentence.","mediaSource":{"base64String":"` + pegasusTestVideoBase64 + `"},"maxOutputTokens":512,"temperature":1,"includeMetering":true}`},
		{"data video without limits",
			pegasusDataVideoRequest(pegasusTestModel, ""),
			`{"inputPrompt":"Answer briefly.\n\nDescribe this video in one short sentence.","mediaSource":{"base64String":"` + pegasusTestVideoBase64 + `"},"includeMetering":true}`},
		{"url video in the string form, over turns, max_completion_tokens lowered to 98304, temperature raised to 0",
			`{"model":"` + pegasusTestModel + `","max_tokens":600,"max_completion_tokens":200000,"temperature":-0.5,"messages":[` +
				`{"role":"user","content":[{"type":"video_url","video_url":"https://videos.example/blue.mp4"},{"type":"text","text":"What colour is it?"}]},` +
				`{"role":"assistant","content":"Blue."},` +
				`{"role":"user","content":"Does it change?"}]}`,
			`{"inputPrompt":"User: What colour is it?\n\nAssistant: Blue.\n\nUser: Does it change?","mediaSource":{"base64String":"` + pegasusTestVideoBase64 + `"},"maxOutputTokens":98304,"temperature":0,"includeMetering":true}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			info := newPegasusTestInfo(pegasusTestModel, false)
			converted, err := (&Adaptor{}).ConvertOpenAIRequest(newAwsTestContext(httptest.NewRecorder(), context.Background()), info, newPegasusTestRequest(t, test.request))
			require.NoError(t, err)
			body, err := common.Marshal(converted)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(body))
		})
	}
}

func TestPegasusRefusesWhatItCannotSend(t *testing.T) {
	originalLoader := awsLoadFileSource
	t.Cleanup(func() { awsLoadFileSource = originalLoader })
	awsLoadFileSource = func(c *gin.Context, source relaytypes.FileSource, reason ...string) (*relaytypes.CachedFileData, error) {
		return relaytypes.NewMemoryCachedData("", "video/mp4", 25*1024*1024+1), nil
	}
	video := `{"type":"video_url","video_url":{"url":"data:video/mp4;base64,` + pegasusTestVideoBase64 + `"}}`

	for _, test := range []struct {
		name    string
		model   string
		request string
		message string
	}{
		{"video over 25 MB", pegasusTestModel,
			`{"messages":[{"role":"user","content":[{"type":"text","text":"Describe it."},{"type":"video_url","video_url":{"url":"https://videos.example/long.mp4"}}]}]}`,
			"messages[0].content[1]: the video is 26214401 bytes; TwelveLabs Pegasus on AWS Bedrock takes at most 26214400 bytes inline"},
		{"two videos", pegasusTestModel,
			`{"messages":[{"role":"user","content":[{"type":"text","text":"Compare them."},` + video + `,` + video + `]}]}`,
			"messages[0].content[2]: TwelveLabs Pegasus takes one video per request; messages[0].content[1] already carries one"},
		{"no video", pegasusTestModel,
			`{"messages":[{"role":"user","content":"Describe it."}]}`,
			"messages: TwelveLabs Pegasus needs one video_url part in a user message"},
		{"image beside the video", pegasusTestModel,
			`{"messages":[{"role":"user","content":[` + video + `,{"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA"}}]}]}`,
			`messages[0].content[1]: content type "image_url" is not supported for TwelveLabs Pegasus on AWS Bedrock`},
		{"video to a Claude model", "us.anthropic.claude-sonnet-4-6",
			`{"messages":[{"role":"user","content":[{"type":"text","text":"Describe it."},` + video + `]}]}`,
			"messages[0].content: video input is supported only for TwelveLabs Pegasus models on AWS Bedrock"},
		{"video to a Nova model", "amazon.nova-pro-v1:0",
			`{"messages":[{"role":"user","content":[{"type":"text","text":"Describe it."},` + video + `]}]}`,
			"messages[0].content: video input is supported only for TwelveLabs Pegasus models on AWS Bedrock"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := newPegasusTestRequest(t, test.request)
			request.Model = test.model
			_, err := (&Adaptor{}).ConvertOpenAIRequest(newAwsTestContext(httptest.NewRecorder(), context.Background()), newPegasusTestInfo(test.model, false), request)
			var loss *relaytypes.ConversionLossError
			require.True(t, errors.As(err, &loss), "want a conversion refusal, got %v", err)
			assert.Equal(t, "conversion would lose protocol semantics: "+test.message, err.Error())
		})
	}

	// A body ConvertOpenAIRequest did not build, such as a passed-through one,
	// is not sent to Pegasus.
	info := newPegasusTestInfo(pegasusTestModel, false)
	_, err := (&Adaptor{}).DoRequest(newAwsTestContext(httptest.NewRecorder(), context.Background()), info, strings.NewReader(pegasusDataVideoRequest(pegasusTestModel, "")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "this request is not supported for TwelveLabs Pegasus on AWS Bedrock")
}

func TestPegasusAnswerBillsBedrockCountsAndVideoSeconds(t *testing.T) {
	info := newPegasusTestInfo(pegasusTestModel, false)

	sentModel, sentBody, recorder, usage, apiErr := runPegasus(t, info, pegasusDataVideoRequest(pegasusTestModel, `"max_tokens":512,"temperature":0,`), func(request *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Amzn-Bedrock-Output-Token-Count": []string{"18"}},
			Body: io.NopCloser(strings.NewReader(`{"input_tokens":108,"output_tokens":18,` +
				`"message":"The video displays a solid blue screen with no visible changes or activity throughout its duration.",` +
				`"stopReason":"stop","outputTokenCount":18,"inputVideoSecond":2}`)),
			Request: request,
		}
	})
	require.Nil(t, apiErr)
	assert.Equal(t, pegasusTestModel, sentModel)
	assert.JSONEq(t, `{"inputPrompt":"Answer briefly.\n\nDescribe this video in one short sentence.","mediaSource":{"base64String":"`+pegasusTestVideoBase64+`"},"maxOutputTokens":512,"temperature":0,"includeMetering":true}`, sentBody)
	assert.Equal(t, 108, usage.PromptTokens)
	assert.Equal(t, 18, usage.CompletionTokens)
	assert.Equal(t, 126, usage.TotalTokens)
	assert.Equal(t, float64(2), usage.InputVideoSeconds)

	var answer dto.OpenAITextResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &answer))
	assert.Equal(t, "chat.completion", answer.Object)
	require.Len(t, answer.Choices, 1)
	assert.Equal(t, "The video displays a solid blue screen with no visible changes or activity throughout its duration.", answer.Choices[0].Message.StringContent())
	assert.Equal(t, "stop", answer.Choices[0].FinishReason)
	assert.Equal(t, 108, answer.Usage.PromptTokens)
	assert.Equal(t, 18, answer.Usage.CompletionTokens)
}

func TestPegasusStreamBillsLastChunkAndVideoSeconds(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() { common.RelayTimeout = originalRelayTimeout })
	info := newPegasusTestInfo(pegasusTestModel, true)
	info.SetEstimatePromptTokens(40)
	chunks := []string{
		`{"message":"","stopReason":""}`,
		`{"message":"","stopReason":""}`,
		`{"message":"","stopReason":""}`,
		`{"message":"The video is ","stopReason":""}`,
		`{"message":"a solid blue screen.","stopReason":""}`,
		`{"message":"","stopReason":"length","outputTokenCount":18,"inputVideoSecond":2,` +
			`"amazon-bedrock-invocationMetrics":{"inputTokenCount":0,"outputTokenCount":18,"invocationLatency":7452,"firstByteLatency":676}}`,
	}

	sentModel, sentBody, recorder, usage, apiErr := runPegasus(t, info, pegasusDataVideoRequest(pegasusTestModel, ""), func(request *http.Request) *http.Response {
		var body bytes.Buffer
		for _, chunk := range chunks {
			require.NoError(t, writeAwsStreamEvent(&body, chunk))
		}
		return newAwsStreamResponse(request, io.NopCloser(bytes.NewReader(body.Bytes())))
	})
	require.Nil(t, apiErr)
	assert.Equal(t, pegasusTestModel, sentModel)
	assert.Contains(t, sentBody, `"includeMetering":true`)
	assert.Equal(t, 40, usage.PromptTokens)
	assert.Equal(t, 18, usage.CompletionTokens)
	assert.Equal(t, 58, usage.TotalTokens)
	assert.Equal(t, float64(2), usage.InputVideoSeconds)

	var text strings.Builder
	var finishReasons []string
	var streamUsage *dto.Usage
	for line := range strings.SplitSeq(recorder.Body.String(), "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "[DONE]" {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.Unmarshal([]byte(data), &chunk))
		for _, choice := range chunk.Choices {
			text.WriteString(choice.Delta.GetContentString())
			if choice.FinishReason != nil {
				finishReasons = append(finishReasons, *choice.FinishReason)
			}
		}
		if chunk.Usage != nil {
			streamUsage = chunk.Usage
		}
	}
	assert.Equal(t, "The video is a solid blue screen.", text.String())
	assert.Equal(t, []string{"length"}, finishReasons)
	require.NotNil(t, streamUsage)
	assert.Equal(t, 18, streamUsage.CompletionTokens)
	assert.Contains(t, recorder.Body.String(), "data: [DONE]")
}

func TestPegasusAlwaysSendsIncludeMetering(t *testing.T) {
	// A parameter override may turn includeMetering off or add Pegasus fields;
	// the fields go through and metering stays on.
	input, err := newPegasusInput(pegasusTestModel, strings.NewReader(`{"inputPrompt":"Describe it.","includeMetering":false,"start_time":0,"end_time":1}`), false)
	require.NoError(t, err)
	invokeInput, ok := input.(*bedrockruntime.InvokeModelInput)
	require.True(t, ok)
	assert.Equal(t, pegasusTestModel, *invokeInput.ModelId)
	assert.JSONEq(t, `{"inputPrompt":"Describe it.","includeMetering":true,"start_time":0,"end_time":1}`, string(invokeInput.Body))
}
