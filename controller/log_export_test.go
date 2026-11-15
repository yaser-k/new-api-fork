package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// callLogHandler runs a log handler as the given user and returns the JSON
// lines of the response.
func callLogHandler(t *testing.T, handler gin.HandlerFunc, userId int, role int, query string) []map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/log?"+query, nil)
	ctx.Set("id", userId)
	ctx.Set("role", role)
	handler(ctx)
	var lines []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(recorder.Body.String()), "\n") {
		var value map[string]any
		require.NoError(t, common.UnmarshalJsonStr(line, &value))
		lines = append(lines, value)
	}
	return lines
}

func exportedRequestIds(lines []map[string]any) []any {
	ids := make([]any, 0, len(lines))
	for _, line := range lines[:len(lines)-1] {
		ids = append(ids, line["request_id"])
	}
	return ids
}

// seedExportLogs stores logs of two users: three in the last hour and one
// from 400 days ago.
func seedExportLogs(t *testing.T, mainDB *gorm.DB, logDB *gorm.DB, now int64) {
	t.Helper()
	require.NoError(t, mainDB.Create(&model.Channel{Id: 9, Name: "primary"}).Error)
	logs := []model.Log{
		{UserId: 1, Username: "alice", CreatedAt: now - 3600, Type: model.LogTypeConsume, ModelName: "gpt-a", ChannelId: 9, Quota: 10, RequestId: "a-1", Other: `{"admin_info":{"use_channel":["9"]}}`},
		{UserId: 2, Username: "bob", CreatedAt: now - 1800, Type: model.LogTypeConsume, ModelName: "gpt-b", ChannelId: 9, Quota: 20, RequestId: "b-1"},
		{UserId: 1, Username: "alice", CreatedAt: now - 60, Type: model.LogTypeError, ModelName: "gpt-a", RequestId: "a-2"},
		{UserId: 1, Username: "alice", CreatedAt: now - 400*24*3600, Type: model.LogTypeConsume, ModelName: "gpt-a", RequestId: "a-old"},
	}
	for i := range logs {
		require.NoError(t, logDB.Create(&logs[i]).Error)
	}
}

func useLogExportDatabases(t *testing.T, mainDB *gorm.DB, logDB *gorm.DB, mainType common.DatabaseType, logType common.DatabaseType) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	previousRedis, previousMemory := common.RedisEnabled, common.MemoryCacheEnabled
	model.DB, model.LOG_DB = mainDB, logDB
	common.SetDatabaseTypes(mainType, logType)
	common.RedisEnabled, common.MemoryCacheEnabled = false, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMain, previousLog)
		common.RedisEnabled, common.MemoryCacheEnabled = previousRedis, previousMemory
	})
}

func TestLogExportDatabaseMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = previousMaster })
	for _, tc := range []struct {
		kind, env string
		typ       common.DatabaseType
	}{
		{"sqlite", "", common.DatabaseTypeSQLite},
		{"mysql", "AUDIT_MYSQL_DSN", common.DatabaseTypeMySQL},
		{"postgres", "AUDIT_POSTGRES_DSN", common.DatabaseTypePostgreSQL},
		{"clickhouse", "AUDIT_CLICKHOUSE_DSN", common.DatabaseTypeClickHouse},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			dsn := os.Getenv(tc.env)
			if tc.env != "" && dsn == "" {
				t.Skip(tc.env + " is not configured")
			}
			db, isolatedDSN := newAuditTestDatabase(t, tc.kind, dsn)
			mainDB, mainType := db, tc.typ
			if tc.kind == "clickhouse" {
				mainDB, _ = newAuditTestDatabase(t, "sqlite", "")
				mainType = common.DatabaseTypeSQLite
			}
			useLogExportDatabases(t, mainDB, db, mainType, tc.typ)
			require.NoError(t, mainDB.AutoMigrate(&model.Channel{}))
			if tc.kind == "clickhouse" {
				t.Setenv("LOG_SQL_DSN", isolatedDSN)
				require.NoError(t, model.InitLogDB())
			} else {
				require.NoError(t, db.AutoMigrate(&model.Log{}))
			}
			versionSQL := "SELECT version()"
			if tc.kind == "sqlite" {
				versionSQL = "SELECT sqlite_version()"
			}
			var version string
			require.NoError(t, model.LOG_DB.Raw(versionSQL).Scan(&version).Error)
			t.Logf("log database: %s %s", tc.kind, version)
			now := time.Now().Unix()
			seedExportLogs(t, mainDB, model.LOG_DB, now)
			query := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-2*24*3600, now+3600)

			listed := callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, query)[0]
			require.Equal(t, true, listed["success"])
			assert.EqualValues(t, 2, listed["data"].(map[string]any)["total"])
			listed = callLogHandler(t, GetAllLogs, 3, common.RoleAdminUser, query+"&username=alice")[0]
			require.Equal(t, true, listed["success"])
			assert.EqualValues(t, 2, listed["data"].(map[string]any)["total"])

			own := callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, query)
			require.Len(t, own, 3)
			assert.Equal(t, []any{"a-2", "a-1"}, exportedRequestIds(own))
			assert.Equal(t, map[string]any{"success": true, "message": "", "data": map[string]any{"count": float64(2)}}, own[2])
			assert.Equal(t, float64(1), own[0]["id"], "user exports carry list positions, not row ids")
			assert.Empty(t, own[1]["channel_name"])
			assert.NotContains(t, own[1]["other"], "admin_info")

			all := callLogHandler(t, ExportAllLogs, 3, common.RoleAdminUser, query+"&model_name=gpt-%25")
			require.Len(t, all, 4)
			assert.Equal(t, []any{"a-2", "b-1", "a-1"}, exportedRequestIds(all))
			assert.Equal(t, "primary", all[2]["channel_name"])
			assert.Contains(t, all[2]["other"], "admin_info")
			assert.Equal(t, true, all[3]["success"])
		})
	}
}

func TestLogExportRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := newAuditTestDatabase(t, "sqlite", "")
	useLogExportDatabases(t, db, db, common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Log{}, &model.QuotaData{}))
	now := time.Now().Unix()
	seedExportLogs(t, db, db, now)
	logSetting := operation_setting.GetLogSetting()
	previousSetting := *logSetting
	t.Cleanup(func() { *logSetting = previousSetting })
	recent := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-2*24*3600, now)
	tooLong := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-366*24*3600, now)

	t.Run("time range", func(t *testing.T) {
		rejected := callLogHandler(t, ExportAllLogs, 3, common.RoleAdminUser, tooLong)
		require.Len(t, rejected, 1)
		assert.Equal(t, false, rejected[0]["success"])
		assert.Equal(t, "The time range cannot exceed {{days}} days", rejected[0]["message_key"])
		assert.Equal(t, map[string]any{"days": float64(365)}, rejected[0]["message_params"])

		assert.Equal(t, false, callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, tooLong)[0]["success"])
		assert.Equal(t, false, callLogHandler(t, GetLogsSelfStat, 1, common.RoleCommonUser, tooLong)[0]["success"])
		assert.Equal(t, true, callLogHandler(t, GetUserLogs, 3, common.RoleAdminUser, tooLong)[0]["success"], "administrators search their own logs without a limit")

		// The dashboard follows the same range instead of its old one-month limit.
		fortyDays := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-40*24*3600, now)
		for _, handler := range []gin.HandlerFunc{GetUserQuotaDates, GetUserFlowQuotaDates} {
			assert.Equal(t, "The time range cannot exceed {{days}} days", callLogHandler(t, handler, 1, common.RoleCommonUser, tooLong)[0]["message_key"])
			assert.Equal(t, true, callLogHandler(t, handler, 1, common.RoleCommonUser, fortyDays)[0]["success"])
		}

		// Without a start, a user's search reaches back the whole range.
		listed := callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, "")[0]
		assert.Equal(t, true, listed["success"])
		assert.EqualValues(t, 2, listed["data"].(map[string]any)["total"])

		logSetting.MaxRangeDays = 400
		assert.Equal(t, true, callLogHandler(t, GetUserLogs, 1, common.RoleCommonUser, tooLong)[0]["success"])
		logSetting.MaxRangeDays = previousSetting.MaxRangeDays
	})

	t.Run("user export switch", func(t *testing.T) {
		logSetting.UserExportEnabled = false
		t.Cleanup(func() { logSetting.UserExportEnabled = previousSetting.UserExportEnabled })
		disabled := callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, recent)
		require.Len(t, disabled, 1)
		assert.Equal(t, "Log export is disabled", disabled[0]["message_key"])
		assert.Equal(t, true, callLogHandler(t, ExportUserLogs, 3, common.RoleAdminUser, recent)[0]["success"], "administrators export their own logs even when users cannot")
	})

	t.Run("one export per user", func(t *testing.T) {
		release, ok, err := service.AcquireLogExportLock(1)
		require.NoError(t, err)
		require.True(t, ok)
		busy := callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, recent)
		require.Len(t, busy, 1)
		assert.Equal(t, "Another log export is running. Try again after it finishes.", busy[0]["message_key"])
		other := callLogHandler(t, ExportUserLogs, 2, common.RoleCommonUser, recent)
		assert.Equal(t, true, other[len(other)-1]["success"], "other users are not blocked")
		release()
		assert.Len(t, callLogHandler(t, ExportUserLogs, 1, common.RoleCommonUser, recent), 3)
	})
}

