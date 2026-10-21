package vertex

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vertex's answers for gemini-3.5-transcribe on a 10.85 s WAV, the second with word timestamps (13 words
// upstream, cut down to three; the offsets after the first word are stand-ins).
const (
	transcribeAnswer = `{"candidates":[{"content":{"role":"model","parts":[{"text":"The quick brown fox jumps over the lazy dog. Testing 1 2 3.",` +
		`"audioTranscription":{"text":"The quick brown fox jumps over the lazy dog. Testing 1 2 3."}}]},"finishReason":"STOP","index":0}],` +
		`"usageMetadata":{"promptTokenCount":272,"candidatesTokenCount":18,"promptTokensDetails":[{"modality":"AUDIO","tokenCount":272}],"candidatesTokensDetails":[{"modality":"TEXT","tokenCount":18}]},` +
		`"modelVersion":"gemini-3.5-transcribe"}`
	transcribeWordsAnswer = `{"candidates":[{"content":{"role":"model","parts":[{"text":"The quick brown fox jumps over the lazy dog. Testing one, two, three.",` +
		`"audioTranscription":{"text":"The quick brown fox jumps over the lazy dog. Testing one, two, three.","words":[` +
		`{"word":"The","startOffset":"0.200s","endOffset":"0.500s"},{"word":"quick","startOffset":"0.500s","endOffset":"0.840s"},{"word":"three.","startOffset":"9.900s","endOffset":"10.600s"}]}}]},` +
		`"finishReason":"STOP","index":0}],` +
		`"usageMetadata":{"promptTokenCount":284,"candidatesTokenCount":17,"promptTokensDetails":[{"modality":"AUDIO","tokenCount":272},{"modality":"TEXT","tokenCount":12}],"candidatesTokensDetails":[{"modality":"TEXT","tokenCount":17}]},` +
		`"modelVersion":"gemini-3.5-transcribe"}`
)

// testWAV is a silent 24 kHz, mono, 16-bit PCM WAV of a quarter second.
func testWAV() []byte {
	const sampleRate, samples = 24000, 6000
	var wav bytes.Buffer
	wav.WriteString("RIFF")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(36+samples*2))
	wav.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(sampleRate), uint32(sampleRate * 2), uint16(2), uint16(16)} {
		_ = binary.Write(&wav, binary.LittleEndian, v)
	}
	wav.WriteString("data")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(samples*2))
	wav.Write(make([]byte, samples*2))
	return wav.Bytes()
}

type transcriptionUpload struct {
	fields      [][2]string
	filename    string
	contentType string
}

// generateContentServer answers every call with answer and records the calls.
type generateContentServer struct {
	*httptest.Server
	calls        []string
	paths        []string
	contentTypes []string
}

func newGenerateContentServer(t *testing.T, answer string) *generateContentServer {
	server := &generateContentServer{}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		server.calls = append(server.calls, string(body))
		server.paths = append(server.paths, r.URL.Path+"?"+r.URL.RawQuery)
		server.contentTypes = append(server.contentTypes, r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)
	return server
}

