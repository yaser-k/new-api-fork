package aws

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math"
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
	relaytypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockruntimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// stubSonicStream stands in for Bedrock's side of a bidirectional stream: it records the input events
// the session sends, and plays the output events a test pushes.
type stubSonicStream struct {
	mu      sync.Mutex
	inputs  []string
	closed  bool
	err     error
	outputs chan bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutput
	// onInput, when set, answers an input event as Bedrock would.
	onInput func(stream *stubSonicStream, name string)
}

type stubSonicWriter struct{ stream *stubSonicStream }

func (w stubSonicWriter) Send(_ context.Context, event bedrockruntimeTypes.InvokeModelWithBidirectionalStreamInput) error {
	data := string(event.(*bedrockruntimeTypes.InvokeModelWithBidirectionalStreamInputMemberChunk).Value.Bytes)
	w.stream.mu.Lock()
	w.stream.inputs = append(w.stream.inputs, data)
	onInput := w.stream.onInput
	w.stream.mu.Unlock()
	if onInput != nil {
		for name := range gjson.Get(data, "event").Map() {
			onInput(w.stream, name)
		}
	}
	return nil
}

func (w stubSonicWriter) Close() error { return nil }
func (w stubSonicWriter) Err() error   { return nil }

type stubSonicReader struct{ stream *stubSonicStream }

func (r stubSonicReader) Events() <-chan bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutput {
	return r.stream.outputs
}

func (r stubSonicReader) Close() error {
	r.stream.end(nil)
	return nil
}

func (r stubSonicReader) Err() error {
	r.stream.mu.Lock()
	defer r.stream.mu.Unlock()
	return r.stream.err
}

func newStubSonicStream() *stubSonicStream {
	return &stubSonicStream{outputs: make(chan bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutput, 64)}
}

// push plays one output event: name and its JSON body.
func (s *stubSonicStream) push(name, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.outputs <- &bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutputMemberChunk{
		Value: bedrockruntimeTypes.BidirectionalOutputPayloadPart{Bytes: []byte(`{"event":{"` + name + `":` + body + `}}`)},
	}
}

// end ends the output, with err as the stream's error.
func (s *stubSonicStream) end(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed, s.err = true, err
	close(s.outputs)
}

func (s *stubSonicStream) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// inputNames are the names of the input events received, in order.
func (s *stubSonicStream) inputNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.inputs))
	for _, input := range s.inputs {
		for name := range gjson.Get(input, "event").Map() {
			names = append(names, name)
		}
	}
	return names
}

func (s *stubSonicStream) input(i int) gjson.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	return gjson.Get(s.inputs[i], "event")
}

func (s *stubSonicStream) awaitInputs(t *testing.T, count int) {
	require.Eventually(t, func() bool { return len(s.inputNames()) >= count }, 5*time.Second, 5*time.Millisecond)
}

type sonicGatewayResult struct {
	usage  *dto.RealtimeUsage
	apiErr *relaytypes.NewAPIError
}

// useStubSonicStream makes the sessions of a test open stream instead of calling Bedrock, and records the
// model ids they open.
func useStubSonicStream(t *testing.T, stream *stubSonicStream) *[]string {
	saved := openNovaSonicStream
	t.Cleanup(func() { openNovaSonicStream = saved })
	var modelIDs []string
	openNovaSonicStream = func(_ context.Context, _ *bedrockruntime.Client, modelID string) (*bedrockruntime.InvokeModelWithBidirectionalStreamEventStream, error) {
		modelIDs = append(modelIDs, modelID)
		return bedrockruntime.NewInvokeModelWithBidirectionalStreamEventStream(func(es *bedrockruntime.InvokeModelWithBidirectionalStreamEventStream) {
			es.Writer = stubSonicWriter{stream: stream}
			es.Reader = stubSonicReader{stream: stream}
		}), nil
	}
	return &modelIDs
}

func sonicRelayInfo(model string, relayMode int) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode:       relayMode,
		RelayFormat:     relaytypes.RelayFormatOpenAIRealtime,
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         9301,
			ApiKey:            "AKIDEXAMPLE|secret|us-east-1",
			UpstreamModelName: model,
		},
	}
}

