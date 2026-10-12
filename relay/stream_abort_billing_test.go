package relay

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// A stream that ends before the upstream sent any event settles at zero and
// returns its pre-consumed hold; a stream that delivered output without usage
// still settles at the local estimate.
func TestStreamAbortSettlement(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		gin.SetMode(oldMode)
		constant.StreamingTimeout = oldTimeout
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Log{}))
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
	oldCache, oldRedis := common.MemoryCacheEnabled, common.RedisEnabled
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.MemoryCacheEnabled, common.RedisEnabled = false, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = oldDB, oldLogDB
		common.SetDatabaseTypes(oldMainType, oldLogType)
		common.MemoryCacheEnabled, common.RedisEnabled = oldCache, oldRedis
		require.NoError(t, sqlDB.Close())
	})

	type streamHandler func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError)
	claudeStream := func(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
		return claude.ClaudeStreamHandler(c, resp, info)
	}
	const (
		startingQuota  = 1_000_000
		reservation    = 50_000
		estimatePrompt = 1_000
	)
	claudeOutput := strings.Join([]string{
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello there"}}`,
	}, "\n")
	chatOutput := `data: {"id":"c1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hello there"}}]}`
	responsesOutput := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"m","created_at":1710000000}}`,
		`data: {"type":"response.output_text.delta","delta":"hello there"}`,
	}, "\n")

	for index, tc := range []struct {
		name       string
		format     types.RelayFormat
		handler    streamHandler
		body       string
		clientGone bool
		charged    bool
	}{
		{name: "claude messages upstream closed before any event", format: types.RelayFormatClaude, handler: claudeStream},
		{name: "claude messages client left before any event", format: types.RelayFormatClaude, handler: claudeStream, clientGone: true},
		{name: "claude to chat upstream closed before any event", format: types.RelayFormatOpenAI, handler: claudeStream},
		{name: "claude output without usage bills the estimate", format: types.RelayFormatClaude, handler: claudeStream, body: claudeOutput, charged: true},
		{name: "chat upstream closed before any event", format: types.RelayFormatOpenAI, handler: openai.OaiStreamHandler},
		{name: "chat client left before any event", format: types.RelayFormatOpenAI, handler: openai.OaiStreamHandler, clientGone: true},
		{name: "chat output without usage bills the estimate", format: types.RelayFormatOpenAI, handler: openai.OaiStreamHandler, body: chatOutput, charged: true},
		{name: "chat via responses upstream closed before any event", format: types.RelayFormatOpenAI, handler: openai.OaiResponsesToChatStreamHandler},
		{name: "chat via responses client left before any event", format: types.RelayFormatOpenAI, handler: openai.OaiResponsesToChatStreamHandler, clientGone: true},
		{name: "chat via responses output without usage bills the estimate", format: types.RelayFormatOpenAI, handler: openai.OaiResponsesToChatStreamHandler, body: responsesOutput, charged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{Username: fmt.Sprintf("stream_abort_%d", index), AffCode: fmt.Sprintf("stream-abort-%d", index), Quota: startingQuota, Status: common.UserStatusEnabled}
			require.NoError(t, db.Create(&user).Error)
			token := model.Token{UserId: user.Id, Key: fmt.Sprintf("stream-abort-%d", index), Name: "stream-abort", RemainQuota: startingQuota, Status: common.TokenStatusEnabled}
			require.NoError(t, db.Create(&token).Error)

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			var body io.ReadCloser = io.NopCloser(strings.NewReader(tc.body))
			if tc.clientGone {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				c.Request = c.Request.WithContext(ctx)
				reader, writer := io.Pipe()
				t.Cleanup(func() { _ = writer.Close() })
				body = reader
			}
			info := &relaycommon.RelayInfo{
				UserId: user.Id, TokenId: token.Id, TokenKey: token.Key,
				OriginModelName: "stream-abort-model", UsingGroup: "default", UserGroup: "default",
				UserSetting:     dto.UserSetting{BillingPreference: "wallet_only"},
				ForcePreConsume: true, StartTime: time.Now(), IsStream: true, DisablePing: true,
				RelayFormat: tc.format, RelayMode: relayconstant.RelayModeChatCompletions,
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "stream-abort-model"},
				PriceData: hosttypes.PriceData{ModelRatio: 1, CompletionRatio: 1, CacheRatio: 0.1,
					GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
			}
			info.SetEstimatePromptTokens(estimatePrompt)
			require.Nil(t, service.PreConsumeBilling(c, reservation, info))
			held, err := model.GetUserQuota(user.Id, true)
			require.NoError(t, err)
			require.Equal(t, startingQuota-reservation, held)

			resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: body}
			usage, apiErr := tc.handler(c, info, resp)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			service.PostTextConsumeQuota(c, info, usage, nil)

			var log model.Log
			require.NoError(t, db.Where("user_id = ?", user.Id).Take(&log).Error)
			require.NoError(t, db.First(&user, user.Id).Error)
			require.NoError(t, db.First(&token, token.Id).Error)
			if !tc.charged {
				assert.Zero(t, usage.TotalTokens)
				assert.Zero(t, log.Quota)
				assert.Equal(t, startingQuota, user.Quota, "the pre-consumed hold must come back")
				assert.Equal(t, startingQuota, token.RemainQuota)
				return
			}
			assert.Equal(t, estimatePrompt, log.PromptTokens)
			assert.Positive(t, log.CompletionTokens)
			assert.Equal(t, estimatePrompt+log.CompletionTokens, log.Quota)
			assert.Equal(t, startingQuota-log.Quota, user.Quota)
		})
	}
}
