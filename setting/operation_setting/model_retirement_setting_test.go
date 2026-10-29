package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateModelRetirementsJSONRefusesInvalidEntries(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		message string
	}{
		{"not an object", `["old-model"]`, "Retired models must be a JSON object"},
		{"entry not an object", `{"old-model": "new-model"}`, `Retired model "old-model" must be a JSON object`},
		{"empty name", `{" ": {"retired_at": "2026-09-30", "successor": "new-model"}}`, "A retired model name must not be empty"},
		{"missing date", `{"old-model": {"successor": "new-model"}}`, `Retired model "old-model": retired_at must be a date written YYYY-MM-DD`},
		{"bad date", `{"old-model": {"retired_at": "2026-9-30", "successor": "new-model"}}`, `Retired model "old-model": retired_at must be a date written YYYY-MM-DD`},
		{"impossible date", `{"old-model": {"retired_at": "2026-02-30", "successor": "new-model"}}`, `Retired model "old-model": retired_at must be a date written YYYY-MM-DD`},
		{"missing successor", `{"old-model": {"retired_at": "2026-09-30"}}`, `Retired model "old-model": successor must name a model`},
		{"blank successor", `{"old-model": {"retired_at": "2026-09-30", "successor": "  "}}`, `Retired model "old-model": successor must name a model`},
		{"successor is itself", `{"old-model": {"retired_at": "2026-09-30", "successor": "old-model"}}`, `Retired model "old-model": the successor must be a different model`},
		{"chained successor", `{"old-model": {"retired_at": "2026-09-30", "successor": "mid-model"}, "mid-model": {"retired_at": "2026-10-01", "successor": "new-model"}}`, `Retired model "old-model": the successor "mid-model" is itself retired`},
		{"http notice", `{"old-model": {"retired_at": "2026-09-30", "successor": "new-model", "notice_url": "http://example.com/notice"}}`, `Retired model "old-model": notice_url must be an https:// URL`},
		{"relative notice", `{"old-model": {"retired_at": "2026-09-30", "successor": "new-model", "notice_url": "/notice"}}`, `Retired model "old-model": notice_url must be an https:// URL`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelRetirementsJSON(tt.value)
			require.Error(t, err)
			assert.Equal(t, tt.message, err.Error())
		})
	}
}

func TestValidateModelRetirementsJSONAcceptsValidList(t *testing.T) {
	for _, value := range []string{"", "{}", `{
		"old-model": {"retired_at": "2026-09-30", "successor": "new-model", "notice_url": "https://example.com/notice", "note": "Moved"},
		"older-model": {"retired_at": "2026-01-15", "successor": "new-model"}
	}`} {
		assert.NoError(t, ValidateModelRetirementsJSON(value), value)
	}
}

func TestLoadModelRetirementsKeepsValidEntries(t *testing.T) {
	t.Cleanup(func() { LoadModelRetirementsFromJSONString("{}") })

	LoadModelRetirementsFromJSONString(`{
		"old-model": {"retired_at": "2026-09-30", "successor": " new-model ", "notice_url": "https://example.com/notice"},
		"broken-model": {"retired_at": "soon", "successor": "new-model"}
	}`)

	retirement, ok := GetModelRetirement("old-model")
	require.True(t, ok)
	assert.Equal(t, ModelRetirement{RetiredAt: "2026-09-30", Successor: "new-model", NoticeURL: "https://example.com/notice"}, retirement)
	assert.False(t, IsModelRetired("broken-model"))
	assert.False(t, IsModelRetired("new-model"))
	assert.Len(t, GetModelRetirementsCopy(), 1)
	assert.Len(t, modelRetirementSetting.Models, 1, "the exported option follows the loaded list")

	LoadModelRetirementsFromJSONString("not json")
	assert.Empty(t, GetModelRetirementsCopy())
}

func TestModelRetirementsEmptyByDefault(t *testing.T) {
	assert.Empty(t, GetModelRetirementsCopy())
	assert.False(t, IsModelRetired(""))
	assert.False(t, IsModelRetired("any-model"))
}
