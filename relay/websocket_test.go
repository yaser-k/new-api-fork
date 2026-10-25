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
)

// A realtime session is settled at the prices of its billing model: the model ratio of the billing name
// and its completion, audio and audio completion ratios, not those of the name sent upstream.
func TestWssHelperSettlesAtTheBillingModelRatios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupRelayChannelDB(t)
	require.NoError(t, database.AutoMigrate(&model.User{}, &model.Token{}, &model.Log{}))
	previousLogDB, previousRedis, previousLogConsume := model.LOG_DB, common.RedisEnabled, common.LogConsumeEnabled
	// The log database is the main one, as with no LOG_SQL_DSN; this also names its columns for SQLite.
	t.Setenv("LOG_SQL_DSN", "")
	require.NoError(t, model.InitLogDB())
	common.RedisEnabled, common.LogConsumeEnabled = false, true
	t.Cleanup(func() {
		model.LOG_DB, common.RedisEnabled, common.LogConsumeEnabled = previousLogDB, previousRedis, previousLogConsume
	})
	// The ratios are priced for the billing name only.
	const billingModel, requestedModel = "acme-voice", "acme-voice-live"
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
	user := &model.User{Username: "realtime-billing", Quota: 1_000_000, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, database.Create(user).Error)
	token := &model.Token{UserId: user.Id, Key: "realtimebillingkey", Name: "realtime", Status: common.TokenStatusEnabled, UnlimitedQuota: true, ExpiredTime: -1}
	require.NoError(t, database.Create(token).Error)

	// The upstream answers one turn: 10 text and 100 audio tokens in, 20 text and 200 audio tokens out.
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.done","response":{"usage":{"total_tokens":330,"input_tokens":110,"output_tokens":220,`+
			`"input_token_details":{"text_tokens":10,"audio_tokens":100},"output_token_details":{"text_tokens":20,"audio_tokens":200}}}}`))
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(upstream.Close)

	results := make(chan *types.NewAPIError, 1)
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer clientConn.Close()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r
		c.Set("token_name", token.Name)
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
		common.SetContextKey(c, constant.ContextKeyChannelId, 9401)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
		common.SetContextKey(c, constant.ContextKeyChannelKey, "sk-upstream")
		common.SetContextKey(c, constant.ContextKeyOriginalModel, requestedModel)
		info := &relaycommon.RelayInfo{
			RelayMode:       relayconstant.RelayModeRealtime,
			RelayFormat:     types.RelayFormatOpenAIRealtime,
			OriginModelName: requestedModel,
			// What ModelPriceHelper sets when the requested name has no price and the billing name has.
			BillingModelName: billingModel,
			UserId:           user.Id,
			TokenId:          token.Id,
			TokenKey:         "sk-" + token.Key,
			UsingGroup:       "default",
			StartTime:        time.Now(),
			ClientWs:         clientConn,
			PriceData:        hosttypes.PriceData{ModelRatio: 1, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
		}
		results <- WssHelper(c, info)
	}))
	t.Cleanup(gateway.Close)

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(gateway.URL, "http")+"/v1/realtime?model="+requestedModel, nil)
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

	var log model.Log
	require.NoError(t, database.Where("user_id = ? AND type = ?", user.Id, model.LogTypeConsume).Take(&log).Error)
	// 10 text in + 20 text out x 2 + 100 audio in x 3 + 200 audio out x 3 x 4, at model and group ratio 1.
	assert.Equal(t, 2750, log.Quota)
	var other map[string]any
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	assert.Equal(t, float64(2), other["completion_ratio"])
	assert.Equal(t, float64(4), other["audio_completion_ratio"])
}
