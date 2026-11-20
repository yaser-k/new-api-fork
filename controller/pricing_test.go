package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestPricingVisibleGroups(t *testing.T) {
	usable := map[string]string{"default": "Default"}
	ratios := map[string]float64{"default": 1, "premium": 1.5}
	pricing := []model.Pricing{
		{ModelName: "shared", EnableGroup: []string{"default", "premium"}},
		{ModelName: "premium-only", EnableGroup: []string{"premium"}},
		{ModelName: "unpriced-group", EnableGroup: []string{"internal"}},
	}

	tests := []struct {
		name       string
		showAll    bool
		wantGroups map[string]string
		wantModels []string
	}{
		{
			name:       "usable groups only by default",
			showAll:    false,
			wantGroups: map[string]string{"default": "Default"},
			wantModels: []string{"shared"},
		},
		{
			name:       "every group with a ratio when enabled",
			showAll:    true,
			wantGroups: map[string]string{"default": "Default", "premium": "premium"},
			wantModels: []string{"shared", "premium-only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := pricingVisibleGroups(usable, ratios, tt.showAll)
			assert.Equal(t, tt.wantGroups, groups)

			var models []string
			for _, item := range filterPricingByUsableGroups(pricing, groups) {
				models = append(models, item.ModelName)
			}
			assert.Equal(t, tt.wantModels, models)
			assert.Equal(t, map[string]string{"default": "Default"}, usable, "the viewer's usable groups must not change")
		})
	}
}
