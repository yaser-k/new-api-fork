package system_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchSettingReadsStoredAllowedPorts(t *testing.T) {
	tests := []struct {
		name   string
		stored string
		want   []string
	}{
		{name: "ports saved as text", stored: `["80","443","8000-9000"]`, want: []string{"80", "443", "8000-9000"}},
		{name: "ports saved as numbers by an older console", stored: `[80,443]`, want: []string{"80", "443"}},
		{name: "empty list allows every port", stored: `[]`, want: []string{}},
		{name: "unreadable value keeps the current ports", stored: `{"port":80}`, want: []string{"8080"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting := FetchSetting{}
			setting.AllowedPorts = append(setting.AllowedPorts, "8080")

			err := config.UpdateConfigFromMap(&setting, map[string]string{"allowed_ports": tt.stored})

			require.NoError(t, err)
			assert.Equal(t, tt.want, []string(setting.AllowedPorts))
		})
	}
}
