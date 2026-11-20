package gemini

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/internal/convdiag"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
	"github.com/QuantumNous/new-api/relaykit/types"
)

// supportedMimeTypes follows the Blob.mimeType list in the Gemini API
// reference (https://ai.google.dev/api/generate-content#Blob). Audio and
// video are listed there as audio/* and video/*; IsSupportedMimeType accepts
// them by prefix.
var supportedMimeTypes = map[string]bool{
	"image/png":                 true,
	"image/jpeg":                true,
	"image/jpg":                 true,
	"image/webp":                true,
	"image/heic":                true,
	"image/heif":                true,
	"image/gif":                 true,
	"image/avif":                true,
	"text/plain":                true,
	"text/html":                 true,
	"text/css":                  true,
	"text/javascript":           true,
	"text/x-typescript":         true,
	"text/csv":                  true,
	"text/markdown":             true,
	"text/x-python":             true,
	"text/xml":                  true,
	"text/rtf":                  true,
	"application/x-javascript":  true,
	"application/x-typescript":  true,
	"application/x-python-code": true,
	"application/json":          true,
	"application/x-ipynb+json":  true,
	"application/rtf":           true,
	"application/pdf":           true,
}

// IsSupportedMimeType reports whether Gemini accepts inline data of the MIME
// type.
func IsSupportedMimeType(mimeType string) bool {
	mimeType = strings.ToLower(mimeType)
	return supportedMimeTypes[mimeType] || strings.HasPrefix(mimeType, "audio/") || strings.HasPrefix(mimeType, "video/")
}

var SafetySettingCategories = []string{
	"HARM_CATEGORY_HARASSMENT",
	"HARM_CATEGORY_HATE_SPEECH",
	"HARM_CATEGORY_SEXUALLY_EXPLICIT",
	"HARM_CATEGORY_DANGEROUS_CONTENT",
}

const ThoughtSignatureBypassValue = "context_engineering_is_the_way_to_go"

func ShouldAttachThoughtSignature(opts *convmeta.Options) bool {
	return opts != nil && opts.Gemini.FunctionCallThoughtSignatureEnabled
}

func AttachThoughtSignatureBypass(opts *convmeta.Options, part *dto.GeminiPart) bool {
	if part == nil || len(part.ThoughtSignature) > 0 || !ShouldAttachThoughtSignature(opts) {
		return false
	}
	part.ThoughtSignature = []byte(strconv.Quote(ThoughtSignatureBypassValue))
	return true
}

func AttachFunctionCallThoughtSignature(opts *convmeta.Options, part *dto.GeminiPart) bool {
	if part == nil || !HasFunctionCallContent(part.FunctionCall) {
		return false
	}
	return AttachThoughtSignatureBypass(opts, part)
}

func AttachFirstTextThoughtSignature(opts *convmeta.Options, parts []dto.GeminiPart) bool {
	if !ShouldAttachThoughtSignature(opts) {
		return false
	}
	for i := range parts {
		if parts[i].Text != "" && len(parts[i].ThoughtSignature) == 0 {
			parts[i].ThoughtSignature = []byte(strconv.Quote(ThoughtSignatureBypassValue))
			return true
		}
	}
	return false
}

