package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetTopUpInfoReturnsTopUpDisabledMessageVerbatim(t *testing.T) {
	oldDB := model.DB
	oldOptionMap := common.OptionMap
	oldMessage := common.TopUpDisabledMessage
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.SystemTask{}))
	model.DB = db
	common.OptionMap = map[string]string{}
	t.Cleanup(func() {
		model.DB = oldDB
		common.OptionMap = oldOptionMap
		common.TopUpDisabledMessage = oldMessage
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	testCases := []struct {
		name    string
		message string
	}{
		{name: "unset keeps the wallet default", message: ""},
		{name: "plain text with line breaks", message: "Top up at the front desk.\nOpen weekdays."},
		{name: "markup stays text", message: `<b>Ask</b> <a href="x">here</a> & wait`},
	}

	gin.SetMode(gin.TestMode)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, model.UpdateOption("TopUpDisabledMessage", tc.message))

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)
			GetTopUpInfo(ctx)

			require.Equal(t, http.StatusOK, recorder.Code)
			var body struct {
				Success bool `json:"success"`
				Data    struct {
					TopUpDisabledMessage *string `json:"topup_disabled_message"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
			require.True(t, body.Success)
			require.NotNil(t, body.Data.TopUpDisabledMessage)
			assert.Equal(t, tc.message, *body.Data.TopUpDisabledMessage)
		})
	}
}
