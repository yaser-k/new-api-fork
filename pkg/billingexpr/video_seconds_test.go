package billingexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The measured TwelveLabs Pegasus 1.5 answer: 18 output tokens and 2 seconds
// of input video, priced at USD 7.50 per 1M output tokens and USD 0.00049 per
// second (vid_sec * 490, since token expressions are divided by 1,000,000).
const videoSecondsExpr = `tier("base", c * 7.5 + vid_sec * 490)`

func TestVideoSecondsVariableCompilesAndPricesSeconds(t *testing.T) {
	_, err := CompileFromCache(videoSecondsExpr)
	require.NoError(t, err)
	assert.True(t, UsedVars(videoSecondsExpr)["vid_sec"])

	cost, trace, err := RunExpr(videoSecondsExpr, TokenParams{P: 108, C: 18, Len: 108, VidSec: 2})
	require.NoError(t, err)
	assert.Equal(t, float64(1115), cost)
	assert.Equal(t, "base", trace.MatchedTier)

	whole, _, err := RunExpr(`tier("base", ceil(vid_sec) * 490)`, TokenParams{VidSec: 1.2})
	require.NoError(t, err)
	assert.Equal(t, float64(980), whole)
}

func TestExpressionWithoutVideoSecondsIgnoresThem(t *testing.T) {
	const tokensOnly = `tier("base", c * 7.5)`
	assert.False(t, UsedVars(tokensOnly)["vid_sec"])
	for _, seconds := range []float64{0, 2, 7200} {
		cost, _, err := RunExpr(tokensOnly, TokenParams{P: 108, C: 18, Len: 108, VidSec: seconds})
		require.NoError(t, err)
		assert.Equal(t, float64(135), cost, "vid_sec=%g", seconds)
	}
}

func TestVideoSecondsSettlementQuotaAndBillingTokens(t *testing.T) {
	params := TokenParams{P: 108, C: 18, Len: 108, VidSec: 2}
	for _, test := range []struct {
		name          string
		expr          string
		groupRatio    float64
		beforeGroup   float64
		afterGroup    int
		billingTokens bool
	}{
		// USD 0.001115 is 557.5 quota at 500,000 a dollar; QuotaRound rounds the
		// exact half away from zero.
		{"seconds priced, ratio 1", videoSecondsExpr, 1, 557.5, 558, true},
		{"seconds priced, ratio 1.5", videoSecondsExpr, 1.5, 557.5, 836, true},
		{"tokens only, ratio 1", `tier("base", c * 7.5)`, 1, 67.5, 68, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			snap := &BillingSnapshot{
				BillingMode:  "tiered_expr",
				ExprString:   test.expr,
				ExprHash:     ExprHashString(test.expr),
				GroupRatio:   test.groupRatio,
				QuotaPerUnit: 500000,
			}
			result, err := ComputeTieredQuota(snap, params)
			require.NoError(t, err)
			assert.Equal(t, test.beforeGroup, result.ActualQuotaBeforeGroup)
			assert.Equal(t, test.afterGroup, result.ActualQuotaAfterGroup)
			if test.billingTokens {
				require.NotNil(t, result.BillingTokens)
				assert.Equal(t, float64(2), result.BillingTokens.VidSec)
			} else {
				assert.Nil(t, result.BillingTokens)
			}
		})
	}
}
