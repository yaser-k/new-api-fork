package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRetiredModels = `{
	"zz-retired-model": {"retired_at": "2026-09-30", "successor": "zz-live-model"},
	"zz-retired-on-channel": {"retired_at": "2026-10-01", "successor": "zz-live-model"}
}`

func withRetiredModels(t *testing.T, value string) {
	t.Helper()
	operation_setting.LoadModelRetirementsFromJSONString(value)
	t.Cleanup(func() { operation_setting.LoadModelRetirementsFromJSONString("{}") })
}

func listModelIDs(t *testing.T, modelType int) []string {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("id", 1101)
	ListModels(ctx, modelType)
	require.Equal(t, http.StatusOK, recorder.Code)

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	ids := make([]string, 0)
	for _, item := range payload.Data {
		ids = append(ids, item.ID)
	}
	for _, item := range payload.Models {
		ids = append(ids, item.Name)
	}
	return ids
}

func userModelNames(t *testing.T) []string {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/models", nil)
	ctx.Set("id", 1101)
	GetUserModels(ctx)
	return decodeUserModelsResponse(t, recorder)
}

func TestModelListsNeverShowRetiredModels(t *testing.T) {
	withSelfUseModeEnabled(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1101,
		Username: "retired-model-list-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	// zz-retired-on-channel is a misconfiguration: retired, yet still on a channel.
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-live-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-retired-on-channel", ChannelId: 1, Enabled: true},
	}).Error)

	modelTypes := []int{constant.ChannelTypeOpenAI, constant.ChannelTypeAnthropic, constant.ChannelTypeGemini}
	for _, modelType := range modelTypes {
		assert.ElementsMatch(t, []string{"zz-live-model", "zz-retired-on-channel"}, listModelIDs(t, modelType), "an empty list changes nothing")
	}
	assert.ElementsMatch(t, []string{"zz-live-model", "zz-retired-on-channel"}, userModelNames(t))

	withRetiredModels(t, testRetiredModels)
	for _, modelType := range modelTypes {
		assert.Equal(t, []string{"zz-live-model"}, listModelIDs(t, modelType))
	}
	assert.Equal(t, []string{"zz-live-model"}, userModelNames(t))
}