// newSonicGateway serves the realtime route as the relay does: the client's WebSocket, the adaptor's
// request, then its response handler, whose usage the relay settles.
func newSonicGateway(t *testing.T, model string) (string, chan sonicGatewayResult) {
	results := make(chan sonicGatewayResult, 1)
	upgrader := websocket.Upgrader{}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer clientConn.Close()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r
		info := sonicRelayInfo(model, relayconstant.RelayModeRealtime)
		info.ClientWs = clientConn

		adaptor := &Adaptor{}
		adaptor.Init(info)
		if _, err := adaptor.DoRequest(c, info, nil); err != nil {
			results <- sonicGatewayResult{apiErr: relaytypes.NewError(err, relaytypes.ErrorCodeDoRequestFailed)}
			return
		}
		usage, apiErr := adaptor.DoResponse(c, nil, info)
		result := sonicGatewayResult{apiErr: apiErr}
		if apiErr == nil {
			result.usage = usage.(*dto.RealtimeUsage)
		}
		results <- result
	}))
	t.Cleanup(gateway.Close)
	return "ws" + strings.TrimPrefix(gateway.URL, "http") + "/v1/realtime?model=" + model, results
}

func dialSonicGateway(t *testing.T, gatewayURL string) *websocket.Conn {
	client, _, err := websocket.DefaultDialer.Dial(gatewayURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	assert.Equal(t, "session.created", readSonicEvent(t, client)["type"])
	return client
}

func readSonicEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	var event map[string]any
	require.NoError(t, common.Unmarshal(message, &event))
	return event
}

func sendSonicEvent(t *testing.T, conn *websocket.Conn, event string) {
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(event)))
}

func awaitSonicResult(t *testing.T, results chan sonicGatewayResult) sonicGatewayResult {
	select {
	case result := <-results:
		return result
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the realtime session did not end")
		return sonicGatewayResult{}
	}
}

// awaitGoroutines waits for the goroutine count to return to baseline; it polls from the test's own
// goroutine, as a polling helper's goroutines would be counted.
func awaitGoroutines(t *testing.T, baseline int) {
	for deadline := time.Now().Add(5 * time.Second); runtime.NumGoroutine() > baseline && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	assert.LessOrEqual(t, runtime.NumGoroutine(), baseline)
}

// sonicTone is pcm16 mono samples of a 440 Hz tone at the given rate, amplitude 10000.
func sonicTone(rate, samples int) []byte {
	data := make([]byte, 0, samples*2)
	for i := range samples {
		value := int16(math.Round(10000 * math.Sin(2*math.Pi*440*float64(i)/float64(rate))))
		data = binary.LittleEndian.AppendUint16(data, uint16(value))
	}
	return data
}

func TestNovaSonicRefusesNonRealtimeRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modelIDs := useStubSonicStream(t, newStubSonicStream())
	for _, model := range []string{"nova-2-sonic-v1:0", "nova-2-5-sonic", "amazon.nova-2-5-sonic"} {
		t.Run(model, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := sonicRelayInfo(model, relayconstant.RelayModeChatCompletions)
			_, err := (&Adaptor{}).DoRequest(c, info, nil)
			apiErr := relaytypes.NewError(err, relaytypes.ErrorCodeDoRequestFailed)
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.Equal(t, "model "+model+" is a speech-to-speech model served only on the realtime route (/v1/realtime)", apiErr.Error())
		})
	}
	t.Run("a realtime request for a model that is not Nova Sonic", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
		_, err := (&Adaptor{}).DoRequest(c, sonicRelayInfo("nova-pro-v1:0", relayconstant.RelayModeRealtime), nil)
		apiErr := relaytypes.NewError(err, relaytypes.ErrorCodeDoRequestFailed)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
		assert.Equal(t, "model nova-pro-v1:0 does not serve the realtime route on AWS: only Nova Sonic models do", apiErr.Error())
	})
	assert.Empty(t, *modelIDs)
}

