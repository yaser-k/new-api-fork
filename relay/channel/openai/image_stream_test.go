package openai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newImageTestContext(t *testing.T, body, contentType string, isStream bool) (*gin.Context, *httptest.ResponseRecorder, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{contentType}},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
		IsStream:    isStream,
	}
	return c, recorder, resp, info
}

func TestImageExpressionUsesCompletedCountAndProtectsAbortedStreams(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	for _, tc := range []struct {
		name, body      string
		stream, abort   bool
		requested, want int
	}{
		{"JSON uses actual count", `{"data":[{"b64_json":"first"},{"b64_json":"second"}]}`, false, false, 3, 2},
		{"JSON wrapped as SSE uses actual count", `{"data":[{"b64_json":"first"}]}`, true, false, 3, 1},
		{"empty response retains request", `{"data":[]}`, false, false, 3, 3},
		{"JSON object data counts one image", `{"data":{"url":"https://example.com/a.png","b64_json":"first"}}`, false, false, 3, 1},
		{"JSON wrapped as SSE counts object data once", `{"data":{"b64_json":"first"}}`, true, false, 3, 1},
		{"completed stream refunds missing images", "data: {\"type\":\"image_generation.completed\",\"b64_json\":\"first\"}\n\ndata: [DONE]\n\n", true, false, 3, 1},
		{"client abort cannot reduce count", "data: {\"type\":\"image_generation.completed\",\"b64_json\":\"first\"}\n\n", true, true, 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contentType := "application/json"
			if strings.HasPrefix(tc.body, "data:") {
				contentType = "text/event-stream"
			}
			c, _, resp, info := newImageTestContext(t, tc.body, contentType, tc.stream)
			if tc.abort {
				c, _, resp, info = newDisconnectingImageStream(t, tc.body, "first")
			}
			info.TieredBillingSnapshot = &billingexpr.BillingSnapshot{EstimatedImageCount: &tc.requested}
			if tc.stream {
				_, err := OpenaiImageStreamHandler(c, info, resp)
				require.Nil(t, err)
			} else {
				_, err := OpenaiImageHandler(c, info, resp)
				require.Nil(t, err)
			}
			count := info.RequestedImageCount()
			if info.BillingImageCount != nil {
				count = *info.BillingImageCount
			}
			assert.Equal(t, tc.want, count)
			assert.Empty(t, info.PriceData.OtherRatios(), "expression quantities must not add a legacy multiplier")
		})
	}
}

func TestImageCacheUsageAcrossResponseFormats(t *testing.T) {
	previousTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = previousTimeout })
	// Contract fixture for compatible upstreams. Live OpenAI cache response
	// verification is a separate release check; this is not a captured response.
	const usageJSON = `{"input_tokens":1000,"output_tokens":100,"total_tokens":1100,"input_tokens_details":{"text_tokens":400,"image_tokens":600,"cached_tokens":300,"cached_tokens_details":{"text_tokens":100,"image_tokens":200}}}`
	for _, mode := range []string{"image_generation", "image_edit"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%t", mode, stream), func(t *testing.T) {
				body := `{"data":[{"b64_json":"image"}],"usage":` + usageJSON + `}`
				contentType := "application/json"
				if stream {
					body = "data: {\"type\":\"" + mode + ".completed\",\"usage\":" + usageJSON + "}\n\ndata: [DONE]\n\n"
					contentType = "text/event-stream"
				}
				ctx, recorder, response, info := newImageTestContext(t, body, contentType, stream)
				info.RelayMode = relayconstant.RelayModeImagesGenerations
				if mode == "image_edit" {
					info.RelayMode = relayconstant.RelayModeImagesEdits
					ctx.Request.URL.Path = "/v1/images/edits"
				}
				result, apiErr := (&Adaptor{}).DoResponse(ctx, response, info)
				require.Nil(t, apiErr)
				usage := result.(*dto.Usage)
				assert.Equal(t, 1000, usage.PromptTokens)
				assert.Equal(t, 300, usage.PromptTokensDetails.CachedTokens)
				require.NotNil(t, usage.PromptTokensDetails.CachedTokensDetails)
				assert.Equal(t, 200, *usage.PromptTokensDetails.CachedTokensDetails.ImageTokens)
				assert.Contains(t, recorder.Body.String(), `"cached_tokens_details":{"text_tokens":100,"image_tokens":200}`)
			})
		}
	}
}

