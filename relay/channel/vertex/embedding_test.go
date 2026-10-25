package vertex

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// embedContentServer answers each embedContent call from answers, keyed by the input text, and records
// the calls in the order they arrive.
type embedContentServer struct {
	*httptest.Server
	mu    sync.Mutex
	paths []string
	calls []string
}

func newEmbedContentServer(t *testing.T, answers map[string]string) *embedContentServer {
	server := &embedContentServer{}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var request struct {
			Content dto.GeminiChatContent `json:"content"`
		}
		require.NoError(t, common.Unmarshal(body, &request))
		server.mu.Lock()
		server.paths = append(server.paths, r.URL.Path+"?"+r.URL.RawQuery)
		server.calls = append(server.calls, string(body))
		server.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		answer := answers[request.Content.Parts[0].Text]
		if strings.Contains(answer, `"error"`) {
			w.WriteHeader(http.StatusBadRequest)
		}
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)
	return server
}

func vertexEmbeddingInfo(baseURL, model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeEmbeddings,
		RelayFormat:     types.RelayFormatEmbedding,
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiVersion:        "global",
			ApiKey:            "test-key",
			ChannelBaseUrl:    baseURL,
			UpstreamModelName: model,
			ChannelOtherSettings: dto.ChannelOtherSettings{
				VertexKeyType:   dto.VertexKeyTypeAPIKey,
				VertexProjectID: "my-project",
			},
		},
	}
}

// relayVertexEmbeddings takes the adaptor through the steps of the embeddings relay: convert, request,
// the upstream error on a status that is not 200, and the answer.
func relayVertexEmbeddings(t *testing.T, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (*httptest.ResponseRecorder, *dto.Usage, *types.NewAPIError) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)

	adaptor := &Adaptor{}
	adaptor.Init(info)
	converted, err := adaptor.ConvertEmbeddingRequest(c, info, request)
	if err != nil {
		return recorder, nil, types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	resp, err := adaptor.DoRequest(c, info, strings.NewReader(string(body)))
	require.NoError(t, err)
	httpResp := resp.(*http.Response)
	if httpResp.StatusCode != http.StatusOK {
		return recorder, nil, service.RelayErrorHandler(c.Request.Context(), httpResp, false)
	}
	usage, apiErr := adaptor.DoResponse(c, httpResp, info)
	if apiErr != nil {
		return recorder, nil, apiErr
	}
	return recorder, usage.(*dto.Usage), nil
}

func TestVertexEmbeddingsCallEmbedContentPerInput(t *testing.T) {
	server := newEmbedContentServer(t, map[string]string{
		"hello world": `{"embedding":{"values":[0.25,-0.5]},"usageMetadata":{"promptTokenCount":2,"totalTokenCount":2,"trafficType":"ON_DEMAND"}}`,
		"first":       `{"embedding":{"values":[1,0]},"usageMetadata":{"promptTokenCount":1,"totalTokenCount":1}}`,
		"second one":  `{"embedding":{"values":[0,1]},"usageMetadata":{"promptTokenCount":2,"totalTokenCount":2}}`,
		"third":       `{"embedding":{"values":[1,1]},"usageMetadata":{"promptTokenCount":3,"totalTokenCount":3}}`,
		"vector":      `{"embedding":{"values":[1.5,-2]},"usageMetadata":{"promptTokenCount":1,"totalTokenCount":1}}`,
		"no usage":    `{"embedding":{"values":[0.5]}}`,
	})
	const path = "/v1/projects/my-project/locations/global/publishers/google/models/gemini-embedding-001:embedContent?key=test-key"

	tests := []struct {
		name      string
		request   dto.EmbeddingRequest
		wantCalls []string
		wantData  string
		wantUsage int
	}{
		{
			name:      "string input",
			request:   dto.EmbeddingRequest{Input: "hello world"},
			wantCalls: []string{`{"content":{"parts":[{"text":"hello world"}]}}`},
			wantData:  `[{"object":"embedding","index":0,"embedding":[0.25,-0.5]}]`,
			wantUsage: 2,
		},
		{
			name:    "list in order with dimensions",
			request: dto.EmbeddingRequest{Input: []any{"first", "second one", "third"}, Dimensions: common.GetPointer(8)},
			wantCalls: []string{
				`{"content":{"parts":[{"text":"first"}]},"outputDimensionality":8}`,
				`{"content":{"parts":[{"text":"second one"}]},"outputDimensionality":8}`,
				`{"content":{"parts":[{"text":"third"}]},"outputDimensionality":8}`,
			},
			wantData: `[{"object":"embedding","index":0,"embedding":[1,0]},` +
				`{"object":"embedding","index":1,"embedding":[0,1]},` +
				`{"object":"embedding","index":2,"embedding":[1,1]}]`,
			wantUsage: 6,
		},
		{
			name:      "base64 encoding",
			request:   dto.EmbeddingRequest{Input: "vector", EncodingFormat: "base64"},
			wantCalls: []string{`{"content":{"parts":[{"text":"vector"}]}}`},
			// 1.5 and -2 as little-endian float32.
			wantData:  `[{"object":"embedding","index":0,"embedding":"AADAPwAAAMA="}]`,
			wantUsage: 1,
		},
		{
			// An answer without a token count bills the request's estimate, never nothing.
			name:      "no usage reported",
			request:   dto.EmbeddingRequest{Input: "no usage"},
			wantCalls: []string{`{"content":{"parts":[{"text":"no usage"}]}}`},
			wantData:  `[{"object":"embedding","index":0,"embedding":[0.5]}]`,
			wantUsage: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.paths, server.calls = nil, nil
			tt.request.Model = "gemini-embedding-001"

			info := vertexEmbeddingInfo(server.URL, "gemini-embedding-001")
			info.SetEstimatePromptTokens(3)
			recorder, usage, apiErr := relayVertexEmbeddings(t, info, tt.request)
			require.Nil(t, apiErr)

			require.Len(t, server.calls, len(tt.wantCalls))
			for i, want := range tt.wantCalls {
				assert.JSONEq(t, want, server.calls[i])
				assert.Equal(t, path, server.paths[i])
			}
			assert.Equal(t, tt.wantUsage, usage.PromptTokens)
			assert.Equal(t, tt.wantUsage, usage.TotalTokens)

			var answer map[string]any
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &answer))
			assert.Equal(t, "list", answer["object"])
			assert.Equal(t, "gemini-embedding-001", answer["model"])
			data, err := common.Marshal(answer["data"])
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantData, string(data))
			answerUsage, ok := answer["usage"].(map[string]any)
			require.True(t, ok)
			assert.EqualValues(t, tt.wantUsage, answerUsage["prompt_tokens"])
			assert.EqualValues(t, tt.wantUsage, answerUsage["total_tokens"])
		})
	}
}

