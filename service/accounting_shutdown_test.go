package service

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatedFunding holds a refund until release is closed, standing in for a
// refund that is still running when shutdown starts.
type gatedFunding struct {
	FundingSource
	release chan struct{}
}

func (f *gatedFunding) Refund() error {
	<-f.release
	return f.FundingSource.Refund()
}

func setShutdownAccountingTestState(t *testing.T, batch bool) {
	t.Helper()
	truncate(t)
	oldBatch, oldInterval := common.BatchUpdateEnabled, common.BatchUpdateInterval
	common.BatchUpdateEnabled = batch
	// The periodic loop must not write before the shutdown flush does.
	common.BatchUpdateInterval = 3600
	if batch {
		model.InitBatchUpdater()
	}
	t.Cleanup(func() {
		model.StopBatchUpdater()
		model.FlushBatchUpdates()
		common.BatchUpdateEnabled, common.BatchUpdateInterval = oldBatch, oldInterval
	})
}

func startGatedRefund(t *testing.T, userID, tokenID int, tokenKey string, quota int) chan struct{} {
	t.Helper()
	release := make(chan struct{})
	session := &BillingSession{
		relayInfo:     &relaycommon.RelayInfo{UserId: userID, TokenId: tokenID, TokenKey: tokenKey},
		funding:       &gatedFunding{FundingSource: &WalletFunding{userId: userID, consumed: quota}, release: release},
		tokenConsumed: quota,
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	session.Refund(c)
	return release
}

func TestShutdownAccountingWritesRefundIssuedDuringShutdown(t *testing.T) {
	tests := []struct {
		name  string
		batch bool
	}{
		{name: "batch on", batch: true},
		{name: "batch off", batch: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setShutdownAccountingTestState(t, tt.batch)
			seedUser(t, 801, 900)
			seedToken(t, 802, 801, "shutdown-refund-key", 400)

			release := startGatedRefund(t, 801, 802, "shutdown-refund-key", 100)
			time.AfterFunc(10*time.Millisecond, func() { close(release) })
			require.NoError(t, ShutdownAccounting(time.Minute))

			var user model.User
			require.NoError(t, model.DB.Select("quota").First(&user, 801).Error)
			assert.Equal(t, 1000, user.Quota)
			var token model.Token
			require.NoError(t, model.DB.Select("remain_quota").First(&token, 802).Error)
			assert.Equal(t, 500, token.RemainQuota)
		})
	}
}

func TestShutdownAccountingGivesUpAfterTimeoutAndStillFlushes(t *testing.T) {
	setShutdownAccountingTestState(t, true)
	seedUser(t, 811, 1000)
	seedToken(t, 812, 811, "shutdown-timeout-key", 400)
	require.NoError(t, model.DecreaseUserQuota(811, 300, false))

	var logs bytes.Buffer
	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logs
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	release := startGatedRefund(t, 811, 812, "shutdown-timeout-key", 50)
	err := ShutdownAccounting(20 * time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 still running")
	common.LogWriterMu.Lock()
	assert.Contains(t, logs.String(), "gave up waiting for async billing jobs")
	common.LogWriterMu.Unlock()

	var user model.User
	require.NoError(t, model.DB.Select("quota").First(&user, 811).Error)
	assert.Equal(t, 700, user.Quota, "deltas queued before the timeout are still flushed")

	close(release)
	require.NoError(t, ShutdownAccounting(time.Minute))
}
