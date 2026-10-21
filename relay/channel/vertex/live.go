package vertex

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

// The Live API serves Gemini Live transcription models over one WebSocket. The realtime route speaks
// OpenAI's Realtime transcription session to the client, and a live session translates between the two:
// the client's pcm16 audio at 24 kHz goes upstream as it is, and the upstream's voice activity and input
// transcriptions come back as OpenAI's events.

// liveSessionLimit is the longest a Live API session lasts; the session is closed when it is reached.
var liveSessionLimit = 10 * time.Minute

const liveAudioMIMEType = "audio/pcm;rate=24000"

var liveAudioStreamEnd = map[string]any{"realtimeInput": map[string]any{"audioStreamEnd": true}}

// isGeminiLiveTranscriptionModel reports whether a model is one of Gemini's Live transcription models,
// the only models the realtime route serves on Vertex AI.
func isGeminiLiveTranscriptionModel(upstreamModelName string) bool {
	return strings.HasPrefix(upstreamModelName, "gemini-") &&
		strings.Contains(upstreamModelName, "transcribe") && strings.Contains(upstreamModelName, "live")
}

// liveRequestURL is the Live API endpoint of a realtime session. The session needs a service account's
// access token and project, so an API key channel is refused, as is any model but a Live transcription one.
func (a *Adaptor) liveRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if !isGeminiLiveTranscriptionModel(info.UpstreamModelName) {
		return "", types.NewErrorWithStatusCode(fmt.Errorf("model %s does not serve the realtime route on Vertex AI: only Gemini Live transcription models do", info.UpstreamModelName), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if info.ChannelOtherSettings.VertexKeyType == dto.VertexKeyTypeAPIKey {
		return "", types.NewErrorWithStatusCode(errors.New("the realtime route on Vertex AI needs a service account key, not an API key"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	adc := &Credentials{}
	if err := common.Unmarshal([]byte(info.ApiKey), adc); err != nil {
		return "", fmt.Errorf("failed to decode credentials file: %w", err)
	}
	a.AccountCredentials = *adc
	return BuildLiveURL(info.ChannelBaseUrl, GetModelRegion(info.ApiVersion, info.OriginModelName)), nil
}

type liveSetupMessage struct {
	Setup struct {
		Model            string `json:"model"`
		GenerationConfig struct {
			ResponseModalities []string `json:"responseModalities"`
		} `json:"generationConfig"`
		InputAudioTranscription struct{} `json:"inputAudioTranscription"`
	} `json:"setup"`
}

type liveServerMessage struct {
	SetupComplete *struct {
		SessionId string `json:"sessionId"`
	} `json:"setupComplete"`
	VoiceActivity *struct {
		Type        string `json:"type"`
		AudioOffset string `json:"audioOffset"`
	} `json:"voiceActivity"`
	ServerContent *struct {
		InterimInputTranscription *struct {
			Text string `json:"text"`
		} `json:"interimInputTranscription"`
		InputTranscription *struct {
			Text string `json:"text"`
		} `json:"inputTranscription"`
	} `json:"serverContent"`
	UsageMetadata *dto.GeminiUsageMetadata `json:"usageMetadata"`
}

type liveClientEvent struct {
	EventId string          `json:"event_id"`
	Type    string          `json:"type"`
	Audio   string          `json:"audio"`
	Session json.RawMessage `json:"session"`
}

type liveTranscriptionUsage struct {
	Type              string `json:"type"`
	InputTokens       int    `json:"input_tokens"`
	InputTokenDetails struct {
		TextTokens  int `json:"text_tokens"`
		AudioTokens int `json:"audio_tokens"`
	} `json:"input_token_details"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// liveClientServerEvent is an OpenAI Realtime server event sent to the client.
type liveClientServerEvent struct {
	EventId      string                  `json:"event_id"`
	Type         string                  `json:"type"`
	ItemId       string                  `json:"item_id,omitempty"`
	ContentIndex *int                    `json:"content_index,omitempty"`
	AudioStartMs *int64                  `json:"audio_start_ms,omitempty"`
	AudioEndMs   *int64                  `json:"audio_end_ms,omitempty"`
	Delta        string                  `json:"delta,omitempty"`
	Transcript   *string                 `json:"transcript,omitempty"`
	Usage        *liveTranscriptionUsage `json:"usage,omitempty"`
	Session      json.RawMessage         `json:"session,omitempty"`
	Error        *liveError              `json:"error,omitempty"`
}

// liveError is the error of an OpenAI Realtime error event; EventId names the client event refused.
type liveError struct {
	Type    string  `json:"type"`
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Param   *string `json:"param"`
	EventId string  `json:"event_id,omitempty"`
}

// saturatingTokenAdd adds an upstream token count to a total without letting either wrap.
func saturatingTokenAdd(total, count int) int {
	return min(total+min(max(count, 0), math.MaxInt32), math.MaxInt32)
}

// stableInterimText is the part of a cumulative interim transcription that the next one repeated, cut
// back to a whole word: the text that can go to the client as deltas, which can only be appended.
func stableInterimText(previous, current string) string {
	n := 0
	for n < len(previous) && n < len(current) && previous[n] == current[n] {
		n++
	}
	if (n == len(current) || current[n] == ' ') && (n == len(previous) || previous[n] == ' ') {
		return current[:n]
	}
	if i := strings.LastIndexByte(current[:n], ' '); i >= 0 {
		return current[:i]
	}
	return ""
}

func liveOffsetMs(offset string) *int64 {
	duration, err := time.ParseDuration(offset)
	if err != nil {
		return nil
	}
	return common.GetPointer(duration.Milliseconds())
}

type liveSession struct {
	client   *websocket.Conn
	upstream *websocket.Conn
	done     chan struct{}
	once     sync.Once

	// mu guards writes to the client and the transcription state below.
	mu          sync.Mutex
	itemID      string
	itemDone    bool
	lastInterim string
	sent        string
	usage       dto.RealtimeUsage
}

func (s *liveSession) close() {
	s.once.Do(func() {
		close(s.done)
		deadline := time.Now().Add(time.Second)
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		_ = s.client.WriteControl(websocket.CloseMessage, closeMessage, deadline)
		_ = s.upstream.WriteControl(websocket.CloseMessage, closeMessage, deadline)
		_ = s.client.Close()
		_ = s.upstream.Close()
	})
}

func (s *liveSession) closed() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// writeUpstream is called from the client reader only, the upstream's single writer.
func (s *liveSession) writeUpstream(message any) error {
	data, err := common.Marshal(message)
	if err != nil {
		return err
	}
	return s.upstream.WriteMessage(websocket.TextMessage, data)
}

// sendLocked writes an event to the client; s.mu is held.
func (s *liveSession) sendLocked(event liveClientServerEvent) {
	event.EventId = "event_" + common.GetUUID()
	data, err := common.Marshal(event)
	if err != nil {
		return
	}
	_ = s.client.WriteMessage(websocket.TextMessage, data)
}

func (s *liveSession) sendError(code, message, clientEventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	errorType := "invalid_request_error"
	if strings.HasPrefix(code, "upstream") || code == "session_expired" {
		errorType = "server_error"
	}
	s.sendLocked(liveClientServerEvent{Type: "error", Error: &liveError{Type: errorType, Code: code, Message: message, EventId: clientEventID}})
}

// currentItemLocked is the item the next transcription belongs to; s.mu is held.
func (s *liveSession) currentItemLocked() string {
	if s.itemID == "" || s.itemDone {
		s.itemID, s.itemDone, s.lastInterim, s.sent = "item_"+common.GetUUID(), false, "", ""
	}
	return s.itemID
}

// relayClient forwards the client's audio upstream until the client ends, and refuses the events a
// transcription session does not take.
func (s *liveSession) relayClient() {
	defer s.close()
	for {
		_, message, err := s.client.ReadMessage()
		if err != nil {
			// The client ended: the upstream gets the end of the audio before the session closes.
			_ = s.writeUpstream(liveAudioStreamEnd)
			return
		}
		var event liveClientEvent
		if err := common.Unmarshal(message, &event); err != nil {
			s.sendError("invalid_event", "the event is not valid JSON", "")
			continue
		}
		switch event.Type {
		case "input_audio_buffer.append":
			if event.Audio == "" {
				s.sendError("invalid_event", "input_audio_buffer.append needs audio", event.EventId)
				continue
			}
			audio := map[string]any{"realtimeInput": map[string]any{"audio": map[string]string{"mimeType": liveAudioMIMEType, "data": event.Audio}}}
			if err := s.writeUpstream(audio); err != nil {
				return
			}
		case "input_audio_buffer.commit":
			if err := s.writeUpstream(liveAudioStreamEnd); err != nil {
				return
			}
			s.mu.Lock()
			s.sendLocked(liveClientServerEvent{Type: "input_audio_buffer.committed", ItemId: s.currentItemLocked()})
			s.mu.Unlock()
		case "transcription_session.update", "session.update":
			// Only the audio format is checked: the upstream session was set up when the client connected.
			format := gjson.GetBytes(event.Session, "input_audio_format").Str
			gaFormat := gjson.GetBytes(event.Session, "audio.input.format")
			if (format != "" && format != "pcm16") ||
				(gaFormat.Get("type").Str != "" && gaFormat.Get("type").Str != "audio/pcm") ||
				(gaFormat.Get("rate").Int() != 0 && gaFormat.Get("rate").Int() != 24000) {
				s.sendError("unsupported_audio_format", "Gemini Live transcription takes pcm16 audio at 24 kHz", event.EventId)
				continue
			}
			s.mu.Lock()
			s.sendLocked(liveClientServerEvent{Type: strings.TrimSuffix(event.Type, "update") + "updated", Session: event.Session})
			s.mu.Unlock()
		default:
			s.sendError("unsupported_event", event.Type+" is not supported by a Gemini Live transcription session", event.EventId)
		}
	}
}

// relayUpstream turns the upstream's messages into the client's events and adds up the usage they carry.
func (s *liveSession) relayUpstream(info *relaycommon.RelayInfo) {
	defer s.close()
	for {
		_, message, err := s.upstream.ReadMessage()
		if err != nil {
			if s.closed() {
				return
			}
			var closeErr *websocket.CloseError
			if !errors.As(err, &closeErr) || closeErr.Code != websocket.CloseNormalClosure {
				s.sendError("upstream_error", fmt.Sprintf("the Live API session ended: %v", err), "")
			}
			return
		}
		info.SetFirstResponseTime()
		var upstream liveServerMessage
		if err := common.Unmarshal(message, &upstream); err != nil {
			continue
		}

		s.mu.Lock()
		if upstream.UsageMetadata != nil {
			metadata := upstream.UsageMetadata
			audioIn, audioOut := 0, 0
			for _, detail := range metadata.PromptTokensDetails {
				if detail.Modality == "AUDIO" {
					audioIn = saturatingTokenAdd(audioIn, detail.TokenCount)
				}
			}
			for _, detail := range metadata.CandidatesTokensDetails {
				if detail.Modality == "AUDIO" {
					audioOut = saturatingTokenAdd(audioOut, detail.TokenCount)
				}
			}
			prompt := saturatingTokenAdd(0, metadata.PromptTokenCount)
			output := saturatingTokenAdd(saturatingTokenAdd(0, metadata.CandidatesTokenCount), metadata.ThoughtsTokenCount)
			audioIn, audioOut = min(audioIn, prompt), min(audioOut, output)
			s.usage.InputTokens = saturatingTokenAdd(s.usage.InputTokens, prompt)
			s.usage.InputTokenDetails.AudioTokens = saturatingTokenAdd(s.usage.InputTokenDetails.AudioTokens, audioIn)
			s.usage.InputTokenDetails.TextTokens = saturatingTokenAdd(s.usage.InputTokenDetails.TextTokens, prompt-audioIn)
			s.usage.OutputTokens = saturatingTokenAdd(s.usage.OutputTokens, output)
			s.usage.OutputTokenDetails.AudioTokens = saturatingTokenAdd(s.usage.OutputTokenDetails.AudioTokens, audioOut)
			s.usage.OutputTokenDetails.TextTokens = saturatingTokenAdd(s.usage.OutputTokenDetails.TextTokens, output-audioOut)
			s.usage.TotalTokens = saturatingTokenAdd(s.usage.TotalTokens, saturatingTokenAdd(prompt, output))
		}
		switch {
		case upstream.SetupComplete != nil:
			session, _ := common.Marshal(map[string]string{"id": upstream.SetupComplete.SessionId, "object": "realtime.transcription_session", "input_audio_format": "pcm16"})
			s.sendLocked(liveClientServerEvent{Type: "transcription_session.created", Session: session})
		case upstream.VoiceActivity != nil && upstream.VoiceActivity.Type == "ACTIVITY_START":
			s.itemDone = true
			s.sendLocked(liveClientServerEvent{Type: "input_audio_buffer.speech_started", ItemId: s.currentItemLocked(), AudioStartMs: liveOffsetMs(upstream.VoiceActivity.AudioOffset)})
		case upstream.VoiceActivity != nil && upstream.VoiceActivity.Type == "ACTIVITY_END":
			s.sendLocked(liveClientServerEvent{Type: "input_audio_buffer.speech_stopped", ItemId: s.itemID, AudioEndMs: liveOffsetMs(upstream.VoiceActivity.AudioOffset)})
		case upstream.ServerContent != nil && upstream.ServerContent.InterimInputTranscription != nil:
			// Interim transcriptions are cumulative and may revise their last words; a word goes out once
			// the next interim repeats it, so what was sent is rarely revised.
			text := upstream.ServerContent.InterimInputTranscription.Text
			item := s.currentItemLocked()
			stable := stableInterimText(s.lastInterim, text)
			s.lastInterim = text
			if delta, ok := strings.CutPrefix(stable, s.sent); ok && delta != "" {
				s.sent = stable
				s.sendLocked(liveClientServerEvent{Type: "conversation.item.input_audio_transcription.delta", ItemId: item, ContentIndex: common.GetPointer(0), Delta: delta})
			}
		case upstream.ServerContent != nil && upstream.ServerContent.InputTranscription != nil:
			transcript := upstream.ServerContent.InputTranscription.Text
			item := s.currentItemLocked()
			// A revision of text already sent cannot be appended; the completed transcript carries it.
			if delta, ok := strings.CutPrefix(transcript, s.sent); ok && delta != "" {
				s.sendLocked(liveClientServerEvent{Type: "conversation.item.input_audio_transcription.delta", ItemId: item, ContentIndex: common.GetPointer(0), Delta: delta})
			}
			completed := liveClientServerEvent{Type: "conversation.item.input_audio_transcription.completed", ItemId: item, ContentIndex: common.GetPointer(0), Transcript: &transcript}
			if metadata := upstream.UsageMetadata; metadata != nil {
				completed.Usage = &liveTranscriptionUsage{
					Type:         "tokens",
					InputTokens:  metadata.PromptTokenCount,
					OutputTokens: metadata.CandidatesTokenCount,
					TotalTokens:  metadata.TotalTokenCount,
				}
				for _, detail := range metadata.PromptTokensDetails {
					switch detail.Modality {
					case "AUDIO":
						completed.Usage.InputTokenDetails.AudioTokens += detail.TokenCount
					case "TEXT":
						completed.Usage.InputTokenDetails.TextTokens += detail.TokenCount
					}
				}
			}
			s.sendLocked(completed)
			s.itemDone = true
		}
		s.mu.Unlock()
	}
}

// liveTranscriptionHandler runs a realtime transcription session on the Live API until either side
// closes or the session limit is reached, then returns its usage. Both connections are closed and both
// relay goroutines have ended when it returns.
func (a *Adaptor) liveTranscriptionHandler(c *gin.Context, info *relaycommon.RelayInfo) (*dto.RealtimeUsage, *types.NewAPIError) {
	if info.ClientWs == nil || info.TargetWs == nil {
		return nil, types.NewError(errors.New("invalid websocket connection"), types.ErrorCodeBadResponse)
	}
	info.IsStream = true
	session := &liveSession{client: info.ClientWs, upstream: info.TargetWs, done: make(chan struct{})}

	var setup liveSetupMessage
	region := normalizeVertexRegion(GetModelRegion(info.ApiVersion, info.OriginModelName))
	setup.Setup.Model = fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", a.AccountCredentials.ProjectID, region, info.UpstreamModelName)
	setup.Setup.GenerationConfig.ResponseModalities = []string{"TEXT"}
	if err := session.writeUpstream(setup); err != nil {
		session.close()
		return nil, types.NewError(fmt.Errorf("send the Live API setup: %w", err), types.ErrorCodeDoRequestFailed)
	}

	var wg sync.WaitGroup
	wg.Go(session.relayClient)
	wg.Go(func() { session.relayUpstream(info) })
	limit := time.NewTimer(liveSessionLimit)
	defer limit.Stop()
	select {
	case <-session.done:
	case <-limit.C:
		session.sendError("session_expired", fmt.Sprintf("the Live API session reached its %s limit", liveSessionLimit), "")
		session.close()
	}
	wg.Wait()

	usage := session.usage
	return &usage, nil
}
