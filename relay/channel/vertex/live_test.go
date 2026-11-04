package vertex

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const liveFinalText = "The quick brown fox jumps over the lazy dog. Testing 1 2 3."

// liveMeasuredSession is the measured gemini-3.5-transcribe-live session after the audio: twelve cumulative
// interims (the first four as measured, one revising the case of the one before; the rest stand-ins
// ending in the measured last one), the final transcription with its usage, and the end of activity.
func liveMeasuredSession() []string {
	interims := []string{
		"The", "The Quick", "The quick", "The quick brown fox", "The quick brown fox jumps",
		"The quick brown fox jumps over", "The quick brown fox jumps over the lazy",
		"The quick brown fox jumps over the lazy dog.", "The quick brown fox jumps over the lazy dog. Testing",
		"The quick brown fox jumps over the lazy dog. Testing 1", "The quick brown fox jumps over the lazy dog. Testing 1 2",
		"The quick brown fox jumps over the lazy dog. Testing 1 2 3",
	}
	messages := []string{`{"voiceActivity":{"type":"ACTIVITY_START","audioOffset":"0.360s"}}`}
	for _, text := range interims {
		messages = append(messages, `{"serverContent":{"interimInputTranscription":{"text":"`+text+`"}}}`)
	}
	return append(messages,
		`{"serverContent":{"inputTranscription":{"text":"`+liveFinalText+`"}},"usageMetadata":{"promptTokenCount":272,"candidatesTokenCount":18,"totalTokenCount":290,`+
			`"promptTokensDetails":[{"modality":"AUDIO","tokenCount":272}],"candidatesTokensDetails":[{"modality":"TEXT","tokenCount":18}]}}`,
		`{"serverContent":{"generationComplete":true}}`,
		`{"voiceActivity":{"type":"ACTIVITY_END","audioOffset":"10.850937500s"}}`,
	)
}

// fakeLiveAPI answers setup with setupComplete and the end of the audio with replay, records what it
// receives, and with closeAfterSetup ends the session itself as an upstream error would.
type fakeLiveAPI struct {
	*httptest.Server
	mu            sync.Mutex
	authorization string
	path          string
	received      []string
	ended         chan struct{}
	// replayDelay is how long the upstream takes to answer the end of the audio.
	replayDelay time.Duration
}

