package controller

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionRejectsInvalidTaskBillingExpressions(t *testing.T) {
	const pluginKey = "billing-save-probe"
	const modelName = "billing-save-model"
	source := `
export const meta = {
  apiVersion: 1, key: "billing-save-probe", name: "Billing Save Probe", version: "1.0.0", author: {name: "Test"},
  models: ["billing-save-model"], fetchMode: "per_task",
  usageSchema: {seconds: {type: "number", unit: "second"}}
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {}; }
`
	_, err := jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(pluginKey) })

	tests := []struct {
		name       string
		expression string
		errorText  string
	}{
		{
			name:       "invalid syntax",
			expression: `tier("base",`,
			errorText:  "expr compile error",
		},
		{
			name:       "undeclared usage key",
			expression: `tier("base", u("clips") * 0.1)`,
			errorText:  `usage key \"clips\" is not declared`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			expressions, marshalErr := common.Marshal(map[string]string{modelName: testCase.expression})
			require.NoError(t, marshalErr)
			body, marshalErr := common.Marshal(OptionUpdateRequest{
				Key:   "billing_setting.billing_expr",
				Value: string(expressions),
			})
			require.NoError(t, marshalErr)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(string(body)))

			UpdateOption(context)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Contains(t, recorder.Body.String(), `"success":false`)
			assert.Contains(t, recorder.Body.String(), modelName)
			assert.Contains(t, recorder.Body.String(), testCase.errorText)
		})
	}
}

func TestUpdateOptionRejectsUsageExpressionWithoutTaskPlugin(t *testing.T) {
	const modelName = "billing-save-model-without-plugin"
	expressions, err := common.Marshal(map[string]string{
		modelName: `u("mode") == "std" ? 1 : 2`,
	})
	require.NoError(t, err)
	body, err := common.Marshal(OptionUpdateRequest{
		Key:   "billing_setting.billing_expr",
		Value: string(expressions),
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/option/",
		strings.NewReader(string(body)),
	)

	UpdateOption(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), modelName)
	assert.Contains(t, recorder.Body.String(), "mode")
	assert.Contains(t, recorder.Body.String(), "no task plugin usage schema")
}

func TestUpdateOptionAliasBillingExprUsesPluginSchema(t *testing.T) {
	database := modelManagementDB(t, "sqlite", "")
	require.NoError(t, database.AutoMigrate(&model.Log{}))
	const pluginKey = "billing-alias-probe"
	source := `
export const meta = {
  apiVersion: 1, key: "billing-alias-probe", name: "Billing Alias Probe", version: "1.0.0", author: {name: "Test"},
  models: ["declared-model"], fetchMode: "per_task",
  usageSchema: {seconds: {type: "number", unit: "second"}, image_count: {type: "number", unit: "count"}},
  usageProfiles: [{models: ["declared-model"], schema: {seconds: {type: "number", unit: "second"}}}]
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {}; }
`
	_, err := jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(pluginKey) })

	mapping := `{"alias-model":"declared-model"}`
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:           1,
		Type:         54,
		Key:          "key-1",
		Status:       common.ChannelStatusEnabled,
		Name:         "ch-1",
		Group:        "default",
		Models:       "alias-model,declared-model",
		ModelMapping: &mapping,
	}).Error)
	model.InitChannelCache()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	putExpr := func(modelName, expression string) *httptest.ResponseRecorder {
		t.Helper()
		expressions, marshalErr := common.Marshal(map[string]string{modelName: expression})
		require.NoError(t, marshalErr)
		body, marshalErr := common.Marshal(OptionUpdateRequest{
			Key:   "billing_setting.billing_expr",
			Value: string(expressions),
		})
		require.NoError(t, marshalErr)
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(string(body)))
		UpdateOption(context)
		return recorder
	}

	accepted := putExpr("alias-model", `u("seconds")`)
	assert.Equal(t, http.StatusOK, accepted.Code)
	assert.Contains(t, accepted.Body.String(), `"success":true`)

	rejectedKey := putExpr("alias-model", `u("image_count")`)
	assert.Equal(t, http.StatusOK, rejectedKey.Code)
	assert.Contains(t, rejectedKey.Body.String(), `"success":false`)
	assert.Contains(t, rejectedKey.Body.String(), `usage key \"image_count\" is not declared`)

	rejectedDeclared := putExpr("declared-model", `u("image_count")`)
	assert.Contains(t, rejectedDeclared.Body.String(), `"success":false`)
	assert.Contains(t, rejectedDeclared.Body.String(), `usage key \"image_count\" is not declared`)

	unresolvable := putExpr("unknown-alias-model", `u("seconds")`)
	assert.Equal(t, http.StatusOK, unresolvable.Code)
	assert.Contains(t, unresolvable.Body.String(), `"success":false`)
	assert.Contains(t, unresolvable.Body.String(), "no task plugin usage schema")
}