func TestNormalizeOpenAIUsageMapsOutputImageTokens(t *testing.T) {
	previousTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = previousTimeout })

	const usageJSON = `{"input_tokens":15,"output_tokens":1352,"total_tokens":1367,"input_tokens_details":{"text_tokens":15,"image_tokens":0},"output_tokens_details":{"image_tokens":1120,"text_tokens":232}}`
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%t", stream), func(t *testing.T) {
			body := `{"data":[{"b64_json":"image"}],"usage":` + usageJSON + `}`
			contentType := "application/json"
			if stream {
				body = "data: {\"type\":\"image_generation.completed\",\"usage\":" + usageJSON + "}\n\ndata: [DONE]\n\n"
				contentType = "text/event-stream"
			}
			ctx, _, response, info := newImageTestContext(t, body, contentType, stream)
			info.RelayMode = relayconstant.RelayModeImagesGenerations
			result, apiErr := (&Adaptor{}).DoResponse(ctx, response, info)
			require.Nil(t, apiErr)
			usage := result.(*dto.Usage)
			assert.Equal(t, 15, usage.PromptTokens)
			assert.Equal(t, 1352, usage.CompletionTokens)
			assert.Equal(t, 1120, usage.CompletionTokenDetails.ImageTokens)
			assert.Equal(t, 232, usage.CompletionTokenDetails.TextTokens)
		})
	}
}

func TestOpenaiImageDoResponseUsesInfoIsStream(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	body := `{"created":1710000000,"data":[{"b64_json":"image"}]}`

	t.Run("non-stream response stays JSON", func(t *testing.T) {
		c, recorder, resp, info := newImageTestContext(t, body, "application/json", false)
		info.RelayMode = relayconstant.RelayModeImagesGenerations

		usage, err := (&Adaptor{}).DoResponse(c, resp, info)

		require.Nil(t, err)
		require.NotNil(t, usage)
		require.Equal(t, body, recorder.Body.String())
	})

	t.Run("stream response converts JSON to SSE", func(t *testing.T) {
		c, recorder, resp, info := newImageTestContext(t, body, "application/json", true)
		info.RelayMode = relayconstant.RelayModeImagesGenerations

		usage, err := (&Adaptor{}).DoResponse(c, resp, info)

		require.Nil(t, err)
		require.NotNil(t, usage)
		require.Contains(t, recorder.Body.String(), `event: image_generation.completed`)
		require.Contains(t, recorder.Body.String(), `data: [DONE]`)
	})
}

