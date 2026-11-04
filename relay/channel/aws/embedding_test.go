package aws

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	relaytypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// awsEmbeddingUpstreamCall is one InvokeModel call the fake Bedrock received.
type awsEmbeddingUpstreamCall struct {
	modelId string
	body    string
}

// awsEmbeddingUpstreamAnswer is what the fake Bedrock answers one call with.
type awsEmbeddingUpstreamAnswer struct {
	tokenCount string
	body       string
}

func newAwsEmbeddingTestInfo(model string, request *dto.EmbeddingRequest) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeEmbeddings,
		RelayFormat:     relaytypes.RelayFormatEmbedding,
		OriginModelName: model,
		Request:         request,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "access-key|secret-key|us-east-1",
			UpstreamModelName: model,
		},
	}
}

// runAwsEmbedding relays an embedding request through the adaptor against a
// fake Bedrock that answers each call in turn.
func runAwsEmbedding(t *testing.T, info *relaycommon.RelayInfo, answers []awsEmbeddingUpstreamAnswer) ([]awsEmbeddingUpstreamCall, *httptest.ResponseRecorder, any, *relaytypes.NewAPIError) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c := newAwsTestContext(recorder, context.Background())
	a := &Adaptor{}
	a.Init(info)

	converted, err := a.ConvertEmbeddingRequest(c, info, *info.Request.(*dto.EmbeddingRequest))
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	response, err := a.DoRequest(c, info, strings.NewReader(string(body)))
	require.NoError(t, err)
	require.Nil(t, response)

	var calls []awsEmbeddingUpstreamCall
	a.AwsClient = newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		requestBody, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		modelId, err := url.PathUnescape(strings.TrimSuffix(strings.TrimPrefix(request.URL.EscapedPath(), "/model/"), "/invoke"))
		require.NoError(t, err)
		calls = append(calls, awsEmbeddingUpstreamCall{modelId: modelId, body: string(requestBody)})
		require.LessOrEqual(t, len(calls), len(answers))
		answer := answers[len(calls)-1]
		header := http.Header{"Content-Type": []string{"application/json"}}
		if answer.tokenCount != "" {
			header.Set("X-Amzn-Bedrock-Input-Token-Count", answer.tokenCount)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     header,
			Body:       io.NopCloser(strings.NewReader(answer.body)),
			Request:    request,
		}, nil
	}))
	usage, apiErr := a.DoResponse(c, nil, info)
	return calls, recorder, usage, apiErr
}

// assertAwsEmbeddingAnswer checks the OpenAI embedding list the client got:
// one embedding object per input, in input order, and its token usage.
func assertAwsEmbeddingAnswer(t *testing.T, recorder *httptest.ResponseRecorder, model string, vectors [][]float64, promptTokens int) {
	t.Helper()
	assert.Equal(t, http.StatusOK, recorder.Code)
	var answer dto.OpenAIEmbeddingResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &answer))
	assert.Equal(t, "list", answer.Object)
	assert.Equal(t, model, answer.Model)
	require.Len(t, answer.Data, len(vectors))
	for i, vector := range vectors {
		assert.Equal(t, dto.OpenAIEmbeddingResponseItem{Object: "embedding", Index: i, Embedding: vector}, answer.Data[i])
	}
	assert.Equal(t, promptTokens, answer.PromptTokens)
	assert.Equal(t, promptTokens, answer.TotalTokens)
}