// Covers configuration persistence, validation, reservation and reconciliation
// through the same public entry points used by the relay.
func TestPreConsumePolicyDatabaseMatrix(t *testing.T) {
	previousConfig := config.GlobalConfig.ExportAllConfigs()
	previousUnit, previousBatch := common.QuotaPerUnit, common.BatchUpdateEnabled
	common.QuotaPerUnit, common.BatchUpdateEnabled = 500000, false
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(previousConfig))
		common.QuotaPerUnit, common.BatchUpdateEnabled = previousUnit, previousBatch
	})
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			require.NoError(t, db.AutoMigrate(&model.Token{}))
			for key, value := range map[string]string{
				"quota_setting.trust_quota_usd":        "10.5",
				"quota_setting.pre_consume_multiplier": "1.5",
			} {
				var response struct{ Success bool }
				modelManagementRequest(t, UpdateOption, http.MethodPut, "/api/option/", OptionUpdateRequest{Key: key, Value: value}, &response)
				require.True(t, response.Success)
				var saved model.Option
				require.NoError(t, db.Where(&model.Option{Key: key}).First(&saved).Error)
				assert.Equal(t, value, saved.Value)
			}
			var savedOptions []model.Option
			require.NoError(t, db.Find(&savedOptions).Error)
			stored := map[string]string{}
			for _, option := range savedOptions {
				stored[option.Key] = option.Value
			}
			for range 2 {
				operation_setting.GetQuotaSetting().TrustQuotaUSD = 10
				operation_setting.GetQuotaSetting().PreConsumeMultiplier = 1
				require.NoError(t, config.GlobalConfig.LoadFromDB(stored))
				assert.Equal(t, 10.5, operation_setting.GetQuotaSetting().TrustQuotaUSD)
				assert.Equal(t, 1.5, operation_setting.GetQuotaSetting().PreConsumeMultiplier)
			}
			for _, tc := range []struct{ key, value string }{
				{"trust_quota_usd", "-1"}, {"trust_quota_usd", "NaN"}, {"trust_quota_usd", "+Inf"},
				{"pre_consume_multiplier", "0"}, {"pre_consume_multiplier", "-0.5"},
				{"pre_consume_multiplier", "NaN"}, {"pre_consume_multiplier", "Inf"},
				{"pre_consume_multiplier", "1e309"}, {"pre_consume_multiplier", ""},
			} {
				key := "quota_setting." + tc.key
				var response struct{ Success bool }
				modelManagementRequest(t, UpdateOption, http.MethodPut, "/api/option/", OptionUpdateRequest{Key: key, Value: tc.value}, &response)
				assert.False(t, response.Success, "%s=%q", key, tc.value)
				var saved model.Option
				require.NoError(t, db.Where(&model.Option{Key: key}).First(&saved).Error)
				assert.Equal(t, stored[key], saved.Value, "invalid saves must preserve the previous value")
			}
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				"billing_setting.billing_mode":    `{"policy-model":"tiered_expr"}`,
				"billing_setting.billing_expr":    `{"policy-model":"tier(\"base\", p * 3 + c * 15)"}`,
				"group_ratio_setting.group_ratio": `{"default":1}`,
			}))
			cases := []struct {
				name                  string
				threshold, multiplier float64
				wallet, token         int
				force, unlimited      bool
				wantHeld              int
			}{
				{"above threshold", 10, 1, 5500000, 5500000, false, false, 0},
				{"custom threshold", 20, 1, 5500000, 5500000, false, false, 1500},
				{"zero disables bypass", 0, 1, 5500000, 5500000, false, false, 1500},
				{"wallet equals threshold", 10, 1, 5000000, 5500000, false, false, 1500},
				{"token equals threshold", 10, 1, 5500000, 5000000, false, false, 1500},
				{"fractional threshold", 10.5, 1, 5250001, 5250001, false, false, 0},
				{"unlimited token", 10, 1, 5500000, 0, false, true, 0},
				{"forced reservation", 10, 1, 5500000, 5500000, true, false, 1500},
				{"half input cost", 0, 0.5, 100000, 100000, false, false, 750},
				{"fractional multiple refunds excess", 0, 2.5, 100000, 100000, false, false, 3750},
			}
			for i, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					require.NoError(t, model.UpdateOption("quota_setting.trust_quota_usd", strconv.FormatFloat(tc.threshold, 'f', -1, 64)))
					require.NoError(t, model.UpdateOption("quota_setting.pre_consume_multiplier", strconv.FormatFloat(tc.multiplier, 'f', -1, 64)))
					user := model.User{Username: fmt.Sprintf("policy-user-%d", i), Quota: tc.wallet, Group: "default", AffCode: fmt.Sprintf("policy-aff-%d", i)}
					require.NoError(t, db.Create(&user).Error)
					token := model.Token{UserId: user.Id, Key: fmt.Sprintf("policy-token-%d", i), RemainQuota: tc.token, UnlimitedQuota: tc.unlimited}
					require.NoError(t, db.Create(&token).Error)
					ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
					ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
					ctx.Set("token_quota", tc.token)
					info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id, TokenKey: token.Key, TokenUnlimited: tc.unlimited, ForcePreConsume: tc.force, OriginModelName: "policy-model", UserGroup: "default", UsingGroup: "default", BillingRequestInput: &billingexpr.RequestInput{}}
					info.UserSetting.BillingPreference = "wallet_only"
					price, err := helper.ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{MaxTokens: 10000})
					require.NoError(t, err)
					require.Nil(t, service.PreConsumeBilling(ctx, price.QuotaToPreConsume, info))
					assert.Equal(t, tc.wantHeld, info.FinalPreConsumedQuota)
					require.NoError(t, db.First(&user, user.Id).Error)
					assert.Equal(t, tc.wallet-tc.wantHeld, user.Quota)
					if !tc.unlimited {
						require.NoError(t, db.First(&token, token.Id).Error)
						assert.Equal(t, tc.token-tc.wantHeld, token.RemainQuota)
					}
					_, actual, _ := service.TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 100, Len: 1000})
					assert.Equal(t, 2250, actual)
					require.NoError(t, info.Billing.Settle(actual))
					require.NoError(t, info.Billing.Settle(actual))
					require.NoError(t, db.First(&user, user.Id).Error)
					assert.Equal(t, tc.wallet-actual, user.Quota)
					if !tc.unlimited {
						require.NoError(t, db.First(&token, token.Id).Error)
						assert.Equal(t, tc.token-actual, token.RemainQuota)
					}
				})
			}
		})
	}
}

