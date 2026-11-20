package i18n

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The web console saves its own language codes (zhCN, zhTW, fa, fr, ...) in
// the user setting; backend messages must follow them, and languages without a
// backend locale fall back to English.
func TestLanguageFromUserSetting(t *testing.T) {
	require.NoError(t, Init())
	for _, tc := range []struct {
		saved string
		want  string
	}{
		{"zhCN", LangZhCN},
		{"zhTW", LangZhTW},
		{"zh-HK", LangZhTW},
		{"zh_TW", LangZhTW},
		{"zh_HK", LangZhTW},
		{"zh-Hant-TW", LangZhTW},
		{"zh", LangZhCN},
		{"en", LangEn},
		{"fa", LangFa},
		{"fa-IR", LangFa},
		{"fa_IR", LangFa},
		{"fr", LangEn},
		{"ja", LangEn},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/v1/chat/completions", nil)
		c.Set(string(constant.ContextKeyUserSetting), dto.UserSetting{Language: tc.saved})
		assert.Equal(t, tc.want, StatedLang(c), tc.saved)
	}
}

// TokenAuth puts the user's settings into the request. A user who saved no
// language must not be loaded again, which would cost a cache or database read
// on every relay request; a request without loaded settings still loads them.
func TestStatedLangLoadsTheUserOnlyWithoutLoadedSettings(t *testing.T) {
	previous := userLangLoaderFunc
	t.Cleanup(func() { userLangLoaderFunc = previous })
	loads := 0
	SetUserLangLoader(func(int) string {
		loads++
		return "zhTW"
	})

	relay, _ := gin.CreateTestContext(httptest.NewRecorder())
	relay.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	relay.Set("id", 7)
	relay.Set(string(constant.ContextKeyUserSetting), dto.UserSetting{})
	assert.Equal(t, "", StatedLang(relay))
	assert.Zero(t, loads)

	dashboard, _ := gin.CreateTestContext(httptest.NewRecorder())
	dashboard.Request = httptest.NewRequest("GET", "/api/user/self", nil)
	dashboard.Set("id", 7)
	assert.Equal(t, LangZhTW, StatedLang(dashboard))
	assert.Equal(t, 1, loads)
}

// A saved language or an Accept-Language header selects the language of a
// backend message. A reader who states neither gets DEFAULT_LANGUAGE, which is
// English when unset.
func TestBackendMessageLanguage(t *testing.T) {
	require.NoError(t, Init())
	t.Cleanup(func() { common.DefaultLanguage = "" })
	quota := map[string]any{"Remaining": "$0.10"}
	const english = "Insufficient user quota, remaining quota: $0.10"
	const chinese = "用户额度不足, 剩余额度: $0.10"
	// The amount is isolated (FSI ... PDI) so it keeps its order in right-to-left text.
	const persian = "سهمیۀ کاربر کافی نیست؛ سهمیۀ باقی‌مانده: \u2068$0.10\u2069"
	for _, tc := range []struct {
		name            string
		defaultLanguage string
		saved           string
		acceptLanguage  string
		want            string
	}{
		{"nothing stated", "", "", "", english},
		{"no preference header", "", "", "*", english},
		{"nothing stated with a Chinese default", "zh-CN", "", "", chinese},
		{"nothing stated with an English default", "en", "", "", english},
		{"header wins over the default", "zh-CN", "", "en-US,en;q=0.9", english},
		{"header in a language without a locale", "zh-CN", "", "fr-FR", english},
		{"saved language wins over the header", "", "zhTW", "en-US", "使用者額度不足，剩餘額度：$0.10"},
		{"saved language", "zh-CN", "en", "", english},
		{"saved Persian", "", "fa", "en-US", persian},
		{"Persian header", "", "", "fa-IR,fa;q=0.9,en;q=0.8", persian},
		{"nothing stated with a Persian default", "fa", "", "", persian},
		{"header wins over a Persian default", "fa", "", "en-US", english},
	} {
		t.Run(tc.name, func(t *testing.T) {
			common.DefaultLanguage = tc.defaultLanguage
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
			if tc.acceptLanguage != "" {
				c.Request.Header.Set("Accept-Language", tc.acceptLanguage)
			}
			if tc.saved != "" {
				c.Set(string(constant.ContextKeyUserSetting), dto.UserSetting{Language: tc.saved})
			}
			assert.Equal(t, tc.want, T(c, MsgQuotaUserInsufficient, quota))
		})
	}
}

// A key missing from a locale, or a template that does not parse, reaches
// clients as the raw key.
func TestLocalesRenderEveryKey(t *testing.T) {
	require.NoError(t, Init())
	keysFile, err := parser.ParseFile(token.NewFileSet(), "keys.go", nil, parser.SkipObjectResolution)
	require.NoError(t, err)
	var declared []string
	ast.Inspect(keysFile, func(node ast.Node) bool {
		if spec, ok := node.(*ast.ValueSpec); ok {
			for _, value := range spec.Values {
				if literal, ok := value.(*ast.BasicLit); ok {
					key, err := strconv.Unquote(literal.Value)
					require.NoError(t, err)
					declared = append(declared, key)
				}
			}
		}
		return true
	})
	slices.Sort(declared)

	keysByLang := make(map[string][]string)
	for _, lang := range SupportedLanguages() {
		data, err := localeFS.ReadFile("locales/" + lang + ".yaml")
		require.NoError(t, err)
		var messages map[string]string
		require.NoError(t, yaml.Unmarshal(data, &messages))
		for key := range messages {
			keysByLang[lang] = append(keysByLang[lang], key)
		}
		slices.Sort(keysByLang[lang])
	}
	for _, lang := range SupportedLanguages() {
		assert.Equal(t, declared, keysByLang[lang], lang)
		for _, key := range keysByLang[lang] {
			text := Translate(lang, key, map[string]any{})
			assert.NotEqual(t, key, text, "%s %s", lang, key)
			assert.False(t, strings.Contains(text, "{{"), "%s %s: %s", lang, key, text)
		}
	}
}

