package vertex

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// Vertex AI serves Gemini embedding models through embedContent only: it has no
// batchEmbedContents. An OpenAI embeddings request is answered by one embedContent call per
// input, made in turn, in the input's order.

// maxEmbedContentInputs bounds the inputs of one embeddings request, since each input is an
// upstream call. It is the request limit of one batchEmbedContents call on the Gemini API.
const maxEmbedContentInputs = 100

type embedContentRequest struct {
	Content              dto.GeminiChatContent `json:"content"`
	OutputDimensionality *int                  `json:"outputDimensionality,omitempty"`
}

// embedContentBatch is the converted embeddings request: the body of each embedContent call.
type embedContentBatch struct {
	Requests []embedContentRequest `json:"requests"`
}

type embedContentResponse struct {
	Embedding     dto.ContentEmbedding    `json:"embedding"`
	UsageMetadata dto.GeminiUsageMetadata `json:"usageMetadata"`
}

// doEmbedContentCalls sends each body of the converted request as its own embedContent call and
// answers with the calls' bodies as one JSON array. The first call that is not a 200 is returned
// as it is, so the request fails with that call's error and nothing partial is answered.
func (a *Adaptor) doEmbedContentCalls(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	// The bodies stay raw so that fields a parameter override added reach the upstream.
	var batch struct {
		Requests []json.RawMessage `json:"requests"`
	}
	if err := common.DecodeJson(requestBody, &batch); err != nil {
		return nil, fmt.Errorf("decode embedContent requests: %w", err)
	}

	answers := make([]json.RawMessage, 0, len(batch.Requests))
	for _, body := range batch.Requests {
		resp, err := channel.DoApiRequest(a, c, info, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return resp, nil
		}
		answer, err := io.ReadAll(resp.Body)
		service.CloseResponseBodyGracefully(resp)
		if err != nil {
			return nil, fmt.Errorf("read embedContent response: %w", err)
		}
		answers = append(answers, answer)
	}

	combined, err := common.Marshal(answers)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(combined)),
	}, nil
}

// embedContentHandler answers the embedContent calls as OpenAI's embeddings list, with the prompt
// tokens the calls report as the request's usage.
func embedContentHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, encodingFormat string) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	var responses []embedContentResponse
	if err := common.DecodeJson(resp.Body, &responses); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	openAIResponse := dto.FlexibleEmbeddingResponse{
		Object: "list",
		Data:   make([]dto.FlexibleEmbeddingResponseItem, 0, len(responses)),
		Model:  info.UpstreamModelName,
	}
	promptTokens := 0
	for i, response := range responses {
		values := response.Embedding.Values
		if len(values) == 0 {
			return nil, types.NewOpenAIError(errors.New("embedContent answered without an embedding"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		var embedding any = values
		if encodingFormat == "base64" {
			// OpenAI's base64 encoding of a vector: its values as little-endian float32.
			raw := make([]byte, 4*len(values))
			for j, value := range values {
				binary.LittleEndian.PutUint32(raw[4*j:], math.Float32bits(float32(value)))
			}
			embedding = base64.StdEncoding.EncodeToString(raw)
		}
		openAIResponse.Data = append(openAIResponse.Data, dto.FlexibleEmbeddingResponseItem{
			Object:    "embedding",
			Index:     i,
			Embedding: embedding,
		})
		// Upstream counts are bounded so their sum cannot wrap.
		count := min(max(response.UsageMetadata.PromptTokenCount, 0), math.MaxInt32)
		promptTokens = min(promptTokens+count, math.MaxInt32)
	}
	if promptTokens == 0 {
		promptTokens = info.GetEstimatePromptTokens()
	}
	usage := &dto.Usage{PromptTokens: promptTokens, TotalTokens: promptTokens}
	openAIResponse.Usage = *usage

	body, err := common.Marshal(openAIResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, body)
	return usage, nil
}