// ApplyThinkingConfig resolves every reasoning control that can reach a
// Gemini request (provider-native thinkingConfig, generic OpenAI fields, and
// host model-name aliases) into the target model's dialect. Precedence is
// model name, then provider-native config, then generic fields; each override
// and each capability coercion is reported through convdiag on ctx.
func ApplyThinkingConfig(ctx context.Context, geminiRequest *dto.GeminiChatRequest, info convmeta.Meta, oaiRequest ...dto.GeneralOpenAIRequest) error {
	opts := convmeta.OptionsOf(info)
	if geminiRequest == nil {
		return nil
	}

	modelName := convmeta.UpstreamModelName(info)
	var source reasoning.Intent
	crossProtocol := len(oaiRequest) > 0
	if len(oaiRequest) > 0 {
		if modelName == "" {
			modelName = oaiRequest[0].Model
		}
		var diagnostics []types.ConversionDiagnostic
		var err error
		source, diagnostics, err = reasoning.FromOpenAIChat(&oaiRequest[0])
		if err != nil {
			return err
		}
		convdiag.Add(ctx, diagnostics...)
	}

	baseModel := modelName
	suffix := reasoning.IntentFromState(convmeta.ReasoningStateOf(info))
	preserveSuffix := opts.ShouldPreserveThinkingSuffix(modelName)
	if info != nil && opts.ShouldPreserveThinkingSuffix(info.GetOriginModelName()) {
		preserveSuffix = true
	}
	if preserveSuffix {
		suffix = reasoning.Intent{}
	}
	// Native Gemini requests already use the target protocol. Without a host
	// modifier, read portable effort metadata without running the capability
	// renderer or rewriting provider-native controls.
	if !crossProtocol && suffix.IsEmpty() {
		if info != nil {
			effort := ""
			if config := geminiRequest.GenerationConfig.ThinkingConfig; config != nil {
				effort = config.ThinkingLevel
				// Gemini accepts thinkingLevel case-insensitively; record the
				// canonical effort so logs match other protocols. Unknown values
				// stay as sent because this path does not validate.
				if canonical, err := reasoning.ParseEffort(effort); err == nil {
					effort = string(canonical)
				}
				if effort == "" && config.ThinkingBudget != nil {
					effort = string(reasoning.EffortFromBudget(*config.ThinkingBudget))
				}
			}
			info.SetReasoningEffort(effort)
		}
		return nil
	}
	// Rewrite the provider-native config into the target model's dialect
	// first, so a budget on Gemini 3 or a level on Gemini 2.5 is converted
	// rather than rejected and later comparisons see what will be sent.
	nativeEffort, diagnostics, err := reasoning.NormalizeGeminiThinkingConfig(baseModel, &geminiRequest.GenerationConfig)
	if err != nil {
		return err
	}
	convdiag.Add(ctx, diagnostics...)
	native, diagnostics, err := reasoning.FromGemini(geminiRequest)
	if err != nil {
		return err
	}
	convdiag.Add(ctx, diagnostics...)
	if native.HasStrength() && source.HasStrength() {
		// Native Gemini configuration is the lossless representation and wins
		// over the generic OpenAI fields. Only a generic effort or budget can
		// disagree with it; a bare enable states no strength to compare.
		if source.Effort != "" || source.BudgetTokens != nil {
			equivalent, compareErr := reasoning.EquivalentGeminiStrength(baseModel, native, source)
			if compareErr != nil {
				return compareErr
			}
			if !equivalent {
				convdiag.Add(ctx, types.ConversionDiagnostic{
					Code:     "native_overrode_standard",
					Path:     "generationConfig.thinkingConfig",
					Message:  fmt.Sprintf("model %q: Gemini thinking_config effort %q overrides the standard reasoning effort %q", modelName, reasoning.EffectiveEffort(native), reasoning.EffectiveEffort(source)),
					Severity: types.ConversionDiagnosticWarning,
				})
			}
		}
		if native.IncludeThoughts == nil {
			native.IncludeThoughts = source.IncludeThoughts
		}
		source = reasoning.Intent{}
	}
	explicit, diagnostics, err := reasoning.MergeExplicit(native, source, modelName)
	if err != nil {
		return err
	}
	convdiag.Add(ctx, diagnostics...)
	if explicit.HasStrength() && suffix.HasStrength() {
		equivalent, compareErr := reasoning.EquivalentGeminiStrength(baseModel, explicit, suffix)
		if compareErr != nil {
			return compareErr
		}
		if equivalent {
			if explicit.IncludeThoughts == nil {
				explicit.IncludeThoughts = suffix.IncludeThoughts
			}
			suffix = reasoning.Intent{}
		}
	}
	requested, diagnostics, err := reasoning.MergeExplicitAndSuffix(explicit, suffix, modelName)
	if err != nil {
		return err
	}
	convdiag.Add(ctx, diagnostics...)
	requested = reasoning.ResolveGeminiEnabledDefault(baseModel, requested, geminiRequest.GenerationConfig.MaxOutputTokens)

	if native.HasStrength() && !suffix.HasStrength() {
		// The normalized native config is what goes upstream; only portable
		// visibility metadata is taken from the standard representation.
		if explicit.IncludeThoughts != nil {
			geminiRequest.GenerationConfig.ThinkingConfig.IncludeThoughts = explicit.IncludeThoughts
		}
		if info != nil && nativeEffort != "" {
			info.SetReasoningEffort(string(nativeEffort))
		}
		return nil
	}
	if requested.IsEmpty() {
		return nil
	}
	rendered, err := reasoning.RenderGemini(
		baseModel,
		requested,
		geminiRequest.GenerationConfig.MaxOutputTokens,
		opts.Gemini.ThinkingAdapterBudgetTokensPercentage,
	)
	if err != nil {
		return err
	}
	convdiag.Add(ctx, rendered.Diagnostics...)
	geminiRequest.GenerationConfig.ThinkingConfig = rendered.Config
	if info != nil && rendered.EffectiveEffort != "" {
		info.SetReasoningEffort(string(rendered.EffectiveEffort))
	}
	return nil
}

