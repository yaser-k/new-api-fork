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
	"plain-old-model": {"retired_at": "2026-08-01", "successor": "new-model"},
	"last-model": {"retired_at": "2026-07-15", "notice_url": "https://example.com/retirements/last-model"},
	"plain-last-model": {"retired_at": "2026-06-01", "successor": ""}
}`

const testRetiredMessage = "The model 'old-model' was retired on 2026-09-30. Use 'new-model' instead. See https://example.com/retirements/old-model."

const testNoSuccessorMessage = "The model 'last-model' was retired on 2026-07-15 and has no successor. See https://example.com/retirements/last-model."

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

	// The bodies are compared byte for byte, as the encoder writes them.
	openAI := func(message string) string {
		return fmt.Sprintf(`{"error":{"code":"model_retired","message":%q,"type":"new_api_error"}}`, message)
	}
	anthropic := func(message string) string {
		return fmt.Sprintf(`{"error":{"message":%q,"type":"model_retired"},"type":"error"}`, message)
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
		{"no successor", "/v1/chat/completions", `{"model":"last-model","messages":[]}`, openAI, testNoSuccessorMessage},
		{"no successor, streaming", "/v1/chat/completions", `{"model":"last-model","stream":true,"messages":[]}`, openAI, testNoSuccessorMessage},
		{"no successor, responses", "/v1/responses", `{"model":"last-model","input":"hi","stream":true}`, openAI, testNoSuccessorMessage},
		{"no successor, without notice url", "/v1/chat/completions", `{"model":"plain-last-model"}`, openAI, "The model 'plain-last-model' was retired on 2026-06-01 and has no successor."},
		{"no successor, modifier variant", "/v1/chat/completions", `{"model":"last-model@thinking:on"}`, openAI, testNoSuccessorMessage},
		{"no successor, anthropic messages", "/v1/messages", `{"model":"last-model","max_tokens":1,"messages":[]}`, anthropic, testNoSuccessorMessage},
		{"no successor, streaming anthropic messages", "/v1/messages", `{"model":"last-model","stream":true,"max_tokens":1,"messages":[]}`, anthropic, testNoSuccessorMessage},
		{"no successor, gemini generate", "/v1beta/models/last-model:generateContent", `{"contents":[]}`, google, testNoSuccessorMessage},
		{"no successor, gemini stream", "/v1beta/models/last-model:streamGenerateContent", `{"contents":[]}`, google, testNoSuccessorMessage},
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
			assert.Equal(t, tc.shape(tc.message+" (request id: "+requestID+")"), recorder.Body.String())
		})
	}
}

// The answer for a model retired with no successor is written in the reader's
// language, the notice sentence after it as with a successor.
func TestDistributeAnswersModelWithoutSuccessorInTheReadersLanguage(t *testing.T) {
	require.NoError(t, i18n.Init())
	withModelRetirements(t, testRetirementList)

	for _, tc := range []struct {
		acceptLanguage, message string
	}{
		{"fa-IR,fa;q=0.9", "مدل «⁨last-model⁩» در ⁨2026-07-15⁩ کنار گذاشته شده است و مدل جایگزینی ندارد. به ⁨https://example.com/retirements/last-model⁩ مراجعه کنید."},
		{"zh-CN", "模型 'last-model' 已于 2026-07-15 下线，没有替代模型。 详见 https://example.com/retirements/last-model。"},
		{"zh-TW", "模型 'last-model' 已於 2026-07-15 下線，沒有替代模型。 詳見 https://example.com/retirements/last-model。"},
	} {
		t.Run(tc.acceptLanguage, func(t *testing.T) {
			recorder, relayed := serveDistribute(t, http.MethodPost, "/v1/chat/completions", `{"model":"last-model"}`, func(c *gin.Context) {
				c.Request.Header.Set("Accept-Language", tc.acceptLanguage)
			})

			require.Equal(t, http.StatusGone, recorder.Code)
			assert.False(t, relayed)
			var body struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
			assert.Equal(t, tc.message+" (request id: "+recorder.Header().Get(common.RequestIdKey)+")", body.Error.Message)
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
