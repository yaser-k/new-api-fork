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
		wantModels map[string][]string
	}{
		{
			name:       "usable groups only by default",
			showAll:    false,
			wantGroups: map[string]string{"default": "Default"},
			wantModels: map[string][]string{"shared": {"default"}},
		},
		{
			name:       "every group with a ratio when enabled",
			showAll:    true,
			wantGroups: map[string]string{"default": "Default", "premium": "premium"},
			wantModels: map[string][]string{"shared": {"default", "premium"}, "premium-only": {"premium"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := pricingVisibleGroups(usable, ratios, tt.showAll)
			assert.Equal(t, tt.wantGroups, groups)

			models := map[string][]string{}
			for _, item := range filterPricingByUsableGroups(pricing, groups) {
				models[item.ModelName] = item.EnableGroup
			}
			assert.Equal(t, tt.wantModels, models)
			assert.Equal(t, map[string]string{"default": "Default"}, usable, "the viewer's usable groups must not change")
		})
	}
}