func TestVertexEmbeddingsRefusedBeforeAnyCall(t *testing.T) {
	server := newEmbedContentServer(t, map[string]string{})
	tooMany := make([]any, 101)
	for i := range tooMany {
		tooMany[i] = "text"
	}

	tests := []struct {
		name       string
		model      string
		input      any
		wantStatus int
		wantError  string
	}{
		{"token array", "gemini-embedding-001", []any{float64(9906), float64(1917)}, http.StatusBadRequest, "vertex embeddings take text inputs, not token arrays"},
		{"list of token arrays", "gemini-embedding-001", []any{[]any{float64(9906)}}, http.StatusBadRequest, "vertex embeddings take text inputs, not token arrays"},
		{"more inputs than the bound", "gemini-embedding-001", tooMany, http.StatusBadRequest, "vertex embeddings take at most 100 inputs in one request, got 101"},
		// Any other model keeps the answer it had.
		{"chat model", "gemini-2.5-flash", "hello", http.StatusInternalServerError, "not implemented"},
		{"text embedding model", "text-embedding-005", "hello", http.StatusInternalServerError, "not implemented"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, apiErr := relayVertexEmbeddings(t, vertexEmbeddingInfo(server.URL, tt.model), dto.EmbeddingRequest{Model: tt.model, Input: tt.input})
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.StatusCode)
			assert.Equal(t, tt.wantError, apiErr.Error())
			assert.Empty(t, server.calls)
		})
	}
}

func TestVertexEmbeddingsFailWithTheFailedCall(t *testing.T) {
	server := newEmbedContentServer(t, map[string]string{
		"first":  `{"embedding":{"values":[1,0]},"usageMetadata":{"promptTokenCount":1,"totalTokenCount":1}}`,
		"second": `{"error":{"code":400,"message":"Request contains an invalid argument.","status":"INVALID_ARGUMENT"}}`,
		"third":  `{"embedding":{"values":[1,1]},"usageMetadata":{"promptTokenCount":1,"totalTokenCount":1}}`,
	})

	recorder, usage, apiErr := relayVertexEmbeddings(t, vertexEmbeddingInfo(server.URL, "gemini-embedding-001"),
		dto.EmbeddingRequest{Model: "gemini-embedding-001", Input: []any{"first", "second", "third"}})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Contains(t, apiErr.Error(), "Request contains an invalid argument.")
	assert.Nil(t, usage)
	assert.Empty(t, recorder.Body.String())
	// The calls stop at the failed one.
	assert.Len(t, server.calls, 2)
}
