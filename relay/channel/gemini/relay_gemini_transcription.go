package gemini

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// transcriptionMIMETypes maps the audio types a Gemini transcription model takes, and their common
// aliases, to the type sent upstream.
var transcriptionMIMETypes = map[string]string{
	"audio/wav": "audio/wav", "audio/x-wav": "audio/wav", "audio/wave": "audio/wav", "audio/vnd.wave": "audio/wav",
	"audio/mp3": "audio/mp3", "audio/mpeg": "audio/mpeg",
	"audio/aiff": "audio/aiff", "audio/x-aiff": "audio/aiff",
	"audio/aac":  "audio/aac",
	"audio/ogg":  "audio/ogg",
	"audio/flac": "audio/flac", "audio/x-flac": "audio/flac",
	"audio/m4a": "audio/m4a", "audio/x-m4a": "audio/m4a",
	"audio/l16":   "audio/l16",
	"audio/opus":  "audio/opus",
	"audio/alaw":  "audio/alaw",
	"audio/mulaw": "audio/mulaw",
	"audio/webm":  "audio/webm",
}

// transcriptionExtensionMIMETypes names the type of an upload that arrives without one.
var transcriptionExtensionMIMETypes = map[string]string{
	".wav": "audio/wav", ".mp3": "audio/mp3", ".mpga": "audio/mpeg", ".mpeg": "audio/mpeg",
	".aiff": "audio/aiff", ".aif": "audio/aiff", ".aac": "audio/aac", ".ogg": "audio/ogg", ".oga": "audio/ogg",
	".flac": "audio/flac", ".m4a": "audio/m4a", ".opus": "audio/opus", ".webm": "audio/webm",
}

var transcriptionResponseFormats = []string{"json", "text", "verbose_json"}

// TranscriptionRequest is what the answer to a transcription needs from its request.
type TranscriptionRequest struct {
	ResponseFormat string
	// Language is the language the client named, the only one known: the upstream does not report one.
	Language       string
	WordTimestamps bool
	// Duration is the upload's length in seconds, read from its header for verbose_json; 0 when unknown.
	Duration float64
}

type geminiTranscriptionRequest struct {
	Contents         []dto.GeminiChatContent              `json:"contents"`
	GenerationConfig *geminiTranscriptionGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiTranscriptionGenerationConfig struct {
	AudioTranscriptionConfig geminiAudioTranscriptionConfig `json:"audioTranscriptionConfig"`
}

// geminiAudioTranscriptionConfig never sets a mode: SMART is refused with word timestamps.
type geminiAudioTranscriptionConfig struct {
	LanguageCodes []string `json:"languageCodes,omitempty"`
	WordTimestamp *bool    `json:"wordTimestamp,omitempty"`
}

type geminiTranscriptionWord struct {
	Word        string `json:"word"`
	StartOffset string `json:"startOffset"`
	EndOffset   string `json:"endOffset"`
}

type geminiTranscriptionPart struct {
	Text               string `json:"text"`
	AudioTranscription *struct {
		Text  string                    `json:"text"`
		Words []geminiTranscriptionWord `json:"words"`
	} `json:"audioTranscription"`
}

type geminiTranscriptionResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiTranscriptionPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback *dto.GeminiChatPromptFeedback `json:"promptFeedback"`
	UsageMetadata  *dto.GeminiUsageMetadata      `json:"usageMetadata"`
}

