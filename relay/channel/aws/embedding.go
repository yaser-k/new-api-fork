package aws

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// The Bedrock embedding models this adaptor knows, by model id without any
// inference profile prefix or version suffix.
const (
	awsEmbeddingCohereV4    = "cohere.embed-v4"
	awsEmbeddingTitanTextV2 = "amazon.titan-embed-text-v2"
)

// cohereEmbedDefaultInputType is sent as Cohere's input_type, which has no
// OpenAI field; a channel's parameter override can replace it.
const cohereEmbedDefaultInputType = "search_document"

// awsInputTokenCountHeader carries the input token count of an InvokeModel
// answer; Cohere Embed returns its count only here.
const awsInputTokenCountHeader = "X-Amzn-Bedrock-Input-Token-Count"

type awsCohereEmbeddingRequest struct {
	Texts           []string `json:"texts"`
	InputType       string   `json:"input_type"`
	EmbeddingTypes  []string `json:"embedding_types"`
	OutputDimension *int     `json:"output_dimension,omitempty"`
}

type awsCohereEmbeddingResponse struct {
	Embeddings struct {
		Float [][]float64 `json:"float"`
	} `json:"embeddings"`
}

// awsTitanEmbeddingRequest is Titan's body, except that InputText may hold a
// list: Titan takes one string per call, so DoRequest sends one call per text.
type awsTitanEmbeddingRequest struct {
	InputText  any  `json:"inputText"`
	Dimensions *int `json:"dimensions,omitempty"`
	Normalize  bool `json:"normalize"`
}

type awsTitanEmbeddingResponse struct {
	Embedding           []float64 `json:"embedding"`
	InputTextTokenCount int       `json:"inputTextTokenCount"`
}

