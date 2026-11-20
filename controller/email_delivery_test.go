package controller

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type emailHTTPTransport func(*http.Request) (*http.Response, error)

func (transport emailHTTPTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func configureEmailDelivery(t *testing.T, options map[string]string) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	original := common.OptionMap
	common.OptionMap = options
	common.OptionMapRWMutex.Unlock()
	originalTransport := http.DefaultTransport
	originalFrom, originalAccount, originalServer := common.SMTPFrom, common.SMTPAccount, common.SMTPServer
	common.SMTPFrom, common.SMTPAccount, common.SMTPServer = "", "", ""
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = original
		common.OptionMapRWMutex.Unlock()
		http.DefaultTransport = originalTransport
		common.SMTPFrom, common.SMTPAccount, common.SMTPServer = originalFrom, originalAccount, originalServer
	})
}

func TestResendDeliversThroughSelectedProvider(t *testing.T) {
	configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": "test-key", "ResendFrom": "New API <sender@example.com>"})
	calls := 0
	http.DefaultTransport = emailHTTPTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		assert.Equal(t, "https://api.resend.com/emails", request.URL.String())
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "Bearer test-key", request.Header.Get("Authorization"))
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
		_, hasDeadline := request.Context().Deadline()
		assert.True(t, hasDeadline, "email delivery must have a bounded wait")
		var body struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			HTML    string   `json:"html"`
		}
		require.NoError(t, common.DecodeJson(request.Body, &body))
		assert.Equal(t, `"New API" <sender@example.com>`, body.From)
		assert.Equal(t, []string{"one@example.com", "two@example.com"}, body.To)
		assert.Equal(t, "验证码", body.Subject)
		assert.Equal(t, "<p>123456</p>", body.HTML)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"email-test-id"}`)), Header: make(http.Header)}, nil
	})
	require.NoError(t, common.SendEmail("验证码", "one@example.com; two@example.com", "<p>123456</p>"))
	assert.Equal(t, 1, calls, "no SMTP configuration is needed and no duplicate request is made")
}

// The verification code only registers an account. The mail's first line names
// the site by the host of the server address, or by the system name when none
// is set, in every backend language; the mail is wrapped in the language and
// direction of the request, and the code stays whole.
func TestEmailVerificationNamesTheSiteToRegisterAt(t *testing.T) {
	require.NoError(t, i18n.Init())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	previousDB, previousRedis, previousAddress := model.DB, common.RedisEnabled, system_setting.ServerAddress
	previousName, previousLanguage, previousMinutes := common.SystemName, common.DefaultLanguage, common.VerificationValidMinutes
	model.DB, common.RedisEnabled = db, false
	common.SystemName, common.DefaultLanguage, common.VerificationValidMinutes = "New <API>", "", 10
	t.Cleanup(func() {
		model.DB, common.RedisEnabled, system_setting.ServerAddress = previousDB, previousRedis, previousAddress
		common.SystemName, common.DefaultLanguage, common.VerificationValidMinutes = previousName, previousLanguage, previousMinutes
	})
	configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": "test-key", "ResendFrom": "sender@example.com"})
	sent := ""
	http.DefaultTransport = emailHTTPTransport(func(request *http.Request) (*http.Response, error) {
		var body struct {
			HTML string `json:"html"`
		}
		require.NoError(t, common.DecodeJson(request.Body, &body))
		sent = body.HTML
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"email-test-id"}`)), Header: make(http.Header)}, nil
	})
	send := func(acceptLanguage string) string {
		sent = ""
		response := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(response)
		context.Request = httptest.NewRequest(http.MethodGet, "/api/verification?email=new@example.com", nil)
		context.Request.Header.Set("Accept-Language", acceptLanguage)
		SendEmailVerification(context)
		require.Contains(t, response.Body.String(), `"success":true`)
		return sent
	}

	const ltr = ` dir="ltr" style="direction:ltr;text-align:left"`
	system_setting.ServerAddress = "https://example.org"
	assert.Regexp(t, "^"+regexp.QuoteMeta(`<div lang="en"`+ltr+`>`+
		`<p`+ltr+`>Hello, you are verifying your email address to register at <span dir="ltr">example.org</span>.</p>`+
		`<p`+ltr+`>Your verification code is: <strong><span dir="ltr">`)+`[0-9a-f]{6}`+regexp.QuoteMeta(`</span></strong></p>`+
		`<p`+ltr+`>The verification code is valid for 10 minutes. If you did not request it, please ignore this email.</p></div>`)+"$",
		send("en-US"))

	for _, tc := range []struct{ address, site string }{
		{"https://example.org", `<span dir="ltr">example.org</span>`},
		{"http://localhost:3000/", `<span dir="ltr">localhost:3000</span>`},
		{"", "New &lt;API&gt;"},
	} {
		system_setting.ServerAddress = tc.address
		for _, lang := range i18n.SupportedLanguages() {
			assert.Contains(t, send(lang), tc.site, "server address %q, %s", tc.address, lang)
		}
	}
}

