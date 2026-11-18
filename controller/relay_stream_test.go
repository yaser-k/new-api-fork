package controller

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayKeepAliveDuringImagePreparation(t *testing.T) {
	settings := operation_setting.GetGeneralSetting()
	previousEnabled, previousInterval := settings.PingIntervalEnabled, settings.PingIntervalSeconds
	previousCount, previousMedia := constant.CountToken, constant.GetMediaToken
	fetchSettings := system_setting.GetFetchSetting()
	previousFetchSettings := *fetchSettings
	fetchSettings.AllowPrivateIp = true
	fetchSettings.AllowedPorts = []string{"1-65535"}
	settings.PingIntervalEnabled, settings.PingIntervalSeconds = true, 1
	constant.CountToken, constant.GetMediaToken = true, true
	t.Cleanup(func() {
		settings.PingIntervalEnabled, settings.PingIntervalSeconds = previousEnabled, previousInterval
		constant.CountToken, constant.GetMediaToken = previousCount, previousMedia
		*fetchSettings = previousFetchSettings
	})
	service.InitHttpClient()

	imageRequested := make(chan struct{})
	releaseImage := make(chan struct{})
	var releaseOnce sync.Once
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(imageRequested)
		<-releaseImage
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer imageServer.Close()
	engine := gin.New()
	engine.POST("/v1/messages", func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyOriginalModel, "claude-sonnet-4-20250514")
		Relay(c, types.RelayFormatClaude)
	})
	gateway := httptest.NewServer(engine)
	defer gateway.Close()
	defer releaseOnce.Do(func() { close(releaseImage) })

	body := `{"model":"claude-sonnet-4-20250514","max_tokens":16,"stream":true,"messages":[{"role":"user","content":[{"type":"image","source":{"type":"url","url":"` + imageServer.URL + `"}}]}]}`
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Post(gateway.URL+"/v1/messages", "application/json", strings.NewReader(body))
	require.NoError(t, err, "the gateway must send keepalive while its image download is blocked")
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))
	reader := bufio.NewReader(response.Body)
	for range 2 {
		line, readErr := reader.ReadString('\n')
		require.NoError(t, readErr)
		require.Equal(t, ": PING\n", line)
		line, readErr = reader.ReadString('\n')
		require.NoError(t, readErr)
		require.Equal(t, "\n", line)
	}
	select {
	case <-imageRequested:
	default:
		t.Fatal("the request never reached image preparation")
	}
	releaseOnce.Do(func() { close(releaseImage) })
	remaining, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Contains(t, string(remaining), "event: error\n")
	require.Contains(t, string(remaining), `"type":"error"`)
	require.Contains(t, string(remaining), "data: ")
}

func TestRelayKeepAliveAcrossUpstreamRetries(t *testing.T) {
	settings := operation_setting.GetGeneralSetting()
	previousEnabled, previousInterval := settings.PingIntervalEnabled, settings.PingIntervalSeconds
	settings.PingIntervalEnabled, settings.PingIntervalSeconds = true, 1
	t.Cleanup(func() {
		settings.PingIntervalEnabled, settings.PingIntervalSeconds = previousEnabled, previousInterval
	})
	service.InitHttpClient()
	var attempts atomic.Int32
	retryStarted, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		close(retryStarted)
		select {
		case <-release:
			_, _ = io.WriteString(w, "data: ok\n\n")
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()
	engine := gin.New()
	engine.POST("/v1/messages", func(c *gin.Context) {
		stop := helper.StartStreamKeepAlive(c, 10*time.Millisecond)
		defer stop()
		if !assert.NoError(t, http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(5*time.Second))) {
			return
		}
		info := &relaycommon.RelayInfo{IsStream: true, ChannelMeta: &relaycommon.ChannelMeta{}}
		for range 2 {
			request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstream.URL, strings.NewReader("{}"))
			if !assert.NoError(t, err) {
				return
			}
			response, err := channel.DoRequest(c, request, info)
			if !assert.NoError(t, err) {
				return
			}
			if response.StatusCode == http.StatusTooManyRequests {
				response.Body.Close()
				continue
			}
			_, _ = io.Copy(c.Writer, response.Body)
			response.Body.Close()
		}
	})
	gateway := httptest.NewServer(engine)
	defer gateway.Close()
	defer releaseOnce.Do(func() { close(release) })
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Post(gateway.URL+"/v1/messages", "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	defer response.Body.Close()
	select {
	case <-retryStarted:
	case <-time.After(time.Second):
		t.Fatal("the upstream retry did not start")
	}
	reader := bufio.NewReader(response.Body)
	for range 2 {
		line, readErr := reader.ReadString('\n')
		require.NoError(t, readErr)
		require.Equal(t, ": PING\n", line)
		line, readErr = reader.ReadString('\n')
		require.NoError(t, readErr)
		require.Equal(t, "\n", line)
	}
	releaseOnce.Do(func() { close(release) })
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Contains(t, string(body), "data: ok\n\n")
	require.Equal(t, int32(2), attempts.Load())
}