func ParseStopSequences(stop any) []string {
	if stop == nil {
		return nil
	}

	switch v := stop.(type) {
	case string:
		if v != "" {
			return []string{v}
		}
	case []string:
		return v
	case []interface{}:
		sequences := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				sequences = append(sequences, str)
			}
		}
		return sequences
	}
	return nil
}

func HasFunctionCallContent(call *dto.FunctionCall) bool {
	if call == nil {
		return false
	}
	if strings.TrimSpace(call.FunctionName) != "" {
		return true
	}

	switch v := call.Arguments.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	case map[string]interface{}:
		return len(v) > 0
	case []interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func SupportedMimeTypesList() []string {
	return append(slices.Sorted(maps.Keys(supportedMimeTypes)), "audio/*", "video/*")
}

// AppendContentPart adds part to the last content when it has the same role,
// so consecutive parts of one role form a single turn, and starts a content
// otherwise. Function calls stay ahead of the other parts of a model turn.
// Function responses stay ahead of the tool media sent after them, in the
// order of the calls they answer in the preceding model turn: Gemini pairs a
// response with its call by position when call ids are absent, as on Vertex AI
// by default.
func AppendContentPart(req *dto.GeminiChatRequest, role string, part dto.GeminiPart) {
	last := len(req.Contents) - 1
	if last < 0 || req.Contents[last].Role != role {
		req.Contents = append(req.Contents, dto.GeminiChatContent{
			Role:  role,
			Parts: []dto.GeminiPart{part},
		})
		return
	}
	parts := req.Contents[last].Parts
	insertAt := len(parts)
	switch {
	case role == "model" && part.FunctionCall != nil:
		insertAt = 0
		for insertAt < len(parts) && parts[insertAt].FunctionCall != nil {
			insertAt++
		}
	case part.FunctionResponse != nil:
		var calls []dto.GeminiPart
		if last > 0 && req.Contents[last-1].Role == "model" {
			calls = req.Contents[last-1].Parts
		}
		position := answeredCallPosition(calls, part.FunctionResponse)
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i].FunctionResponse == nil {
				continue
			}
			if answeredCallPosition(calls, parts[i].FunctionResponse) <= position {
				insertAt = i + 1
				break
			}
			insertAt = i
		}
	}
	req.Contents[last].Parts = slices.Insert(parts, insertAt, part)
}

// answeredCallPosition returns the index, among the function calls in calls,
// of the call whose id the response carries, or len(calls) when none matches.
func answeredCallPosition(calls []dto.GeminiPart, response *dto.GeminiFunctionResponse) int {
	id := kitutil.JsonRawMessageToString(response.ID)
	if id == "" {
		return len(calls)
	}
	position := 0
	for _, call := range calls {
		if call.FunctionCall == nil {
			continue
		}
		if call.FunctionCall.ID == id {
			return position
		}
		position++
	}
	return len(calls)
}
