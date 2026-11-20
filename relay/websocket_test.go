package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The ratios of realtime sessions are priced for acme-voice only; clients ask for acme-voice-live.
const realtimeBillingModel, realtimeRequestedModel = "acme-voice", "acme-voice-live"

// realtimeSession is the relay of one realtime session through WssHelper, with the database it settles in.
type realtimeSession struct {
	database *gorm.DB
	user     *model.User
	token    *model.Token
	// upstreamModels receives the model each upstream connection asked for in its query.
	upstreamModels chan string
	upstreamURL    string
}

// newRealtimeSession prepares the database, a user and its token, the ratios priced for acme-voice, and an
// upstream that answers one turn: 10 text and 100 audio tokens in, 20 text and 200 audio tokens out.
func newRealtimeSession(t *testing.T) *realtimeSession {
	gin.SetMode(gin.TestMode)
	session := &realtimeSession{database: setupRelayChannelDB(t), upstreamModels: make(chan string, 1)}
	require.NoError(t, session.database.AutoMigrate(&model.User{}, &model.Token{}, &model.Log{}))
	previousLogDB, previousRedis, previousLogConsume := model.LOG_DB, common.RedisEnabled, common.LogConsumeEnabled
	// The log database is the main one, as with no LOG_SQL_DSN; this also names its columns for SQLite.
	t.Setenv("LOG_SQL_DSN", "")
	require.NoError(t, model.InitLogDB())
	common.RedisEnabled, common.LogConsumeEnabled = false, true
	t.Cleanup(func() {
		model.LOG_DB, common.RedisEnabled, common.LogConsumeEnabled = previousLogDB, previousRedis, previousLogConsume
	})
	for _, ratios := range []struct {
		saved  string
		update func(string) error
		value  string
	}{
		{ratio_setting.CompletionRatio2JSONString(), ratio_setting.UpdateCompletionRatioByJSONString, `{"acme-voice":2}`},
		{ratio_setting.AudioRatio2JSONString(), ratio_setting.UpdateAudioRatioByJSONString, `{"acme-voice":3}`},
		{ratio_setting.AudioCompletionRatio2JSONString(), ratio_setting.UpdateAudioCompletionRatioByJSONString, `{"acme-voice":4}`},
	} {
		require.NoError(t, ratios.update(ratios.value))
		t.Cleanup(func() { require.NoError(t, ratios.update(ratios.saved)) })
	}
	session.user = &model.User{Username: "realtime-billing", Quota: 1_000_000, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, session.database.Create(session.user).Error)
	session.token = &model.Token{UserId: session.user.Id, Key: "realtimebillingkey", Name: "realtime", Status: common.TokenStatusEnabled, UnlimitedQuota: true, ExpiredTime: -1}
	require.NoError(t, session.database.Create(session.token).Error)

	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		session.upstreamModels <- r.URL.Query().Get("model")
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.done","response":{"usage":{"total_tokens":330,"input_tokens":110,"output_tokens":220,`+
			`"input_token_details":{"text_tokens":10,"audio_tokens":100},"output_token_details":{"text_tokens":20,"audio_tokens":200}}}}`))
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(upstream.Close)
	session.upstreamURL = upstream.URL
	return session
}

// run relays one session for acme-voice-live on a channel of channelType at baseURL, as the distributor
// selected it, with the channel's model mapping and the billing name ModelPriceHelper resolved.
func (s *realtimeSession) run(t *testing.T, channelType int, baseURL, modelMapping, billingModel string) {
	results := make(chan *types.NewAPIError, 1)
	upgrader := websocket.Upgrader{}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer clientConn.Close()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r
		c.Set("token_name", s.token.Name)
		c.Set("model_mapping", modelMapping)
		common.SetContextKey(c, constant.ContextKeyChannelType, channelType)
		common.SetContextKey(c, constant.ContextKeyChannelId, 9401)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, baseURL)
		common.SetContextKey(c, constant.ContextKeyChannelKey, "sk-upstream")
		common.SetContextKey(c, constant.ContextKeyOriginalModel, realtimeRequestedModel)
		info := &relaycommon.RelayInfo{
			RelayMode:        relayconstant.RelayModeRealtime,
			RelayFormat:      types.RelayFormatOpenAIRealtime,
			RequestURLPath:   r.URL.String(),
			OriginModelName:  realtimeRequestedModel,
			BillingModelName: billingModel,
			UserId:           s.user.Id,
			TokenId:          s.token.Id,
			TokenKey:         "sk-" + s.token.Key,
			UsingGroup:       "default",
			StartTime:        time.Now(),
			ClientWs:         clientConn,
			PriceData:        hosttypes.PriceData{ModelRatio: 1, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
		}
		results <- WssHelper(c, info)
	}))
	t.Cleanup(gateway.Close)

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(gateway.URL, "http")+"/v1/realtime?model="+realtimeRequestedModel, nil)
	require.NoError(t, err)
	require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, message, err := client.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(message), `"response.done"`)
	require.NoError(t, client.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
	_ = client.Close()
	select {
	case apiErr := <-results:
		require.Nil(t, apiErr)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the realtime session did not end")
	}
}

func (s *realtimeSession) consumeLog(t *testing.T) (model.Log, map[string]any) {
	var log model.Log
	require.NoError(t, s.database.Where("user_id = ? AND type = ?", s.user.Id, model.LogTypeConsume).Take(&log).Error)
	var other map[string]any
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	return log, other
}

// A realtime session is settled at the prices of its billing model: the model ratio of the billing name
// and its completion, audio and audio completion ratios, not those of the name sent upstream.
func TestWssHelperSettlesAtTheBillingModelRatios(t *testing.T) {
	session := newRealtimeSession(t)
	// The billing name is what ModelPriceHelper sets when the requested name has no price and acme-voice has.
	session.run(t, constant.ChannelTypeOpenAI, session.upstreamURL, "", realtimeBillingModel)

	log, other := session.consumeLog(t)
	// 10 text in + 20 text out x 2 + 100 audio in x 3 + 200 audio out x 3 x 4, at model and group ratio 1.
	assert.Equal(t, 2750, log.Quota)
	// The log shows the ratios the quota was computed with.
	assert.Equal(t, float64(2), other["completion_ratio"])
	assert.Equal(t, float64(3), other["audio_ratio"])
	assert.Equal(t, float64(4), other["audio_completion_ratio"])
}

// A channel's model mapping applies on the realtime route: the adaptor builds the upstream request for the
// mapped name, while the session is still settled at the prices of the name the client asked for.
func TestWssHelperAppliesTheChannelModelMapping(t *testing.T) {
	session := newRealtimeSession(t)
	// A Custom channel's realtime URL names the upstream model through its {model} placeholder.
	session.run(t, constant.ChannelTypeCustom, session.upstreamURL+"/v1/realtime?model={model}",
		`{"acme-voice-live":"acme-voice-2026-10"}`, realtimeBillingModel)

	assert.Equal(t, "acme-voice-2026-10", <-session.upstreamModels)
	log, _ := session.consumeLog(t)
	assert.Equal(t, 2750, log.Quota)
	assert.Equal(t, realtimeBillingModel, log.ModelName)
}