// TestOpenaiImageStreamHandlerForwardsSSEAndUsage covers the core SSE path:
// chunks are forwarded with rebuilt event lines, usage is extracted and
// normalized (input_tokens -> prompt_tokens with details), and [DONE] is
// re-emitted to the client.
func TestOpenaiImageStreamHandlerForwardsSSEAndUsage(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`event: image_generation.partial_image`,
		`data: {"type":"image_generation.partial_image","b64_json":"partial"}`,
		``,
		`data: {"usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7,"input_tokens_details":{"image_tokens":2,"text_tokens":1}}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	c, recorder, resp, info := newImageTestContext(t, body, "text/event-stream", true)
	info.PriceData.UsePrice = true
	info.PriceData.AddOtherRatio("n", 3)

	usage, err := OpenaiImageStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 4, usage.CompletionTokens)
	require.Equal(t, 7, usage.TotalTokens)
	require.Equal(t, 2, usage.PromptTokensDetails.ImageTokens)
	require.Equal(t, 1, usage.PromptTokensDetails.TextTokens)
	require.Contains(t, recorder.Body.String(), `event: image_generation.partial_image`)
	require.Contains(t, recorder.Body.String(), `data: {"type":"image_generation.partial_image","b64_json":"partial"}`)
	require.Contains(t, recorder.Body.String(), `data: {"usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7,"input_tokens_details":{"image_tokens":2,"text_tokens":1}}}`)
	require.Contains(t, recorder.Body.String(), `data: [DONE]`)
	require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	require.Equal(t, 3.0, info.PriceData.OtherRatios()["n"], "streams without completed events keep the requested count")
}

func TestOpenaiImageStreamHandlerUsesCompletedEventCount(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`data: {"type":"image_generation.partial_image","partial_image_index":0,"b64_json":"partial"}`,
		``,
		`data: {"type":"image_generation.completed","b64_json":"first"}`,
		``,
		`data: {"type":"image_edit.completed","b64_json":"second","usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	c, _, resp, info := newImageTestContext(t, body, "text/event-stream", true)
	info.PriceData.UsePrice = true
	info.PriceData.AddOtherRatio("n", 3)

	usage, err := OpenaiImageStreamHandler(c, info, resp)

	require.Nil(t, err)
	require.Equal(t, 7, usage.TotalTokens)
	require.Equal(t, 2.0, info.PriceData.OtherRatios()["n"])
}

// blockingBody serves one SSE chunk, then blocks until Close (the scanner's
// cleanup) and returns EOF — keeping the upstream "open" while the client-side
// disconnect is simulated elsewhere.
type blockingBody struct {
	mu     sync.Mutex
	sent   bool
	chunk  []byte
	closed chan struct{}
}

func (b *blockingBody) Read(p []byte) (int, error) {
	b.mu.Lock()
	if !b.sent {
		b.sent = true
		n := copy(p, b.chunk)
		b.mu.Unlock()
		return n, nil
	}
	b.mu.Unlock()
	<-b.closed
	return 0, io.EOF
}

func (b *blockingBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-b.closed:
	default:
		close(b.closed)
	}
	return nil
}

// cancelAfterWriter cancels the request context right after the payload
// containing needle has been written to the client, simulating a client that
// disconnects after receiving that event. Cancelling from the write side (not
// the upstream read side) makes the abort deterministic: the handler has
// already processed and counted the event when the disconnect fires.
type cancelAfterWriter struct {
	gin.ResponseWriter
	needle string
	cancel context.CancelFunc
	once   sync.Once
}

func (w *cancelAfterWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if strings.Contains(string(p), w.needle) {
		w.once.Do(w.cancel)
	}
	return n, err
}

func (w *cancelAfterWriter) WriteString(s string) (int, error) {
	n, err := io.WriteString(w.ResponseWriter, s)
	if strings.Contains(s, w.needle) {
		w.once.Do(w.cancel)
	}
	return n, err
}

func newDisconnectingImageStream(t *testing.T, sseBody, disconnectAfter string) (*gin.Context, *httptest.ResponseRecorder, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()
	c, recorder, resp, info := newImageTestContext(t, "", "text/event-stream", true)
	ctx, cancel := context.WithCancel(c.Request.Context())
	t.Cleanup(cancel)
	c.Request = c.Request.WithContext(ctx)
	c.Writer = &cancelAfterWriter{ResponseWriter: c.Writer, needle: disconnectAfter, cancel: cancel}
	resp.Body = &blockingBody{
		chunk:  []byte(sseBody),
		closed: make(chan struct{}),
	}
	return c, recorder, resp, info
}

// TestOpenaiImageStreamHandlerClientDisconnectKeepsRequestedCount guards the
// billing invariant: completed-event counting must not lower the charge when
// the client aborts the stream. Upstream already generated (and charged for)
// all requested images, so a disconnect after the first completed event keeps
// the requested n instead of dropping it to 1.
func TestOpenaiImageStreamHandlerClientDisconnectKeepsRequestedCount(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := "data: {\"type\":\"image_generation.completed\",\"b64_json\":\"first\"}\n\n"
	c, recorder, resp, info := newDisconnectingImageStream(t, body, "first")
	info.PriceData.UsePrice = true
	info.PriceData.AddOtherRatio("n", 3)

	usage, err := OpenaiImageStreamHandler(c, info, resp)

	require.Nil(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, info.StreamStatus)
	// A client abort surfaces as client_gone (main-loop ctx watch) or
	// handler_stop (failed client write); both must be treated as untrusted.
	require.Contains(t,
		[]relaycommon.StreamEndReason{relaycommon.StreamEndReasonClientGone, relaycommon.StreamEndReasonHandlerStop},
		info.StreamStatus.EndReason)
	require.Contains(t, recorder.Body.String(), `"b64_json":"first"`)
	require.Equal(t, 3.0, info.PriceData.OtherRatios()["n"], "client abort must not reduce the billed image count")
}

// TestOpenaiImageStreamHandlerClientDisconnectRaisesCount covers the other
// direction of the abort guard: when completed events already exceed the
// recorded n, the higher actual count is billed even though the client aborted.
func TestOpenaiImageStreamHandlerClientDisconnectRaisesCount(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`data: {"type":"image_generation.completed","b64_json":"first"}`,
		``,
		`data: {"type":"image_generation.completed","b64_json":"second"}`,
		``,
		``,
	}, "\n")
	c, _, resp, info := newDisconnectingImageStream(t, body, "second")
	info.PriceData.UsePrice = true
	info.PriceData.AddOtherRatio("n", 1)

	usage, err := OpenaiImageStreamHandler(c, info, resp)

	require.Nil(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, info.StreamStatus)
	require.Contains(t,
		[]relaycommon.StreamEndReason{relaycommon.StreamEndReasonClientGone, relaycommon.StreamEndReasonHandlerStop},
		info.StreamStatus.EndReason)
	require.Equal(t, 2.0, info.PriceData.OtherRatios()["n"], "completed events beyond the recorded n must raise the charge even on abort")
}

// TestOpenaiImageStreamHandlerWrapsJSONResponse covers the non-SSE fallback:
// a JSON upstream response is wrapped into pseudo-SSE completed events.
func TestOpenaiImageStreamHandlerWrapsJSONResponse(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	body := `{"created":1710000000,"data":[{"b64_json":"first","revised_prompt":"draw a cat"},{"b64_json":"second"}],"usage":{"input_tokens":3,"output_tokens":4,"total_tokens":7,"input_tokens_details":{"image_tokens":2,"text_tokens":1}}}`

	c, recorder, resp, info := newImageTestContext(t, body, "application/json", true)
	info.PriceData.UsePrice = true
	info.PriceData.AddOtherRatio("n", 3)

	usage, err := OpenaiImageStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 4, usage.CompletionTokens)
	require.Equal(t, 7, usage.TotalTokens)
	require.Equal(t, 2, usage.PromptTokensDetails.ImageTokens)
	require.Equal(t, 1, usage.PromptTokensDetails.TextTokens)
	require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	require.Empty(t, recorder.Header().Get("Content-Length"))
	require.Contains(t, recorder.Body.String(), `event: image_generation.completed`)
	require.Contains(t, recorder.Body.String(), `"type":"image_generation.completed"`)
	require.Contains(t, recorder.Body.String(), `"b64_json":"first"`)
	require.Contains(t, recorder.Body.String(), `"b64_json":"second"`)
	require.Contains(t, recorder.Body.String(), `"revised_prompt":"draw a cat"`)
	require.Contains(t, recorder.Body.String(), `data: [DONE]`)
	require.Equal(t, 2, strings.Count(recorder.Body.String(), `event: image_generation.completed`))
	require.Equal(t, 2.0, info.PriceData.OtherRatios()["n"])
}

func TestOpenaiImageHandlerUsesPositiveActualCountForFixedPrice(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })
	longImage := strings.Repeat("a", 4096)

	tests := []struct {
		name      string
		body      string
		usePrice  bool
		wantCount float64
	}{
		{
			name:      "fixed price uses data length",
			body:      `{"data":[{"b64_json":"` + longImage + `"},{"b64_json":"second"}]}`,
			usePrice:  true,
			wantCount: 2,
		},
		{
			name:      "empty data keeps requested count",
			body:      `{"data":[]}`,
			usePrice:  true,
			wantCount: 3,
		},
		{
			name:      "ratio billing ignores data length",
			body:      `{"data":[{"b64_json":"first"},{"b64_json":"second"}]}`,
			usePrice:  false,
			wantCount: 3,
		},
		{
			name:      "object data with url and b64_json counts one image",
			body:      `{"data":{"url":"https://example.com/a.png","b64_json":"` + longImage + `"}}`,
			usePrice:  true,
			wantCount: 1,
		},
		{
			name:      "url and b64_json split across entries count one image",
			body:      `{"data":[{"url":"https://example.com/a.png"},{"b64_json":"` + longImage + `"}]}`,
			usePrice:  true,
			wantCount: 1,
		},
		{
			name:      "entries without image payload keep requested count",
			body:      `{"data":[{"revised_prompt":"draw a cat"}]}`,
			usePrice:  true,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder, resp, info := newImageTestContext(t, tt.body, "application/json", false)
			info.PriceData.UsePrice = tt.usePrice
			info.PriceData.AddOtherRatio("n", 3)

			_, err := OpenaiImageHandler(c, info, resp)

			require.Nil(t, err)
			require.Equal(t, tt.wantCount, info.PriceData.OtherRatios()["n"])
			require.Equal(t, tt.body, recorder.Body.String())
		})
	}
}

// TestOpenaiImageStreamHandlerWrapsNonStandardDataShapes covers the JSON-to-SSE
// fallback when upstream returns data as an object, splits one image across a
// url entry and a b64_json entry, or returns entries without any image payload.
// Forwarded events and the billed quantity must both follow the real images.
func TestOpenaiImageStreamHandlerWrapsNonStandardDataShapes(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	for _, tc := range []struct {
		name       string
		body       string
		wantEvents int
		wantCount  float64
		wantBody   []string
	}{
		{
			name:       "object data is forwarded as one completed event",
			body:       `{"data":{"url":"https://example.com/a.png","b64_json":"first"}}`,
			wantEvents: 1,
			wantCount:  1,
			wantBody:   []string{`"url":"https://example.com/a.png"`, `"b64_json":"first"`},
		},
		{
			name:       "split url and b64_json entries bill one image",
			body:       `{"data":[{"url":"https://example.com/a.png"},{"b64_json":"first"}]}`,
			wantEvents: 2,
			wantCount:  1,
			wantBody:   []string{`"url":"https://example.com/a.png"`, `"b64_json":"first"`},
		},
		{
			name:       "entries without image payload are dropped and keep requested count",
			body:       `{"data":[{"revised_prompt":"draw a cat"}]}`,
			wantEvents: 0,
			wantCount:  3,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, recorder, resp, info := newImageTestContext(t, tc.body, "application/json", true)
			info.PriceData.UsePrice = true
			info.PriceData.AddOtherRatio("n", 3)

			_, err := OpenaiImageStreamHandler(c, info, resp)
			require.Nil(t, err)
			out := recorder.Body.String()
			assert.Equal(t, tc.wantEvents, strings.Count(out, "event: image_generation.completed"))
			assert.Equal(t, tc.wantEvents, info.ReceivedResponseCount)
			for _, want := range tc.wantBody {
				assert.Contains(t, out, want)
			}
			assert.True(t, strings.HasSuffix(out, "data: [DONE]\n\n"))
			assert.Equal(t, tc.wantCount, info.PriceData.OtherRatios()["n"])
		})
	}
}

// TestOpenaiImageHandlersReturnJSONError covers JSON error responses for both
// entry points: the non-streaming handler and the stream handler's non-SSE
// fallback. Neither must leak the error body to the client.
func TestOpenaiImageHandlersReturnJSONError(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	body := `{"error":{"message":"content moderation failed","type":"upstream_error","code":"content_moderation_failed","status":502}}`

	t.Run("non-streaming handler", func(t *testing.T) {
		c, recorder, resp, info := newImageTestContext(t, body, "application/json", false)

		usage, err := OpenaiImageHandler(c, info, resp)
		require.Nil(t, usage)
		require.NotNil(t, err)
		require.Equal(t, http.StatusOK, err.StatusCode)
		oaiError := err.ToOpenAIError()
		require.Equal(t, "content moderation failed", oaiError.Message)
		require.Equal(t, "upstream_error", oaiError.Type)
		require.Equal(t, "content_moderation_failed", oaiError.Code)
		require.Empty(t, recorder.Body.String())
	})

	t.Run("stream handler JSON fallback", func(t *testing.T) {
		c, recorder, resp, info := newImageTestContext(t, body, "application/json", true)

		usage, err := OpenaiImageStreamHandler(c, info, resp)
		require.Nil(t, usage)
		require.NotNil(t, err)
		require.Equal(t, http.StatusOK, err.StatusCode)
		require.Equal(t, "content moderation failed", err.ToOpenAIError().Message)
		require.Empty(t, recorder.Body.String())
	})

	t.Run("stream handler non-2xx stays JSON error", func(t *testing.T) {
		c, recorder, resp, info := newImageTestContext(t, body, "application/json", true)
		resp.StatusCode = http.StatusBadGateway

		usage, err := OpenaiImageStreamHandler(c, info, resp)
		require.Nil(t, usage)
		require.NotNil(t, err)
		require.Equal(t, http.StatusBadGateway, err.StatusCode)
		require.Equal(t, "content moderation failed", err.ToOpenAIError().Message)
		require.Empty(t, recorder.Body.String())
		require.NotContains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
	})
}

// TestOpenaiImageStreamHandlerRecordsUpstreamErrorEvent verifies that an error
// event inside the SSE stream is recorded as a soft error while the payload is
// still forwarded to the client.
func TestOpenaiImageStreamHandlerRecordsUpstreamErrorEvent(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`event: image_generation.partial_image`,
		`data: {"type":"image_generation.partial_image","b64_json":"partial"}`,
		``,
		`event: error`,
		`data: {"type":"upstream_error","error":{"message":"stream error: stream ID 77; INTERNAL_ERROR; received from peer"}}`,
		``,
	}, "\n")

	c, recorder, resp, info := newImageTestContext(t, body, "text/event-stream", true)

	usage, err := OpenaiImageStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, info.StreamStatus)
	require.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.EndReason)
	require.True(t, info.StreamStatus.HasErrors())
	require.Equal(t, 1, info.StreamStatus.TotalErrorCount())
	require.Contains(t, info.StreamStatus.Errors[0].Message, "INTERNAL_ERROR")
	// The scanner strips the upstream "event: error" line; the event name is
	// rebuilt from the JSON "type" field (upstream_error). The error message
	// is still forwarded in the data: payload (stream ID 77).
	require.Contains(t, recorder.Body.String(), `event: upstream_error`)
	require.Contains(t, recorder.Body.String(), `stream ID 77`)
}

// The answers below keep the shape an OpenAI-compatible upstream returned on
// two dates; the base64 payloads and the links are cut. Every answer carries
// fields of the upstream's own (billing, request_id, id, generation_id) that a
// rewritten answer must not pass on.
const (
	// A per-image model, first measurement: a url only, whatever
	// response_format asked for, and no usage.
	measuredPerImageURLAnswer = `{"created":1761480000,"data":[{"url":"{{IMAGE_URL}}"}],"id":"0b9d6f42-4a4f-4f43-9d7e-2d1f8f3c5a10","request_id":"7c2f3e1a-5b6d-4e8f-9a0b-1c2d3e4f5a6b","billing":{"status":"settled","billed_cost_usd":"0.025400","currency":"USD"}}`
	// The same model, second measurement: b64_json with or without
	// response_format, and a usage of zeros.
	measuredPerImageB64Answer = `{"created":1761998400,"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAE"}],"usage":{"total_tokens":0,"input_tokens":0,"output_tokens":0},"request_id":"1e2d3c4b-5a69-4788-9a1b-2c3d4e5f6a7b","billing":{"billed_cost_usd":"0.025500"}}`
	// A per-token model, first measurement: an Images usage without an
	// output breakdown.
	measuredPerTokenAnswer = `{"created":1761480000,"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAI"}],"request_id":"2f3e4d5c-6b7a-4899-8a0b-1c2d3e4f5a6b","usage":{"total_tokens":212,"input_tokens":16,"output_tokens":196,"input_tokens_details":{"text_tokens":16,"image_tokens":0}},"billing":{"billed_cost_usd":"0.005066"}}`
	// The per-token model, second measurement: an output breakdown and the
	// generation parameters echoed back.
	measuredPerTokenDetailedAnswer = `{"created":1761998400,"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAM","generation_id":"gen_0a1b2c3d"}],"background":"opaque","output_format":"png","quality":"low","size":"1024x1024","request_id":"3a4b5c6d-7e8f-4a0b-9c1d-2e3f4a5b6c7d","usage":{"total_tokens":210,"input_tokens":14,"output_tokens":196,"input_tokens_details":{"text_tokens":14,"image_tokens":0},"output_tokens_details":{"image_tokens":196,"text_tokens":0}},"billing":{"billed_cost_usd":"0.005057"}}`
)

var testPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR-image-fixture")

// newImageObjectServer serves testPNG under /object/image.png and counts the
// requests; /object/expired answers like an expired signed link.
func newImageObjectServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	fetchSetting := system_setting.GetFetchSetting()
	previousFetchSetting := *fetchSetting
	fetchSetting.AllowPrivateIp = true
	fetchSetting.AllowedPorts = []string{"1-65535"}
	previousLimit := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 64
	service.InitHttpClient()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		assert.Empty(t, r.Header.Get("Authorization"), "an image link is fetched without credentials")
		switch r.URL.Path {
		case "/object/image.png":
			w.Header().Set("Content-Type", "binary/octet-stream")
			_, _ = w.Write(testPNG)
		case "/object/page":
			_, _ = w.Write([]byte("<html>not an image</html>"))
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	t.Cleanup(func() {
		server.Close()
		*fetchSetting = previousFetchSetting
		constant.MaxFileDownloadMB = previousLimit
	})
	return server, &hits
}

func newOpenAIChannelImageContext(t *testing.T, body string, channelType int, inline bool, responseFormat string, stream bool) (*gin.Context, *httptest.ResponseRecorder, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()
	contentType := "application/json"
	if strings.HasPrefix(body, "data:") {
		contentType = "text/event-stream"
	}
	c, recorder, resp, info := newImageTestContext(t, body, contentType, stream)
	info.RelayMode = relayconstant.RelayModeImagesGenerations
	info.ChannelMeta.ChannelType = channelType
	info.ChannelMeta.ChannelOtherSettings.ImageResponseInline = inline
	info.Request = &dto.ImageRequest{Model: "image-model", Prompt: "a lighthouse", Size: "1024x1024", ResponseFormat: responseFormat}
	info.PriceData.UsePrice = true
	info.PriceData.AddOtherRatio("n", 1)
	return c, recorder, resp, info
}

func TestOpenaiImageAnswersWithoutUpstreamLinksOrFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, hits := newImageObjectServer(t)
	imageURL := server.URL + "/object/image.png?Expires=1761566400&Signature=cut"
	urlAnswer := strings.ReplaceAll(measuredPerImageURLAnswer, "{{IMAGE_URL}}", imageURL)
	fetched := base64.StdEncoding.EncodeToString(testPNG)

	for _, tc := range []struct {
		name           string
		body           string
		channelType    int
		inline         bool
		responseFormat string
		wantForwarded  bool
		wantB64        string
		wantURL        string
		wantFetches    int32
		wantKept       []string
		wantRevised    string
	}{
		{name: "url answer, setting on", body: urlAnswer, channelType: constant.ChannelTypeOpenAI, inline: true, wantB64: fetched, wantFetches: 1},
		{name: "url answer, b64_json asked, setting off", body: urlAnswer, channelType: constant.ChannelTypeOpenAI, responseFormat: "b64_json", wantB64: fetched, wantFetches: 1},
		{name: "url answer, url asked, setting off, keeps the link", body: urlAnswer, channelType: constant.ChannelTypeOpenAI, responseFormat: "url", wantURL: imageURL},
		{name: "url answer, no response_format, setting off, keeps the link", body: urlAnswer, channelType: constant.ChannelTypeOpenAI, wantURL: imageURL},
		{name: "url answer on another channel type, as today", body: urlAnswer, channelType: constant.ChannelTypeCustom, inline: true, responseFormat: "b64_json", wantForwarded: true},
		{name: "b64 answer, setting on", body: measuredPerImageB64Answer, channelType: constant.ChannelTypeOpenAI, inline: true, wantB64: "iVBORw0KGgoAAAANSUhEUgAAAAE", wantKept: []string{"usage"}},
		{name: "b64 answer, setting off", body: measuredPerImageB64Answer, channelType: constant.ChannelTypeOpenAI, wantB64: "iVBORw0KGgoAAAANSUhEUgAAAAE", wantKept: []string{"usage"}},
		{name: "split url and b64_json entries return the one billed image", body: `{"created":1761480000,"data":[{"url":"` + imageURL + `"},{"b64_json":"aW1hZ2U=","revised_prompt":"a lighthouse at dusk"}],"request_id":"r"}`, channelType: constant.ChannelTypeOpenAI, inline: true, wantB64: "aW1hZ2U=", wantRevised: "a lighthouse at dusk"},
		{name: "per-token answer, setting on", body: measuredPerTokenDetailedAnswer, channelType: constant.ChannelTypeOpenAI, inline: true, wantB64: "iVBORw0KGgoAAAANSUhEUgAAAAM", wantKept: []string{"usage", "background", "output_format", "quality", "size"}},
		{name: "per-token answer, setting off", body: measuredPerTokenDetailedAnswer, channelType: constant.ChannelTypeOpenAI, wantB64: "iVBORw0KGgoAAAANSUhEUgAAAAM", wantKept: []string{"usage", "background", "output_format", "quality", "size"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := hits.Load()
			c, recorder, resp, info := newOpenAIChannelImageContext(t, tc.body, tc.channelType, tc.inline, tc.responseFormat, false)
			resp.Header.Set("X-Request-Id", "upstream-request-id")
			resp.Header.Set("Set-Cookie", "upstream-session=1")

			usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tc.wantFetches, hits.Load()-before)
			assert.Equal(t, 1.0, info.PriceData.OtherRatios()["n"], "billed per image returned")
			out := recorder.Body.String()
			if tc.wantForwarded {
				assert.Equal(t, tc.body, out)
				assert.Equal(t, "upstream-request-id", recorder.Header().Get("X-Request-Id"))
				return
			}
			require.True(t, gjson.Valid(out), out)
			answer := gjson.Parse(out)
			assert.Equal(t, int64(1), answer.Get("data.#").Int())
			assert.Equal(t, tc.wantB64, answer.Get("data.0.b64_json").String())
			assert.Equal(t, tc.wantURL, answer.Get("data.0.url").String())
			assert.Equal(t, gjson.Get(tc.body, "created").Int(), answer.Get("created").Int())
			if tc.wantURL == "" {
				assert.NotContains(t, out, server.URL)
			}
			for _, field := range []string{"billing", "request_id", "id", "data.0.generation_id"} {
				assert.False(t, answer.Get(field).Exists(), field)
			}
			for _, field := range tc.wantKept {
				assert.JSONEq(t, gjson.Get(tc.body, field).Raw, answer.Get(field).Raw, field)
			}
			assert.Equal(t, tc.wantRevised, answer.Get("data.0.revised_prompt").String())
			assert.Equal(t, fmt.Sprint(len(out)), recorder.Header().Get("Content-Length"))
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			assert.Empty(t, recorder.Header().Get("X-Request-Id"), "no upstream header reaches the client")
			assert.Empty(t, recorder.Header().Get("Set-Cookie"), "no upstream header reaches the client")
		})
	}
}

func TestOpenaiImageFetchFailureIsAnErrorBilledAsTheUpstreamBilled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, _ := newImageObjectServer(t)
	for _, tc := range []struct {
		name, path string
		stream     bool
	}{
		{"expired link", "/object/expired?Expires=1&Signature=cut", false},
		{"not an image", "/object/page", false},
		{"expired link wrapped as SSE", "/object/expired?Expires=1&Signature=cut", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.ReplaceAll(measuredPerImageURLAnswer, "{{IMAGE_URL}}", server.URL+tc.path)
			c, recorder, resp, info := newOpenAIChannelImageContext(t, body, constant.ChannelTypeOpenAI, true, "", tc.stream)
			info.PriceData.AddOtherRatio("n", 3)

			usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)

			require.NotNil(t, apiErr)
			assert.True(t, errors.Is(apiErr, ErrGeneratedImageUndelivered))
			assert.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
			assert.True(t, types.IsSkipRetryError(apiErr), "a generated image must not be generated again on another channel")
			assert.NotContains(t, apiErr.Error(), server.URL)
			require.NotNil(t, usage, "the usage of the answer the upstream billed is returned for settlement")
			assert.Equal(t, 1.0, info.PriceData.OtherRatios()["n"], "the image the upstream returned is billed")
			assert.Empty(t, recorder.Body.String(), "no empty data list is returned as success")
		})
	}
}

func TestOpenaiImageStreamsCarryTheImageWithoutTheLink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	server, _ := newImageObjectServer(t)
	imageURL := server.URL + "/object/image.png?Expires=1761566400&Signature=cut"
	fetched := base64.StdEncoding.EncodeToString(testPNG)
	const upstreamFields = `"id":"evt_1","request_id":"7c2f3e1a","generation_id":"gen_0a1b","billing":{"status":"settled","billed_cost_usd":"0.025400"}`
	const usage = `{"input_tokens":16,"output_tokens":196,"total_tokens":212}`

	for _, tc := range []struct {
		name, body string
		keepLink   bool
		wantEvents []string
	}{
		{
			name:       "JSON answer wrapped as SSE",
			body:       strings.ReplaceAll(measuredPerImageURLAnswer, "{{IMAGE_URL}}", imageURL),
			wantEvents: []string{`{"type":"image_generation.completed","created_at":1761480000,"b64_json":"` + fetched + `"}`},
		},
		{
			name: "SSE events with a url and the upstream's own fields",
			body: "data: {\"type\":\"image_generation.partial_image\",\"partial_image_index\":0,\"b64_json\":\"cGFydGlhbA==\"," + upstreamFields + "}\n\n" +
				"data: {\"type\":\"image_generation.completed\",\"created_at\":1761480000,\"url\":\"" + imageURL + "\",\"usage\":" + usage + "," + upstreamFields + "}\n\ndata: [DONE]\n\n",
			wantEvents: []string{
				`{"type":"image_generation.partial_image","partial_image_index":0,"b64_json":"cGFydGlhbA=="}`,
				`{"type":"image_generation.completed","created_at":1761480000,"usage":` + usage + `,"b64_json":"` + fetched + `"}`,
			},
		},
		{
			name:       "SSE event nesting the image in a data list",
			body:       "data: {\"type\":\"image_generation.completed\",\"data\":[{\"url\":\"" + imageURL + "\",\"generation_id\":\"gen_0a1b\"}]," + upstreamFields + "}\n\ndata: [DONE]\n\n",
			wantEvents: []string{`{"type":"image_generation.completed","b64_json":"` + fetched + `"}`},
		},
		{
			name:       "setting off keeps the link without the upstream's fields",
			body:       "data: {\"type\":\"image_generation.completed\",\"url\":\"" + imageURL + "\"," + upstreamFields + "}\n\ndata: [DONE]\n\n",
			keepLink:   true,
			wantEvents: []string{`{"type":"image_generation.completed","url":"` + imageURL + `"}`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, recorder, resp, info := newOpenAIChannelImageContext(t, tc.body, constant.ChannelTypeOpenAI, !tc.keepLink, "", true)

			_, apiErr := (&Adaptor{}).DoResponse(c, resp, info)

			require.Nil(t, apiErr)
			out := recorder.Body.String()
			var events []string
			for line := range strings.SplitSeq(out, "\n") {
				if payload, ok := strings.CutPrefix(line, "data: "); ok && payload != "[DONE]" {
					events = append(events, payload)
				}
			}
			require.Len(t, events, len(tc.wantEvents), out)
			for index, want := range tc.wantEvents {
				assert.JSONEq(t, want, events[index])
			}
			leaked := []string{"billing", "request_id", "generation_id", `"id"`, `"data"`}
			if !tc.keepLink {
				leaked = append(leaked, server.URL)
			}
			for _, field := range leaked {
				assert.NotContains(t, out, field)
			}
			assert.Equal(t, 1, strings.Count(out, "event: image_generation.completed"))
			assert.Equal(t, 1.0, info.PriceData.OtherRatios()["n"])
		})
	}
}

// TestOpenaiImagePerTokenUsageBillsImageOutputAtItsRate settles both measured
// per-token answers with an expression that prices image output on its own
// (img_o) and with one that prices all output as c, as before.
func TestOpenaiImagePerTokenUsageBillsImageOutputAtItsRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const imageRate = `tier("base", p * 4.25 + img_o * 25.5)`
	const outputRate = `tier("base", p * 4.25 + c * 25.5)`
	for _, tc := range []struct {
		name            string
		body            string
		channelType     int
		wantPrompt      int
		wantImageTokens int
		wantImageRate   float64
	}{
		{"first measurement", measuredPerTokenAnswer, constant.ChannelTypeOpenAI, 16, 196, 16*4.25 + 196*25.5},
		{"second measurement", measuredPerTokenDetailedAnswer, constant.ChannelTypeOpenAI, 14, 196, 14*4.25 + 196*25.5},
		{"first measurement on another channel type, as today", measuredPerTokenAnswer, constant.ChannelTypeCustom, 16, 0, 16 * 4.25},
		{"first measurement streamed", "data: {\"type\":\"image_generation.completed\",\"b64_json\":\"aW1hZ2U=\",\"usage\":" + gjson.Get(measuredPerTokenAnswer, "usage").Raw + "}\n\ndata: [DONE]\n\n", constant.ChannelTypeOpenAI, 16, 196, 16*4.25 + 196*25.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stream := strings.HasPrefix(tc.body, "data:")
			if stream {
				oldTimeout := constant.StreamingTimeout
				constant.StreamingTimeout = 30
				t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
			}
			c, _, resp, info := newOpenAIChannelImageContext(t, tc.body, tc.channelType, false, "b64_json", stream)
			result, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
			require.Nil(t, apiErr)
			usage := result.(*dto.Usage)
			assert.Equal(t, tc.wantPrompt, usage.PromptTokens)
			assert.Equal(t, 196, usage.CompletionTokens)
			assert.Equal(t, tc.wantImageTokens, usage.CompletionTokenDetails.ImageTokens)

			for expression, want := range map[string]float64{imageRate: tc.wantImageRate, outputRate: float64(tc.wantPrompt)*4.25 + 196*25.5} {
				params := service.BuildTieredTokenParams(usage, false, billingexpr.UsedVars(expression))
				cost, _, err := billingexpr.RunExpr(expression, params)
				require.NoError(t, err)
				assert.InDelta(t, want, cost, 1e-9, expression)
			}
		})
	}
}
