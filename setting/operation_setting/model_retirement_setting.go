package operation_setting

import (
	"encoding/json"
	"maps"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// ---------------------------------------------------------------------------
// Model retirement list
// DB key: model_retirement_setting.models
//
// A JSON object mapping each retired model name to its retirement:
//
//	{"example-model-v1": {"retired_at": "2026-09-30", "successor": "example-model-v2",
//	  "notice_url": "https://example.com/retirements", "note": "Moved to v2"}}
//
// Empty (the default) changes nothing. A listed name is answered by the
// distributor with HTTP 410 model_retired, is left out of the model lists and
// is shown as retired on the pricing page.
// ---------------------------------------------------------------------------

const ModelRetirementOptionKey = "model_retirement_setting.models"

const modelRetirementDateLayout = "2006-01-02"

type ModelRetirement struct {
	RetiredAt string `json:"retired_at"`
	Successor string `json:"successor"`
	NoticeURL string `json:"notice_url,omitempty"`
	Note      string `json:"note,omitempty"`
}

// ModelRetirementSetting is managed by config.GlobalConfig.Register.
type ModelRetirementSetting struct {
	Models map[string]ModelRetirement `json:"models"`
}

var modelRetirementSetting = ModelRetirementSetting{
	Models: make(map[string]ModelRetirement),
}

// modelRetirements is the list the request path reads, replaced whole on each
// update so a reader never sees a half-written map.
var modelRetirements atomic.Pointer[map[string]ModelRetirement]

func init() {
	config.GlobalConfig.Register("model_retirement_setting", &modelRetirementSetting)
	storeModelRetirements(map[string]ModelRetirement{})
}

func storeModelRetirements(models map[string]ModelRetirement) {
	modelRetirementSetting.Models = models
	modelRetirements.Store(&models)
}

func validateModelRetirement(name string, retirement ModelRetirement, models map[string]ModelRetirement) error {
	if strings.TrimSpace(name) == "" {
		return common.NewMessage("A retired model name must not be empty")
	}
	if _, err := time.Parse(modelRetirementDateLayout, retirement.RetiredAt); err != nil {
		return common.NewMessage(`Retired model "{{name}}": retired_at must be a date written YYYY-MM-DD`, map[string]any{"name": name})
	}
	successor := strings.TrimSpace(retirement.Successor)
	if successor == "" {
		return common.NewMessage(`Retired model "{{name}}": successor must name a model`, map[string]any{"name": name})
	}
	if successor == name {
		return common.NewMessage(`Retired model "{{name}}": the successor must be a different model`, map[string]any{"name": name})
	}
	if _, retired := models[successor]; retired {
		return common.NewMessage(`Retired model "{{name}}": the successor "{{successor}}" is itself retired`, map[string]any{"name": name, "successor": successor})
	}
	if retirement.NoticeURL != "" {
		parsed, err := url.Parse(retirement.NoticeURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return common.NewMessage(`Retired model "{{name}}": notice_url must be an https:// URL`, map[string]any{"name": name})
		}
	}
	return nil
}

func decodeModelRetirementsJSON(value string, ignoreInvalidEntries bool) (map[string]ModelRetirement, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]ModelRetirement{}, nil
	}
	rawValue := json.RawMessage(value)
	if common.GetJsonType(rawValue) != "object" {
		return nil, common.NewMessage("Retired models must be a JSON object")
	}
	var rawModels map[string]json.RawMessage
	if err := common.Unmarshal(rawValue, &rawModels); err != nil {
		return nil, common.NewMessage("Failed to parse retired models: {{error}}", map[string]any{"error": err.Error()})
	}

	decoded := make(map[string]ModelRetirement, len(rawModels))
	var entryErrs []error
	for name, rawEntry := range rawModels {
		var retirement ModelRetirement
		if common.GetJsonType(rawEntry) != "object" || common.Unmarshal(rawEntry, &retirement) != nil {
			entryErrs = append(entryErrs, common.NewMessage(`Retired model "{{name}}" must be a JSON object`, map[string]any{"name": name}))
			continue
		}
		retirement.Successor = strings.TrimSpace(retirement.Successor)
		decoded[name] = retirement
	}

	names := make([]string, 0, len(decoded))
	for name := range decoded {
		names = append(names, name)
	}
	sort.Strings(names)
	models := make(map[string]ModelRetirement, len(decoded))
	for _, name := range names {
		if err := validateModelRetirement(name, decoded[name], decoded); err != nil {
			entryErrs = append(entryErrs, err)
			continue
		}
		models[name] = decoded[name]
	}

	if len(entryErrs) > 0 && !ignoreInvalidEntries {
		return nil, entryErrs[0]
	}
	for _, err := range entryErrs {
		common.SysError(err.Error())
	}
	return models, nil
}

// ValidateModelRetirementsJSON validates an operator-supplied retirement list
// and refuses it whole, naming the first invalid entry.
func ValidateModelRetirementsJSON(value string) error {
	_, err := decodeModelRetirementsJSON(value, false)
	return err
}

// LoadModelRetirementsFromJSONString replaces the retirement list. Invalid
// stored entries are dropped one by one and logged, so valid siblings survive.
func LoadModelRetirementsFromJSONString(value string) {
	models, err := decodeModelRetirementsJSON(value, true)
	if err != nil {
		common.SysError(common.LogText("failed to load retired models, none is retired: %s", err.Error()))
		models = map[string]ModelRetirement{}
	}
	storeModelRetirements(models)
}

// GetModelRetirement returns the retirement of an exactly named model.
func GetModelRetirement(name string) (ModelRetirement, bool) {
	models := modelRetirements.Load()
	if models == nil || name == "" {
		return ModelRetirement{}, false
	}
	retirement, ok := (*models)[name]
	return retirement, ok
}

// IsModelRetired reports whether a model name is on the retirement list.
func IsModelRetired(name string) bool {
	_, ok := GetModelRetirement(name)
	return ok
}

// GetModelRetirementsCopy returns a copy of the retirement list.
func GetModelRetirementsCopy() map[string]ModelRetirement {
	models := modelRetirements.Load()
	if models == nil {
		return map[string]ModelRetirement{}
	}
	return maps.Clone(*models)
}