func newFakeLiveAPI(t *testing.T, replay []string, closeAfterSetup bool) *fakeLiveAPI {
	api := &fakeLiveAPI{ended: make(chan struct{})}
	upgrader := websocket.Upgrader{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(api.ended)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		api.mu.Lock()
		api.authorization, api.path = r.Header.Get("Authorization"), r.URL.Path
		api.mu.Unlock()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			api.mu.Lock()
			api.received = append(api.received, string(message))
			api.mu.Unlock()
			switch {
			case strings.Contains(string(message), `"setup"`):
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"setupComplete":{"sessionId":"5d6f0c3e-live"}}`))
				if closeAfterSetup {
					_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "internal error"), time.Now().Add(time.Second))
					return
				}
			case strings.Contains(string(message), `"audioStreamEnd"`) && replay != nil:
				api.mu.Lock()
				delay := api.replayDelay
				api.mu.Unlock()
				time.Sleep(delay)
				for _, event := range replay {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(event))
				}
				replay = nil
			}
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func (api *fakeLiveAPI) messages() []string {
	api.mu.Lock()
	defer api.mu.Unlock()
	return append([]string(nil), api.received...)
}

type liveGatewayResult struct {
	usage  *dto.RealtimeUsage
	apiErr *types.NewAPIError
}

// newLiveGateway serves the realtime route as the relay does: the client's WebSocket, the adaptor's
// upstream dial, then its response handler, whose usage the relay settles.
func newLiveGateway(t *testing.T, upstreamURL string, channelID int) (string, chan liveGatewayResult) {
	results := make(chan liveGatewayResult, 1)
	upgrader := websocket.Upgrader{}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer clientConn.Close()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r
		info := liveRelayInfo(upstreamURL, "gemini-3.5-transcribe-live", channelID)
		info.ClientWs = clientConn

		adaptor := &Adaptor{}
		adaptor.Init(info)
		target, err := adaptor.DoRequest(c, info, nil)
		if err != nil {
			results <- liveGatewayResult{apiErr: types.NewError(err, types.ErrorCodeDoRequestFailed)}
			return
		}
		info.TargetWs = target.(*websocket.Conn)
		defer info.TargetWs.Close()
		usage, apiErr := adaptor.DoResponse(c, nil, info)
		result := liveGatewayResult{apiErr: apiErr}
		if apiErr == nil {
			result.usage = usage.(*dto.RealtimeUsage)
		}
		results <- result
	}))
	t.Cleanup(gateway.Close)
	return "ws" + strings.TrimPrefix(gateway.URL, "http") + "/v1/realtime?model=gemini-3.5-transcribe-live", results
}

func liveRelayInfo(baseURL, model string, channelID int) *relaycommon.RelayInfo {
	// The service account's token is the one cached for the channel, so no call reaches Google.
	Cache.SetDefault(fmt.Sprintf("access-token-%d", channelID), "test-token")
	return &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeRealtime,
		RelayFormat:     types.RelayFormatOpenAIRealtime,
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         channelID,
			ApiVersion:        "global",
			ApiKey:            `{"project_id":"my-project","client_email":"relay@my-project.iam.gserviceaccount.com"}`,
			ChannelBaseUrl:    baseURL,
			UpstreamModelName: model,
		},
	}
}

func readLiveEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	var event map[string]any
	require.NoError(t, common.Unmarshal(message, &event))
	return event
}

func sendLiveEvent(t *testing.T, conn *websocket.Conn, event string) {
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(event)))
}

func awaitLiveResult(t *testing.T, results chan liveGatewayResult, api *fakeLiveAPI) liveGatewayResult {
	var result liveGatewayResult
	select {
	case result = <-results:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the realtime session did not end")
	}
	select {
	case <-api.ended:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the upstream session was not closed")
	}
	return result
}

func TestVertexLiveTranscriptionSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseline := runtime.NumGoroutine()

	t.Run("audio in, transcription and usage out", func(t *testing.T) {
		api := newFakeLiveAPI(t, liveMeasuredSession(), false)
		gatewayURL, results := newLiveGateway(t, api.URL, 9101)
		client, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
		require.NoError(t, err)

		created := readLiveEvent(t, client)
		assert.Equal(t, "transcription_session.created", created["type"])
		sendLiveEvent(t, client, `{"type":"transcription_session.update","session":{"input_audio_format":"pcm16","input_audio_transcription":{"model":"gemini-3.5-transcribe-live"}}}`)
		assert.Equal(t, "transcription_session.updated", readLiveEvent(t, client)["type"])
		sendLiveEvent(t, client, `{"event_id":"evt_7","type":"response.create"}`)
		refused := readLiveEvent(t, client)
		assert.Equal(t, "error", refused["type"])
		assert.Equal(t, map[string]any{"type": "invalid_request_error", "code": "unsupported_event", "message": "response.create is not supported by a Gemini Live transcription session", "param": nil, "event_id": "evt_7"}, refused["error"])

		var frames []string
		for i := range 3 {
			frame := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{byte(i + 1)}, 4800))
			frames = append(frames, frame)
			sendLiveEvent(t, client, `{"type":"input_audio_buffer.append","audio":"`+frame+`"}`)
		}
		sendLiveEvent(t, client, `{"type":"input_audio_buffer.commit"}`)

		var deltas strings.Builder
		var started, completed, stopped map[string]any
		committed := false
		for stopped == nil || !committed {
			event := readLiveEvent(t, client)
			switch event["type"] {
			case "input_audio_buffer.committed":
				committed = true
			case "input_audio_buffer.speech_started":
				started = event
			case "conversation.item.input_audio_transcription.delta":
				deltas.WriteString(event["delta"].(string))
			case "conversation.item.input_audio_transcription.completed":
				completed = event
			case "input_audio_buffer.speech_stopped":
				stopped = event
			default:
				require.Failf(t, "unexpected event", "%v", event)
			}
		}
		require.NotNil(t, started)
		require.NotNil(t, completed)
		assert.EqualValues(t, 360, started["audio_start_ms"])
		assert.EqualValues(t, 10850, stopped["audio_end_ms"])
		assert.Equal(t, started["item_id"], completed["item_id"])
		assert.Equal(t, started["item_id"], stopped["item_id"])
		// The deltas the client appends rebuild the final text, the revised word included.
		assert.Equal(t, liveFinalText, deltas.String())
		assert.Equal(t, liveFinalText, completed["transcript"])
		assert.Equal(t, map[string]any{"type": "tokens", "input_tokens": float64(272), "input_token_details": map[string]any{"text_tokens": float64(0), "audio_tokens": float64(272)}, "output_tokens": float64(18), "total_tokens": float64(290)}, completed["usage"])

		require.NoError(t, client.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
		_ = client.Close()
		result := awaitLiveResult(t, results, api)
		require.Nil(t, result.apiErr)
		// The usage the relay settles with PostWssConsumeQuota: audio in as audio, text out as text.
		assert.Equal(t, dto.RealtimeUsage{
			TotalTokens:        290,
			InputTokens:        272,
			OutputTokens:       18,
			InputTokenDetails:  dto.InputTokenDetails{AudioTokens: 272},
			OutputTokenDetails: dto.OutputTokenDetails{TextTokens: 18},
		}, *result.usage)

		api.mu.Lock()
		assert.Equal(t, "Bearer test-token", api.authorization)
		assert.Equal(t, LiveAPIPath, api.path)
		api.mu.Unlock()
		received := api.messages()
		require.GreaterOrEqual(t, len(received), 5)
		assert.JSONEq(t, `{"setup":{"model":"projects/my-project/locations/global/publishers/google/models/gemini-3.5-transcribe-live","generationConfig":{"responseModalities":["TEXT"]},"inputAudioTranscription":{}}}`, received[0])
		for i, frame := range frames {
			assert.JSONEq(t, `{"realtimeInput":{"audio":{"mimeType":"audio/pcm;rate=24000","data":"`+frame+`"}}}`, received[1+i])
		}
		assert.JSONEq(t, `{"realtimeInput":{"audioStreamEnd":true}}`, received[4])
	})

	t.Run("a client close closes the upstream", func(t *testing.T) {
		api := newFakeLiveAPI(t, nil, false)
		gatewayURL, results := newLiveGateway(t, api.URL, 9102)
		client, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
		require.NoError(t, err)
		assert.Equal(t, "transcription_session.created", readLiveEvent(t, client)["type"])
		_ = client.Close()

		result := awaitLiveResult(t, results, api)
		require.Nil(t, result.apiErr)
		assert.Equal(t, dto.RealtimeUsage{}, *result.usage)
		received := api.messages()
		assert.JSONEq(t, `{"realtimeInput":{"audioStreamEnd":true}}`, received[len(received)-1])
	})

	t.Run("an upstream close closes the client", func(t *testing.T) {
		api := newFakeLiveAPI(t, nil, true)
		gatewayURL, results := newLiveGateway(t, api.URL, 9103)
		client, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
		require.NoError(t, err)
		defer client.Close()
		assert.Equal(t, "transcription_session.created", readLiveEvent(t, client)["type"])
		failure := readLiveEvent(t, client)
		assert.Equal(t, "error", failure["type"])
		assert.Equal(t, "upstream_error", failure["error"].(map[string]any)["code"])
		_, _, err = client.ReadMessage()
		assert.True(t, websocket.IsCloseError(err, websocket.CloseNormalClosure), "%v", err)

		result := awaitLiveResult(t, results, api)
		require.Nil(t, result.apiErr)
	})

	t.Run("the session limit closes both sides", func(t *testing.T) {
		saved := liveSessionLimit
		liveSessionLimit = 300 * time.Millisecond
		t.Cleanup(func() { liveSessionLimit = saved })
		api := newFakeLiveAPI(t, nil, false)
		gatewayURL, results := newLiveGateway(t, api.URL, 9104)
		client, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
		require.NoError(t, err)
		defer client.Close()
		assert.Equal(t, "transcription_session.created", readLiveEvent(t, client)["type"])
		expired := readLiveEvent(t, client)
		assert.Equal(t, "session_expired", expired["error"].(map[string]any)["code"])

		result := awaitLiveResult(t, results, api)
		require.Nil(t, result.apiErr)
	})

	// Every relay goroutine and connection of the sessions above has ended. The count is polled from this
	// goroutine: a polling helper's own goroutines would be counted.
	for deadline := time.Now().Add(5 * time.Second); runtime.NumGoroutine() > baseline && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	assert.LessOrEqual(t, runtime.NumGoroutine(), baseline)
}

func TestVertexRealtimeRefusesWhatLiveCannotServe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := newFakeLiveAPI(t, nil, false)
	tests := []struct {
		name      string
		model     string
		apiKey    bool
		wantError string
	}{
		{"chat model", "gemini-2.5-flash", false, "model gemini-2.5-flash does not serve the realtime route on Vertex AI: only Gemini Live transcription models do"},
		{"batch transcription model", "gemini-3.5-transcribe", false, "model gemini-3.5-transcribe does not serve the realtime route on Vertex AI: only Gemini Live transcription models do"},
		{"api key channel", "gemini-3.5-transcribe-live", true, "the realtime route on Vertex AI needs a service account key, not an API key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime?model="+tt.model, nil)
			info := liveRelayInfo(api.URL, tt.model, 9105)
			if tt.apiKey {
				info.ChannelOtherSettings.VertexKeyType = dto.VertexKeyTypeAPIKey
			}
			adaptor := &Adaptor{}
			adaptor.Init(info)

			_, err := adaptor.DoRequest(c, info, nil)
			apiErr := types.NewError(err, types.ErrorCodeDoRequestFailed)
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.Equal(t, tt.wantError, apiErr.Error())
			assert.Empty(t, api.messages())
		})
	}
}

// A session that ends before the upstream reports the usage of its audio is still billed: the upstream
// stays open for the grace period, and audio it never reports is billed by its length.
func TestVertexLiveTranscriptionGracePeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseline := runtime.NumGoroutine()

	t.Run("usage that arrives after the client closed is settled", func(t *testing.T) {
		api := newFakeLiveAPI(t, liveMeasuredSession(), false)
		api.mu.Lock()
		api.replayDelay = 200 * time.Millisecond
		api.mu.Unlock()
		gatewayURL, results := newLiveGateway(t, api.URL, 9201)
		client, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
		require.NoError(t, err)
		assert.Equal(t, "transcription_session.created", readLiveEvent(t, client)["type"])
		for i := range 3 {
			sendLiveEvent(t, client, `{"type":"input_audio_buffer.append","audio":"`+base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{byte(i + 1)}, 4800))+`"}`)
		}
		// The client closes right after its last audio frame, before any transcription.
		_ = client.Close()

		result := awaitLiveResult(t, results, api)
		require.Nil(t, result.apiErr)
		assert.Equal(t, dto.RealtimeUsage{
			TotalTokens:        290,
			InputTokens:        272,
			OutputTokens:       18,
			InputTokenDetails:  dto.InputTokenDetails{AudioTokens: 272},
			OutputTokenDetails: dto.OutputTokenDetails{TextTokens: 18},
		}, *result.usage)
		received := api.messages()
		require.Len(t, received, 5)
		assert.JSONEq(t, `{"realtimeInput":{"audioStreamEnd":true}}`, received[4])
	})

	t.Run("audio the upstream never reports is billed by its length", func(t *testing.T) {
		saved := liveUsageGrace
		liveUsageGrace = 300 * time.Millisecond
		t.Cleanup(func() { liveUsageGrace = saved })
		api := newFakeLiveAPI(t, nil, false)
		gatewayURL, results := newLiveGateway(t, api.URL, 9202)
		client, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
		require.NoError(t, err)
		assert.Equal(t, "transcription_session.created", readLiveEvent(t, client)["type"])
		// 10.85 s of pcm16 at 24 kHz, the measured sample's length: 108 frames of 100 ms and one of 50 ms.
		frame := base64.StdEncoding.EncodeToString(make([]byte, 4800))
		for range 108 {
			sendLiveEvent(t, client, `{"type":"input_audio_buffer.append","audio":"`+frame+`"}`)
		}
		sendLiveEvent(t, client, `{"type":"input_audio_buffer.append","audio":"`+base64.StdEncoding.EncodeToString(make([]byte, 2400))+`"}`)
		_ = client.Close()

		// The upstream never closes by itself: the session closes it when the grace period ends.
		result := awaitLiveResult(t, results, api)
		require.Nil(t, result.apiErr)
		// 520800 bytes / 48000 = 10.85 s at 25 tokens a second, rounded up: the measured 272 audio tokens.
		assert.Equal(t, dto.RealtimeUsage{
			TotalTokens:       272,
			InputTokens:       272,
			InputTokenDetails: dto.InputTokenDetails{AudioTokens: 272},
		}, *result.usage)
	})

	// No goroutine or connection of the sessions above outlives the grace period.
	for deadline := time.Now().Add(5 * time.Second); runtime.NumGoroutine() > baseline && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	assert.LessOrEqual(t, runtime.NumGoroutine(), baseline)
}