// relayVertexTranscription takes the adaptor through the steps of the audio relay: convert, request, the
// upstream error on a status that is not 200, and the answer.
func relayVertexTranscription(t *testing.T, baseURL, model string, relayMode int, upload transcriptionUpload, audio []byte) (*httptest.ResponseRecorder, *dto.Usage, *types.NewAPIError) {
	gin.SetMode(gin.TestMode)
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	require.NoError(t, writer.WriteField("model", model))
	for _, field := range upload.fields {
		require.NoError(t, writer.WriteField(field[0], field[1]))
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+upload.filename+`"`)
	header.Set("Content-Type", upload.contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(audio)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", &form)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	info := &relaycommon.RelayInfo{
		RelayMode:       relayMode,
		RelayFormat:     types.RelayFormatOpenAIAudio,
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

	adaptor := &Adaptor{}
	adaptor.Init(info)
	body, err := adaptor.ConvertAudioRequest(c, info, dto.AudioRequest{Model: model})
	if err != nil {
		return recorder, nil, types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}
	resp, err := adaptor.DoRequest(c, info, body)
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

func TestVertexTranscriptionByGenerateContent(t *testing.T) {
	audio := testWAV()
	inlineAudio := `{"inlineData":{"mimeType":"audio/wav","data":"` + base64.StdEncoding.EncodeToString(audio) + `"}}`

	tests := []struct {
		name            string
		answer          string
		upload          transcriptionUpload
		wantCall        string
		wantContentType string
		wantBody        string
		wantPromptText  int
		wantCompletion  int
		wantTotal       int
	}{
		{
			name:            "json by default",
			answer:          transcribeAnswer,
			upload:          transcriptionUpload{filename: "fox.wav", contentType: "audio/wav"},
			wantCall:        `{"contents":[{"role":"user","parts":[` + inlineAudio + `]}]}`,
			wantContentType: "application/json",
			wantBody: `{"text":"The quick brown fox jumps over the lazy dog. Testing 1 2 3.",` +
				`"usage":{"type":"tokens","input_tokens":272,"input_token_details":{"text_tokens":0,"audio_tokens":272},"output_tokens":18,"total_tokens":290}}`,
			wantCompletion: 18,
			wantTotal:      290,
		},
		{
			name:            "text with a language",
			answer:          transcribeAnswer,
			upload:          transcriptionUpload{fields: [][2]string{{"response_format", "text"}, {"language", "en"}, {"prompt", "fox, dog"}}, filename: "fox.wav", contentType: "audio/x-wav"},
			wantCall:        `{"contents":[{"role":"user","parts":[` + inlineAudio + `]}],"generationConfig":{"audioTranscriptionConfig":{"languageCodes":["en"]}}}`,
			wantContentType: "text/plain; charset=utf-8",
			wantBody:        "The quick brown fox jumps over the lazy dog. Testing 1 2 3.",
			wantCompletion:  18,
			wantTotal:       290,
		},
		{
			name:   "verbose_json with words",
			answer: transcribeWordsAnswer,
			upload: transcriptionUpload{
				fields:   [][2]string{{"response_format", "verbose_json"}, {"language", "en"}, {"timestamp_granularities[]", "word"}},
				filename: "fox.wav", contentType: "application/octet-stream",
			},
			wantCall:        `{"contents":[{"role":"user","parts":[` + inlineAudio + `]}],"generationConfig":{"audioTranscriptionConfig":{"languageCodes":["en"],"wordTimestamp":true}}}`,
			wantContentType: "application/json",
			wantBody: `{"task":"transcribe","language":"en","duration":0.25,"text":"The quick brown fox jumps over the lazy dog. Testing one, two, three.",` +
				`"words":[{"word":"The","start":0.2,"end":0.5},{"word":"quick","start":0.5,"end":0.84},{"word":"three.","start":9.9,"end":10.6}]}`,
			wantPromptText: 12,
			wantCompletion: 17,
			wantTotal:      301,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newGenerateContentServer(t, tt.answer)

			recorder, usage, apiErr := relayVertexTranscription(t, server.URL, "gemini-3.5-transcribe", relayconstant.RelayModeAudioTranscription, tt.upload, audio)
			require.Nil(t, apiErr)

			require.Len(t, server.calls, 1)
			assert.JSONEq(t, tt.wantCall, server.calls[0])
			assert.Equal(t, "/v1/projects/my-project/locations/global/publishers/google/models/gemini-3.5-transcribe:generateContent?key=test-key", server.paths[0])
			assert.Equal(t, "application/json", server.contentTypes[0])
			assert.Equal(t, tt.wantContentType, recorder.Header().Get("Content-Type"))
			if tt.wantContentType == "application/json" {
				assert.JSONEq(t, tt.wantBody, recorder.Body.String())
			} else {
				assert.Equal(t, tt.wantBody, recorder.Body.String())
			}

			// Audio prompt tokens send the usage to the audio quota, priced with the audio ratios.
			assert.Equal(t, 272, usage.PromptTokensDetails.AudioTokens)
			assert.Equal(t, tt.wantPromptText, usage.PromptTokensDetails.TextTokens)
			assert.Equal(t, tt.wantCompletion, usage.CompletionTokenDetails.TextTokens)
			assert.Equal(t, tt.wantTotal, usage.TotalTokens)
		})
	}
}

func TestVertexTranscriptionRefusedBeforeAnyCall(t *testing.T) {
	audio := testWAV()
	tests := []struct {
		name       string
		model      string
		relayMode  int
		upload     transcriptionUpload
		wantStatus int
		wantError  string
	}{
		{
			name: "unsupported response_format", model: "gemini-3.5-transcribe", relayMode: relayconstant.RelayModeAudioTranscription,
			upload:     transcriptionUpload{fields: [][2]string{{"response_format", "srt"}}, filename: "fox.wav", contentType: "audio/wav"},
			wantStatus: http.StatusBadRequest, wantError: `response_format "srt" is not supported by gemini transcription models`,
		},
		{
			name: "stream", model: "gemini-3.5-transcribe", relayMode: relayconstant.RelayModeAudioTranscription,
			upload:     transcriptionUpload{fields: [][2]string{{"stream", "true"}}, filename: "fox.wav", contentType: "audio/wav"},
			wantStatus: http.StatusBadRequest, wantError: "gemini transcription models do not stream",
		},
		{
			name: "unsupported audio type", model: "gemini-3.5-transcribe", relayMode: relayconstant.RelayModeAudioTranscription,
			upload:     transcriptionUpload{filename: "fox.mp4", contentType: "video/mp4"},
			wantStatus: http.StatusBadRequest, wantError: `audio type "video/mp4" (fox.mp4) is not supported by gemini transcription models`,
		},
		// Any other model or audio route keeps the answer it had.
		{
			name: "chat model", model: "gemini-2.5-flash", relayMode: relayconstant.RelayModeAudioTranscription,
			upload:     transcriptionUpload{filename: "fox.wav", contentType: "audio/wav"},
			wantStatus: http.StatusInternalServerError, wantError: "not implemented",
		},
		{
			name: "translation", model: "gemini-3.5-transcribe", relayMode: relayconstant.RelayModeAudioTranslation,
			upload:     transcriptionUpload{filename: "fox.wav", contentType: "audio/wav"},
			wantStatus: http.StatusInternalServerError, wantError: "not implemented",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newGenerateContentServer(t, transcribeAnswer)

			_, _, apiErr := relayVertexTranscription(t, server.URL, tt.model, tt.relayMode, tt.upload, audio)
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.StatusCode)
			assert.Equal(t, tt.wantError, apiErr.Error())
			assert.Empty(t, server.calls)
		})
	}
}