func TestAcquireLogExportLock(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%v", useRedis), func(t *testing.T) {
			previousRedis, previousClient := common.RedisEnabled, common.RDB
			common.RedisEnabled = useRedis
			if useRedis {
				server := miniredis.RunT(t)
				common.RDB = redis.NewClient(&redis.Options{Addr: server.Addr()})
				require.NoError(t, common.RDB.Ping(context.Background()).Err())
			}
			t.Cleanup(func() { common.RedisEnabled, common.RDB = previousRedis, previousClient })

			release, ok, err := service.AcquireLogExportLock(1)
			require.NoError(t, err)
			require.True(t, ok)
			_, ok, err = service.AcquireLogExportLock(1)
			require.NoError(t, err)
			assert.False(t, ok, "a second export of the same user must wait")
			releaseOther, ok, err := service.AcquireLogExportLock(2)
			require.NoError(t, err)
			assert.True(t, ok)
			releaseOther()
			release()
			releaseAgain, ok, err := service.AcquireLogExportLock(1)
			require.NoError(t, err)
			assert.True(t, ok, "a finished export frees the user")
			releaseAgain()
		})
	}
}

// useBillingExpressions loads billing modes, billing expressions and public
// price expressions for one test and restores the previous settings after it.
func useBillingExpressions(t *testing.T, modes, expressions, public map[string]string) {
	t.Helper()
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(saved)) })
	encoded := map[string]string{}
	for key, values := range map[string]map[string]string{
		"billing_setting.billing_mode": modes, "billing_setting.billing_expr": expressions, billing_setting.PublicExprOption: public,
	} {
		value, err := common.Marshal(values)
		require.NoError(t, err)
		encoded[key] = string(value)
	}
	require.NoError(t, config.GlobalConfig.LoadFromDB(encoded))
}

