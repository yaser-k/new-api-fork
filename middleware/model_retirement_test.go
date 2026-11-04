package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRetirementList = `{
	"old-model": {"retired_at": "2026-09-30", "successor": "new-model", "notice_url": "https://example.com/retirements/old-model"},
	"plain-old-model": {"retired_at": "2026-08-01", "successor": "new-model"}
}`

const testRetiredMessage = "The model 'old-model' was retired on 2026-09-30. Use 'new-model' instead. See https://example.com/retirements/old-model."

func withModelRetirements(t *testing.T, value string) {
	t.Helper()
	operation_setting.LoadModelRetirementsFromJSONString(value)
	t.Cleanup(func() { operation_setting.LoadModelRetirementsFromJSONString("{}") })
}

// countChannelSelections replaces the distributor's channel selector with one
// that counts its calls and then selects as before.
func countChannelSelections(t *testing.T) *int {
	t.Helper()
	calls := 0
	original := selectChannelForRequest
	selectChannelForRequest = func(c *gin.Context, modelName string, param *service.RetryParam) (*model.Channel, string, *service.ChannelSelectError) {
		calls++
		return original(c, modelName, param)
	}
	t.Cleanup(func() { selectChannelForRequest = original })
	return &calls
}

// serveDistribute runs one request through Distribute and reports whether the
// relay handler after it ran.
func serveDistribute(t *testing.T, method, path, body string, setup func(c *gin.Context)) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	relayed := false
	router := gin.New()
	router.Handle(method, path, RequestId(), func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
		if setup != nil {
			setup(c)
		}
	}, Distribute(), func(c *gin.Context) {
		relayed = true
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept-Language", "en")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder, relayed
}

func TestDistributeAnswersRetiredModelWithGone(t *testing.T) {
	require.NoError(t, i18n.Init())
	withModelRetirements(t, testRetirementList)

	openAI := func(message string) string {
		return fmt.Sprintf(`{"error":{"message":%q,"type":"new_api_error","code":"model_retired"}}`, message)
	}
	anthropic := func(message string) string {
		return fmt.Sprintf(`{"type":"error","error":{"type":"model_retired","message":%q}}`, message)
	}
	google := func(message string) string {
		return fmt.Sprintf(`{"error":{"code":410,"message":%q,"status":"NOT_FOUND"}}`, message)
	}

	for _, tc := range []struct {
		name, path, body string
		shape            func(string) string
		message          string
	}{
		{"chat completions", "/v1/chat/completions", `{"model":"old-model","messages":[]}`, openAI, testRetiredMessage},
		{"streaming chat completions", "/v1/chat/completions", `{"model":"old-model","stream":true,"messages":[]}`, openAI, testRetiredMessage},
		{"completions", "/v1/completions", `{"model":"old-model","prompt":"hi"}`, openAI, testRetiredMessage},
		{"responses", "/v1/responses", `{"model":"old-model","input":"hi","stream":true}`, openAI, testRetiredMessage},
		{"embeddings", "/v1/embeddings", `{"model":"old-model","input":"hi"}`, openAI, testRetiredMessage},
		{"without notice url", "/v1/chat/completions", `{"model":"plain-old-model"}`, openAI, "The model 'plain-old-model' was retired on 2026-08-01. Use 'new-model' instead."},
		{"modifier variant", "/v1/chat/completions", `{"model":"old-model@thinking:on"}`, openAI, testRetiredMessage},
		{"anthropic messages", "/v1/messages", `{"model":"old-model","max_tokens":1,"messages":[]}`, anthropic, testRetiredMessage},
		{"streaming anthropic messages", "/v1/messages", `{"model":"old-model","stream":true,"max_tokens":1,"messages":[]}`, anthropic, testRetiredMessage},
		{"gemini generate", "/v1beta/models/old-model:generateContent", `{"contents":[]}`, google, testRetiredMessage},
		{"gemini stream", "/v1beta/models/old-model:streamGenerateContent", `{"contents":[]}`, google, testRetiredMessage},
		{"gemini on v1", "/v1/models/old-model:generateContent", `{"contents":[]}`, google, testRetiredMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selections := countChannelSelections(t)
			recorder, relayed := serveDistribute(t, http.MethodPost, tc.path, tc.body, nil)

			require.Equal(t, http.StatusGone, recorder.Code)
			assert.Equal(t, 0, *selections, "no channel may be selected for a retired model")
			assert.False(t, relayed, "a retired model must stop before the relay, its billing and its retries")
			assert.True(t, strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json"))
			requestID := recorder.Header().Get(common.RequestIdKey)
			require.NotEmpty(t, requestID)
			assert.JSONEq(t, tc.shape(tc.message+" (request id: "+requestID+")"), recorder.Body.String())
		})
	}
}

func TestDistributeAnswersRetiredModelBeforeTokenModelLimit(t *testing.T) {
	require.NoError(t, i18n.Init())
	withModelRetirements(t, testRetirementList)
	selections := countChannelSelections(t)

	recorder, relayed := serveDistribute(t, http.MethodPost, "/v1/chat/completions", `{"model":"old-model"}`, func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyTokenModelLimitEnabled, true)
		common.SetContextKey(c, constant.ContextKeyTokenModelLimit, map[string]bool{"new-model": true})
	})

	require.Equal(t, http.StatusGone, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":"model_retired"`)
	assert.Equal(t, 0, *selections)
	assert.False(t, relayed)
}

func TestDistributeLeavesModelsOffTheListUnchanged(t *testing.T) {
	require.NoError(t, i18n.Init())
	previousCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() { common.MemoryCacheEnabled = previousCacheEnabled })

	for _, list := range []string{"{}", testRetirementList} {
		t.Run(list, func(t *testing.T) {
			withModelRetirements(t, list)
			selections := countChannelSelections(t)

			recorder, relayed := serveDistribute(t, http.MethodPost, "/v1/chat/completions", `{"model":"new-model"}`, nil)

			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			requestID := recorder.Header().Get(common.RequestIdKey)
			assert.JSONEq(t, fmt.Sprintf(`{"error":{"message":%q,"type":"new_api_error","code":"model_not_found"}}`,
				"No available channel for model new-model under group default (distributor) (request id: "+requestID+")"), recorder.Body.String())
			assert.Equal(t, 1, *selections)
			assert.False(t, relayed)
		})
	}
}