func TestRelayKeepAlivePreservesValidationErrors(t *testing.T) {
	settings := operation_setting.GetGeneralSetting()
	previousEnabled := settings.PingIntervalEnabled
	settings.PingIntervalEnabled = true
	t.Cleanup(func() { settings.PingIntervalEnabled = previousEnabled })
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-4-20250514","stream":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	Relay(c, types.RelayFormatClaude)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.NotContains(t, recorder.Body.String(), ": PING")
}

func TestStreamKeepAlivePreservesFastHTTPFailures(t *testing.T) {
	settings := operation_setting.GetGeneralSetting()
	previousEnabled, previousInterval := settings.PingIntervalEnabled, settings.PingIntervalSeconds
	settings.PingIntervalEnabled, settings.PingIntervalSeconds = true, 60
	t.Cleanup(func() {
		settings.PingIntervalEnabled, settings.PingIntervalSeconds = previousEnabled, previousInterval
	})
	service.InitHttpClient()
	for _, status := range []int{http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"error":"request rejected"}`)
			}))
			defer upstream.Close()
			engine := gin.New()
			engine.POST("/v1/messages", func(c *gin.Context) {
				stop := helper.StartStreamKeepAlive(c, time.Minute)
				defer stop()
				request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstream.URL, strings.NewReader("{}"))
				if !assert.NoError(t, err) {
					return
				}
				info := &relaycommon.RelayInfo{IsStream: true, ChannelMeta: &relaycommon.ChannelMeta{}}
				response, err := channel.DoRequest(c, request, info)
				if !assert.NoError(t, err) {
					return
				}
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				if assert.NoError(t, err) {
					c.Header("Content-Type", "application/json")
					c.Data(response.StatusCode, "application/json", body)
				}
			})
			gateway := httptest.NewServer(engine)
			defer gateway.Close()
			client := &http.Client{Timeout: 4 * time.Second}
			response, err := client.Post(gateway.URL+"/v1/messages", "application/json", strings.NewReader("{}"))
			require.NoError(t, err)
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, status, response.StatusCode)
			require.Equal(t, "application/json", response.Header.Get("Content-Type"))
			require.JSONEq(t, `{"error":"request rejected"}`, string(body))
		})
	}
}

func TestRelayKeepAliveHonorsStreamSettings(t *testing.T) {
	settings := operation_setting.GetGeneralSetting()
	previousEnabled := settings.PingIntervalEnabled
	previousCount, previousMedia, previousNonStreamMedia := constant.CountToken, constant.GetMediaToken, constant.GetMediaTokenNotStream
	constant.CountToken, constant.GetMediaToken, constant.GetMediaTokenNotStream = true, true, true
	t.Cleanup(func() {
		settings.PingIntervalEnabled = previousEnabled
		constant.CountToken, constant.GetMediaToken, constant.GetMediaTokenNotStream = previousCount, previousMedia, previousNonStreamMedia
	})
	for _, test := range []struct {
		name    string
		enabled bool
		stream  bool
	}{
		{name: "ping disabled", enabled: false, stream: true},
		{name: "non-streaming", enabled: true, stream: false},
		{name: "fast streaming failure", enabled: true, stream: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			settings.PingIntervalEnabled = test.enabled
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			body := `{"model":"claude-sonnet-4-20250514","max_tokens":16,"stream":` + strconv.FormatBool(test.stream) + `,"messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"!"}}]}]}`
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, "claude-sonnet-4-20250514")
			Relay(c, types.RelayFormatClaude)
			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
			require.NotContains(t, recorder.Body.String(), ": PING")
			var response map[string]any
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, "error", response["type"])
		})
	}
}

func TestRelayStreamErrorAfterKeepAlive(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/stream", nil)
			helper.SetEventStreamHeaders(c)
			require.NoError(t, helper.PingData(c))
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://provider.example")
			common.SetContextKey(c, constant.ContextKeyUpstreamRequestOrigin, "https://provider.example")
			apiError := types.NewErrorWithStatusCode(errors.New("upstream rejected input at https://provider.example/v1/messages"), types.ErrorCodeBadResponseStatusCode, http.StatusBadRequest)
			writeRelayStreamError(c, format, apiError)
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
			require.Contains(t, recorder.Body.String(), "upstream rejected input")
			require.NotContains(t, recorder.Body.String(), "provider.example")
			status, ok := common.GetContextKeyType[*relaycommon.StreamStatus](c, constant.ContextKeyResponseStreamStatus)
			require.True(t, ok)
			require.True(t, status.ResponseFailed(), "stream errors must not consume the successful-request rate limit")
			require.Equal(t, http.StatusBadRequest, status.OutcomeSnapshot().ErrorStatus)
			if format == types.RelayFormatClaude {
				require.Contains(t, recorder.Body.String(), "event: error\n")
			} else {
				require.Contains(t, recorder.Body.String(), "data: [DONE]")
			}
		})
	}
}

