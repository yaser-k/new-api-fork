package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProcessChannelErrorUsesSnapshotWithoutLeakingChannelMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousErrorLogEnabled := constant.ErrorLogEnabled

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&model.User{}, &model.Log{}))
	model.DB, model.LOG_DB = database, database
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	constant.ErrorLogEnabled = true
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		constant.ErrorLogEnabled = previousErrorLogEnabled
		require.NoError(t, sqlDB.Close())
	})

	require.NoError(t, database.Create(&model.User{Id: 7, Username: "log-owner", Group: "default"}).Error)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("id", 7)
	ctx.Set("username", "log-owner")
	ctx.Set("token_name", "test-token")
	ctx.Set("token_id", 11)
	ctx.Set("original_model", "gpt-test")
	ctx.Set("group", "default")
	ctx.Set("channel_id", 202)
	ctx.Set("channel_name", "mutable-context-channel")
	ctx.Set("channel_type", 9)
	ctx.Set("use_channel", []string{"101"})
	common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Now().Add(-time.Second))

	channelSnapshot := types.ChannelError{
		ChannelId:   101,
		ChannelType: 1,
		ChannelName: "snapshot-channel",
		AutoBan:     false,
	}
	apiErr := types.NewOpenAIError(errors.New("upstream failed"), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway)

	processChannelError(ctx, channelSnapshot, apiErr, nil)

	var stored model.Log
	require.NoError(t, database.First(&stored).Error)
	assert.Equal(t, channelSnapshot.ChannelId, stored.ChannelId)
	storedOther, err := common.StrToMap(stored.Other)
	require.NoError(t, err)
	assert.Equal(t, float64(http.StatusBadGateway), storedOther["status_code"])
	for _, key := range []string{"channel_id", "channel_name", "channel_type"} {
		assert.NotContains(t, storedOther, key)
	}
	adminInfo, ok := storedOther["admin_info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"101"}, adminInfo["use_channel"])

	logs, total, err := model.GetUserLogs(7, model.LogTypeError, 0, 0, "", "", 0, 10, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, channelSnapshot.ChannelId, logs[0].ChannelId)
	assert.Empty(t, logs[0].ChannelName)
	userOther, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, userOther, "admin_info")
	for _, key := range []string{"channel_id", "channel_name", "channel_type"} {
		assert.NotContains(t, userOther, key)
	}
}

func TestRetriedAttemptErrorLogsAreHiddenFromTheUser(t *testing.T) {
	for _, tc := range []struct {
		name           string
		failures       int64
		failureStatus  int
		noChannelLeft  bool
		userLogTypes   []int
		adminLogTypes  []int
		requestsInStat int
	}{
		{"retried twice then answered", 2, http.StatusInternalServerError, false, []int{model.LogTypeConsume}, []int{model.LogTypeError, model.LogTypeError, model.LogTypeConsume}, 1},
		{"retried twice and still failing", 3, http.StatusInternalServerError, false, []int{model.LogTypeError}, []int{model.LogTypeError, model.LogTypeError, model.LogTypeError}, 0},
		{"failed without retry", 1, http.StatusBadRequest, false, []int{model.LogTypeError}, []int{model.LogTypeError}, 0},
		{"retry decided but no channel left", 1, http.StatusInternalServerError, true, []int{model.LogTypeError}, []int{model.LogTypeError}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Upstream translates the error of a request no channel is left for.
			require.NoError(t, i18n.Init())
			fixture := newResponsesWSBillingTest(t, `tier("request", fixed(0.002))`, func(*websocket.Conn, *http.Request) {})
			previousRetries, previousErrorLog := common.RetryTimes, constant.ErrorLogEnabled
			common.RetryTimes, constant.ErrorLogEnabled = 2, true
			t.Cleanup(func() { common.RetryTimes, constant.ErrorLogEnabled = previousRetries, previousErrorLog })
			var attempts atomic.Int64
			fixture.httpUpstream = func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if attempts.Add(1) <= tc.failures {
					if tc.noChannelLeft {
						require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ?", fixture.channel.Id).Update("enabled", false).Error)
					}
					w.WriteHeader(tc.failureStatus)
					_, _ = fmt.Fprint(w, `{"error":{"type":"server_error","code":"server_error","message":"upstream detail"}}`)
					return
				}
				_, _ = fmt.Fprint(w, `{"id":"completed","status":"completed","usage":{"input_tokens":10,"output_tokens":1,"total_tokens":11}}`)
			}
			request, err := http.NewRequest(http.MethodPost, fixture.gatewayURL+"/v1/responses", strings.NewReader(`{"model":"ws-billing","input":"hello"}`))
			require.NoError(t, err)
			request.Header.Set("Authorization", "Bearer sk-"+fixture.token.Key)
			request.Header.Set("Content-Type", "application/json")
			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			_, err = io.Copy(io.Discard, response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			select {
			case <-fixture.httpDone:
			case <-time.After(3 * time.Second):
				t.Fatal("HTTP request did not finish")
			}
			require.Equal(t, int64(len(tc.adminLogTypes)), attempts.Load())

			logTypes := func(handler gin.HandlerFunc, target string, configure func(*gin.Context)) []int {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodGet, target, nil)
				configure(c)
				handler(c)
				var body struct {
					Success bool `json:"success"`
					Data    json.RawMessage
				}
				require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
				require.True(t, body.Success, recorder.Body.String())
				var page struct {
					Items []model.Log `json:"items"`
					Total int         `json:"total"`
				}
				if err := common.Unmarshal(body.Data, &page); err != nil {
					require.NoError(t, common.Unmarshal(body.Data, &page.Items))
					page.Total = len(page.Items)
				}
				types := make([]int, 0, len(page.Items))
				for i := len(page.Items) - 1; i >= 0; i-- {
					types = append(types, page.Items[i].Type)
				}
				assert.Equal(t, len(page.Items), page.Total)
				return types
			}
			asUser := func(c *gin.Context) {
				c.Set("id", fixture.user.Id)
				c.Set("username", fixture.user.Username)
			}
			assert.Equal(t, tc.userLogTypes, logTypes(GetUserLogs, "/api/log/self", asUser))
			assert.Equal(t, tc.userLogTypes, logTypes(GetLogByKey, "/api/log/token", func(c *gin.Context) { c.Set("token_id", fixture.token.Id) }))
			assert.Equal(t, tc.adminLogTypes, logTypes(GetAllLogs, "/api/log/", func(c *gin.Context) { c.Set("role", common.RoleAdminUser) }))

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/log/self/stat", nil)
			asUser(c)
			GetLogsSelfStat(c)
			var stat struct {
				Data model.Stat `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &stat))
			assert.Equal(t, tc.requestsInStat, stat.Data.Rpm, "statistics count only consume rows")
		})
	}
}