type openAITranscriptionUsage struct {
	Type              string `json:"type"`
	InputTokens       int    `json:"input_tokens"`
	InputTokenDetails struct {
		TextTokens  int `json:"text_tokens"`
		AudioTokens int `json:"audio_tokens"`
	} `json:"input_token_details"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type openAITranscriptionWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type openAIVerboseTranscription struct {
	Task     string                    `json:"task"`
	Language string                    `json:"language,omitempty"`
	Duration *float64                  `json:"duration,omitempty"`
	Text     string                    `json:"text"`
	Words    []openAITranscriptionWord `json:"words,omitempty"`
}

func transcriptionRequestError(err error) error {
	return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}

// ConvertTranscriptionRequest turns an OpenAI multipart transcription request into the generateContent
// body of a Gemini transcription model: the upload as one inline audio part, the language as its
// language code and word granularity as word timestamps. The prompt is not sent: Gemini's custom
// vocabulary is a list of terms, not the free text OpenAI's prompt is.
func ConvertTranscriptionRequest(c *gin.Context, info *relaycommon.RelayInfo) (*TranscriptionRequest, io.Reader, error) {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, nil, transcriptionRequestError(fmt.Errorf("error parsing multipart form: %w", err))
	}
	field := func(name string) string {
		if values := form.Value[name]; len(values) > 0 {
			return strings.TrimSpace(values[0])
		}
		return ""
	}

	if info.IsStream || strings.EqualFold(field("stream"), "true") {
		return nil, nil, transcriptionRequestError(errors.New("gemini transcription models do not stream"))
	}
	transcription := &TranscriptionRequest{
		ResponseFormat: field("response_format"),
		Language:       field("language"),
	}
	if transcription.ResponseFormat == "" {
		transcription.ResponseFormat = "json"
	}
	if !slices.Contains(transcriptionResponseFormats, transcription.ResponseFormat) {
		return nil, nil, transcriptionRequestError(fmt.Errorf("response_format %q is not supported by gemini transcription models", transcription.ResponseFormat))
	}
	transcription.WordTimestamps = slices.Contains(form.Value["timestamp_granularities[]"], "word") ||
		slices.Contains(form.Value["timestamp_granularities"], "word")

	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return nil, nil, transcriptionRequestError(errors.New("file is required"))
	}
	fileHeader := fileHeaders[0]
	mimeType := ""
	mediaType, params, mediaErr := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if mediaErr != nil || mediaType == "application/octet-stream" {
		mimeType = transcriptionExtensionMIMETypes[strings.ToLower(filepath.Ext(fileHeader.Filename))]
	} else if upstreamType, ok := transcriptionMIMETypes[mediaType]; ok {
		mimeType = mime.FormatMediaType(upstreamType, params)
	}
	if mimeType == "" {
		return nil, nil, transcriptionRequestError(fmt.Errorf("audio type %q (%s) is not supported by gemini transcription models", fileHeader.Header.Get("Content-Type"), fileHeader.Filename))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("error opening audio file: %w", err)
	}
	defer file.Close()
	audio, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading audio file: %w", err)
	}
	if transcription.ResponseFormat == "verbose_json" {
		// The header's length is shown to the client only; it never reaches billing.
		duration, err := common.GetAudioDuration(c.Request.Context(), bytes.NewReader(audio), filepath.Ext(fileHeader.Filename))
		if err == nil && duration > 0 && !math.IsInf(duration, 0) {
			transcription.Duration = duration
		}
	}

	body := geminiTranscriptionRequest{
		Contents: []dto.GeminiChatContent{{
			Role: "user",
			Parts: []dto.GeminiPart{{InlineData: &dto.GeminiInlineData{
				MimeType: mimeType,
				Data:     base64.StdEncoding.EncodeToString(audio),
			}}},
		}},
	}
	if transcription.Language != "" || transcription.WordTimestamps {
		config := geminiAudioTranscriptionConfig{}
		if transcription.Language != "" {
			config.LanguageCodes = []string{transcription.Language}
		}
		if transcription.WordTimestamps {
			config.WordTimestamp = common.GetPointer(true)
		}
		body.GenerationConfig = &geminiTranscriptionGenerationConfig{AudioTranscriptionConfig: config}
	}
	jsonBody, err := common.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	return transcription, bytes.NewReader(jsonBody), nil
}

// GeminiTranscriptionHandler answers a Gemini transcription model's generateContent response in the
// response_format the client asked for. The usage keeps the prompt's audio tokens apart from its text
// tokens, so the audio relay settles it with the model's audio ratios.
func GeminiTranscriptionHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, transcription *TranscriptionRequest) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
	var geminiResponse geminiTranscriptionResponse
	if err := common.Unmarshal(responseBody, &geminiResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	info.ObserveResponseModel(gjson.GetBytes(responseBody, "modelVersion").Str)

	if geminiResponse.PromptFeedback != nil && geminiResponse.PromptFeedback.BlockReason != nil {
		return nil, types.NewOpenAIError(errors.New("request blocked by Gemini API: "+*geminiResponse.PromptFeedback.BlockReason), types.ErrorCodePromptBlocked, http.StatusBadRequest)
	}
	if len(geminiResponse.Candidates) == 0 {
		return nil, types.NewOpenAIError(errors.New("empty response from Gemini API"), types.ErrorCodeEmptyResponse, http.StatusInternalServerError)
	}
	candidate := geminiResponse.Candidates[0]
	var text strings.Builder
	var words []openAITranscriptionWord
	for _, part := range candidate.Content.Parts {
		if part.AudioTranscription == nil {
			text.WriteString(part.Text)
			continue
		}
		text.WriteString(part.AudioTranscription.Text)
		for _, word := range part.AudioTranscription.Words {
			start, _ := time.ParseDuration(word.StartOffset)
			end, _ := time.ParseDuration(word.EndOffset)
			words = append(words, openAITranscriptionWord{Word: word.Word, Start: start.Seconds(), End: end.Seconds()})
		}
	}
	// Silence transcribes to an empty text that finishes with STOP; any other empty finish withheld it.
	if text.Len() == 0 && candidate.FinishReason != "" && candidate.FinishReason != "STOP" {
		return nil, types.NewOpenAIError(errors.New("transcription withheld by Gemini API: "+candidate.FinishReason), types.ErrorCodePromptBlocked, http.StatusBadRequest)
	}

	usage := buildUsageFromGeminiMetadata(geminiResponse.UsageMetadata, info.GetEstimatePromptTokens())
	// The audio quota prices the text and audio details only, so every token is in one of them.
	usage.PromptTokensDetails.TextTokens = max(usage.PromptTokens-usage.PromptTokensDetails.AudioTokens, 0)
	usage.CompletionTokenDetails.TextTokens = max(usage.CompletionTokens-usage.CompletionTokenDetails.AudioTokens, 0)
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	var answer any
	switch transcription.ResponseFormat {
	case "text":
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(text.String()))
		return &usage, nil
	case "verbose_json":
		verbose := openAIVerboseTranscription{
			Task:     "transcribe",
			Language: transcription.Language,
			Text:     text.String(),
		}
		if transcription.Duration > 0 {
			verbose.Duration = common.GetPointer(transcription.Duration)
		}
		if transcription.WordTimestamps {
			verbose.Words = words
		}
		answer = verbose
	default:
		openAIUsage := openAITranscriptionUsage{
			Type:         "tokens",
			InputTokens:  usage.PromptTokens,
			OutputTokens: usage.CompletionTokens,
			TotalTokens:  usage.TotalTokens,
		}
		openAIUsage.InputTokenDetails.TextTokens = usage.PromptTokensDetails.TextTokens
		openAIUsage.InputTokenDetails.AudioTokens = usage.PromptTokensDetails.AudioTokens
		answer = struct {
			Text  string                   `json:"text"`
			Usage openAITranscriptionUsage `json:"usage"`
		}{Text: text.String(), Usage: openAIUsage}
	}
	body, err := common.Marshal(answer)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	c.Data(http.StatusOK, "application/json", body)
	return &usage, nil
}