func TestNovaSonicSessionRefusals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseline := runtime.NumGoroutine()

	tests := []struct {
		name    string
		model   string
		event   string
		code    string
		message string
	}{
		{"g711 input audio", "nova-2-sonic-v1:0", `{"event_id":"evt_1","type":"session.update","session":{"input_audio_format":"g711_ulaw"}}`,
			"unsupported_audio_format", "Nova Sonic sessions take and return pcm16 audio at 24 kHz only; g711_ulaw and g711_alaw are not supported"},
		{"16 kHz input audio", "nova-2-5-sonic", `{"event_id":"evt_1","type":"session.update","session":{"audio":{"input":{"format":{"type":"audio/pcm","rate":16000}}}}}`,
			"unsupported_audio_format", "Nova Sonic sessions take and return pcm16 audio at 24 kHz only; g711_ulaw and g711_alaw are not supported"},
		{"text to Nova 2 Sonic", "nova-2-sonic-v1:0", `{"event_id":"evt_1","type":"conversation.item.create","item":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}}`,
			"text_input_not_supported", "nova-2-sonic-v1:0 takes audio input only"},
		{"tools", "nova-2-5-sonic", `{"event_id":"evt_1","type":"session.update","session":{"tools":[{"type":"function","name":"get_weather","parameters":{}}]}}`,
			"tools_not_supported", "Nova Sonic sessions on this gateway do not take tools"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := newStubSonicStream()
			useStubSonicStream(t, stream)
			gatewayURL, results := newSonicGateway(t, tt.model)
			client := dialSonicGateway(t, gatewayURL)

			sendSonicEvent(t, client, tt.event)
			refused := readSonicEvent(t, client)
			assert.Equal(t, "error", refused["type"])
			assert.Equal(t, map[string]any{"type": "invalid_request_error", "code": tt.code, "message": tt.message, "param": nil, "event_id": "evt_1"}, refused["error"])
			_ = client.Close()

			result := awaitSonicResult(t, results)
			require.Nil(t, result.apiErr)
			assert.Equal(t, dto.RealtimeUsage{}, *result.usage)
			// Nothing reached Bedrock: the session had not started.
			assert.Empty(t, stream.inputNames())
			assert.True(t, stream.isClosed())
		})
	}

	t.Run("an upstream error mid-session ends both sides", func(t *testing.T) {
		stream := newStubSonicStream()
		useStubSonicStream(t, stream)
		gatewayURL, results := newSonicGateway(t, "nova-2-sonic-v1:0")
		client := dialSonicGateway(t, gatewayURL)

		sendSonicEvent(t, client, `{"type":"input_audio_buffer.append","audio":"`+base64.StdEncoding.EncodeToString(sonicTone(24000, 2400))+`"}`)
		stream.awaitInputs(t, 4)
		stream.push("usageEvent", `{"details":{"total":{"input":{"speechTokens":7,"textTokens":0},"output":{"speechTokens":0,"textTokens":0}}}}`)
		stream.end(errors.New("ModelStreamErrorException: the model stream failed"))

		failure := readSonicEvent(t, client)
		assert.Equal(t, "error", failure["type"])
		assert.Equal(t, map[string]any{"type": "server_error", "code": "upstream_error", "message": "the Nova Sonic stream ended: ModelStreamErrorException: the model stream failed", "param": nil}, failure["error"])
		_, _, err := client.ReadMessage()
		assert.True(t, websocket.IsCloseError(err, websocket.CloseNormalClosure), "%v", err)

		result := awaitSonicResult(t, results)
		require.Nil(t, result.apiErr)
		// The usage reported before the error is billed.
		assert.Equal(t, dto.RealtimeUsage{TotalTokens: 7, InputTokens: 7, InputTokenDetails: dto.InputTokenDetails{AudioTokens: 7}}, *result.usage)
		// The upstream has ended, so no closing event is sent to it.
		assert.Equal(t, []string{"sessionStart", "promptStart", "contentStart", "audioInput"}, stream.inputNames())
	})

	awaitGoroutines(t, baseline)
}

func TestNovaSonicSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseline := runtime.NumGoroutine()

	t.Run("audio and text in, speech, transcripts and usage out", func(t *testing.T) {
		stream := newStubSonicStream()
		// Bedrock answers the end of the session with the session's last usage, then ends its output.
		stream.onInput = func(stream *stubSonicStream, name string) {
			if name == "sessionEnd" {
				stream.push("usageEvent", `{"details":{"total":{"input":{"speechTokens":160,"textTokens":34},"output":{"speechTokens":95,"textTokens":12}}}}`)
				stream.end(nil)
			}
		}
		modelIDs := useStubSonicStream(t, stream)
		gatewayURL, results := newSonicGateway(t, "nova-2-5-sonic")
		client := dialSonicGateway(t, gatewayURL)

		sendSonicEvent(t, client, `{"type":"session.update","session":{"instructions":"Answer briefly.","voice":"tiffany","temperature":0.6,"max_response_output_tokens":512,`+
			`"turn_detection":{"type":"semantic_vad","eagerness":"high"},"input_audio_format":"pcm16","output_audio_format":"pcm16"}}`)
		updated := readSonicEvent(t, client)
		assert.Equal(t, "session.updated", updated["type"])
		assert.Equal(t, "tiffany", updated["session"].(map[string]any)["voice"])

		// One second of a 440 Hz tone at 24 kHz, in chunks that split samples.
		tone := sonicTone(24000, 24000)
		for _, chunk := range [][]byte{tone[:15999], tone[15999:32000], tone[32000:]} {
			sendSonicEvent(t, client, `{"type":"input_audio_buffer.append","audio":"`+base64.StdEncoding.EncodeToString(chunk)+`"}`)
		}
		sendSonicEvent(t, client, `{"type":"conversation.item.create","item":{"type":"message","role":"user","content":[{"type":"input_text","text":"What time is it?"}]}}`)
		stream.awaitInputs(t, 12)

		assert.Equal(t, []string{"amazon.nova-2-5-sonic"}, *modelIDs)
		names := stream.inputNames()
		assert.Equal(t, []string{"sessionStart", "promptStart", "contentStart", "textInput", "contentEnd", "contentStart", "audioInput", "audioInput", "audioInput", "contentStart", "textInput", "contentEnd"}, names)
		assert.JSONEq(t, `{"inferenceConfiguration":{"maxTokens":512,"temperature":0.6},"turnDetectionConfiguration":{"endpointingSensitivity":"HIGH"}}`, stream.input(0).Get("sessionStart").Raw)
		promptName := stream.input(1).Get("promptStart.promptName").Str
		require.NotEmpty(t, promptName)
		assert.JSONEq(t, `{"promptName":"`+promptName+`","textOutputConfiguration":{"mediaType":"text/plain"},`+
			`"audioOutputConfiguration":{"mediaType":"audio/lpcm","sampleRateHertz":24000,"sampleSizeBits":16,"channelCount":1,"voiceId":"tiffany","encoding":"base64","audioType":"SPEECH"}}`,
			stream.input(1).Get("promptStart").Raw)
		systemName := stream.input(2).Get("contentStart.contentName").Str
		assert.JSONEq(t, `{"promptName":"`+promptName+`","contentName":"`+systemName+`","type":"TEXT","interactive":false,"role":"SYSTEM","textInputConfiguration":{"mediaType":"text/plain"}}`, stream.input(2).Get("contentStart").Raw)
		assert.JSONEq(t, `{"promptName":"`+promptName+`","contentName":"`+systemName+`","content":"Answer briefly."}`, stream.input(3).Get("textInput").Raw)
		assert.JSONEq(t, `{"promptName":"`+promptName+`","contentName":"`+systemName+`"}`, stream.input(4).Get("contentEnd").Raw)
		audioName := stream.input(5).Get("contentStart.contentName").Str
		assert.JSONEq(t, `{"promptName":"`+promptName+`","contentName":"`+audioName+`","type":"AUDIO","interactive":true,"role":"USER",`+
			`"audioInputConfiguration":{"mediaType":"audio/lpcm","sampleRateHertz":16000,"sampleSizeBits":16,"channelCount":1,"audioType":"SPEECH","encoding":"base64"}}`,
			stream.input(5).Get("contentStart").Raw)

		// The tone reaches Bedrock resampled to 16 kHz: its 16000 samples, still a 440 Hz tone.
		var resampled []byte
		for i := 6; i < 9; i++ {
			assert.Equal(t, audioName, stream.input(i).Get("audioInput.contentName").Str)
			chunk, err := base64.StdEncoding.DecodeString(stream.input(i).Get("audioInput.content").Str)
			require.NoError(t, err)
			resampled = append(resampled, chunk...)
		}
		require.Len(t, resampled, 2*16000)
		for i := 0; i < len(resampled); i += 2 {
			want := 10000 * math.Sin(2*math.Pi*440*float64(i/2)/16000)
			require.InDelta(t, want, float64(int16(binary.LittleEndian.Uint16(resampled[i:]))), 200, "sample %d", i/2)
		}
		textName := stream.input(9).Get("contentStart.contentName").Str
		assert.JSONEq(t, `{"promptName":"`+promptName+`","contentName":"`+textName+`","type":"TEXT","interactive":true,"role":"USER","textInputConfiguration":{"mediaType":"text/plain"}}`, stream.input(9).Get("contentStart").Raw)
		assert.JSONEq(t, `{"promptName":"`+promptName+`","contentName":"`+textName+`","content":"What time is it?"}`, stream.input(10).Get("textInput").Raw)

		// A turn: the user's transcript, the assistant's speculative text (not sent), speech, final text, usage.
		stream.push("contentStart", `{"contentId":"c1","type":"TEXT","role":"USER","additionalModelFields":"{\"generationStage\":\"FINAL\"}"}`)
		stream.push("textOutput", `{"contentId":"c1","role":"USER","content":"what time is it"}`)
		stream.push("contentEnd", `{"contentId":"c1","type":"TEXT","stopReason":"END_TURN"}`)
		stream.push("contentStart", `{"contentId":"c2","type":"TEXT","role":"ASSISTANT","additionalModelFields":"{\"generationStage\":\"SPECULATIVE\"}"}`)
		stream.push("textOutput", `{"contentId":"c2","role":"ASSISTANT","content":"It is noon."}`)
		stream.push("contentEnd", `{"contentId":"c2","type":"TEXT","stopReason":"PARTIAL_TURN"}`)
		stream.push("contentStart", `{"contentId":"c3","type":"AUDIO","role":"ASSISTANT"}`)
		stream.push("audioOutput", `{"contentId":"c3","content":"AAAB"}`)
		stream.push("audioOutput", `{"contentId":"c3","content":"AAAC"}`)
		stream.push("contentEnd", `{"contentId":"c3","type":"AUDIO","stopReason":"PARTIAL_TURN"}`)
		stream.push("contentStart", `{"contentId":"c4","type":"TEXT","role":"ASSISTANT","additionalModelFields":"{\"generationStage\":\"FINAL\"}"}`)
		stream.push("textOutput", `{"contentId":"c4","role":"ASSISTANT","content":"It is noon."}`)
		stream.push("usageEvent", `{"details":{"delta":{"input":{"speechTokens":100,"textTokens":30},"output":{"speechTokens":60,"textTokens":8}},"total":{"input":{"speechTokens":100,"textTokens":30},"output":{"speechTokens":60,"textTokens":8}}}}`)
		stream.push("contentEnd", `{"contentId":"c4","type":"TEXT","stopReason":"END_TURN"}`)

		var types []string
		var transcription, audioDelta, transcriptDelta, done map[string]any
		for done == nil {
			event := readSonicEvent(t, client)
			types = append(types, event["type"].(string))
			switch event["type"] {
			case "conversation.item.input_audio_transcription.completed":
				transcription = event
			case "response.audio.delta":
				audioDelta = event
			case "response.audio_transcript.delta":
				transcriptDelta = event
			case "response.done":
				done = event
			}
		}
		assert.Equal(t, []string{
			"conversation.item.input_audio_transcription.completed", "response.created", "response.output_item.added",
			"response.audio.delta", "response.audio.delta", "response.audio_transcript.delta",
			"response.audio.done", "response.audio_transcript.done", "response.output_item.done", "response.done",
		}, types)
		assert.Equal(t, "what time is it", transcription["transcript"])
		assert.Equal(t, "AAAC", audioDelta["delta"])
		assert.Equal(t, "It is noon.", transcriptDelta["delta"])
		response := done["response"].(map[string]any)
		assert.Equal(t, "completed", response["status"])
		assert.Equal(t, audioDelta["response_id"], response["id"])
		assert.Equal(t, map[string]any{
			"total_tokens": float64(198), "input_tokens": float64(130), "output_tokens": float64(68),
			"input_token_details":  map[string]any{"cached_tokens": float64(0), "text_tokens": float64(30), "audio_tokens": float64(100), "image_tokens": float64(0)},
			"output_token_details": map[string]any{"text_tokens": float64(8), "audio_tokens": float64(60), "image_tokens": float64(0), "reasoning_tokens": float64(0)},
		}, response["usage"])

		// A barge-in: the user speaks over the assistant's speech.
		stream.push("contentStart", `{"contentId":"c5","type":"AUDIO","role":"ASSISTANT"}`)
		stream.push("audioOutput", `{"contentId":"c5","content":"AAAD"}`)
		stream.push("contentStart", `{"contentId":"c6","type":"TEXT","role":"ASSISTANT","additionalModelFields":"{\"generationStage\":\"FINAL\"}"}`)
		stream.push("textOutput", `{"contentId":"c6","role":"ASSISTANT","content":"{ \"interrupted\" : true }"}`)
		stream.push("contentEnd", `{"contentId":"c5","type":"AUDIO","stopReason":"INTERRUPTED"}`)
		types = nil
		done = nil
		for done == nil {
			event := readSonicEvent(t, client)
			types = append(types, event["type"].(string))
			if event["type"] == "response.done" {
				done = event
			}
		}
		assert.Equal(t, []string{
			"response.created", "response.output_item.added", "response.audio.delta", "input_audio_buffer.speech_started",
			"response.audio.done", "response.audio_transcript.done", "response.output_item.done", "response.done",
		}, types)
		assert.Equal(t, "cancelled", done["response"].(map[string]any)["status"])
		assert.Equal(t, map[string]any{"type": "cancelled", "reason": "turn_detected"}, done["response"].(map[string]any)["status_details"])

		// The client closes: the open audio content, the prompt and the session are closed, in order.
		require.NoError(t, client.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
		result := awaitSonicResult(t, results)
		require.Nil(t, result.apiErr)
		names = stream.inputNames()
		require.Len(t, names, 15)
		assert.Equal(t, []string{"contentEnd", "promptEnd", "sessionEnd"}, names[12:])
		assert.JSONEq(t, `{"promptName":"`+promptName+`","contentName":"`+audioName+`"}`, stream.input(12).Get("contentEnd").Raw)
		assert.JSONEq(t, `{"promptName":"`+promptName+`"}`, stream.input(13).Get("promptEnd").Raw)
		assert.JSONEq(t, `{}`, stream.input(14).Get("sessionEnd").Raw)
		// The usage the relay settles with PostWssConsumeQuota is the session's last total, reported after the
		// close: speech as audio tokens, text as text tokens.
		assert.Equal(t, dto.RealtimeUsage{
			TotalTokens:        301,
			InputTokens:        194,
			OutputTokens:       107,
			InputTokenDetails:  dto.InputTokenDetails{AudioTokens: 160, TextTokens: 34},
			OutputTokenDetails: dto.OutputTokenDetails{AudioTokens: 95, TextTokens: 12},
		}, *result.usage)

	})

	awaitGoroutines(t, baseline)
}

func TestNovaSonicSessionLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseline := runtime.NumGoroutine()

	t.Run("the session limit closes both sides", func(t *testing.T) {
		saved := novaSonicSessionLimit
		novaSonicSessionLimit = 300 * time.Millisecond
		t.Cleanup(func() { novaSonicSessionLimit = saved })
		savedGrace := novaSonicUsageGrace
		novaSonicUsageGrace = 300 * time.Millisecond
		t.Cleanup(func() { novaSonicUsageGrace = savedGrace })

		// Bedrock does not end its output by itself: the session closes it when the grace period ends.
		stream := newStubSonicStream()
		useStubSonicStream(t, stream)
		gatewayURL, results := newSonicGateway(t, "nova-2-sonic-v1:0")
		client := dialSonicGateway(t, gatewayURL)
		sendSonicEvent(t, client, `{"type":"input_audio_buffer.append","audio":"`+base64.StdEncoding.EncodeToString(sonicTone(24000, 2400))+`"}`)

		expired := readSonicEvent(t, client)
		assert.Equal(t, "session_expired", expired["error"].(map[string]any)["code"])
		result := awaitSonicResult(t, results)
		require.Nil(t, result.apiErr)
		assert.Equal(t, []string{"sessionStart", "promptStart", "contentStart", "audioInput", "contentEnd", "promptEnd", "sessionEnd"}, stream.inputNames())
		assert.True(t, stream.isClosed())

	})

	awaitGoroutines(t, baseline)
}

func TestPCM16ResamplerKeepsLengthAcrossChunks(t *testing.T) {
	tone := sonicTone(24000, 24000)
	tests := []struct {
		name   string
		chunks []int
	}{
		{"one chunk", []int{48000}},
		{"chunks splitting samples", []int{15999, 16001, 16000}},
		{"100 ms frames", []int{4800, 4800, 4800, 4800, 4800, 4800, 4800, 4800, 4800, 4800}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resampler := newPCM16Resampler(24000, 16000)
			var out []byte
			offset := 0
			for _, size := range tt.chunks {
				out = append(out, resampler.resample(tone[offset:offset+size])...)
				offset += size
			}
			// One second at 24 kHz is one second at 16 kHz.
			require.Len(t, out, 2*16000)
			for i := 0; i < len(out); i += 2 {
				want := 10000 * math.Sin(2*math.Pi*440*float64(i/2)/16000)
				require.InDelta(t, want, float64(int16(binary.LittleEndian.Uint16(out[i:]))), 200, "sample %d", i/2)
			}
		})
	}
}
