package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type perfMetricsResponse struct {
	Success bool                    `json:"success"`
	Data    perfmetrics.QueryResult `json:"data"`
}

type perfMetricsSummaryResponse struct {
	Success bool                         `json:"success"`
	Data    perfmetrics.SummaryAllResult `json:"data"`
}

func setupPerfMetricsVisibilityTest(t *testing.T) {
	t.Helper()

	initModelListColumnNames(t)
	gin.SetMode(gin.TestMode)
	oldDB, oldLogDB, oldRedis := model.DB, model.LOG_DB, common.RedisEnabled
	oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
	oldGroupRatio := ratio_setting.GroupRatio2JSONString()
	oldUsableGroups := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.RedisEnabled = oldDB, oldLogDB, oldRedis
		common.SetDatabaseTypes(oldMainType, oldLogType)
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatio))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUsableGroups))
	})

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.PerfMetric{}))

	// Every group has a ratio; only "default" is offered to every user.
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"hidden":1}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"default"}`))
	require.NoError(t, db.Create(&model.User{Id: 1, Username: "hidden-user", Password: "password", Group: "hidden", Status: common.UserStatusEnabled}).Error)

	hour := time.Now().Unix() - time.Now().Unix()%3600 - 3600
	for _, row := range []model.PerfMetric{
		{ModelName: "perf-shared-model", Group: "default", BucketTs: hour, RequestCount: 10, SuccessCount: 10, TotalLatencyMs: 10000},
		{ModelName: "perf-shared-model", Group: "hidden", BucketTs: hour, RequestCount: 10, SuccessCount: 0, TotalLatencyMs: 90000},
		{ModelName: "perf-hidden-model", Group: "hidden", BucketTs: hour, RequestCount: 5, SuccessCount: 5, TotalLatencyMs: 5000},
	} {
		require.NoError(t, model.UpsertPerfMetric(&row))
	}
}

func performPerfMetricsRequest(t *testing.T, handler gin.HandlerFunc, target string, userID int, out any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	if userID != 0 {
		c.Set("id", userID)
	}
	handler(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), out))
}

func TestPerfMetricsOnlyCountGroupsTheViewerCanUse(t *testing.T) {
	setupPerfMetricsVisibilityTest(t)

	groupNames := func(groups []perfmetrics.GroupResult) []string {
		names := make([]string, 0, len(groups))
		for _, group := range groups {
			names = append(names, group.Group)
		}
		return names
	}

	var signedOut perfMetricsResponse
	performPerfMetricsRequest(t, GetPerfMetrics, "/api/perf-metrics?model=perf-shared-model", 0, &signedOut)
	require.True(t, signedOut.Success)
	assert.Equal(t, []string{"default"}, groupNames(signedOut.Data.Groups))
	require.NotNil(t, signedOut.Data.Summary)
	assert.Equal(t, 100.0, signedOut.Data.Summary.SuccessRate)
	assert.Equal(t, int64(1000), signedOut.Data.Summary.AvgLatencyMs)
	require.Len(t, signedOut.Data.Series, 1)
	assert.Equal(t, 100.0, signedOut.Data.Series[0].SuccessRate)

	var signedOutHiddenGroup perfMetricsResponse
	performPerfMetricsRequest(t, GetPerfMetrics, "/api/perf-metrics?model=perf-shared-model&group=hidden", 0, &signedOutHiddenGroup)
	assert.Empty(t, signedOutHiddenGroup.Data.Groups)
	assert.Nil(t, signedOutHiddenGroup.Data.Summary)

	var signedIn perfMetricsResponse
	performPerfMetricsRequest(t, GetPerfMetrics, "/api/perf-metrics?model=perf-shared-model", 1, &signedIn)
	assert.ElementsMatch(t, []string{"default", "hidden"}, groupNames(signedIn.Data.Groups))
	require.NotNil(t, signedIn.Data.Summary)
	assert.Equal(t, 50.0, signedIn.Data.Summary.SuccessRate)

	modelNames := func(models []perfmetrics.ModelSummary) map[string]perfmetrics.ModelSummary {
		byName := make(map[string]perfmetrics.ModelSummary, len(models))
		for _, item := range models {
			byName[item.ModelName] = item
		}
		return byName
	}

	var signedOutSummary perfMetricsSummaryResponse
	performPerfMetricsRequest(t, GetPerfMetricsSummary, "/api/perf-metrics/summary", 0, &signedOutSummary)
	signedOutModels := modelNames(signedOutSummary.Data.Models)
	assert.NotContains(t, signedOutModels, "perf-hidden-model")
	require.Contains(t, signedOutModels, "perf-shared-model")
	assert.Equal(t, 100.0, signedOutModels["perf-shared-model"].SuccessRate)

	var signedInSummary perfMetricsSummaryResponse
	performPerfMetricsRequest(t, GetPerfMetricsSummary, "/api/perf-metrics/summary", 1, &signedInSummary)
	signedInModels := modelNames(signedInSummary.Data.Models)
	assert.Contains(t, signedInModels, "perf-hidden-model")
	require.Contains(t, signedInModels, "perf-shared-model")
	assert.Equal(t, 50.0, signedInModels["perf-shared-model"].SuccessRate)
}
