package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func ImageHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)
	info.BillingImageCount = nil
	info.ImageRequestCount = 0

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)
	imageCount, err := request.ImageCount(false)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	var requestBody io.Reader
	var jsonData []byte

	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
			requestBody = common.NewReplayableBodyReader(storage)
		} else {
			jsonData, err = storage.Bytes()
			if err != nil {
				return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
		}
	} else {
		convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
		if err != nil {
			// An adaptor that already classified its rejection (status code
			// and retry policy) keeps that classification instead of being
			// downgraded to a retryable conversion failure.
			var apiErr *types.NewAPIError
			if errors.As(err, &apiErr) {
				return apiErr
			}
			return types.NewError(err, types.ErrorCodeConvertRequestFailed)
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

		switch convertedRequest.(type) {
		case *bytes.Buffer:
			requestBody = convertedRequest.(io.Reader)
		default:
			jsonData, err = common.Marshal(convertedRequest)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}

			// apply param override
			if len(info.ParamOverride) > 0 {
				jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
				if err != nil {
					return newAPIErrorFromParamOverride(err)
				}
			}

		}
	}
	if jsonData != nil {
		// This is a different trust boundary from ingress: channel overrides
		// and pass-through bodies can change the quantity actually submitted.
		var outbound struct {
			N *uint `json:"n"`
		}
		if err := common.Unmarshal(jsonData, &outbound); err != nil {
			return types.NewErrorWithStatusCode(fmt.Errorf("invalid image billing parameters: %w", err), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		quantityRequest := dto.ImageRequest{N: outbound.N}
		if quantityRequest.N == nil {
			quantityRequest.N = common.GetPointer(uint(imageCount))
		}
		imageCount, err = quantityRequest.ImageCount(false)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		logger.LogDebug(c, "image request body: %s", jsonData)
		body, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		defer closer.Close()
		requestBody = body
	}
	// Price what is actually sent upstream: channel overrides and
	// pass-through bodies can change it. A body without a field keeps the
	// client's value.
	selection := imagePriceSelection{
		Size:       request.Size,
		Quality:    request.Quality,
		Style:      gjson.ParseBytes(request.Style).String(),
		InputImage: info.RelayMode == relayconstant.RelayModeImagesEdits || imageInputPresent(gjson.ParseBytes(request.Image)) || imageInputPresent(gjson.ParseBytes(request.Images)),
	}
	if jsonData != nil {
		for field, target := range map[string]*string{"size": &selection.Size, "quality": &selection.Quality, "style": &selection.Style} {
			if value := gjson.GetBytes(jsonData, field); value.Exists() {
				*target = value.String()
			}
		}
		selection.InputImage = selection.InputImage || imageInputPresent(gjson.GetBytes(jsonData, "image")) || imageInputPresent(gjson.GetBytes(jsonData, "images"))
	}
	price, keyed, priceErr := resolveImageKeyPrice(info.GetBillingModelName(), info.PriceData.UsePrice, selection)
	if priceErr != nil {
		return types.NewErrorWithStatusCode(priceErr, types.ErrorCodeModelPriceError, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if keyed {
		info.PriceData.ModelPrice = price
	}
	if billingErr := service.PrepareImageBillingForRequest(c, info, imageCount); billingErr != nil {
		return billingErr
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			if httpResp.StatusCode == http.StatusCreated && info.ApiType == constant.APITypeReplicate {
				// replicate channel returns 201 Created when using Prefer: wait, treat it as success.
				httpResp.StatusCode = http.StatusOK
			} else {
				newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
				// reset status code 重置状态码
				service.ResetStatusCode(newAPIError, statusCodeMappingStr)
				return newAPIError
			}
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		// The upstream completed and billed an image the gateway could not
		// deliver: the client gets the error, and the request still settles
		// as the upstream billed it.
		if billed, ok := usage.(*dto.Usage); !ok || billed == nil || !errors.Is(newAPIError, openai.ErrGeneratedImageUndelivered) {
			return newAPIError
		}
	}

	// The log content shows the settled count: the count the handler derived
	// from the upstream response when it did, otherwise the reserved quantity.
	imageN := info.RequestedImageCount()
	if info.BillingImageCount != nil {
		imageN = *info.BillingImageCount
	} else if count, ok := info.PriceData.OtherRatios()["n"]; ok && info.PriceData.UsePrice && count >= 1 && count <= dto.MaxImageN {
		imageN = common.QuotaRound(count)
	}

	if usage.(*dto.Usage).TotalTokens == 0 {
		usage.(*dto.Usage).TotalTokens = 1
	}
	if usage.(*dto.Usage).PromptTokens == 0 {
		usage.(*dto.Usage).PromptTokens = 1
	}

	quality := request.Quality
	if quality == "" {
		quality = "standard"
	}

	var logContent []*common.Message

	if len(request.Size) > 0 {
		logContent = append(logContent, common.NewMessage("Size {{size}}", map[string]any{"size": request.Size}))
	}
	if len(quality) > 0 {
		logContent = append(logContent, common.NewMessage("Quality {{quality}}", map[string]any{"quality": quality}))
	}
	if imageN > 0 {
		logContent = append(logContent, common.NewMessage("Image count {{count}}", map[string]any{"count": imageN}))
	}

	service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), logContent)
	return newAPIError
}

// imagePriceKeySeparator joins a model's billing name and an image price key
// in the per-call price table: "image-model#1024X1024" prices that model's
// 1024x1024 images, "image-model#LOW" its low quality ones and
// "image-model#2048X2048+LOW" the two together. A key is the request value in
// upper case.
const imagePriceKeySeparator = "#"

// imageSizePriceKey and imageQualityPriceKeys recognise the keys of the two
// dimensions a request selects itself: its size (pixels, a 1K-style tier or a
// megapixel count, as the request names it) and its quality. Any other key,
// such as a style or a reference-image surcharge, prices something the
// gateway cannot select.
var (
	imageSizePriceKey     = regexp.MustCompile(`^(\d+X\d+|\d+K|\d+MP)$`)
	imageQualityPriceKeys = []string{"LOW", "MEDIUM", "HIGH", "STANDARD", "HD"}
)

// imagePriceSelection is what an outbound image request selects among a
// model's keyed prices.
type imagePriceSelection struct {
	Size, Quality, Style string
	InputImage           bool
}

// imageInputPresent reports whether a request field carries an input image.
func imageInputPresent(value gjson.Result) bool {
	if !value.Exists() || value.Type == gjson.Null {
		return false
	}
	if value.IsArray() {
		return len(value.Array()) > 0
	}
	return value.String() != "" || value.IsObject()
}

// resolveImageKeyPrice returns the per-image price of a model that has keyed
// image prices, for what the outbound request selects; keyed is false when
// the model has none, so its price applies as before. Keyed prices need a
// per-call price of the model's own, which a request that selects no key
// pays (the price of the upstream's default). A dimension the model prices
// and the request names must have a rate: the SIZE+QUALITY key when the
// request names both, else the SIZE or QUALITY key. A dimension the model
// does not price is ignored. A request is refused before the upstream is
// called when that rate is missing, when it matches a size price and a
// quality price at once, and when it names a style or carries an input image
// while the model has keys of another kind.
func resolveImageKeyPrice(billingModel string, usePrice bool, selection imagePriceSelection) (price float64, keyed bool, err error) {
	prefix := billingModel + imagePriceKeySeparator
	rows := make(map[string]float64)
	for name, rowPrice := range ratio_setting.GetModelPriceCopy() {
		if key, found := strings.CutPrefix(name, prefix); found && key != "" {
			rows[key] = rowPrice
		}
	}
	if len(rows) == 0 {
		return 0, false, nil
	}
	own, hasOwn := ratio_setting.GetModelPrice(billingModel, false)
	if !usePrice || !hasOwn {
		return 0, true, fmt.Errorf("model %s has keyed image prices but no per-call price of its own; set a per-call price for %s, the price of a request that selects no key", billingModel, billingModel)
	}
	var pricesSize, pricesQuality, pricesOther bool
	for key := range rows {
		size, quality, combined := strings.Cut(key, "+")
		switch {
		case combined && imageSizePriceKey.MatchString(size) && slices.Contains(imageQualityPriceKeys, quality):
			pricesSize, pricesQuality = true, true
		case !combined && imageSizePriceKey.MatchString(key):
			pricesSize = true
		case !combined && slices.Contains(imageQualityPriceKeys, key):
			pricesQuality = true
		default:
			pricesOther = true
		}
	}
	if pricesOther && (strings.TrimSpace(selection.Style) != "" || selection.InputImage) {
		return 0, true, fmt.Errorf("model %s has image prices the gateway cannot select, such as a style or a reference image; a request with a style or an input image is refused", billingModel)
	}
	var size, quality string
	if pricesSize {
		size = strings.ToUpper(strings.TrimSpace(selection.Size))
	}
	if pricesQuality {
		quality = strings.ToUpper(strings.TrimSpace(selection.Quality))
	}
	switch {
	case size == "" && quality == "":
		return own, true, nil
	case size != "" && quality != "":
		if combined, found := rows[size+"+"+quality]; found {
			return combined, true, nil
		}
		_, sizeRow := rows[size]
		_, qualityRow := rows[quality]
		if sizeRow && qualityRow {
			return 0, true, fmt.Errorf("model %s prices image size %s and quality %s separately; set a price for %s+%s", billingModel, size, quality, size, quality)
		}
		return 0, true, fmt.Errorf("model %s has no image price for %s+%s", billingModel, size, quality)
	case size != "":
		if sizePrice, found := rows[size]; found {
			return sizePrice, true, nil
		}
		return 0, true, fmt.Errorf("model %s has no image price for %s", billingModel, size)
	default:
		if qualityPrice, found := rows[quality]; found {
			return qualityPrice, true, nil
		}
		return 0, true, fmt.Errorf("model %s has no image price for %s", billingModel, quality)
	}
}