func TestPreConsumeMultiplierRejectsInvalidRuntimeAndOverflow(t *testing.T) {
	previous := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(previous)) })
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":    `{"policy-overflow":"tiered_expr"}`,
		"billing_setting.billing_expr":    `{"policy-overflow":"tier(\"base\", p * 3)"}`,
		"group_ratio_setting.group_ratio": `{"default":1}`,
	}))
	for _, multiplier := range []float64{0, -0.5, math.NaN(), math.Inf(1), math.MaxFloat64} {
		operation_setting.GetQuotaSetting().PreConsumeMultiplier = multiplier
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		info := &relaycommon.RelayInfo{OriginModelName: "policy-overflow", UserGroup: "default", UsingGroup: "default", BillingRequestInput: &billingexpr.RequestInput{}}
		_, err := helper.ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
		require.Error(t, err)
		assert.Nil(t, info.Billing)
	}
}

// A model's public price expression is saved through the same option endpoint
// and per-model pricing API as its billing expression. A save that does not
// compile, reads the request, or leaves the model without a billing expression
// is refused with a message naming the model, and stores nothing.
func TestUpdateOptionValidatesPublicPriceExpressions(t *testing.T) {
	db := modelManagementDB(t, "sqlite", "")
	previousConfig := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(previousConfig)) })
	const publicKey = "billing_setting.public_expr"
	put := func(key string, value map[string]string) (bool, string) {
		t.Helper()
		encoded, err := common.Marshal(value)
		require.NoError(t, err)
		var response struct {
			Success bool
			Message string
		}
		modelManagementRequest(t, UpdateOption, http.MethodPut, "/api/option/", OptionUpdateRequest{Key: key, Value: string(encoded)}, &response)
		return response.Success, response.Message
	}
	stored := func() string {
		t.Helper()
		var options []model.Option
		require.NoError(t, db.Where(&model.Option{Key: publicKey}).Find(&options).Error)
		if len(options) == 0 {
			return ""
		}
		return options[0].Value
	}
	const billing = `param("route") == "b" ? tier("route_b", p * 4 + c * 20) : tier("base", p * 3 + c * 15)`
	const public = `tier("public", p * 4 + c * 20)`
	ok, message := put("billing_setting.billing_expr", map[string]string{"public-model": billing})
	require.True(t, ok, message)
	ok, message = put("billing_setting.billing_mode", map[string]string{"public-model": "tiered_expr"})
	require.True(t, ok, message)

	for _, tc := range []struct{ name, model, expression, reason string }{
		{"does not compile", "public-model", `tier("public",`, "compile"},
		{"is empty", "public-model", " ", "required"},
		{"reads a request parameter", "public-model", `(tier("public", p * 4 + c * 20)) * (param("route") == "b" ? 2 : 1)`, "request"},
		{"reads a request header", "public-model", `has(header("x-route"), "b") ? tier("b", p * 4) : tier("a", p * 3)`, "request"},
		{"reads the time", "public-model", `hour("UTC") < 8 ? tier("night", p * 2) : tier("day", p * 4)`, "request"},
		{"cannot price the model", "public-model", `tier("public", p * 4 - c * 20)`, "non-negative"},
		{"has no billing expression", "unpriced-model", public, "billing expression"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok, message := put(publicKey, map[string]string{tc.model: tc.expression})
			assert.False(t, ok, "the save must be refused")
			assert.Contains(t, message, tc.model)
			assert.Contains(t, message, tc.reason)
			assert.NotContains(t, stored(), tc.model)
			assert.NotContains(t, config.GlobalConfig.ExportAllConfigs()[publicKey], tc.model)
		})
	}

	ok, message = put(publicKey, map[string]string{"public-model": public})
	require.True(t, ok, message)
	want, err := common.Marshal(map[string]string{"public-model": public})
	require.NoError(t, err)
	assert.JSONEq(t, string(want), stored())
	assert.JSONEq(t, string(want), config.GlobalConfig.ExportAllConfigs()[publicKey])

	ok, message = put("billing_setting.billing_expr", map[string]string{})
	assert.False(t, ok, "removing the billing expression behind a public one must be refused")
	assert.Contains(t, message, "public-model")

	var snapshot struct {
		Success bool
		Data    model.ModelPricingSnapshot
	}
	modelManagementRequest(t, GetModelPricingConfig, http.MethodGet, "/api/option/model_pricing?model=public-model", nil, &snapshot)
	require.True(t, snapshot.Success)
	require.Len(t, snapshot.Data.Entries, 1)
	entry := snapshot.Data.Entries[0]
	assert.Equal(t, public, entry.Configured[publicKey])
	assert.Equal(t, billing, entry.Configured["billing_setting.billing_expr"])

	patch := func(pricing model.PricingValues) (bool, string) {
		t.Helper()
		var response struct {
			Success bool
			Message string
		}
		modelManagementRequest(t, UpdateModelPricingConfig, http.MethodPatch, "/api/option/model_pricing", gin.H{
			"changes": []model.ModelPricingChange{{ModelName: "public-model", ExpectedVersion: entry.Version, Pricing: pricing}},
		}, &response)
		return response.Success, response.Message
	}
	draft := model.PricingValues{}
	for key, value := range entry.Configured {
		draft[key] = value
	}
	draft[publicKey] = `day("UTC") == 1 ? tier("monday", p * 2) : tier("base", p * 4)`
	ok, message = patch(draft)
	assert.False(t, ok, "a draft whose public expression reads the request must be refused")
	assert.Contains(t, message, "public-model")
	assert.JSONEq(t, string(want), stored())

	delete(draft, publicKey)
	ok, message = patch(draft)
	require.True(t, ok, message)
	assert.JSONEq(t, `{}`, stored())
}