func TestUserLogsWithholdInternalBillingExpressions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := newAuditTestDatabase(t, "sqlite", "")
	useLogExportDatabases(t, db, db, common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Log{}, &model.Task{}))

	const routed = `param("route") == "b" ? tier("route_b", p * 4 + c * 20) : tier("base", p * 3 + c * 15)`
	const routedTask = `param("route") == "b" ? tier("route_b", u("seconds") * 0.6) : tier("base", u("seconds") * 0.4)`
	const plain = `tier("base", p * 2 + c * 8)`
	useBillingExpressions(t,
		map[string]string{"routed-model": "tiered_expr", "plain-model": "tiered_expr", "retired-public-model": "tiered_expr", "routed-task-model": "tiered_expr"},
		map[string]string{"routed-model": routed, "plain-model": plain, "retired-public-model": routed, "routed-task-model": routedTask},
		map[string]string{"routed-model": `tier("public", p * 4 + c * 20)`, "routed-task-model": `tier("public", u("seconds") * 0.6)`},
	)
	encode := func(expression string) string { return base64.StdEncoding.EncodeToString([]byte(expression)) }
	rules := `[{"cond":"param(\"route\") == \"b\"","multiplier":2,"matched":true}]`
	now := time.Now().Unix()
	// Rows in the order the lists return them, newest first. Each internal
	// row stands for one way a row becomes internal.
	rows := []struct {
		name, model, other string
		quota              int
		internal           bool
	}{
		{"written before this change, its model has a public expression now", "routed-model",
			`{"admin_info":{"use_channel":["9"]},"billing_mode":"tiered_expr","expr_b64":"` + encode(routed) + `","matched_tier":"route_b","request_rules":` + rules + `}`, 120, true},
		{"written while its model had a public expression", "retired-public-model",
			`{"billing_mode":"tiered_expr","expr_b64":"` + encode(routed) + `","expr_internal":true,"matched_tier":"base","request_rules":` + rules + `}`, 30, true},
		{"its model has no public expression", "plain-model",
			`{"billing_mode":"tiered_expr", "expr_b64":"` + encode(plain) + `","matched_tier":"base","request_rules":` + rules + `,"big":12345678901234567890}`, 40, false},
		{"a task of a model with a public expression", "routed-task-model",
			`{"is_task":true,"billing_mode":"tiered_expr","expr_b64":"` + encode(routedTask) + `","matched_tier":"route_b","usage_facts":{"seconds":5}}`, 50, true},
		{"a task priced by the model it maps to", "task-alias",
			`{"is_task":true,"is_model_mapped":true,"upstream_model_name":"routed-task-model","billing_mode":"tiered_expr","expr_b64":"` + encode(routedTask) + `","matched_tier":"base"}`, 60, true},
	}
	for i := len(rows) - 1; i >= 0; i-- {
		require.NoError(t, db.Create(&model.Log{UserId: 1, Username: "alice", TokenId: 5, CreatedAt: now - int64(i), Type: model.LogTypeConsume,
			ModelName: rows[i].model, Quota: rows[i].quota, RequestId: fmt.Sprintf("r-%d", i), Other: rows[i].other}).Error)
	}
	query := fmt.Sprintf("start_timestamp=%d&end_timestamp=%d", now-3600, now+3600)

	// checkRows compares the rows a caller received with the stored rows.
	checkRows := func(t *testing.T, received []any, withheld bool) {
		t.Helper()
		require.Len(t, received, len(rows))
		for i, row := range rows {
			item := received[i].(map[string]any)
			assert.EqualValues(t, row.quota, item["quota"], row.name)
			other := item["other"].(string)
			if !row.internal {
				assert.Equal(t, row.other, other, "%s: the row is returned byte for byte", row.name)
				continue
			}
			var values, stored map[string]any
			require.NoError(t, common.UnmarshalJsonStr(other, &values))
			require.NoError(t, common.UnmarshalJsonStr(row.other, &stored))
			assert.Equal(t, "tiered_expr", values["billing_mode"], row.name)
			if withheld {
				// A tier's label is chosen with the expression and can name
				// an internal condition.
				assert.NotContains(t, values, "expr_b64", row.name)
				assert.NotContains(t, values, "request_rules", row.name)
				assert.NotContains(t, values, "matched_tier", row.name)
			} else {
				assert.Contains(t, values, "expr_b64", row.name)
				assert.Equal(t, strings.Contains(row.other, "request_rules"), values["request_rules"] != nil, row.name)
				assert.Equal(t, stored["matched_tier"], values["matched_tier"], row.name)
			}
		}
	}
	listed := func(t *testing.T, role int) []any {
		page := callLogHandler(t, GetUserLogs, 1, role, query)[0]
		require.Equal(t, true, page["success"])
		return page["data"].(map[string]any)["items"].([]any)
	}
	exported := func(t *testing.T, role int) []any {
		lines := callLogHandler(t, ExportUserLogs, 1, role, query)
		require.Equal(t, true, lines[len(lines)-1]["success"])
		items := make([]any, 0, len(lines)-1)
		for _, line := range lines[:len(lines)-1] {
			items = append(items, line)
		}
		return items
	}

	t.Run("an ordinary user's own log list", func(t *testing.T) {
		checkRows(t, listed(t, common.RoleCommonUser), true)
	})
	t.Run("an ordinary user's own log export", func(t *testing.T) {
		checkRows(t, exported(t, common.RoleCommonUser), true)
	})
	t.Run("the log list read with an API key", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/log/token", nil)
		ctx.Set("id", 1)
		ctx.Set("token_id", 5)
		GetLogByKey(ctx)
		var response map[string]any
		require.NoError(t, common.UnmarshalJsonStr(recorder.Body.String(), &response))
		require.Equal(t, true, response["success"])
		checkRows(t, response["data"].([]any), true)
	})
	t.Run("an administrator's own log list and export", func(t *testing.T) {
		checkRows(t, listed(t, common.RoleAdminUser), false)
		checkRows(t, exported(t, common.RoleAdminUser), false)
	})
	t.Run("an administrator's log list of every user", func(t *testing.T) {
		page := callLogHandler(t, GetAllLogs, 3, common.RoleAdminUser, query)[0]
		require.Equal(t, true, page["success"])
		items := page["data"].(map[string]any)["items"].([]any)
		require.Len(t, items, len(rows))
		for i, row := range rows {
			assert.Contains(t, items[i].(map[string]any)["other"], `"expr_b64"`, row.name)
		}
	})
	t.Run("statistics carry the charge of internal rows and no expression", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/log/self/stat?"+query, nil)
		ctx.Set("id", 1)
		ctx.Set("username", "alice")
		ctx.Set("role", common.RoleCommonUser)
		GetLogsSelfStat(ctx)
		assert.JSONEq(t, `{"success":true,"message":"","data":{"quota":300,"rpm":5,"tpm":0}}`, recorder.Body.String())
	})
	t.Run("an ordinary user's own task list", func(t *testing.T) {
		snapshot := &billingexpr.BillingSnapshot{BillingMode: "tiered_expr", ModelName: "routed-task-model", ExprString: routedTask, EstimatedTier: "route_b"}
		require.NoError(t, db.Create(&model.Task{TaskID: "task-1", UserId: 1, Quota: 50, Status: model.TaskStatusSuccess, SubmitTime: now,
			Properties:  model.Properties{OriginModelName: "routed-task-model"},
			PrivateData: model.TaskPrivateData{BillingContext: &model.TaskBillingContext{TieredSnapshot: snapshot}}}).Error)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/task/self", nil)
		ctx.Set("id", 1)
		ctx.Set("role", common.RoleCommonUser)
		GetUserTask(ctx)
		body := recorder.Body.String()
		require.Contains(t, body, `"task_id":"task-1"`)
		assert.NotContains(t, body, `param(`)
		assert.NotContains(t, body, encode(routedTask))
	})
}