// A mail names the language its text is written in, and the wrapper and every
// paragraph carry that language's direction: a right-to-left language reads
// right to left and a left-to-right one keeps its layout. A language without a
// backend locale is written, and laid out, in English.
func TestEmailBodyCarriesLanguageAndDirection(t *testing.T) {
	require.NoError(t, Init())
	previous := common.DefaultLanguage
	t.Cleanup(func() { common.DefaultLanguage = previous })
	common.DefaultLanguage = ""
	const ltr = ` dir="ltr" style="direction:ltr;text-align:left"`
	const rtl = ` dir="rtl" style="direction:rtl;text-align:right"`

	assert.Equal(t, common.EmailBody(`<div lang="en"`+ltr+`><p`+ltr+`>Hello.</p><p`+ltr+`>Bye.</p></div>`), EmailBody("en-US", "Hello.", "Bye."))
	assert.Equal(t, common.EmailBody(`<div lang="fa"`+rtl+`><p`+rtl+`>سلام.</p><p`+rtl+`>خداحافظ.</p></div>`), emailBodyIn("fa", []string{"سلام.", "خداحافظ."}))

	for _, tc := range []struct {
		defaultLanguage string
		lang            string
		want            string
	}{
		{"", "fr", `<div lang="en"` + ltr + `>`},
		{"", "zhTW", `<div lang="zh-TW"` + ltr + `>`},
		{"", "", `<div lang="en"` + ltr + `>`},
		{"zh-CN", "", `<div lang="zh-CN"` + ltr + `>`},
	} {
		common.DefaultLanguage = tc.defaultLanguage
		assert.True(t, strings.HasPrefix(string(EmailBody(tc.lang, "x")), tc.want), "%q with default %q", tc.lang, tc.defaultLanguage)
	}

	// The direction comes from the script a language tag names or implies.
	for lang, direction := range map[string]string{
		"ar-EG": rtl, "he": rtl, "ur": rtl, "ckb": rtl, "az-Arab": rtl, "dv": rtl,
		"ku": ltr, "az": ltr, "zh-CN": ltr, "ja": ltr,
	} {
		assert.Equal(t, common.EmailBody(`<div lang="`+lang+`"`+direction+`></div>`), emailBodyIn(lang, nil), lang)
	}
}

// A code, a link or a host name stays whole and left to right inside a mail of
// either direction: escaped, and with no added character to be copied with it.
func TestEmailLTRKeepsLeftToRightTextWhole(t *testing.T) {
	assert.Equal(t, `<span dir="ltr">https://example.org/user/reset?email=a@example.org&amp;token=3fa9</span>`,
		EmailLTR("https://example.org/user/reset?email=a@example.org&token=3fa9"))
	assert.Equal(t, `<span dir="ltr">&lt;b&gt;3fa9c1</span>`, EmailLTR("<b>3fa9c1"))
}

// Web console responses carry the English source text as message_key plus its
// params, and the web console translates them. The backend sends the same body
// whatever language the request asks for.
func TestWebConsoleMessageResponse(t *testing.T) {
	require.NoError(t, Init())
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{
			name: "params",
			err:  common.NewMessage("Channel {{name}} (#{{id}}) does not exist", map[string]any{"name": "{{id}}", "id": 7}),
			want: `{"success":false,"message":"Channel {{id}} (#7) does not exist","message_key":"Channel {{name}} (#{{id}}) does not exist","message_params":{"id":7,"name":"{{id}}"}}`,
		},
		{
			name: "no params",
			err:  common.NewMessage("Invalid parameters"),
			want: `{"success":false,"message":"Invalid parameters","message_key":"Invalid parameters"}`,
		},
		{
			name: "plain error",
			err:  errors.New("record not found"),
			want: `{"success":false,"message":"record not found"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("GET", "/api/channel/7", nil)
			c.Request.Header.Set("Accept-Language", "zh-CN")
			common.ApiError(c, tc.err)
			assert.JSONEq(t, tc.want, recorder.Body.String())
		})
	}
}

// Server log lines are English. DEFAULT_LANGUAGE=zh-CN prints the Chinese text of
// the lines that have one in common/log_text.zh-CN.json; Persian has none.
func TestServerLogLanguage(t *testing.T) {
	t.Cleanup(func() { common.SetLogLanguage("") })
	const line = "channel #%d has %d unfinished tasks"
	assert.Equal(t, "channel #7 has 2 unfinished tasks", common.LogText(line, 7, 2))
	common.SetLogLanguage("zh-CN")
	assert.Equal(t, "渠道 #7 未完成的任务有: 2", common.LogText(line, 7, 2))
	assert.Equal(t, "a line without a Chinese text: x", common.LogText("a line without a Chinese text: %s", "x"))
	assert.Equal(t, "任务进度轮询开始", common.LogText("task progress polling started"))
	common.SetLogLanguage("en")
	assert.Equal(t, "channel #7 has 2 unfinished tasks", common.LogText(line, 7, 2))
	common.SetLogLanguage("fa")
	assert.Equal(t, "channel #7 has 2 unfinished tasks", common.LogText(line, 7, 2))
}