func newAwsEmbeddingRequestError(message string) error {
	return types.NewErrorWithStatusCode(errors.New(message), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}

// awsEmbeddingModel returns which known embedding model a Bedrock model id
// names, read after any ARN path and inference profile prefix (global., eu.,
// us. ...). An id it does not know is refused rather than sent a guessed body.
func awsEmbeddingModel(modelId string) (string, error) {
	name := modelId[strings.LastIndex(modelId, "/")+1:]
	_, unprefixed, _ := strings.Cut(name, ".")
	for _, model := range []string{awsEmbeddingCohereV4, awsEmbeddingTitanTextV2} {
		for _, candidate := range []string{name, unprefixed} {
			if candidate == model || strings.HasPrefix(candidate, model+":") {
				return model, nil
			}
		}
	}
	return "", newAwsEmbeddingRequestError(fmt.Sprintf("embedding model %q is not supported on AWS Bedrock; supported models: %s, %s", modelId, awsEmbeddingCohereV4, awsEmbeddingTitanTextV2))
}

// convertToAwsEmbeddingRequest converts an OpenAI embedding request to the
// InvokeModel body of the Bedrock model it names.
func convertToAwsEmbeddingRequest(modelId string, request dto.EmbeddingRequest) (any, error) {
	model, err := awsEmbeddingModel(modelId)
	if err != nil {
		return nil, err
	}
	switch request.EncodingFormat {
	case "", "float", "base64":
	default:
		return nil, newAwsEmbeddingRequestError(fmt.Sprintf("encoding_format %q is not supported; use float or base64", request.EncodingFormat))
	}

	var texts []string
	switch input := request.Input.(type) {
	case string:
		texts = []string{input}
	case []any:
		for i, item := range input {
			text, ok := item.(string)
			if !ok {
				return nil, newAwsEmbeddingRequestError(fmt.Sprintf("input[%d] must be a string; token arrays are not supported on AWS Bedrock", i))
			}
			texts = append(texts, text)
		}
	default:
		return nil, newAwsEmbeddingRequestError("input must be a string or an array of strings")
	}
	if len(texts) == 0 {
		return nil, newAwsEmbeddingRequestError("input must not be empty")
	}

	if model == awsEmbeddingCohereV4 {
		return &awsCohereEmbeddingRequest{
			Texts:           texts,
			InputType:       cohereEmbedDefaultInputType,
			EmbeddingTypes:  []string{"float"},
			OutputDimension: request.Dimensions,
		}, nil
	}
	titanRequest := &awsTitanEmbeddingRequest{InputText: texts, Dimensions: request.Dimensions, Normalize: true}
	if text, ok := request.Input.(string); ok {
		titanRequest.InputText = text
	}
	return titanRequest, nil
}

// doAwsEmbeddingRequest builds the InvokeModel calls of an embeddings request
// with the channel's SDK client: one call for Cohere Embed, one per text for
// Titan. A key of <api-key>|<region> takes the client's bearer token path
// whichever key type the channel is set to, so the API-key mode needs no
// HTTP call of its own.
func doAwsEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor, requestBody io.Reader) error {
	if info.ChannelOtherSettings.AwsApiType == dto.AwsApiTypeMantle {
		return newAwsEmbeddingRequestError("embeddings are not available through the AWS Mantle API; use the InvokeModel API type")
	}
	awsCli, err := newAwsClient(c, info)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelAwsClientError)
	}
	a.AwsClient = awsCli

	awsModelId := getAwsModelID(info.UpstreamModelName)
	awsRegionPrefix := getAwsRegionPrefix(awsCli.Options().Region)
	if awsModelCanCrossRegion(awsModelId, awsRegionPrefix) {
		awsModelId = awsModelCrossRegion(awsModelId, awsRegionPrefix)
	}
	model, err := awsEmbeddingModel(awsModelId)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(requestBody)
	if err != nil {
		return types.NewError(errors.Wrap(err, "read embedding request"), types.ErrorCodeReadRequestBodyFailed)
	}
	bodies := [][]byte{body}
	if model == awsEmbeddingTitanTextV2 {
		// Other fields, including any a parameter override added, go with each text.
		var titanBody map[string]json.RawMessage
		if err := common.Unmarshal(body, &titanBody); err != nil {
			return types.NewErrorWithStatusCode(errors.Wrap(err, "decode embedding request"), types.ErrorCodeBadRequestBody, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		if common.GetJsonType(titanBody["inputText"]) == "array" {
			var texts []string
			if err := common.Unmarshal(titanBody["inputText"], &texts); err != nil {
				return newAwsEmbeddingRequestError("inputText must be a string or an array of strings")
			}
			if len(texts) == 0 {
				return newAwsEmbeddingRequestError("input must not be empty")
			}
			bodies = make([][]byte, 0, len(texts))
			for _, text := range texts {
				if titanBody["inputText"], err = common.Marshal(text); err != nil {
					return types.NewError(errors.Wrap(err, "marshal embedding request"), types.ErrorCodeJsonMarshalFailed)
				}
				textBody, err := common.Marshal(titanBody)
				if err != nil {
					return types.NewError(errors.Wrap(err, "marshal embedding request"), types.ErrorCodeJsonMarshalFailed)
				}
				bodies = append(bodies, textBody)
			}
		}
	}

	inputs := make([]*bedrockruntime.InvokeModelInput, 0, len(bodies))
	for _, body := range bodies {
		inputs = append(inputs, &bedrockruntime.InvokeModelInput{
			ModelId:     aws.String(awsModelId),
			Accept:      aws.String("application/json"),
			ContentType: aws.String("application/json"),
			Body:        body,
		})
	}
	a.AwsReq = inputs
	return nil
}

// awsInputTokenCount reads the input token count Bedrock sends in the
// X-Amzn-Bedrock-Input-Token-Count header. InvokeModelOutput has no field for
// it, so it is read from the raw HTTP response the SDK keeps in the result
// metadata. A missing, zero or out-of-range count is not reported.
func awsInputTokenCount(metadata middleware.Metadata) (int, bool) {
	rawResponse, ok := awsmiddleware.GetRawResponse(metadata).(*smithyhttp.Response)
	if !ok || rawResponse == nil || rawResponse.Response == nil {
		return 0, false
	}
	count, err := strconv.ParseInt(strings.TrimSpace(rawResponse.Header.Get(awsInputTokenCountHeader)), 10, 32)
	if err != nil || count <= 0 {
		return 0, false
	}
	return int(count), true
}

// awsEmbeddingHandler sends the calls DoRequest built, in order, and answers an
// OpenAI embedding list billed on the input tokens Bedrock counted: Cohere's
// from the response header, Titan's from each body (the header if a body has
// none), summed over the texts. Without a count for every call the request is
// billed on the local estimate of its input.
func awsEmbeddingHandler(c *gin.Context, info *relaycommon.RelayInfo, a *Adaptor) (*types.NewAPIError, *dto.Usage) {
	inputs, ok := a.AwsReq.([]*bedrockruntime.InvokeModelInput)
	if !ok || len(inputs) == 0 {
		return types.NewError(errors.New("no AWS embedding request was built"), types.ErrorCodeInvalidRequest), nil
	}
	requestContext := c.Request.Context()
	ctx, cancel := newAwsInvokeContext(requestContext)
	defer cancel()

	model, err := awsEmbeddingModel(aws.ToString(inputs[0].ModelId))
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest), nil
	}

	var vectors [][]float64
	promptTokens := 0
	counted := true
	for _, input := range inputs {
		awsResp, err := a.AwsClient.InvokeModel(ctx, input)
		if err != nil {
			return newAwsInvokeError(requestContext, err, "InvokeModel"), nil
		}
		tokenCount, hasCount := awsInputTokenCount(awsResp.ResultMetadata)
		if model == awsEmbeddingCohereV4 {
			var cohereResp awsCohereEmbeddingResponse
			if err := common.Unmarshal(awsResp.Body, &cohereResp); err != nil {
				return types.NewOpenAIError(errors.Wrap(err, "unmarshal cohere embedding response"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError), nil
			}
			var sent awsCohereEmbeddingRequest
			if err := common.Unmarshal(input.Body, &sent); err != nil || len(cohereResp.Embeddings.Float) != len(sent.Texts) {
				return types.NewOpenAIError(fmt.Errorf("cohere embedding response has %d float embeddings for %d texts", len(cohereResp.Embeddings.Float), len(sent.Texts)), types.ErrorCodeBadResponseBody, http.StatusInternalServerError), nil
			}
			vectors = append(vectors, cohereResp.Embeddings.Float...)
		} else {
			var titanResp awsTitanEmbeddingResponse
			if err := common.Unmarshal(awsResp.Body, &titanResp); err != nil {
				return types.NewOpenAIError(errors.Wrap(err, "unmarshal titan embedding response"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError), nil
			}
			if len(titanResp.Embedding) == 0 {
				return types.NewOpenAIError(errors.New("titan embedding response has no embedding"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError), nil
			}
			vectors = append(vectors, titanResp.Embedding)
			if titanResp.InputTextTokenCount > 0 && titanResp.InputTextTokenCount <= math.MaxInt32 {
				tokenCount, hasCount = titanResp.InputTextTokenCount, true
			}
		}
		counted = counted && hasCount
		promptTokens = min(promptTokens+tokenCount, math.MaxInt32)
	}

	usage := &dto.Usage{PromptTokens: promptTokens, TotalTokens: promptTokens}
	if !counted {
		usage = service.ResponseText2Usage(c, "", info.UpstreamModelName, info.GetEstimatePromptTokens())
	}

	encodeBase64 := false
	if request, ok := info.Request.(*dto.EmbeddingRequest); ok {
		encodeBase64 = request.EncodingFormat == "base64"
	}
	response := dto.FlexibleEmbeddingResponse{
		Object: "list",
		Data:   make([]dto.FlexibleEmbeddingResponseItem, 0, len(vectors)),
		Model:  info.UpstreamModelName,
		Usage:  *usage,
	}
	for i, vector := range vectors {
		item := dto.FlexibleEmbeddingResponseItem{Object: "embedding", Index: i, Embedding: vector}
		if encodeBase64 {
			// OpenAI's base64 encoding: the vector as little-endian float32.
			packed := make([]byte, 0, 4*len(vector))
			for _, value := range vector {
				packed = binary.LittleEndian.AppendUint32(packed, math.Float32bits(float32(value)))
			}
			item.Embedding = base64.StdEncoding.EncodeToString(packed)
		}
		response.Data = append(response.Data, item)
	}
	responseBody, err := common.Marshal(response)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError), nil
	}
	c.Data(http.StatusOK, "application/json", responseBody)
	return nil, usage
}
