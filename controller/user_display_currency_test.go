package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const displayCurrencyTestSetting = `{"notify_type":"email","quota_warning_threshold":5,"gotify_priority":0,"sidebar_modules":"{\"chat\":true}","language":"fr"}`

func storedUserSetting(t *testing.T, userID int) map[string]any {
	t.Helper()
	stored, err := model.GetUserById(userID, false)
	require.NoError(t, err)
	setting := map[string]any{}
	require.NoError(t, common.UnmarshalJsonStr(stored.Setting, &setting))
	return setting
}

func TestUpdateSelfSavesDisplayCurrency(t *testing.T) {
	for _, choice := range []string{"site", "USD"} {
		t.Run(choice, func(t *testing.T) {
			user, identity := setupSecurityEnrollmentTest(t)
			require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("setting", displayCurrencyTestSetting).Error)
			before := storedUserSetting(t, user.Id)

			response := securityEnrollmentRequest(http.MethodPut, "/api/user/self", `{"display_currency":"`+choice+`"}`, "", identity, UpdateSelf)

			var result securityEnrollmentResponse
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			require.True(t, result.Success, response.Body.String())
			after := storedUserSetting(t, user.Id)
			assert.Equal(t, choice, after["display_currency"])
			delete(after, "display_currency")
			assert.Equal(t, before, after)
		})
	}
}

func TestUpdateSelfRefusesUnknownDisplayCurrency(t *testing.T) {
	for _, body := range []string{
		`{"display_currency":"EUR"}`,
		`{"display_currency":"usd"}`,
		`{"display_currency":""}`,
		`{"display_currency":1}`,
		`{"display_currency":null}`,
	} {
		t.Run(body, func(t *testing.T) {
			user, identity := setupSecurityEnrollmentTest(t)
			require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("setting", displayCurrencyTestSetting).Error)
			before := storedUserSetting(t, user.Id)

			response := securityEnrollmentRequest(http.MethodPut, "/api/user/self", body, "", identity, UpdateSelf)

			var result securityEnrollmentResponse
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.False(t, result.Success, response.Body.String())
			assert.Equal(t, "Invalid parameters", result.MessageKey)
			assert.Equal(t, before, storedUserSetting(t, user.Id))
		})
	}
}