func TestAwsEmbeddingRequestBodies(t *testing.T) {
	dimensions := 256
	for _, test := range []struct {
		name       string
		model      string
		input      any
		dimensions *int
		want       string
	}{
		{"cohere string", "global.cohere.embed-v4:0", "hello world", nil,
			`{"texts":["hello world"],"input_type":"search_document","embedding_types":["float"]}`},
		{"cohere array with dimensions", "eu.cohere.embed-v4:0", []any{"hello world", "a second short text"}, &dimensions,
			`{"texts":["hello world","a second short text"],"input_type":"search_document","embedding_types":["float"],"output_dimension":256}`},
		{"titan string with dimensions", "amazon.titan-embed-text-v2:0", "hello world", &dimensions,
			`{"inputText":"hello world","dimensions":256,"normalize":true}`},
		{"titan array", "amazon.titan-embed-text-v2:0", []any{"a", "b"}, nil,
			`{"inputText":["a","b"],"normalize":true}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := dto.EmbeddingRequest{Model: test.model, Input: test.input, Dimensions: test.dimensions}
			converted, err := (&Adaptor{}).ConvertEmbeddingRequest(nil, newAwsEmbeddingTestInfo(test.model, &request), request)
			require.NoError(t, err)
			body, err := common.Marshal(converted)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(body))
		})
	}
}

func TestAwsEmbeddingRefusesWhatItCannotSend(t *testing.T) {
	for _, test := range []struct {
		name    string
		request dto.EmbeddingRequest
		message string
	}{
		{"unknown embedding model", dto.EmbeddingRequest{Model: "cohere.embed-english-v3", Input: "hi"},
			`embedding model "cohere.embed-english-v3" is not supported on AWS Bedrock; supported models: cohere.embed-v4, amazon.titan-embed-text-v2`},
		{"chat model", dto.EmbeddingRequest{Model: "us.anthropic.claude-sonnet-4-6", Input: "hi"},
			`embedding model "us.anthropic.claude-sonnet-4-6" is not supported on AWS Bedrock; supported models: cohere.embed-v4, amazon.titan-embed-text-v2`},
		{"token array", dto.EmbeddingRequest{Model: "global.cohere.embed-v4:0", Input: []any{[]any{float64(1), float64(2)}}},
			"input[0] must be a string; token arrays are not supported on AWS Bedrock"},
		{"empty array", dto.EmbeddingRequest{Model: "amazon.titan-embed-text-v2:0", Input: []any{}},
			"input must not be empty"},
		{"encoding format", dto.EmbeddingRequest{Model: "amazon.titan-embed-text-v2:0", Input: "hi", EncodingFormat: "int8"},
			`encoding_format "int8" is not supported; use float or base64`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := (&Adaptor{}).ConvertEmbeddingRequest(nil, newAwsEmbeddingTestInfo(test.request.Model, &test.request), test.request)
			require.Error(t, err)
			apiErr := relaytypes.NewError(err, relaytypes.ErrorCodeConvertRequestFailed)
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.Equal(t, test.message, err.Error())
		})
	}

	info := newAwsEmbeddingTestInfo("amazon.titan-embed-text-v2:0", &dto.EmbeddingRequest{Input: "hi"})
	info.ChannelOtherSettings.AwsApiType = dto.AwsApiTypeMantle
	_, err := (&Adaptor{}).DoRequest(newAwsTestContext(httptest.NewRecorder(), context.Background()), info, strings.NewReader(`{"inputText":"hi","normalize":true}`))
	require.Error(t, err)
	assert.Equal(t, "embeddings are not available through the AWS Mantle API; use the InvokeModel API type", err.Error())
}

func TestAwsEmbeddingCohereBillsHeaderCountInInputOrder(t *testing.T) {
	for _, keyType := range []dto.AwsKeyType{"", dto.AwsKeyTypeApiKey} {
		t.Run(string(keyType), func(t *testing.T) {
			dimensions := 256
			info := newAwsEmbeddingTestInfo("global.cohere.embed-v4:0", &dto.EmbeddingRequest{
				Model:      "global.cohere.embed-v4:0",
				Input:      []any{"hello world", "a second short text"},
				Dimensions: &dimensions,
			})
			if keyType == dto.AwsKeyTypeApiKey {
				info.ApiKey = "bedrock-api-key|us-east-1"
				info.ChannelOtherSettings.AwsKeyType = keyType
			}

			calls, recorder, usage, apiErr := runAwsEmbedding(t, info, []awsEmbeddingUpstreamAnswer{{
				tokenCount: "6",
				body:       `{"id":"x","texts":["hello world","a second short text"],"embeddings":{"float":[[0.1,0.2],[0.3,0.4]]},"response_type":"embeddings_by_type"}`,
			}})
			require.Nil(t, apiErr)
			require.Len(t, calls, 1)
			assert.Equal(t, "global.cohere.embed-v4:0", calls[0].modelId)
			assert.JSONEq(t, `{"texts":["hello world","a second short text"],"input_type":"search_document","embedding_types":["float"],"output_dimension":256}`, calls[0].body)
			assert.Equal(t, &dto.Usage{PromptTokens: 6, TotalTokens: 6}, usage)
			assertAwsEmbeddingAnswer(t, recorder, "global.cohere.embed-v4:0", [][]float64{{0.1, 0.2}, {0.3, 0.4}}, 6)
		})
	}
}

func TestAwsEmbeddingTitanSendsOneCallPerTextAndSumsBodyCounts(t *testing.T) {
	dimensions := 256
	info := newAwsEmbeddingTestInfo("amazon.titan-embed-text-v2:0", &dto.EmbeddingRequest{
		Model:      "amazon.titan-embed-text-v2:0",
		Input:      []any{"hello world", "hi"},
		Dimensions: &dimensions,
	})

	calls, recorder, usage, apiErr := runAwsEmbedding(t, info, []awsEmbeddingUpstreamAnswer{
		{tokenCount: "3", body: `{"embedding":[0.1,0.2],"embeddingsByType":{"float":[0.1,0.2]},"inputTextTokenCount":3}`},
		{body: `{"embedding":[0.3,0.4],"embeddingsByType":{"float":[0.3,0.4]},"inputTextTokenCount":2}`},
	})
	require.Nil(t, apiErr)
	require.Len(t, calls, 2)
	assert.Equal(t, "amazon.titan-embed-text-v2:0", calls[0].modelId)
	assert.JSONEq(t, `{"inputText":"hello world","dimensions":256,"normalize":true}`, calls[0].body)
	assert.JSONEq(t, `{"inputText":"hi","dimensions":256,"normalize":true}`, calls[1].body)
	assert.Equal(t, &dto.Usage{PromptTokens: 5, TotalTokens: 5}, usage)
	assertAwsEmbeddingAnswer(t, recorder, "amazon.titan-embed-text-v2:0", [][]float64{{0.1, 0.2}, {0.3, 0.4}}, 5)
}

func TestAwsEmbeddingWithoutUpstreamCountBillsLocalEstimate(t *testing.T) {
	info := newAwsEmbeddingTestInfo("global.cohere.embed-v4:0", &dto.EmbeddingRequest{Model: "global.cohere.embed-v4:0", Input: "hello world"})
	info.SetEstimatePromptTokens(4)

	_, _, usage, apiErr := runAwsEmbedding(t, info, []awsEmbeddingUpstreamAnswer{
		{tokenCount: "-1", body: `{"embeddings":{"float":[[0.1,0.2]]}}`},
	})
	require.Nil(t, apiErr)
	assert.Equal(t, &dto.Usage{PromptTokens: 4, TotalTokens: 4}, usage)
}

func TestAwsEmbeddingRefusesAnswerMissingAnEmbedding(t *testing.T) {
	info := newAwsEmbeddingTestInfo("global.cohere.embed-v4:0", &dto.EmbeddingRequest{Model: "global.cohere.embed-v4:0", Input: []any{"a", "b"}})

	_, _, usage, apiErr := runAwsEmbedding(t, info, []awsEmbeddingUpstreamAnswer{
		{tokenCount: "2", body: `{"embeddings":{"int8":[[1],[2]]}}`},
	})
	require.NotNil(t, apiErr)
	assert.Nil(t, usage)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	assert.Equal(t, "cohere embedding response has 0 float embeddings for 2 texts", apiErr.Error())
}

func TestAwsEmbeddingEncodesBase64AsLittleEndianFloat32(t *testing.T) {
	info := newAwsEmbeddingTestInfo("amazon.titan-embed-text-v2:0", &dto.EmbeddingRequest{
		Model:          "amazon.titan-embed-text-v2:0",
		Input:          "hello world",
		EncodingFormat: "base64",
	})

	_, recorder, _, apiErr := runAwsEmbedding(t, info, []awsEmbeddingUpstreamAnswer{
		{body: `{"embedding":[0.5,-1.25],"inputTextTokenCount":3}`},
	})
	require.Nil(t, apiErr)
	var response struct {
		Data []struct {
			Embedding string `json:"embedding"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Data, 1)
	packed, err := base64.StdEncoding.DecodeString(response.Data[0].Embedding)
	require.NoError(t, err)
	require.Len(t, packed, 8)
	assert.Equal(t, float32(0.5), math.Float32frombits(binary.LittleEndian.Uint32(packed[0:])))
	assert.Equal(t, float32(-1.25), math.Float32frombits(binary.LittleEndian.Uint32(packed[4:])))
}
