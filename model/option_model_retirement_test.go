package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelRetirementOptionIsValidatedAndLoaded(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		require.NoError(t, updateOptionMap(operation_setting.ModelRetirementOptionKey, "{}"))
	})

	err := validateOptionValue(operation_setting.ModelRetirementOptionKey,
		`{"old-model": {"retired_at": "2026-09-30", "successor": "old-model"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"old-model"`)

	value := `{"old-model": {"retired_at": "2026-09-30", "successor": "new-model"}}`
	require.NoError(t, validateOptionValue(operation_setting.ModelRetirementOptionKey, value))
	require.NoError(t, updateOptionMap(operation_setting.ModelRetirementOptionKey, value))

	retirement, ok := operation_setting.GetModelRetirement("old-model")
	require.True(t, ok)
	assert.Equal(t, "new-model", retirement.Successor)
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	assert.Equal(t, value, common.OptionMap[operation_setting.ModelRetirementOptionKey])
}
