package aws

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/wsmanager"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockruntimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

// Bedrock serves the Nova Sonic speech-to-speech models only over InvokeModelWithBidirectionalStream.
// The realtime route speaks OpenAI's Realtime API to the client, and a Nova Sonic session translates
// between the two: the client's session settings open the prompt, its pcm16 audio at 24 kHz goes
// upstream resampled to 16 kHz, and the model's speech, transcripts and usage come back as OpenAI's events.

// novaSonicSessionLimit is the longest a session lasts. Bedrock keeps the bidirectional stream open for
// 8 minutes; the session ends 15 seconds before that, so that its closing events are sent and the usage
// that follows them arrives before Bedrock cuts the stream.
var novaSonicSessionLimit = 8*time.Minute - 15*time.Second

// novaSonicUsageGrace is how long the upstream stays open, once the closing events are sent, for the usage
// it has not reported yet. A variable so that tests can shorten it.
var novaSonicUsageGrace = 10 * time.Second

// novaSonicOpenLimit bounds the opening of the stream. The SDK returns the stream as its request is about
// to be sent, within milliseconds; a call that fails before that (it cannot build, sign or resolve the
// request) is reported to no one and never returns. A variable so that tests can shorten it.
var novaSonicOpenLimit = 10 * time.Second

// OpenNovaSonicStream opens the bidirectional stream of a session. A variable so that tests, here and of
// the realtime route, can stub it.
var OpenNovaSonicStream = func(ctx context.Context, client *bedrockruntime.Client, modelID string) (*bedrockruntime.InvokeModelWithBidirectionalStreamEventStream, error) {
	output, err := client.InvokeModelWithBidirectionalStream(ctx, &bedrockruntime.InvokeModelWithBidirectionalStreamInput{ModelId: aws.String(modelID)})
	if err != nil {
		return nil, err
	}
	return output.GetStream(), nil
}

const (
	// realtimePCM16SampleRate is the rate of OpenAI Realtime's pcm16 audio, in and out.
	realtimePCM16SampleRate = 24000
	// novaSonicInputSampleRate is the rate of the LPCM audio sent upstream.
	novaSonicInputSampleRate = 16000
	novaSonicDefaultVoice    = "matthew"
)

// novaSonicSettings are the client's session settings that open the prompt.
type novaSonicSettings struct {
	instructions           string
	voice                  string
	temperature            *float64
	maxTokens              *int32
	endpointingSensitivity string
	turnDetection          json.RawMessage
}

type novaSonicSessionUpdate struct {
	Instructions            *string           `json:"instructions"`
	Voice                   *string           `json:"voice"`
	Temperature             *float64          `json:"temperature"`
	MaxResponseOutputTokens json.RawMessage   `json:"max_response_output_tokens"`
	TurnDetection           json.RawMessage   `json:"turn_detection"`
	Tools                   []json.RawMessage `json:"tools"`
}

