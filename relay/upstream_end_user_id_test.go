package relay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

const (
	endUserTestSecret = "end-user-test-secret"
	endUserAlice      = 424242
	endUserBob        = 515151
)

func expectedUpstreamEndUserID(userID int) string {
	mac := hmac.New(sha256.New, []byte(endUserTestSecret))
	mac.Write([]byte("upstream-end-user:" + strconv.Itoa(userID)))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// endUserFixture runs relay handlers against a local upstream that records
// every request body, with users and settlement backed by an in-memory database.
type endUserFixture struct {
	upstreamURL string
	bodies      chan []byte
}

func newEndUserFixture(t *testing.T) *endUserFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousSecret, previousRedis, previousDB, previousLogDB := common.CryptoSecret, common.RedisEnabled, model.DB, model.LOG_DB
	t.Cleanup(func() {
		common.CryptoSecret, common.RedisEnabled, model.DB, model.LOG_DB = previousSecret, previousRedis, previousDB, previousLogDB
	})
	common.CryptoSecret = endUserTestSecret
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.Token{}, &model.Channel{}))
	require.NoError(t, db.Create(&model.User{Id: endUserAlice, Username: "alice-example", AffCode: "aff-alice", Quota: 1000000}).Error)
	require.NoError(t, db.Create(&model.User{Id: endUserBob, Username: "bob-example", AffCode: "aff-bob", Quota: 1000000}).Error)
	model.DB, model.LOG_DB = db, db

	fixture := &endUserFixture{bodies: make(chan []byte, 4)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fixture.bodies <- body
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/messages":
			_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":2}}`))
		case "/v1/responses":
			_, _ = w.Write([]byte(`{"id":"resp_1","object":"response","status":"completed","model":"gpt-4o","output":[{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`))
		default:
			_, _ = w.Write([]byte(`{"id":"chatcmpl_1","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`))
		}
	}))
	t.Cleanup(server.Close)
	fixture.upstreamURL = server.URL
	return fixture
}

type endUserRequest struct {
	path          string
	userID        int
	channelID     int
	channelType   int
	model         string
	field         string
	paramOverride map[string]any
}

func (f *endUserFixture) context(request endUserRequest) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, request.path, nil)
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyUserId, request.userID)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, request.model)
	common.SetContextKey(c, constant.ContextKeyChannelId, request.channelID)
	common.SetContextKey(c, constant.ContextKeyChannelType, request.channelType)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, f.upstreamURL)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, dto.ChannelOtherSettings{UpstreamEndUserIDField: request.field})
	if request.paramOverride != nil {
		common.SetContextKey(c, constant.ContextKeyChannelParamOverride, request.paramOverride)
	}
	return c
}

// chat sends a Chat Completions request through TextHelper and returns the
// body the upstream received.
func (f *endUserFixture) chat(t *testing.T, request endUserRequest, body string, channelTest bool) []byte {
	t.Helper()
	request.path, request.channelType, request.model = "/v1/chat/completions", constant.ChannelTypeOpenAI, "gpt-4o"
	if request.channelID == 0 {
		request.channelID = 1
	}
	c := f.context(request)
	var chatRequest dto.GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalJsonStr(body, &chatRequest))
	info := relaycommon.GenRelayInfoOpenAI(c, &chatRequest)
	info.IsChannelTest = channelTest
	require.Nil(t, TextHelper(c, info))
	return <-f.bodies
}

func assertUpstreamEndUserID(t *testing.T, body []byte, path string, userID int) {
	t.Helper()
	value := gjson.GetBytes(body, path)
	require.True(t, value.Exists(), "%s is written: %s", path, body)
	assert.Regexp(t, `^[0-9a-f]{32}$`, value.String())
	assert.Equal(t, expectedUpstreamEndUserID(userID), value.String())
	assert.NotContains(t, string(body), strconv.Itoa(userID), "the user id never appears in clear")
	assert.NotContains(t, string(body), "-example", "the username never appears")
}

