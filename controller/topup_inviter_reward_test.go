package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTopUpInfoTellsWhetherInvitingEarnsARewardWithoutTheAmount(t *testing.T) {
	paymentSetting := operation_setting.GetPaymentSetting()
	oldConfirmed := paymentSetting.ComplianceConfirmed
	oldTermsVersion := paymentSetting.ComplianceTermsVersion
	oldQuotaForInviter := common.QuotaForInviter
	t.Cleanup(func() {
		paymentSetting.ComplianceConfirmed = oldConfirmed
		paymentSetting.ComplianceTermsVersion = oldTermsVersion
		common.QuotaForInviter = oldQuotaForInviter
	})

	testCases := []struct {
		name             string
		quotaForInviter  int
		complianceIsDone bool
		wantEnabled      bool
	}{
		{name: "reward set and compliance confirmed", quotaForInviter: 731_517, complianceIsDone: true, wantEnabled: true},
		{name: "no reward set", quotaForInviter: 0, complianceIsDone: true, wantEnabled: false},
		{name: "reward set but compliance not confirmed, so none is paid", quotaForInviter: 731_517, complianceIsDone: false, wantEnabled: false},
	}

	gin.SetMode(gin.TestMode)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			common.QuotaForInviter = tc.quotaForInviter
			paymentSetting.ComplianceConfirmed = tc.complianceIsDone
			paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)
			GetTopUpInfo(ctx)

			require.Equal(t, http.StatusOK, recorder.Code)
			var body struct {
				Success bool           `json:"success"`
				Data    map[string]any `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
			require.True(t, body.Success)
			assert.Equal(t, tc.wantEnabled, body.Data["inviter_reward_enabled"])
			if tc.quotaForInviter > 0 {
				assert.NotContains(t, recorder.Body.String(), strconv.Itoa(tc.quotaForInviter))
			}
		})
	}
}
