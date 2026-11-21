package service

import (
	"math"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The measured TwelveLabs Pegasus 1.5 answer on Bedrock and its published
// prices: USD 7.50 per 1M output tokens and USD 0.00049 per second of video.
const (
	videoSecondsTestExpr  = `tier("base", c * 7.5 + vid_sec * 490)`
	videoSecondsTokenExpr = `tier("base", c * 7.5)`
)

func newVideoSecondsTestUsage() *dto.Usage {
	return &dto.Usage{PromptTokens: 108, CompletionTokens: 18, TotalTokens: 126, InputVideoSeconds: 2}
}

func newVideoSecondsTestRelayInfo(expr string, groupRatio float64) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: "twelvelabs.pegasus-1-5-v1:0",
		PriceData:       hosttypes.PriceData{GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: groupRatio}},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:   "tiered_expr",
			ExprString:    expr,
			ExprHash:      billingexpr.ExprHashString(expr),
			GroupRatio:    groupRatio,
			QuotaPerUnit:  500000,
			EstimatedTier: "base",
		},
	}
}

func TestBuildTieredTokenParamsCarriesVideoSecondsOutsideTokens(t *testing.T) {
	usedVars := billingexpr.UsedVars(videoSecondsTestExpr)
	params := BuildTieredTokenParams(newVideoSecondsTestUsage(), false, usedVars)
	assert.Equal(t, float64(2), params.VidSec)
	assert.Equal(t, float64(108), params.P)
	assert.Equal(t, float64(18), params.C)
	assert.Equal(t, float64(108), params.Len)

	for _, seconds := range []float64{-2, math.NaN(), math.Inf(1), math.Inf(-1)} {
		usage := newVideoSecondsTestUsage()
		usage.InputVideoSeconds = seconds
		assert.Zero(t, BuildTieredTokenParams(usage, false, usedVars).VidSec, "seconds=%v", seconds)
	}
}

func TestTryTieredSettleBillsMeasuredVideoAnswer(t *testing.T) {
	for _, test := range []struct {
		name       string
		expr       string
		groupRatio float64
		want       int
	}{
		// USD 0.001115 is exactly 557.5 quota at 500,000 a dollar, rounded half
		// away from zero.
		{"seconds priced, ratio 1", videoSecondsTestExpr, 1, 558},
		{"seconds priced, ratio 1.5", videoSecondsTestExpr, 1.5, 836},
		// USD 0.000135 is 67.5 quota.
		{"tokens only, ratio 1", videoSecondsTokenExpr, 1, 68},
	} {
		t.Run(test.name, func(t *testing.T) {
			info := newVideoSecondsTestRelayInfo(test.expr, test.groupRatio)
			params := BuildTieredTokenParams(newVideoSecondsTestUsage(), false, billingexpr.UsedVars(test.expr))
			ok, quota, result := TryTieredSettle(info, params)
			require.True(t, ok)
			require.NotNil(t, result, "the expression must evaluate, not fall back to the reservation")
			assert.Equal(t, test.want, quota)
		})
	}
}

func TestRatioPricingIgnoresVideoSeconds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		OriginModelName: "twelvelabs.pegasus-1-5-v1:0",
		PriceData: hosttypes.PriceData{
			ModelRatio: 3.75, CompletionRatio: 2,
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
		},
	}
	withSeconds := calculateTextQuotaSummary(ctx, info, newVideoSecondsTestUsage())
	// (108 + 18 * 2) * 3.75 = 540, as without seconds.
	assert.Equal(t, 540, withSeconds.Quota)

	videoOnly := calculateTextQuotaSummary(ctx, info, &dto.Usage{InputVideoSeconds: 2})
	assert.False(t, videoOnly.hasBillableUsage())
	assert.Zero(t, videoOnly.Quota)
}

func TestVideoSecondsAloneAreBillableWhenPriced(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	usage := &dto.Usage{InputVideoSeconds: 2}

	priced := calculateTextQuotaSummary(ctx, newVideoSecondsTestRelayInfo(videoSecondsTestExpr, 1), usage)
	assert.True(t, priced.hasBillableUsage())

	unpriced := calculateTextQuotaSummary(ctx, newVideoSecondsTestRelayInfo(videoSecondsTokenExpr, 1), usage)
	assert.False(t, unpriced.hasBillableUsage())
}

func TestEffectiveBillingUsageKeepsVideoSeconds(t *testing.T) {
	usage := &dto.Usage{
		InputVideoSeconds: 2,
		BillingUsage:      dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{InputTokens: 108, OutputTokens: 18}),
	}
	billingUsage := effectiveBillingUsage(usage)
	require.NotSame(t, usage, billingUsage, "the snapshot must be the billing usage")
	assert.Equal(t, 18, billingUsage.CompletionTokens)
	assert.Equal(t, float64(2), billingUsage.InputVideoSeconds)
}

func TestTieredLogRecordsVideoSecondsWhenPriced(t *testing.T) {
	for _, test := range []struct {
		name    string
		expr    string
		seconds any
	}{
		{"seconds priced", videoSecondsTestExpr, float64(2)},
		{"tokens only", videoSecondsTokenExpr, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			info := newVideoSecondsTestRelayInfo(test.expr, 1)
			params := BuildTieredTokenParams(newVideoSecondsTestUsage(), false, billingexpr.UsedVars(test.expr))
			_, _, result := TryTieredSettle(info, params)
			require.NotNil(t, result)

			other := model.NewLogOther()
			InjectTieredBillingInfo(other, info, result)
			logged := other.Snapshot()
			assert.NotContains(t, logged, "image_cache_tokens")
			if test.seconds == nil {
				assert.NotContains(t, logged, "billing_tokens")
				return
			}
			billingTokens, ok := logged["billing_tokens"].(map[string]float64)
			require.True(t, ok)
			assert.Equal(t, test.seconds, billingTokens["vid_sec"])
			assert.Equal(t, float64(18), billingTokens["c"])
		})
	}
}