// An administrator, or a script, sets the mail frame through the option API and
// reads it back. The next mail is placed in it, and an empty value sends the
// body as it is again.
func TestEmailFrameOptionIsSavedReadBackAndUsed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	previousDB, previousRedis, previousName := model.DB, common.RedisEnabled, common.SystemName
	model.DB, common.RedisEnabled, common.SystemName = db, false, "New API"
	t.Cleanup(func() { model.DB, common.RedisEnabled, common.SystemName = previousDB, previousRedis, previousName })
	configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": "test-key", "ResendFrom": "sender@example.com", common.EmailFrameOption: ""})
	sent := ""
	http.DefaultTransport = emailHTTPTransport(func(request *http.Request) (*http.Response, error) {
		var body struct {
			HTML string `json:"html"`
		}
		require.NoError(t, common.DecodeJson(request.Body, &body))
		sent = body.HTML
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"email-test-id"}`)), Header: make(http.Header)}, nil
	})
	save := func(value string) string {
		body, err := common.Marshal(OptionUpdateRequest{Key: common.EmailFrameOption, Value: value})
		require.NoError(t, err)
		response := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(response)
		context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(string(body)))
		UpdateOption(context)
		return response.Body.String()
	}
	const frame = `{"*":"<div style=\"text-align:{{align}}\">{{system_name}}</div>{{body}}","fa":"<div dir=\"rtl\">{{body}}</div>"}`
	body := i18n.EmailBody("en", "Hello.")

	assert.Contains(t, save(frame), `"success":true`)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/option/", nil)
	GetOptions(context)
	var options struct {
		Data []model.Option `json:"data"`
	}
	require.NoError(t, common.DecodeJson(response.Body, &options))
	assert.Contains(t, options.Data, model.Option{Key: common.EmailFrameOption, Value: frame})
	require.NoError(t, common.SendEmail("Hello", "one@example.com", body))
	assert.Equal(t, `<div style="text-align:left">New API</div>`+string(body), sent)

	assert.Contains(t, save(""), `"success":true`)
	require.NoError(t, common.SendEmail("Hello", "one@example.com", body))
	assert.Equal(t, string(body), sent)
}

// A mail frame that breaks a rule is refused when saved, with a message that
// names the rule, and the saved frame stays as it was.
func TestEmailFrameOptionRefusesEachBrokenRule(t *testing.T) {
	const shape = "The mail frame must be a JSON object that maps language tags, or * for every other language, to HTML"
	for _, tc := range []struct {
		name, value, key, message string
	}{
		{"HTML that is not JSON", `<p>{{body}}</p>`, shape, shape},
		{"a JSON array", `["{{body}}"]`, shape, shape},
		{"null", `null`, shape, shape},
		{"a frame that is not text", `{"*":1}`, shape, shape},
		{"a key that is not a language tag", `{"default":"{{body}}"}`,
			"The mail frame key {{key}} is not a language tag or *", "The mail frame key default is not a language tag or *"},
		{"a language named twice", `{"fa":"{{body}}","FA":"{{body}}"}`,
			"The mail frame names the language {{lang}} more than once", "The mail frame names the language fa more than once"},
		{"no body placeholder", `{"*":"<p>{{system_name}}</p>"}`,
			"The mail frame for {{key}} has no {{placeholder}} placeholder for the mail body", "The mail frame for * has no {{body}} placeholder for the mail body"},
		{"the body placeholder twice", `{"fa":"{{body}}<hr>{{body}}"}`,
			"The mail frame for {{key}} has the {{placeholder}} placeholder more than once", "The mail frame for fa has the {{body}} placeholder more than once"},
		{"a value past the size bound", `{"*":"{{body}}` + strings.Repeat("x", common.EmailFrameMaxBytes) + `"}`,
			"The mail frame must be at most {{max}} bytes", "The mail frame must be at most 32768 bytes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const saved = `{"*":"{{body}}"}`
			configureEmailDelivery(t, map[string]string{common.EmailFrameOption: saved})
			body, err := common.Marshal(OptionUpdateRequest{Key: common.EmailFrameOption, Value: tc.value})
			require.NoError(t, err)
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(string(body)))
			require.NotPanics(t, func() { UpdateOption(context) }, "a refused frame must fail before accessing storage")
			var result struct {
				Success    bool   `json:"success"`
				Message    string `json:"message"`
				MessageKey string `json:"message_key"`
			}
			require.NoError(t, common.DecodeJson(response.Body, &result))
			assert.False(t, result.Success)
			assert.Equal(t, tc.key, result.MessageKey)
			assert.Equal(t, tc.message, result.Message)
			assert.Equal(t, saved, common.OptionMap[common.EmailFrameOption])
		})
	}
}

func TestResendFailuresDoNotRetryFallbackOrExposeSensitiveContent(t *testing.T) {
	for _, status := range []int{302, 400, 401, 403, 429, 500} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": "test-key", "ResendFrom": "sender@example.com"})
			calls := 0
			http.DefaultTransport = emailHTTPTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{"https://untrusted.example/emails"}}, Body: io.NopCloser(strings.NewReader(`{"message":"test-key 123456 reset-token"}`))}, nil
			})
			err := common.SendEmail("verification", "one@example.com", "123456 reset-token")
			require.EqualError(t, err, "Resend email delivery failed (HTTP "+strconv.Itoa(status)+")")
			assert.Equal(t, 1, calls, "do not retry or follow redirects")
		})
	}
}

func TestResendRejectsIncompleteOrInvalidConfigurationBeforeSending(t *testing.T) {
	tests := []struct{ name, key, from, receiver string }{
		{"missing key", "", "sender@example.com", "one@example.com"},
		{"header injection", "key\r\nInjected: value", "sender@example.com", "one@example.com"},
		{"missing sender", "test-key", "", "one@example.com"},
		{"invalid sender", "test-key", "bad-address", "one@example.com"},
		{"invalid recipient", "test-key", "sender@example.com", "bad-address"},
		{"empty recipient", "test-key", "sender@example.com", ""},
		{"too many recipients", "test-key", "sender@example.com", strings.Repeat("one@example.com;", 50) + "two@example.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": test.key, "ResendFrom": test.from})
			http.DefaultTransport = emailHTTPTransport(func(*http.Request) (*http.Response, error) {
				t.Fatal("invalid settings must not make a request")
				return nil, errors.New("unexpected request")
			})
			err := common.SendEmail("verification", test.receiver, "123456")
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "123456")
			assert.NotContains(t, err.Error(), "test-key")
		})
	}
}

func TestResendRejectsMalformedSuccessAndSanitizesTransportErrors(t *testing.T) {
	for _, body := range []string{`{}`, `not json`, `{"id":false}`, `{"id":""}`} {
		t.Run(body, func(t *testing.T) {
			configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": "test-key", "ResendFrom": "sender@example.com"})
			http.DefaultTransport = emailHTTPTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			require.EqualError(t, common.SendEmail("verification", "one@example.com", "123456"), "Invalid Resend delivery response")
		})
	}
	configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": "test-key", "ResendFrom": "sender@example.com"})
	http.DefaultTransport = emailHTTPTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("test-key 123456 reset-token") })
	require.EqualError(t, common.SendEmail("verification", "one@example.com", "123456"), "Resend request failed")
}

func TestSMTPAndLegacyConfigurationNeverUseResend(t *testing.T) {
	for _, provider := range []string{"", "smtp", "invalid"} {
		t.Run("provider="+provider, func(t *testing.T) {
			configureEmailDelivery(t, map[string]string{"EmailProvider": provider, "ResendApiKey": "test-key", "ResendFrom": "sender@example.com"})
			http.DefaultTransport = emailHTTPTransport(func(*http.Request) (*http.Response, error) {
				t.Fatal("SMTP and unknown providers must not invoke Resend")
				return nil, errors.New("unexpected request")
			})
			require.Error(t, common.SendEmail("verification", "one@example.com", "123456"))
		})
	}
}

func TestEmailOptionsNeverReturnResendCredentials(t *testing.T) {
	configureEmailDelivery(t, map[string]string{"EmailProvider": "resend", "ResendApiKey": "private-email-credential", "ResendFrom": "sender@example.com"})
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/option/", nil)
	GetOptions(context)
	require.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "private-email-credential")
	assert.NotContains(t, response.Body.String(), "ResendApiKey")
	assert.Contains(t, response.Body.String(), "ResendFrom")
	assert.Contains(t, response.Body.String(), `"key":"ResendKeyConfigured","value":"true"`)
}

func TestEmailProviderOptionRejectsInvalidValues(t *testing.T) {
	for _, provider := range []string{"", "both", "SMTP", "smtp,resend"} {
		t.Run(provider, func(t *testing.T) {
			configureEmailDelivery(t, map[string]string{"EmailProvider": "smtp"})
			body, err := common.Marshal(OptionUpdateRequest{Key: "EmailProvider", Value: provider})
			require.NoError(t, err)
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(string(body)))
			UpdateOption(context)
			assert.Contains(t, response.Body.String(), `"success":false`)
			assert.Contains(t, response.Body.String(), `"message_key":"Email sending method must be SMTP or Resend"`)
			assert.Equal(t, "smtp", common.OptionMap["EmailProvider"])
		})
	}
}

func TestEmailProviderCannotActivateIncompleteResend(t *testing.T) {
	for name, settings := range map[string]map[string]string{
		"missing key":    {"ResendFrom": "sender@example.com"},
		"missing sender": {"ResendApiKey": "test-key"},
		"invalid sender": {"ResendApiKey": "test-key", "ResendFrom": "invalid"},
		"invalid key":    {"ResendApiKey": "key\r\nInjected: value", "ResendFrom": "sender@example.com"},
	} {
		t.Run(name, func(t *testing.T) {
			settings["EmailProvider"] = "smtp"
			configureEmailDelivery(t, settings)
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(`{"key":"EmailProvider","value":"resend"}`))
			require.NotPanics(t, func() { UpdateOption(context) }, "invalid activation must fail before accessing storage")
			assert.Contains(t, response.Body.String(), `"success":false`)
			assert.Contains(t, response.Body.String(), `"message_key":"Resend `)
			assert.Equal(t, "smtp", common.OptionMap["EmailProvider"])
			assert.NotContains(t, response.Body.String(), "test-key")
		})
	}
}