// Pricing rows of the given caller by model name. role 0 is a signed-out caller.
func pricingRowsFor(t *testing.T, role, userID int) (map[string]string, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	if role != 0 {
		context.Set("role", role)
		context.Set("id", userID)
	}
	GetPricing(context)
	var response struct {
		Success bool
		Data    []json.RawMessage
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response), recorder.Body.String())
	require.True(t, response.Success)
	// The pricing version marks whichever row comes first; it is not a price.
	version := regexp.MustCompile(`,"pricing_version":"[0-9a-f]+"`)
	rows := make(map[string]string, len(response.Data))
	for _, raw := range response.Data {
		var row struct {
			ModelName string `json:"model_name"`
		}
		require.NoError(t, common.Unmarshal(raw, &row))
		rows[row.ModelName] = version.ReplaceAllString(string(raw), "")
	}
	return rows, recorder.Body.String()
}

// Everyone who is not an administrator sees a model's public price expression
// wherever its billing expression would be returned, and nothing of the
// billing expression; an administrator sees both. Models without a public
// expression are returned exactly as before, to everyone.
func TestPricingShowsPublicPriceExpressionsToNonAdministrators(t *testing.T) {
	db := modelManagementDB(t, "sqlite", "")
	previousConfig := config.GlobalConfig.ExportAllConfigs()
	previousExpose := ratio_setting.IsExposeRatioEnabled()
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(previousConfig))
		ratio_setting.SetExposeRatioEnabled(previousExpose)
		model.InvalidatePricingCache()
	})
	for _, key := range []string{"public-alpha", "public-beta"} {
		source := fmt.Sprintf(`
export const meta = {
  apiVersion: 1, key: %q, name: %q, version: "1.0.0", author: {name: "Test"},
  models: ["public-task-model"], fetchMode: "per_task",
  usageSchema: {seconds: {type: "number", unit: "second"}}
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {}; }
`, key, key)
		_, err := jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
		require.NoError(t, err)
		t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(key) })
	}
	require.NoError(t, db.Create(&model.Channel{Id: 1, Type: constant.ChannelTypeOpenAI, Key: "key-1", Status: common.ChannelStatusEnabled, Name: "ch-1"}).Error)
	require.NoError(t, db.Create(&model.Channel{Id: 2, Type: constant.ChannelTypeTaskPlugin, Key: "key-2", Status: common.ChannelStatusEnabled, Name: "ch-2"}).Error)
	for _, ability := range []model.Ability{
		{Group: "default", Model: "public-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "ratio-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "public-task-model", ChannelId: 2, Enabled: true},
	} {
		require.NoError(t, db.Create(&ability).Error)
	}
	user := model.User{Username: "public-price-user", Group: "default", AffCode: "public-price-aff", Role: common.RoleCommonUser}
	require.NoError(t, db.Create(&user).Error)
	admin := model.User{Username: "public-price-admin", Group: "default", AffCode: "public-price-admin-aff", Role: common.RoleAdminUser}
	require.NoError(t, db.Create(&admin).Error)
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"ratio-model":1.5}`))

	const billing = `param("route") == "b" ? tier("route_b", p * 4 + c * 20) : tier("base", p * 3 + c * 15)`
	const public = `tier("public", p * 4 + c * 20)`
	const taskBilling = `param("route") == "b" ? tier("route_b", u("seconds") * 0.6) : tier("base", u("seconds") * 0.4)`
	const taskPublic = `tier("public", u("seconds") * 0.6)`
	expressions, err := common.Marshal(map[string]string{
		"public-model": billing, "expr-model": `tier("base", p * 2 + c * 8)`, "public-task-model": taskBilling,
	})
	require.NoError(t, err)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":          `{"public-model":"tiered_expr","expr-model":"tiered_expr","public-task-model":"tiered_expr"}`,
		"billing_setting.billing_expr":          string(expressions),
		billing_setting.PluginBillingExprOption: `{}`,
		"billing_setting.public_expr":           `{}`,
	}))
	ratio_setting.SetExposeRatioEnabled(true)
	model.InvalidatePricingCache()

	callers := []struct {
		name     string
		role, id int
	}{{"signed out", 0, 0}, {"ordinary user", common.RoleCommonUser, user.Id}, {"administrator", common.RoleAdminUser, admin.Id}}
	before := make(map[string]map[string]string)
	for _, caller := range callers {
		before[caller.name], _ = pricingRowsFor(t, caller.role, caller.id)
	}

	publicExpressions, err := common.Marshal(map[string]string{"public-model": public, "public-task-model": taskPublic})
	require.NoError(t, err)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"billing_setting.public_expr": string(publicExpressions)}))
	model.InvalidatePricingCache()

	type pricingRow struct {
		BillingExpr string  `json:"billing_expr"`
		PublicExpr  *string `json:"public_expr"`
		Variants    []struct {
			PluginKey   string `json:"plugin_key"`
			BillingExpr string `json:"billing_expr"`
		} `json:"billing_plugin_variants"`
	}
	for _, caller := range callers {
		t.Run(caller.name, func(t *testing.T) {
			rows, body := pricingRowsFor(t, caller.role, caller.id)
			for _, name := range []string{"expr-model", "ratio-model"} {
				require.Contains(t, rows, name)
				assert.Equal(t, before[caller.name][name], rows[name], "%s must be returned exactly as before", name)
			}
			var priced, task pricingRow
			require.NoError(t, common.UnmarshalJsonStr(rows["public-model"], &priced))
			require.NoError(t, common.UnmarshalJsonStr(rows["public-task-model"], &task))
			require.Len(t, task.Variants, 2)
			if caller.role >= common.RoleAdminUser {
				assert.Equal(t, before[caller.name]["public-model"], strings.Replace(rows["public-model"], `,"public_expr":`+strconv.Quote(public), "", 1))
				assert.Equal(t, billing, priced.BillingExpr)
				require.NotNil(t, priced.PublicExpr)
				assert.Equal(t, public, *priced.PublicExpr)
				assert.Equal(t, taskBilling, task.Variants[0].BillingExpr)
				assert.Equal(t, taskBilling, task.Variants[1].BillingExpr)
				require.NotNil(t, task.PublicExpr)
				assert.Equal(t, taskPublic, *task.PublicExpr)
				return
			}
			assert.Equal(t, public, priced.BillingExpr)
			assert.Nil(t, priced.PublicExpr)
			assert.Equal(t, taskPublic, task.BillingExpr)
			assert.Equal(t, taskPublic, task.Variants[0].BillingExpr)
			assert.Equal(t, taskPublic, task.Variants[1].BillingExpr)
			assert.Nil(t, task.PublicExpr)
			assert.NotContains(t, body, "route_b")
			assert.NotContains(t, body, "param(")
		})
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/ratio_config", nil)
	GetRatioConfig(context)
	var ratioConfig struct {
		Success bool
		Data    struct {
			BillingExpr map[string]string `json:"billing_expr"`
		}
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &ratioConfig), recorder.Body.String())
	require.True(t, ratioConfig.Success)
	assert.Equal(t, public, ratioConfig.Data.BillingExpr["public-model"])
	assert.Equal(t, taskPublic, ratioConfig.Data.BillingExpr["public-task-model"])
	assert.Equal(t, `tier("base", p * 2 + c * 8)`, ratioConfig.Data.BillingExpr["expr-model"])
	assert.NotContains(t, recorder.Body.String(), "route_b")
}

// Billing never reads the public price expression: reservation and settlement
// of the same requests give the same quota, and settle and log the billing
// expression, with and without a public expression set.
func TestPublicPriceExpressionLeavesBillingUnchanged(t *testing.T) {
	previousConfig := config.GlobalConfig.ExportAllConfigs()
	previousUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(previousConfig))
		common.QuotaPerUnit = previousUnit
	})
	const billing = `param("route") == "b" ? tier("route_b", p * 4 + c * 20) : tier("base", p * 3 + c * 15)`
	const public = `tier("public", p * 8 + c * 40)`
	encoded, err := common.Marshal(map[string]string{"public-billing-model": billing})
	require.NoError(t, err)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":         `{"public-billing-model":"tiered_expr"}`,
		"billing_setting.billing_expr":         string(encoded),
		"billing_setting.public_expr":          `{}`,
		"group_ratio_setting.group_ratio":      `{"default":1}`,
		"quota_setting.pre_consume_multiplier": "1",
	}))
	usage := billingexpr.TokenParams{P: 1000, C: 100, Len: 1000}
	type charge struct {
		reserved, settled int
		tier, expression  string
	}
	chargeFor := func(body string) charge {
		t.Helper()
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		info := &relaycommon.RelayInfo{OriginModelName: "public-billing-model", UserGroup: "default", UsingGroup: "default",
			BillingRequestInput: &billingexpr.RequestInput{Body: []byte(body)}}
		price, err := helper.ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
		require.NoError(t, err)
		ok, settled, result := service.TryTieredSettle(info, usage)
		require.True(t, ok)
		require.NotNil(t, result)
		return charge{price.QuotaToPreConsume, settled, result.MatchedTier, info.TieredBillingSnapshot.ExprString}
	}
	bodies := []string{`{}`, `{"route":"b"}`}
	without := make([]charge, len(bodies))
	for i, body := range bodies {
		without[i] = chargeFor(body)
	}
	assert.Equal(t, charge{1500, 2250, "base", billing}, without[0])
	assert.Equal(t, charge{2000, 3000, "route_b", billing}, without[1])

	publicExpressions, err := common.Marshal(map[string]string{"public-billing-model": public})
	require.NoError(t, err)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"billing_setting.public_expr": string(publicExpressions)}))
	require.JSONEq(t, string(publicExpressions), config.GlobalConfig.ExportAllConfigs()["billing_setting.public_expr"], "the public expression must be in effect")
	publicCost, _, err := billingexpr.RunExpr(public, usage)
	require.NoError(t, err)
	billingCost, _, err := billingexpr.RunExpr(billing, usage)
	require.NoError(t, err)
	require.NotEqual(t, billingCost, publicCost, "the public expression must price differently for the test to tell them apart")
	for i, body := range bodies {
		assert.Equal(t, without[i], chargeFor(body), "request %s", body)
	}
}
