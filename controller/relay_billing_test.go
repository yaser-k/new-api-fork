package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// panicOnFirstWrite stands for any panic raised while the relay answers the
// client: after pre-consume and before settlement.
type panicOnFirstWrite struct {
	gin.ResponseWriter
	panicked bool
}

func (w *panicOnFirstWrite) Write(data []byte) (int, error) {
	if !w.panicked {
		w.panicked = true
		panic("client writer failed")
	}
	return w.ResponseWriter.Write(data)
}

func (w *panicOnFirstWrite) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

func TestRelayReleasesPreConsumedQuotaOnFailureAndPanic(t *testing.T) {
	for _, tc := range []struct {
		name           string
		upstreamStatus int
		panicOnWrite   bool
		wantStatus     int
		wantBody       string
		wantCharged    int
		wantRefunds    int
	}{
		{name: "success settles", upstreamStatus: http.StatusOK, wantStatus: http.StatusOK, wantBody: `"OK"`, wantCharged: 1000},
		{name: "failure refunds", upstreamStatus: http.StatusBadRequest, wantStatus: http.StatusBadRequest, wantBody: "upstream rejected", wantRefunds: 1},
		{name: "panic refunds and reaches the recovery middleware", upstreamStatus: http.StatusOK, panicOnWrite: true, wantStatus: http.StatusInternalServerError, wantBody: "Panic detected, error: client writer failed", wantRefunds: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user, token := setupResponsesWSRequestTest(t)
			previousBatch, previousLogs, previousQuotaPerUnit := common.BatchUpdateEnabled, common.LogConsumeEnabled, common.QuotaPerUnit
			common.BatchUpdateEnabled, common.LogConsumeEnabled, common.QuotaPerUnit = false, true, 500000
			previousPrices := ratio_setting.ModelPrice2JSONString()
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"panic-refund":0.002}`))
			t.Cleanup(func() {
				common.BatchUpdateEnabled, common.LogConsumeEnabled, common.QuotaPerUnit = previousBatch, previousLogs, previousQuotaPerUnit
				require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrices))
			})
			service.InitHttpClient()
			require.NoError(t, model.DB.AutoMigrate(&model.Channel{}, &model.Ability{}))
			require.NoError(t, model.LOG_DB.AutoMigrate(&model.Log{}))
			require.NoError(t, model.DB.Model(user).Updates(map[string]any{"quota": 100000, "setting": `{"billing_preference":"wallet_only"}`}).Error)
			require.NoError(t, model.DB.Model(token).Update("remain_quota", 3000).Error)

			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.upstreamStatus)
				if tc.upstreamStatus != http.StatusOK {
					_, _ = io.WriteString(w, `{"error":{"message":"upstream rejected","type":"invalid_request_error","code":"rejected"}}`)
					return
				}
				_, _ = io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"panic-refund","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`)
			}))
			t.Cleanup(upstream.Close)
			channel := &model.Channel{Name: "panic-refund-upstream", Key: "upstream-key", Status: common.ChannelStatusEnabled, Type: constant.ChannelTypeOpenAI, Group: "default", Models: "panic-refund", BaseURL: &upstream.URL}
			require.NoError(t, model.DB.Create(channel).Error)
			require.NoError(t, model.DB.Create(&model.Ability{ChannelId: channel.Id, Model: "panic-refund", Group: "default", Enabled: true}).Error)

			// Refund is asynchronous. Count committed refunds and wait on them
			// instead of a fixed delay.
			refunds := make(chan string, 8)
			require.NoError(t, model.DB.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("relay-panic-refund", func(tx *gorm.DB) {
				sql := tx.Statement.SQL.String()
				if tx.Error == nil && (strings.Contains(sql, "=remain_quota + ") || strings.Contains(sql, "=quota + ")) {
					refunds <- tx.Statement.Table
				}
			}))

			engine := gin.New()
			engine.Use(middleware.RelayPanicRecover())
			engine.POST("/v1/chat/completions", middleware.TokenAuth(), middleware.Distribute(), func(c *gin.Context) {
				if tc.panicOnWrite {
					c.Writer = &panicOnFirstWrite{ResponseWriter: c.Writer}
				}
				Relay(c, types.RelayFormatOpenAI)
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"panic-refund","messages":[{"role":"user","content":"Say OK"}]}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			engine.ServeHTTP(recorder, request)

			assert.Equal(t, tc.wantStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tc.wantBody)
			refunded := map[string]int{}
			for range tc.wantRefunds * 2 {
				select {
				case table := <-refunds:
					refunded[table]++
				case <-time.After(3 * time.Second):
					t.Fatalf("the pre-consumed quota was not released, refunds so far: %v", refunded)
				}
			}
			if tc.wantRefunds > 0 {
				assert.Equal(t, map[string]int{"tokens": tc.wantRefunds, "users": tc.wantRefunds}, refunded)
			}
			require.NoError(t, model.DB.First(token, token.Id).Error)
			require.NoError(t, model.DB.First(user, user.Id).Error)
			assert.Equal(t, 3000-tc.wantCharged, token.RemainQuota)
			assert.Equal(t, 100000-tc.wantCharged, user.Quota)
			var consumeLogs int64
			require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("type = ?", model.LogTypeConsume).Count(&consumeLogs).Error)
			assert.Equal(t, int64(min(tc.wantCharged, 1)), consumeLogs)
			assert.Empty(t, refunds, "the hold is released once")
		})
	}
}