func TestResendSettingsChangesKeepActiveDeliveryWorking(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	previousDB, previousRedis := model.DB, common.RedisEnabled
	model.DB, common.RedisEnabled = db, false
	t.Cleanup(func() { model.DB, common.RedisEnabled = previousDB, previousRedis })

	tests := []struct {
		name, provider, key, value string
		saved                      bool
	}{
		{"active sender the parser rejects", "resend", "ResendFrom", "notify..team@example.com", false},
		{"active key cleared", "resend", "ResendApiKey", " ", false},
		{"active sender renamed", "resend", "ResendFrom", `"New API" <notify@example.com>`, true},
		{"inactive sender saved before its key", "smtp", "ResendFrom", "notify@example.com", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := map[string]string{"EmailProvider": test.provider, "ResendApiKey": "test-key", "ResendFrom": "sender@example.com"}
			if test.provider == "smtp" {
				settings["ResendApiKey"] = ""
			}
			configureEmailDelivery(t, settings)
			before := settings[test.key]
			body, err := common.Marshal(OptionUpdateRequest{Key: test.key, Value: test.value})
			require.NoError(t, err)
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(string(body)))
			UpdateOption(context)
			if test.saved {
				assert.Contains(t, response.Body.String(), `"success":true`)
				assert.Equal(t, test.value, common.OptionMap[test.key])
				return
			}
			assert.Contains(t, response.Body.String(), `"message_key":"Resend `)
			assert.Equal(t, before, common.OptionMap[test.key])
		})
	}
}

func TestEmailKeyStatusIsDerivedAndReadOnly(t *testing.T) {
	configureEmailDelivery(t, map[string]string{"EmailProvider": "smtp", "ResendKeyConfigured": "true"})
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/option/", nil)
	GetOptions(context)
	assert.Contains(t, response.Body.String(), `"key":"ResendKeyConfigured","value":"false"`)
	assert.NotContains(t, response.Body.String(), `"value":"true"`)

	response = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(`{"key":"ResendKeyConfigured","value":true}`))
	UpdateOption(context)
	assert.Contains(t, response.Body.String(), `"success":false`)
	assert.Contains(t, response.Body.String(), "read-only")
}