func TestUpstreamEndUserIDChatCompletions(t *testing.T) {
	f := newEndUserFixture(t)
	const plain = `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
	const withClientFields = `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"user":"client-user","metadata":{"trace":"t1","user_id":"client-meta"},"safety_identifier":"client-safety"}`

	t.Run("setting off keeps the body as it is today", func(t *testing.T) {
		assert.Equal(t, plain, string(f.chat(t, endUserRequest{userID: endUserAlice}, plain, false)))
		assert.Equal(t,
			`{"messages":[{"content":"hi","role":"user"}],"metadata":{"trace":"t1","user_id":"client-meta"},"model":"gpt-4o","user":"client-user"}`,
			string(f.chat(t, endUserRequest{userID: endUserAlice}, withClientFields, false)))
	})
	t.Run("safety_identifier replaces the filtered client value", func(t *testing.T) {
		body := f.chat(t, endUserRequest{userID: endUserAlice, field: "safety_identifier"}, withClientFields, false)
		assertUpstreamEndUserID(t, body, "safety_identifier", endUserAlice)
		assert.Equal(t, "client-user", gjson.GetBytes(body, "user").String(), "fields not chosen are left as they are")
	})
	t.Run("user replaces the client value", func(t *testing.T) {
		body := f.chat(t, endUserRequest{userID: endUserAlice, field: "user"}, withClientFields, false)
		assertUpstreamEndUserID(t, body, "user", endUserAlice)
		assert.False(t, gjson.GetBytes(body, "safety_identifier").Exists())
	})
	t.Run("metadata.user_id keeps the other metadata keys", func(t *testing.T) {
		body := f.chat(t, endUserRequest{userID: endUserAlice, field: "metadata.user_id"}, withClientFields, false)
		assertUpstreamEndUserID(t, body, "metadata.user_id", endUserAlice)
		assert.Equal(t, "t1", gjson.GetBytes(body, "metadata.trace").String())
	})
	t.Run("metadata.user_id creates metadata", func(t *testing.T) {
		body := f.chat(t, endUserRequest{userID: endUserAlice, field: "metadata.user_id"}, plain, false)
		assertUpstreamEndUserID(t, body, "metadata.user_id", endUserAlice)
	})
	t.Run("a param_override of the same field wins", func(t *testing.T) {
		body := f.chat(t, endUserRequest{userID: endUserAlice, field: "safety_identifier", paramOverride: map[string]any{"safety_identifier": "operator-value"}}, plain, false)
		assert.Equal(t, "operator-value", gjson.GetBytes(body, "safety_identifier").String())
	})
	t.Run("same user on two channels gets one value, two users get two", func(t *testing.T) {
		first := gjson.GetBytes(f.chat(t, endUserRequest{userID: endUserAlice, channelID: 1, field: "user"}, plain, false), "user").String()
		second := gjson.GetBytes(f.chat(t, endUserRequest{userID: endUserAlice, channelID: 2, field: "user"}, plain, false), "user").String()
		other := gjson.GetBytes(f.chat(t, endUserRequest{userID: endUserBob, channelID: 1, field: "user"}, plain, false), "user").String()
		assert.Equal(t, first, second)
		assert.NotEqual(t, first, other)
		assert.Equal(t, expectedUpstreamEndUserID(endUserBob), other)
	})
	t.Run("a channel test writes nothing", func(t *testing.T) {
		assert.Equal(t, plain, string(f.chat(t, endUserRequest{userID: endUserAlice, field: "user"}, plain, true)))
	})
	t.Run("a request without a user writes nothing", func(t *testing.T) {
		assert.Equal(t, plain, string(f.chat(t, endUserRequest{field: "user"}, plain, false)))
	})
}

func TestUpstreamEndUserIDClaudeMessages(t *testing.T) {
	f := newEndUserFixture(t)
	c := f.context(endUserRequest{path: "/v1/messages", userID: endUserAlice, channelID: 1, channelType: constant.ChannelTypeAnthropic, model: "claude-opus-5-5", field: "metadata.user_id"})
	var request dto.ClaudeRequest
	require.NoError(t, common.UnmarshalJsonStr(`{"model":"claude-opus-5-5","max_tokens":16,"messages":[{"role":"user","content":"hi"}],"metadata":{"user_id":"client-meta"}}`, &request))
	info := relaycommon.GenRelayInfoClaude(c, &request)
	require.Nil(t, ClaudeHelper(c, info))
	assertUpstreamEndUserID(t, <-f.bodies, "metadata.user_id", endUserAlice)
}

func TestUpstreamEndUserIDResponses(t *testing.T) {
	f := newEndUserFixture(t)
	c := f.context(endUserRequest{path: "/v1/responses", userID: endUserAlice, channelID: 1, channelType: constant.ChannelTypeOpenAI, model: "gpt-4o", field: "safety_identifier"})
	request := &dto.OpenAIResponsesRequest{Model: "gpt-4o", Input: []byte(`"hi"`), SafetyIdentifier: []byte(`"client-safety"`)}
	info := relaycommon.GenRelayInfoResponses(c, request)
	_, body, closer, apiErr := PrepareResponsesRequest(c, info, request)
	require.Nil(t, apiErr)
	defer closer.Close()
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assertUpstreamEndUserID(t, data, "safety_identifier", endUserAlice)
}

func TestUpstreamEndUserIDChatCompletionsViaResponses(t *testing.T) {
	f := newEndUserFixture(t)
	viaResponses := func(t *testing.T, field string, paramOverride map[string]any) []byte {
		c := f.context(endUserRequest{path: "/v1/chat/completions", userID: endUserAlice, channelID: 1, channelType: constant.ChannelTypeOpenAI, model: "gpt-4o", field: field, paramOverride: paramOverride})
		var request dto.GeneralOpenAIRequest
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"user":"client-user"}`, &request))
		info := relaycommon.GenRelayInfoOpenAI(c, &request)
		info.InitChannelMeta(c)
		adaptor := GetAdaptor(info.ApiType)
		require.NotNil(t, adaptor)
		adaptor.Init(info)
		_, apiErr := textRequestViaResponses(c, info, adaptor, &request)
		require.Nil(t, apiErr)
		return <-f.bodies
	}

	t.Run("safety_identifier reaches the Responses body", func(t *testing.T) {
		assertUpstreamEndUserID(t, viaResponses(t, "safety_identifier", nil), "safety_identifier", endUserAlice)
	})
	t.Run("safety_identifier is written after a chat param_override", func(t *testing.T) {
		assertUpstreamEndUserID(t, viaResponses(t, "safety_identifier", map[string]any{"temperature": 0.5}), "safety_identifier", endUserAlice)
	})
	t.Run("a chat param_override of user wins", func(t *testing.T) {
		body := viaResponses(t, "user", map[string]any{"user": "operator-value"})
		assert.Equal(t, "operator-value", gjson.GetBytes(body, "user").String())
	})
	t.Run("user replaces the client value", func(t *testing.T) {
		assertUpstreamEndUserID(t, viaResponses(t, "user", nil), "user", endUserAlice)
	})
}