func TestStreamKeepAliveCancellation(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Request = httptest.NewRequest(http.MethodPost, "/stream", nil).WithContext(requestContext)
	stop := helper.StartStreamKeepAlive(c, time.Hour)
	cancel()
	stopped := make(chan struct{})
	go func() {
		stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("keepalive did not stop after client cancellation")
	}
	_, err := c.Writer.WriteString("data: done\n\n")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(recorder.Body.String(), "data: done\n\n"))
}

// A client that leaves ends its non-streaming upstream request: the upstream sees the request
// cancelled at once, no other attempt starts, and the reservation is returned because no answer
// reached the client. A slow answer the client still waits for is read in full and billed.
func TestRelayUpstreamRequestFollowsTheClient(t *testing.T) {
	const hold = 2 * time.Second
	for _, tc := range []struct {
		name         string
		headersFirst bool
		clientLeaves bool
	}{
		{name: "client leaves before the upstream headers", clientLeaves: true},
		{name: "client leaves after the upstream headers", headersFirst: true, clientLeaves: true},
		{name: "slow but live answer", headersFirst: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newResponsesWSBillingTest(t, `tier("request", fixed(0.002))`, func(*websocket.Conn, *http.Request) {})
			previousRetries, previousErrorLog := common.RetryTimes, constant.ErrorLogEnabled
			common.RetryTimes, constant.ErrorLogEnabled = 2, true
			t.Cleanup(func() { common.RetryTimes, constant.ErrorLogEnabled = previousRetries, previousErrorLog })
			var attempts atomic.Int64
			requested, upstreamCancelled := make(chan struct{}, 3), make(chan struct{}, 3)
			fixture.httpUpstream = func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				// Reading the request lets the server notice when the gateway drops it.
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("Content-Type", "application/json")
				if tc.headersFirst {
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
				}
				requested <- struct{}{}
				wait := hold
				if !tc.clientLeaves {
					wait = 300 * time.Millisecond
				}
				select {
				case <-r.Context().Done():
					upstreamCancelled <- struct{}{}
					return
				case <-time.After(wait):
				}
				_, _ = fmt.Fprint(w, `{"id":"resp_1","object":"response","status":"completed","model":"gpt-4o","output":[],"usage":{"input_tokens":10,"output_tokens":1,"total_tokens":11}}`)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, fixture.gatewayURL+"/v1/responses", strings.NewReader(`{"model":"ws-billing","input":"hello"}`))
			require.NoError(t, err)
			request.Header.Set("Authorization", "Bearer sk-"+fixture.token.Key)
			request.Header.Set("Content-Type", "application/json")
			status := make(chan int, 1)
			go func() {
				response, err := http.DefaultClient.Do(request)
				if err != nil {
					status <- 0
					return
				}
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
				status <- response.StatusCode
			}()
			select {
			case <-requested:
			case <-time.After(3 * time.Second):
				t.Fatal("the upstream never received the request")
			}
			if tc.clientLeaves {
				cancel()
				select {
				case <-upstreamCancelled:
				case <-time.After(hold / 2):
					t.Error("the upstream request was not cancelled when the client left")
				}
			}
			select {
			case <-fixture.httpDone:
			case <-time.After(2 * hold):
				t.Fatal("HTTP request did not finish")
			}
			if !tc.clientLeaves {
				assert.Equal(t, http.StatusOK, <-status)
			}

			charged, consumeRows, errorRows := 0, int64(0), int64(1)
			if !tc.clientLeaves {
				charged, consumeRows, errorRows = 1000, 1, 0
			}
			assert.Equal(t, int64(1), attempts.Load(), "no other attempt starts")
			// A refund is applied asynchronously.
			quotas := func() (int, int) {
				var user model.User
				var token model.Token
				_ = model.DB.First(&user, fixture.user.Id).Error
				_ = model.DB.First(&token, fixture.token.Id).Error
				return user.Quota, token.RemainQuota
			}
			assert.Eventually(t, func() bool {
				userQuota, tokenQuota := quotas()
				return userQuota == 100000-charged && tokenQuota == 3000-charged
			}, 2*time.Second, 20*time.Millisecond)
			userQuota, tokenQuota := quotas()
			assert.Equal(t, 100000-charged, userQuota, "user quota")
			assert.Equal(t, 3000-charged, tokenQuota, "token quota")
			countRows := func(logType int) int64 {
				var count int64
				require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("token_id = ? AND type = ?", fixture.token.Id, logType).Count(&count).Error)
				return count
			}
			assert.Equal(t, consumeRows, countRows(model.LogTypeConsume), "consume rows")
			assert.Equal(t, errorRows, countRows(model.LogTypeError), "error rows: one per attempt")
		})
	}
}
