package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func servePricing(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	GetPricing(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	return recorder
}

// basePricingBody renders the anonymous /api/pricing answer exactly as the
// handler wrote it before the retirement list existed.
func basePricingBody(t *testing.T) []byte {
	t.Helper()
	usableGroup := service.GetUserUsableGroups("")
	pricing := filterPricingByUsableGroups(model.GetPricing(), usableGroup)
	groupRatio := ratio_setting.GetGroupRatioCopy()
	for group := range groupRatio {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        service.GetUserAutoGroup(""),
		"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
	})
	return recorder.Body.Bytes()
}

type pricingRetirementResponse struct {
	Data []struct {
		ModelName string `json:"model_name"`
	} `json:"data"`
	RetiredModels []pricingRetiredModel `json:"retired_models"`
}

func setupPricingRetirementTest(t *testing.T) {
	t.Helper()
	withSelfUseModeEnabled(t)
	db := setupModelListControllerTestDB(t)
	// zz-priced-retired-model is a misconfiguration: retired, yet still priced.
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-live-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-priced-retired-model", ChannelId: 1, Enabled: true},
	}).Error)
	model.InvalidatePricingCache()
	t.Cleanup(model.InvalidatePricingCache)
}

func TestGetPricingListsRetiredModelsNewestFirst(t *testing.T) {
	setupPricingRetirementTest(t)
	withRetiredModels(t, `{
		"zz-old-model": {"retired_at": "2026-03-01", "successor": "zz-live-model"},
		"zz-priced-retired-model": {"retired_at": "2026-09-30", "successor": "zz-live-model", "notice_url": "https://example.com/notice", "note": "Replaced"},
		"zz-another-model": {"retired_at": "2026-09-30", "successor": "zz-live-model"}
	}`)

	var logs bytes.Buffer
	common.LogWriterMu.Lock()
	previousWriter := gin.DefaultWriter
	gin.DefaultWriter = &logs
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultWriter = previousWriter
		common.LogWriterMu.Unlock()
	})

	for range 2 {
		var payload pricingRetirementResponse
		require.NoError(t, common.Unmarshal(servePricing(t).Body.Bytes(), &payload))

		names := make([]string, 0, len(payload.Data))
		for _, item := range payload.Data {
			names = append(names, item.ModelName)
		}
		assert.Contains(t, names, "zz-live-model")
		assert.NotContains(t, names, "zz-priced-retired-model", "a retired model is shown as retired, not priced")
		assert.Equal(t, []pricingRetiredModel{
			{ModelName: "zz-another-model", RetiredAt: "2026-09-30", Successor: "zz-live-model"},
			{ModelName: "zz-priced-retired-model", RetiredAt: "2026-09-30", Successor: "zz-live-model", NoticeURL: "https://example.com/notice", Note: "Replaced"},
			{ModelName: "zz-old-model", RetiredAt: "2026-03-01", Successor: "zz-live-model"},
		}, payload.RetiredModels)
	}
	assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte(`retired model "zz-priced-retired-model" still has live pricing`)), logs.String())
}

func TestGetPricingRetiredModelsWithShowAllGroups(t *testing.T) {
	setupPricingRetirementTest(t)
	// svip has a group ratio but is not a group the anonymous viewer may use.
	require.NoError(t, model.DB.Create(&[]model.Ability{
		{Group: "svip", Model: "zz-svip-live-model", ChannelId: 1, Enabled: true},
		{Group: "svip", Model: "zz-svip-retired-model", ChannelId: 1, Enabled: true},
	}).Error)
	model.InvalidatePricingCache()
	withRetiredModels(t, `{
		"zz-priced-retired-model": {"retired_at": "2026-09-30", "successor": "zz-live-model"},
		"zz-svip-retired-model": {"retired_at": "2026-09-30", "successor": "zz-svip-live-model"}
	}`)
	setting := operation_setting.GetGeneralSetting()
	original := setting.PricingShowAllGroups
	t.Cleanup(func() { setting.PricingShowAllGroups = original })

	for _, showAll := range []bool{false, true} {
		setting.PricingShowAllGroups = showAll
		var payload pricingRetirementResponse
		require.NoError(t, common.Unmarshal(servePricing(t).Body.Bytes(), &payload))

		names := make([]string, 0, len(payload.Data))
		for _, item := range payload.Data {
			names = append(names, item.ModelName)
		}
		assert.Contains(t, names, "zz-live-model")
		if showAll {
			assert.Contains(t, names, "zz-svip-live-model", "the extra group's models are listed")
		} else {
			assert.NotContains(t, names, "zz-svip-live-model")
		}
		assert.NotContains(t, names, "zz-priced-retired-model")
		assert.NotContains(t, names, "zz-svip-retired-model", "a retired model is never priced, whatever the group")
		assert.Equal(t, []pricingRetiredModel{
			{ModelName: "zz-priced-retired-model", RetiredAt: "2026-09-30", Successor: "zz-live-model"},
			{ModelName: "zz-svip-retired-model", RetiredAt: "2026-09-30", Successor: "zz-svip-live-model"},
		}, payload.RetiredModels)
	}
}

func TestGetPricingUnchangedWithoutRetiredModels(t *testing.T) {
	setupPricingRetirementTest(t)
	require.Empty(t, operation_setting.GetModelRetirementsCopy())

	body := servePricing(t).Body.Bytes()
	assert.Equal(t, string(basePricingBody(t)), string(body))
	assert.NotContains(t, string(body), "retired_models")
	assert.Contains(t, string(body), "zz-priced-retired-model")
}