type novaSonicClientEvent struct {
	EventId string          `json:"event_id"`
	Type    string          `json:"type"`
	Audio   string          `json:"audio"`
	Session json.RawMessage `json:"session"`
	Item    *struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"item"`
}

type novaSonicTokens struct {
	SpeechTokens int `json:"speechTokens"`
	TextTokens   int `json:"textTokens"`
}

type novaSonicTokenCounts struct {
	Input  novaSonicTokens `json:"input"`
	Output novaSonicTokens `json:"output"`
}

type novaSonicOutput struct {
	Event struct {
		ContentStart *struct {
			ContentId             string `json:"contentId"`
			Type                  string `json:"type"`
			Role                  string `json:"role"`
			AdditionalModelFields string `json:"additionalModelFields"`
		} `json:"contentStart"`
		TextOutput *struct {
			ContentId string `json:"contentId"`
			Role      string `json:"role"`
			Content   string `json:"content"`
		} `json:"textOutput"`
		AudioOutput *struct {
			ContentId string `json:"contentId"`
			Content   string `json:"content"`
		} `json:"audioOutput"`
		ContentEnd *struct {
			ContentId  string `json:"contentId"`
			StopReason string `json:"stopReason"`
		} `json:"contentEnd"`
		UsageEvent *struct {
			Details struct {
				Delta *novaSonicTokenCounts `json:"delta"`
				Total *novaSonicTokenCounts `json:"total"`
			} `json:"details"`
		} `json:"usageEvent"`
	} `json:"event"`
}

// novaSonicContent is an upstream content the session has seen start.
type novaSonicContent struct {
	role        string
	speculative bool
}

// novaSonicServerEvent is an OpenAI Realtime server event sent to the client.
type novaSonicServerEvent struct {
	EventId      string          `json:"event_id"`
	Type         string          `json:"type"`
	ResponseId   string          `json:"response_id,omitempty"`
	ItemId       string          `json:"item_id,omitempty"`
	OutputIndex  *int            `json:"output_index,omitempty"`
	ContentIndex *int            `json:"content_index,omitempty"`
	AudioStartMs *int64          `json:"audio_start_ms,omitempty"`
	Delta        string          `json:"delta,omitempty"`
	Transcript   *string         `json:"transcript,omitempty"`
	Item         *novaSonicItem  `json:"item,omitempty"`
	Response     *novaSonicReply `json:"response,omitempty"`
	Session      any             `json:"session,omitempty"`
	Error        *novaSonicError `json:"error,omitempty"`
}

type novaSonicItem struct {
	Id      string                 `json:"id"`
	Object  string                 `json:"object"`
	Type    string                 `json:"type"`
	Status  string                 `json:"status"`
	Role    string                 `json:"role"`
	Content []novaSonicItemContent `json:"content"`
}

type novaSonicItemContent struct {
	Type       string `json:"type"`
	Transcript string `json:"transcript"`
}

// novaSonicReply is the response object of OpenAI's response.created and response.done events.
type novaSonicReply struct {
	Id            string             `json:"id"`
	Object        string             `json:"object"`
	Status        string             `json:"status"`
	StatusDetails any                `json:"status_details"`
	Output        []novaSonicItem    `json:"output"`
	Usage         *dto.RealtimeUsage `json:"usage,omitempty"`
}

// novaSonicError is the error of an OpenAI Realtime error event; EventId names the client event refused.
type novaSonicError struct {
	Type    string  `json:"type"`
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Param   *string `json:"param"`
	EventId string  `json:"event_id,omitempty"`
}

// novaSonicResponse is the assistant turn being sent to the client.
type novaSonicResponse struct {
	id         string
	itemID     string
	audio      bool
	transcript strings.Builder
}

// saturatingTokenAdd adds an upstream token count to a total without letting either wrap.
func saturatingTokenAdd(total, count int) int {
	return min(total+min(max(count, 0), math.MaxInt32), math.MaxInt32)
}

// pcm16Resampler converts mono 16-bit little-endian PCM between two sample rates by linear interpolation.
// It keeps its position across chunks, so audio split anywhere, even within a sample, stays continuous.
type pcm16Resampler struct {
	inRate, outRate int64
	// pending holds the input samples from number base on that the next output still needs.
	pending []int16
	base    int64
	// next is the number of the next output sample.
	next int64
	// odd is the first byte of a sample split between two chunks.
	odd []byte
}

func newPCM16Resampler(inRate, outRate int) *pcm16Resampler {
	return &pcm16Resampler{inRate: int64(inRate), outRate: int64(outRate)}
}

// resample returns the output samples that the input so far determines; the last one or two wait for the
// next chunk.
func (r *pcm16Resampler) resample(chunk []byte) []byte {
	data := append(r.odd, chunk...)
	r.odd = nil
	if len(data)%2 == 1 {
		r.odd = []byte{data[len(data)-1]}
		data = data[:len(data)-1]
	}
	for i := 0; i < len(data); i += 2 {
		r.pending = append(r.pending, int16(binary.LittleEndian.Uint16(data[i:])))
	}
	var out []byte
	for {
		position := r.next * r.inRate
		index, fraction := position/r.outRate-r.base, position%r.outRate
		if index+1 >= int64(len(r.pending)) {
			break
		}
		a, b := int64(r.pending[index]), int64(r.pending[index+1])
		out = binary.LittleEndian.AppendUint16(out, uint16(int16(a+(b-a)*fraction/r.outRate)))
		r.next++
	}
	if consumed := r.next*r.inRate/r.outRate - r.base; consumed > 0 {
		r.pending = append(r.pending[:0], r.pending[consumed:]...)
		r.base += consumed
	}
	return out
}

type novaSonicSession struct {
	client *websocket.Conn
	stream *bedrockruntime.InvokeModelWithBidirectionalStreamEventStream
	model  novaSonicModel
	// modelName is the model the client asked for, the one it is shown; the stream is opened on the
	// Bedrock id the channel maps it to.
	modelName string
	// ctx is the stream's; clientCtx bounds the client reader's sends and is cancelled when the session ends.
	ctx          context.Context
	cancel       context.CancelFunc
	clientCtx    context.Context
	clientCancel context.CancelFunc
	// clientDone and upstreamDone are closed when the reader of that side has ended.
	clientDone   chan struct{}
	upstreamDone chan struct{}
	// ending is closed when the session starts to end; an upstream error after it is not reported.
	ending       chan struct{}
	endingOnce   sync.Once
	clientOnce   sync.Once
	upstreamOnce sync.Once

	// The fields below belong to the client reader, and to the session's end once that reader has ended.
	settings         novaSonicSettings
	started          bool
	promptName       string
	audioContentName string
	resampler        *pcm16Resampler

	// mu guards writes to the client and the state below.
	mu       sync.Mutex
	contents map[string]novaSonicContent
	response *novaSonicResponse
	usage    dto.RealtimeUsage
	// reported is the usage already sent to the client in response.done events.
	reported dto.RealtimeUsage
}

func (s *novaSonicSession) closeClient(code int, reason string) {
	s.clientOnce.Do(func() {
		closeMessage := websocket.FormatCloseMessage(code, reason)
		_ = s.client.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second))
		_ = s.client.Close()
	})
}

func (s *novaSonicSession) closeUpstream() {
	s.upstreamOnce.Do(func() {
		_ = s.stream.Close()
		s.cancel()
	})
}

func (s *novaSonicSession) startEnding() {
	s.endingOnce.Do(func() {
		close(s.ending)
		s.clientCancel()
	})
}

// sendUpstream sends one Nova Sonic input event. It has one caller at a time: the client reader, then
// the session's end.
func (s *novaSonicSession) sendUpstream(ctx context.Context, name string, body any) error {
	data, err := common.Marshal(map[string]any{"event": map[string]any{name: body}})
	if err != nil {
		return err
	}
	return s.stream.Send(ctx, &bedrockruntimeTypes.InvokeModelWithBidirectionalStreamInputMemberChunk{
		Value: bedrockruntimeTypes.BidirectionalInputPayloadPart{Bytes: data},
	})
}

// sendTextContent sends a text as one whole content of the prompt.
func (s *novaSonicSession) sendTextContent(ctx context.Context, role, text string, interactive bool) error {
	contentName := "text-" + common.GetUUID()
	if err := s.sendUpstream(ctx, "contentStart", map[string]any{
		"promptName":             s.promptName,
		"contentName":            contentName,
		"type":                   "TEXT",
		"interactive":            interactive,
		"role":                   role,
		"textInputConfiguration": map[string]string{"mediaType": "text/plain"},
	}); err != nil {
		return err
	}
	if err := s.sendUpstream(ctx, "textInput", map[string]any{"promptName": s.promptName, "contentName": contentName, "content": text}); err != nil {
		return err
	}
	return s.sendUpstream(ctx, "contentEnd", map[string]any{"promptName": s.promptName, "contentName": contentName})
}

// startPrompt sends the events that open the session, from the client's settings: the session, the
// prompt, the instructions as its system content, and the user audio content kept open for the session.
func (s *novaSonicSession) startPrompt(ctx context.Context) error {
	inference := map[string]any{}
	if s.settings.maxTokens != nil {
		inference["maxTokens"] = *s.settings.maxTokens
	}
	if s.settings.temperature != nil {
		inference["temperature"] = *s.settings.temperature
	}
	sessionStart := map[string]any{"inferenceConfiguration": inference}
	if s.settings.endpointingSensitivity != "" {
		sessionStart["turnDetectionConfiguration"] = map[string]string{"endpointingSensitivity": s.settings.endpointingSensitivity}
	}
	if err := s.sendUpstream(ctx, "sessionStart", sessionStart); err != nil {
		return err
	}
	s.promptName, s.audioContentName = "prompt-"+common.GetUUID(), "audio-"+common.GetUUID()
	voice := s.settings.voice
	if voice == "" {
		voice = novaSonicDefaultVoice
	}
	if err := s.sendUpstream(ctx, "promptStart", map[string]any{
		"promptName":              s.promptName,
		"textOutputConfiguration": map[string]string{"mediaType": "text/plain"},
		"audioOutputConfiguration": map[string]any{
			"mediaType":       "audio/lpcm",
			"sampleRateHertz": realtimePCM16SampleRate,
			"sampleSizeBits":  16,
			"channelCount":    1,
			"voiceId":         voice,
			"encoding":        "base64",
			"audioType":       "SPEECH",
		},
	}); err != nil {
		return err
	}
	if s.settings.instructions != "" {
		if err := s.sendTextContent(ctx, "SYSTEM", s.settings.instructions, false); err != nil {
			return err
		}
	}
	if err := s.sendUpstream(ctx, "contentStart", map[string]any{
		"promptName":  s.promptName,
		"contentName": s.audioContentName,
		"type":        "AUDIO",
		"interactive": true,
		"role":        "USER",
		"audioInputConfiguration": map[string]any{
			"mediaType":       "audio/lpcm",
			"sampleRateHertz": novaSonicInputSampleRate,
			"sampleSizeBits":  16,
			"channelCount":    1,
			"audioType":       "SPEECH",
			"encoding":        "base64",
		},
	}); err != nil {
		return err
	}
	s.started = true
	return nil
}

// endPrompt sends the events that close an opened session: the audio content, the prompt, the session.
func (s *novaSonicSession) endPrompt(ctx context.Context) error {
	if err := s.sendUpstream(ctx, "contentEnd", map[string]any{"promptName": s.promptName, "contentName": s.audioContentName}); err != nil {
		return err
	}
	if err := s.sendUpstream(ctx, "promptEnd", map[string]any{"promptName": s.promptName}); err != nil {
		return err
	}
	return s.sendUpstream(ctx, "sessionEnd", map[string]any{})
}

// sendLocked writes an event to the client; s.mu is held.
func (s *novaSonicSession) sendLocked(event novaSonicServerEvent) {
	event.EventId = "event_" + common.GetUUID()
	data, err := common.Marshal(event)
	if err != nil {
		return
	}
	_ = s.client.WriteMessage(websocket.TextMessage, data)
}

func (s *novaSonicSession) sendError(code, message, clientEventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	errorType := "invalid_request_error"
	if strings.HasPrefix(code, "upstream") || code == "session_expired" {
		errorType = "server_error"
	}
	s.sendLocked(novaSonicServerEvent{Type: "error", Error: &novaSonicError{Type: errorType, Code: code, Message: message, EventId: clientEventID}})
}

// sessionObject is the OpenAI Realtime session the client's settings make.
func (s *novaSonicSession) sessionObject() map[string]any {
	voice := s.settings.voice
	if voice == "" {
		voice = novaSonicDefaultVoice
	}
	session := map[string]any{
		"object":              "realtime.session",
		"model":               s.modelName,
		"modalities":          []string{"text", "audio"},
		"instructions":        s.settings.instructions,
		"voice":               voice,
		"input_audio_format":  "pcm16",
		"output_audio_format": "pcm16",
		"tools":               []any{},
	}
	if s.settings.temperature != nil {
		session["temperature"] = *s.settings.temperature
	}
	if s.settings.maxTokens != nil {
		session["max_response_output_tokens"] = *s.settings.maxTokens
	}
	if s.settings.turnDetection != nil {
		session["turn_detection"] = s.settings.turnDetection
	}
	return session
}

// applySessionUpdate merges a client's session.update into the settings, or says why it is refused.
func (s *novaSonicSession) applySessionUpdate(raw json.RawMessage) (code, message string) {
	if s.started {
		return "session_already_started", "Nova Sonic takes the session settings when its input starts; they cannot change after"
	}
	inFormat, outFormat := gjson.GetBytes(raw, "input_audio_format").Str, gjson.GetBytes(raw, "output_audio_format").Str
	gaIn, gaOut := gjson.GetBytes(raw, "audio.input.format"), gjson.GetBytes(raw, "audio.output.format")
	if (inFormat != "" && inFormat != "pcm16") || (outFormat != "" && outFormat != "pcm16") ||
		(gaIn.Get("type").Str != "" && gaIn.Get("type").Str != "audio/pcm") ||
		(gaOut.Get("type").Str != "" && gaOut.Get("type").Str != "audio/pcm") ||
		(gaIn.Get("rate").Int() != 0 && gaIn.Get("rate").Int() != realtimePCM16SampleRate) ||
		(gaOut.Get("rate").Int() != 0 && gaOut.Get("rate").Int() != realtimePCM16SampleRate) {
		return "unsupported_audio_format", "Nova Sonic sessions take and return pcm16 audio at 24 kHz only; g711_ulaw and g711_alaw are not supported"
	}
	var update novaSonicSessionUpdate
	if err := common.Unmarshal(raw, &update); err != nil {
		return "invalid_event", "the session is not valid: " + err.Error()
	}
	if len(update.Tools) > 0 {
		return "tools_not_supported", "Nova Sonic sessions on this gateway do not take tools"
	}
	settings := s.settings
	if update.Instructions != nil {
		settings.instructions = *update.Instructions
	}
	if update.Voice != nil {
		settings.voice = *update.Voice
	}
	if update.Temperature != nil {
		settings.temperature = update.Temperature
	}
	switch maxTokens := update.MaxResponseOutputTokens; {
	case len(maxTokens) == 0 || string(maxTokens) == "null" || string(maxTokens) == `"inf"`:
		settings.maxTokens = nil
	default:
		var value int32
		if err := common.Unmarshal(maxTokens, &value); err != nil || value < 1 {
			return "invalid_value", `max_response_output_tokens must be a positive integer or "inf"`
		}
		settings.maxTokens = &value
	}
	if len(update.TurnDetection) > 0 {
		// Nova Sonic detects turns itself; semantic VAD's eagerness is its endpointing sensitivity.
		settings.turnDetection = update.TurnDetection
		settings.endpointingSensitivity = ""
		if gjson.GetBytes(update.TurnDetection, "type").Str == "semantic_vad" {
			switch eagerness := gjson.GetBytes(update.TurnDetection, "eagerness").Str; eagerness {
			case "low", "medium", "high":
				settings.endpointingSensitivity = strings.ToUpper(eagerness)
			}
		}
	}
	s.settings = settings
	return "", ""
}

// relayClient forwards the client's events upstream until the client ends or the upstream refuses one.
func (s *novaSonicSession) relayClient() {
	defer close(s.clientDone)
	for {
		_, message, err := s.client.ReadMessage()
		if err != nil {
			return
		}
		var event novaSonicClientEvent
		if err := common.Unmarshal(message, &event); err != nil {
			s.sendError("invalid_event", "the event is not valid JSON", "")
			continue
		}
		switch event.Type {
		case "session.update":
			if code, message := s.applySessionUpdate(event.Session); code != "" {
				s.sendError(code, message, event.EventId)
				continue
			}
			s.mu.Lock()
			s.sendLocked(novaSonicServerEvent{Type: "session.updated", Session: s.sessionObject()})
			s.mu.Unlock()
		case "input_audio_buffer.append":
			audio, err := base64.StdEncoding.DecodeString(event.Audio)
			if err != nil || len(audio) == 0 {
				s.sendError("invalid_event", "input_audio_buffer.append needs base64 pcm16 audio", event.EventId)
				continue
			}
			if !s.started {
				if err := s.startPrompt(s.clientCtx); err != nil {
					s.reportSendError(err)
					return
				}
			}
			resampled := s.resampler.resample(audio)
			if len(resampled) == 0 {
				continue
			}
			if err := s.sendUpstream(s.clientCtx, "audioInput", map[string]any{
				"promptName":  s.promptName,
				"contentName": s.audioContentName,
				"content":     base64.StdEncoding.EncodeToString(resampled),
			}); err != nil {
				s.reportSendError(err)
				return
			}
		case "conversation.item.create":
			if event.Item == nil || event.Item.Type != "message" || event.Item.Role != "user" ||
				len(event.Item.Content) != 1 || event.Item.Content[0].Type != "input_text" {
				s.sendError("unsupported_item", "Nova Sonic sessions take only a user message with one input_text content", event.EventId)
				continue
			}
			if !s.model.textInput {
				s.sendError("text_input_not_supported", s.modelName+" takes audio input only", event.EventId)
				continue
			}
			if !s.started {
				if err := s.startPrompt(s.clientCtx); err != nil {
					s.reportSendError(err)
					return
				}
			}
			if err := s.sendTextContent(s.clientCtx, "USER", event.Item.Content[0].Text, true); err != nil {
				s.reportSendError(err)
				return
			}
		case "input_audio_buffer.commit", "response.create":
			// Nova Sonic detects the end of the user's turn and answers it by itself.
		default:
			s.sendError("unsupported_event", event.Type+" is not supported by a Nova Sonic session", event.EventId)
		}
	}
}

// reportSendError tells the client the upstream refused input, unless the session was already ending.
func (s *novaSonicSession) reportSendError(err error) {
	select {
	case <-s.ending:
	default:
		s.sendError("upstream_error", fmt.Sprintf("the Nova Sonic stream refused input: %v", err), "")
	}
}

// currentResponseLocked is the assistant turn the next output belongs to, opened on its first output;
// s.mu is held.
func (s *novaSonicSession) currentResponseLocked() *novaSonicResponse {
	if s.response != nil {
		return s.response
	}
	s.response = &novaSonicResponse{id: "resp_" + common.GetUUID(), itemID: "item_" + common.GetUUID()}
	s.sendLocked(novaSonicServerEvent{Type: "response.created", Response: &novaSonicReply{Id: s.response.id, Object: "realtime.response", Status: "in_progress", Output: []novaSonicItem{}}})
	s.sendLocked(novaSonicServerEvent{Type: "response.output_item.added", ResponseId: s.response.id, OutputIndex: common.GetPointer(0),
		Item: &novaSonicItem{Id: s.response.itemID, Object: "realtime.item", Type: "message", Status: "in_progress", Role: "assistant", Content: []novaSonicItemContent{}}})
	return s.response
}

// finishResponseLocked ends the open assistant turn with the usage reported since the last one ended;
// s.mu is held.
func (s *novaSonicSession) finishResponseLocked(status string) {
	response := s.response
	if response == nil {
		return
	}
	s.response = nil
	transcript := response.transcript.String()
	if response.audio {
		s.sendLocked(novaSonicServerEvent{Type: "response.audio.done", ResponseId: response.id, ItemId: response.itemID, OutputIndex: common.GetPointer(0), ContentIndex: common.GetPointer(0)})
	}
	s.sendLocked(novaSonicServerEvent{Type: "response.audio_transcript.done", ResponseId: response.id, ItemId: response.itemID, OutputIndex: common.GetPointer(0), ContentIndex: common.GetPointer(0), Transcript: &transcript})
	itemStatus := "completed"
	if status != "completed" {
		itemStatus = "incomplete"
	}
	item := novaSonicItem{Id: response.itemID, Object: "realtime.item", Type: "message", Status: itemStatus, Role: "assistant",
		Content: []novaSonicItemContent{{Type: "audio", Transcript: transcript}}}
	s.sendLocked(novaSonicServerEvent{Type: "response.output_item.done", ResponseId: response.id, OutputIndex: common.GetPointer(0), Item: &item})

	usage := dto.RealtimeUsage{
		InputTokens:  max(s.usage.InputTokens-s.reported.InputTokens, 0),
		OutputTokens: max(s.usage.OutputTokens-s.reported.OutputTokens, 0),
		TotalTokens:  max(s.usage.TotalTokens-s.reported.TotalTokens, 0),
	}
	usage.InputTokenDetails.TextTokens = max(s.usage.InputTokenDetails.TextTokens-s.reported.InputTokenDetails.TextTokens, 0)
	usage.InputTokenDetails.AudioTokens = max(s.usage.InputTokenDetails.AudioTokens-s.reported.InputTokenDetails.AudioTokens, 0)
	usage.OutputTokenDetails.TextTokens = max(s.usage.OutputTokenDetails.TextTokens-s.reported.OutputTokenDetails.TextTokens, 0)
	usage.OutputTokenDetails.AudioTokens = max(s.usage.OutputTokenDetails.AudioTokens-s.reported.OutputTokenDetails.AudioTokens, 0)
	s.reported = s.usage
	reply := &novaSonicReply{Id: response.id, Object: "realtime.response", Status: status, Output: []novaSonicItem{item}, Usage: &usage}
	if status == "cancelled" {
		reply.StatusDetails = map[string]string{"type": "cancelled", "reason": "turn_detected"}
	}
	s.sendLocked(novaSonicServerEvent{Type: "response.done", Response: reply})
}

// applyUsageLocked takes a usage event's counts into the session's usage: the session total when the
// event carries it, else the delta added to what came before; s.mu is held.
func (s *novaSonicSession) applyUsageLocked(delta, total *novaSonicTokenCounts) {
	var usage dto.RealtimeUsage
	counts := total
	if counts == nil {
		if delta == nil {
			return
		}
		usage, counts = s.usage, delta
	}
	usage.InputTokenDetails.AudioTokens = saturatingTokenAdd(usage.InputTokenDetails.AudioTokens, counts.Input.SpeechTokens)
	usage.InputTokenDetails.TextTokens = saturatingTokenAdd(usage.InputTokenDetails.TextTokens, counts.Input.TextTokens)
	usage.OutputTokenDetails.AudioTokens = saturatingTokenAdd(usage.OutputTokenDetails.AudioTokens, counts.Output.SpeechTokens)
	usage.OutputTokenDetails.TextTokens = saturatingTokenAdd(usage.OutputTokenDetails.TextTokens, counts.Output.TextTokens)
	usage.InputTokens = saturatingTokenAdd(usage.InputTokenDetails.AudioTokens, usage.InputTokenDetails.TextTokens)
	usage.OutputTokens = saturatingTokenAdd(usage.OutputTokenDetails.AudioTokens, usage.OutputTokenDetails.TextTokens)
	usage.TotalTokens = saturatingTokenAdd(usage.InputTokens, usage.OutputTokens)
	// A session total never goes down: a total that does is not taken.
	if usage.TotalTokens >= s.usage.TotalTokens {
		s.usage = usage
	}
}

// relayUpstream turns the upstream's events into the client's and adds up the usage they carry.
func (s *novaSonicSession) relayUpstream(info *relaycommon.RelayInfo) {
	defer close(s.upstreamDone)
	for event := range s.stream.Events() {
		chunk, ok := event.(*bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutputMemberChunk)
		if !ok {
			continue
		}
		info.SetFirstResponseTime()
		var output novaSonicOutput
		if err := common.Unmarshal(chunk.Value.Bytes, &output); err != nil {
			continue
		}
		upstream := output.Event

		s.mu.Lock()
		switch {
		case upstream.ContentStart != nil:
			start := upstream.ContentStart
			s.contents[start.ContentId] = novaSonicContent{
				role:        start.Role,
				speculative: gjson.Get(start.AdditionalModelFields, "generationStage").Str == "SPECULATIVE",
			}
		case upstream.AudioOutput != nil:
			response := s.currentResponseLocked()
			response.audio = true
			s.sendLocked(novaSonicServerEvent{Type: "response.audio.delta", ResponseId: response.id, ItemId: response.itemID, OutputIndex: common.GetPointer(0), ContentIndex: common.GetPointer(0), Delta: upstream.AudioOutput.Content})
		case upstream.TextOutput != nil:
			text := upstream.TextOutput
			content := s.contents[text.ContentId]
			role := text.Role
			if role == "" {
				role = content.role
			}
			// Speculative text is what the model plans to say; the final text is what it said.
			if content.speculative {
				break
			}
			switch role {
			case "USER":
				transcript := text.Content
				s.sendLocked(novaSonicServerEvent{Type: "conversation.item.input_audio_transcription.completed", ItemId: "item_" + common.GetUUID(), ContentIndex: common.GetPointer(0), Transcript: &transcript})
			case "ASSISTANT":
				// A barge-in is reported in the text as {"interrupted": true}; its contentEnd ends the turn.
				if gjson.Get(text.Content, "interrupted").Bool() {
					break
				}
				response := s.currentResponseLocked()
				response.transcript.WriteString(text.Content)
				s.sendLocked(novaSonicServerEvent{Type: "response.audio_transcript.delta", ResponseId: response.id, ItemId: response.itemID, OutputIndex: common.GetPointer(0), ContentIndex: common.GetPointer(0), Delta: text.Content})
			}
		case upstream.ContentEnd != nil:
			end := upstream.ContentEnd
			content := s.contents[end.ContentId]
			delete(s.contents, end.ContentId)
			if content.role != "ASSISTANT" || s.response == nil {
				break
			}
			switch end.StopReason {
			case "END_TURN":
				s.finishResponseLocked("completed")
			case "INTERRUPTED":
				s.sendLocked(novaSonicServerEvent{Type: "input_audio_buffer.speech_started", ItemId: "item_" + common.GetUUID(), AudioStartMs: common.GetPointer[int64](0)})
				s.finishResponseLocked("cancelled")
			}
		case upstream.UsageEvent != nil:
			s.applyUsageLocked(upstream.UsageEvent.Details.Delta, upstream.UsageEvent.Details.Total)
		}
		s.mu.Unlock()
	}
	select {
	case <-s.ending:
		return
	default:
	}
	if err := s.stream.Err(); err != nil {
		s.sendError("upstream_error", fmt.Sprintf("the Nova Sonic stream ended: %v", err), "")
	}
}

// openNovaSonicSession opens the bidirectional stream of a realtime session on a Nova Sonic model.
func (a *Adaptor) openNovaSonicSession(c *gin.Context, info *relaycommon.RelayInfo) error {
	if info.ChannelOtherSettings.AwsApiType == dto.AwsApiTypeMantle {
		return types.NewErrorWithStatusCode(errors.New("the realtime route on AWS needs the Bedrock runtime API, not Mantle"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	modelID := getAwsModelID(info.UpstreamModelName)
	model, ok := novaSonicModels[modelID]
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("model %s does not serve the realtime route on AWS: only Nova Sonic models do", info.OriginModelName), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	client, err := newAwsClient(c, info)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelAwsClientError)
	}
	// The SDK signs every event of the stream with SigV4 and refuses a call without SigV4 credentials,
	// which a Bedrock API key's bearer token is not. The channel's other models are unaffected, so the
	// error neither disables it nor stops another channel from being tried.
	if client.Options().BearerAuthTokenProvider != nil {
		return types.NewErrorWithStatusCode(errors.New("Nova Sonic on the realtime route needs an AWS channel keyed with an access key (AK|SK|region): Bedrock's bidirectional stream signs every event with SigV4, which a Bedrock API key cannot"), types.ErrorCodeAwsInvokeError, http.StatusInternalServerError)
	}
	type openedStream struct {
		stream *bedrockruntime.InvokeModelWithBidirectionalStreamEventStream
		err    error
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	opened := make(chan openedStream, 1)
	go func() {
		stream, err := OpenNovaSonicStream(ctx, client, modelID)
		// A stream that opens after the session gave up on it is closed.
		if stream != nil && ctx.Err() != nil {
			_ = stream.Close()
		}
		opened <- openedStream{stream, err}
	}()
	limit := time.NewTimer(novaSonicOpenLimit)
	defer limit.Stop()
	var stream *bedrockruntime.InvokeModelWithBidirectionalStreamEventStream
	select {
	case result := <-opened:
		stream, err = result.stream, result.err
	case <-limit.C:
		// The SDK's call is left blocked; the session ends with an error instead of hanging with it.
		err = fmt.Errorf("the stream did not open within %s", novaSonicOpenLimit)
	}
	if err != nil {
		cancel()
		return newAwsInvokeError(c.Request.Context(), err, "InvokeModelWithBidirectionalStream")
	}
	a.AwsClient, a.AwsModelId = client, modelID
	a.sonicModel, a.sonicStream, a.sonicCancel = model, stream, cancel
	return nil
}

// novaSonicHandler runs a realtime session on the opened stream until either side ends or the session
// limit is reached, then closes the prompt and returns the session's usage, waiting at most
// novaSonicUsageGrace for the usage that follows. Both sides are closed and both relay goroutines have
// ended when it returns.
func (a *Adaptor) novaSonicHandler(c *gin.Context, info *relaycommon.RelayInfo) (*dto.RealtimeUsage, *types.NewAPIError) {
	if info.ClientWs == nil || a.sonicStream == nil {
		return nil, types.NewError(errors.New("invalid realtime session"), types.ErrorCodeBadResponse)
	}
	info.IsStream = true
	clientCtx, clientCancel := context.WithCancel(c.Request.Context())
	session := &novaSonicSession{
		client:       info.ClientWs,
		stream:       a.sonicStream,
		model:        a.sonicModel,
		modelName:    info.OriginModelName,
		ctx:          c.Request.Context(),
		cancel:       a.sonicCancel,
		clientCtx:    clientCtx,
		clientCancel: clientCancel,
		clientDone:   make(chan struct{}),
		upstreamDone: make(chan struct{}),
		ending:       make(chan struct{}),
		resampler:    newPCM16Resampler(realtimePCM16SampleRate, novaSonicInputSampleRate),
		contents:     map[string]novaSonicContent{},
	}
	unregister := wsmanager.Register(info.ChannelId, wsmanager.KindRealtime, func(reason string) {
		session.closeClient(websocket.ClosePolicyViolation, reason)
	})
	defer unregister()

	session.mu.Lock()
	session.sendLocked(novaSonicServerEvent{Type: "session.created", Session: session.sessionObject()})
	session.mu.Unlock()
	go session.relayClient()
	go session.relayUpstream(info)
	limit := time.NewTimer(novaSonicSessionLimit)
	defer limit.Stop()
	select {
	case <-session.clientDone:
	case <-session.upstreamDone:
	case <-limit.C:
		session.sendError("session_expired", fmt.Sprintf("the Nova Sonic session reached its %s limit", novaSonicSessionLimit), "")
	}
	session.startEnding()
	session.closeClient(websocket.CloseNormalClosure, "")
	// The client reader has ended, so the session is now the upstream's only writer.
	<-session.clientDone

	select {
	case <-session.upstreamDone:
	default:
		if session.started {
			ctx, cancel := context.WithTimeout(session.ctx, novaSonicUsageGrace)
			if err := session.endPrompt(ctx); err == nil {
				// The upstream ends its output once the session is closed, after the usage it still owes.
				select {
				case <-session.upstreamDone:
				case <-ctx.Done():
				}
			}
			cancel()
		}
	}
	session.closeUpstream()
	<-session.upstreamDone

	session.mu.Lock()
	usage := session.usage
	session.mu.Unlock()
	return &usage, nil
}
