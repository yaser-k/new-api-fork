package relay

import (
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
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// A Claude refusal that arrives before any output settles at zero and returns
// the pre-consumed hold, unless its category is one Anthropic bills; a
// mid-stream refusal is billed as usual. Streamed and non-streamed responses
// settle the same amount on the Anthropic and Bedrock paths, for Messages and
// Chat Completions clients.
func TestClaudeRefusalSettlement(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	claudeSettings := model_setting.GetClaudeSettings()
	oldWaiver, oldCategories := claudeSettings.RefusalBillingWaiverEnabled, claudeSettings.RefusalBilledCategories
	t.Cleanup(func() {
		gin.SetMode(oldMode)
		constant.StreamingTimeout = oldTimeout
		claudeSettings.RefusalBillingWaiverEnabled, claudeSettings.RefusalBilledCategories = oldWaiver, oldCategories
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

	// noDetails stands for an upstream that sends no stop_details at all.
	const noDetails = "-"
	stopDetails := func(category string) string {
		switch category {
		case noDetails:
			return `null`
		case "":
			return `{"type":"refusal","category":null,"explanation":null}`
		}
		return fmt.Sprintf(`{"type":"refusal","category":%q,"explanation":"declined"}`, category)
	}
	nonStreamBody := func(category, text string, outputTokens int) string {
		content := `[]`
		if text != "" {
			content = fmt.Sprintf(`[{"type":"text","text":%q}]`, text)
		}
		return fmt.Sprintf(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-fable-5","content":%s,"stop_reason":"refusal","stop_details":%s,"usage":{"input_tokens":412,"output_tokens":%d}}`,
			content, stopDetails(category), outputTokens)
	}
	streamBody := func(category, text string, outputTokens int) string {
		lines := []string{
			`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-fable-5","content":[],"stop_reason":null,"usage":{"input_tokens":412,"output_tokens":1}}}`,
		}
		if text != "" {
			lines = append(lines,
				`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
				fmt.Sprintf(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`, text),
				`data: {"type":"content_block_stop","index":0}`,
			)
		}
		lines = append(lines,
			fmt.Sprintf(`data: {"type":"message_delta","delta":{"stop_reason":"refusal","stop_sequence":null,"stop_details":%s},"usage":{"output_tokens":%d}}`, stopDetails(category), outputTokens),
			`data: {"type":"message_stop"}`,
		)
		return strings.Join(lines, "\n")
	}
	httpResponse := func(body string) *http.Response {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
	}

	type run func(*gin.Context, *relaycommon.RelayInfo, string) (*dto.Usage, *types.NewAPIError)
	anthropic := func(c *gin.Context, info *relaycommon.RelayInfo, body string) (*dto.Usage, *types.NewAPIError) {
		return claude.ClaudeHandler(c, httpResponse(body), info)
	}
	anthropicStream := func(c *gin.Context, info *relaycommon.RelayInfo, body string) (*dto.Usage, *types.NewAPIError) {
		return claude.ClaudeStreamHandler(c, httpResponse(body), info)
	}
	responsesStream := func(c *gin.Context, info *relaycommon.RelayInfo, body string) (*dto.Usage, *types.NewAPIError) {
		return claude.ClaudeResponsesStreamHandler(c, httpResponse(body), info)
	}
	// Bedrock decodes its own event stream and feeds the shared Claude handlers.
	bedrock := func(c *gin.Context, info *relaycommon.RelayInfo, body string) (*dto.Usage, *types.NewAPIError) {
		claudeInfo := &claude.ClaudeResponseInfo{Usage: &dto.Usage{}}
		return claudeInfo.Usage, claude.HandleClaudeResponseData(c, info, claudeInfo, nil, []byte(body))
	}
	bedrockStream := func(c *gin.Context, info *relaycommon.RelayInfo, body string) (*dto.Usage, *types.NewAPIError) {
		claudeInfo := &claude.ClaudeResponseInfo{Usage: &dto.Usage{}}
		for line := range strings.SplitSeq(body, "\n") {
			if apiErr := claude.HandleStreamResponseData(c, info, claudeInfo, strings.TrimPrefix(line, "data: ")); apiErr != nil {
				return nil, apiErr
			}
		}
		claude.HandleStreamFinalResponse(c, info, claudeInfo)
		return claudeInfo.Usage, nil
	}
	transports := []struct {
		name   string
		format types.RelayFormat
		stream bool
		run    run
	}{
		{"messages", types.RelayFormatClaude, false, anthropic},
		{"messages stream", types.RelayFormatClaude, true, anthropicStream},
		{"chat", types.RelayFormatOpenAI, false, anthropic},
		{"chat stream", types.RelayFormatOpenAI, true, anthropicStream},
		{"responses stream", types.RelayFormatOpenAIResponses, true, responsesStream},
		{"bedrock messages", types.RelayFormatClaude, false, bedrock},
		{"bedrock messages stream", types.RelayFormatClaude, true, bedrockStream},
		{"bedrock chat stream", types.RelayFormatOpenAI, true, bedrockStream},
	}

	const (
		startingQuota = 1_000_000
		reservation   = 50_000
	)
	index := 0
	for _, scenario := range []struct {
		name           string
		category, text string
		outputTokens   int
		waiverOff      bool
		want           int
		wantReason     string
	}{
		{name: "unbilled category before output", category: "cyber", wantReason: "claude_refusal_before_output category=cyber"},
		{name: "null category before output", wantReason: "claude_refusal_before_output category=null"},
		{name: "billed category before output", category: "bio", want: 412},
		{name: "unknown category without stop_details", category: noDetails, want: 412},
		{name: "mid-stream refusal", category: "cyber", text: "Sure, here", outputTokens: 5, want: 417},
		{name: "waiver switched off", category: "cyber", waiverOff: true, want: 412},
	} {
		for _, transport := range transports {
			index++
			t.Run(scenario.name+"/"+transport.name, func(t *testing.T) {
				claudeSettings.RefusalBillingWaiverEnabled = !scenario.waiverOff
				claudeSettings.RefusalBilledCategories = []string{"bio", "frontier_llm", "reasoning_extraction"}

				user := model.User{Username: fmt.Sprintf("refusal_%d", index), AffCode: fmt.Sprintf("refusal-%d", index), Quota: startingQuota, Status: common.UserStatusEnabled}
				require.NoError(t, db.Create(&user).Error)
				token := model.Token{UserId: user.Id, Key: fmt.Sprintf("refusal-%d", index), Name: "refusal", RemainQuota: startingQuota, Status: common.TokenStatusEnabled}
				require.NoError(t, db.Create(&token).Error)

				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
				info := &relaycommon.RelayInfo{
					UserId: user.Id, TokenId: token.Id, TokenKey: token.Key,
					OriginModelName: "claude-fable-5", UsingGroup: "default", UserGroup: "default",
					UserSetting:     dto.UserSetting{BillingPreference: "wallet_only"},
					ForcePreConsume: true, StartTime: time.Now(), IsStream: transport.stream, DisablePing: true,
					RelayFormat: transport.format,
					ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "claude-fable-5"},
					PriceData: hosttypes.PriceData{ModelRatio: 1, CompletionRatio: 1,
						GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
				}
				info.SetEstimatePromptTokens(400)
				require.Nil(t, service.PreConsumeBilling(c, reservation, info))

				body := nonStreamBody(scenario.category, scenario.text, scenario.outputTokens)
				if transport.stream {
					body = streamBody(scenario.category, scenario.text, scenario.outputTokens)
				}
				usage, apiErr := transport.run(c, info, body)
				require.Nil(t, apiErr)
				require.NotNil(t, usage)
				service.PostTextConsumeQuota(c, info, usage, nil)

				var log model.Log
				require.NoError(t, db.Where("user_id = ?", user.Id).Take(&log).Error)
				require.NoError(t, db.First(&user, user.Id).Error)
				require.NoError(t, db.First(&token, token.Id).Error)
				assert.Equal(t, scenario.want, log.Quota)
				assert.Equal(t, startingQuota-scenario.want, user.Quota)
				assert.Equal(t, startingQuota-scenario.want, token.RemainQuota)
				assert.Equal(t, 412, log.PromptTokens, "reported tokens stay on the consume log")
				assert.Equal(t, scenario.outputTokens, log.CompletionTokens)
				var other map[string]any
				require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
				if scenario.wantReason == "" {
					assert.NotContains(t, other, "billing_exempt_reason")
					return
				}
				assert.Equal(t, scenario.wantReason, other["billing_exempt_reason"])
				const exemptKey = "Upstream did not bill this request, so nothing was charged: {{reason}}"
				assert.Equal(t, "Upstream did not bill this request, so nothing was charged: "+scenario.wantReason, log.Content)
				assert.Contains(t, other["content_parts"], map[string]any{"key": exemptKey, "params": map[string]any{"reason": scenario.wantReason}})
				assert.Equal(t, 1, user.RequestCount)
			})
		}
	}
}
