package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// lookupModelRetirement finds a requested model on the retirement list, by
// its exact name or by the routing name channel matching falls back to (@
// modifiers and legacy aliases stripped).
func lookupModelRetirement(modelName string) (string, operation_setting.ModelRetirement, bool) {
	if retirement, ok := operation_setting.GetModelRetirement(modelName); ok {
		return modelName, retirement, true
	}
	if routingName := ratio_setting.RoutingMatchModelName(modelName); routingName != modelName {
		if retirement, ok := operation_setting.GetModelRetirement(routingName); ok {
			return routingName, retirement, true
		}
	}
	return "", operation_setting.ModelRetirement{}, false
}

func modelRetiredMessage(c *gin.Context, modelName string, retirement operation_setting.ModelRetirement) string {
	message := i18n.T(c, i18n.MsgDistributorModelRetired, map[string]any{
		"Model":     modelName,
		"RetiredAt": retirement.RetiredAt,
		"Successor": retirement.Successor,
	})
	if retirement.NoticeURL != "" {
		message += " " + i18n.T(c, i18n.MsgDistributorModelRetiredNotice, map[string]any{"NoticeURL": retirement.NoticeURL})
	}
	return message
}

// abortWithModelRetired answers a request for a retired model with HTTP 410
// in the error shape of the endpoint's dialect: Anthropic's on /v1/messages
// (as SystemPerformanceCheck decides it), Google's on the Gemini-native paths
// (relay mode Gemini), OpenAI's everywhere else. It is written directly, not
// through NewAPIError, so the notice URL is not masked.
func abortWithModelRetired(c *gin.Context, modelName string, retirement operation_setting.ModelRetirement) {
	message := modelRetiredMessage(c, modelName, retirement)
	body := common.MessageWithRequestId(message, c.GetString(common.RequestIdKey))
	path := c.Request.URL.Path
	switch {
	case strings.HasPrefix(path, "/v1/messages"):
		c.JSON(http.StatusGone, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    string(types.ErrorCodeModelRetired),
				"message": body,
			},
		})
	case relayconstant.Path2RelayMode(path) == relayconstant.RelayModeGemini:
		c.JSON(http.StatusGone, gin.H{
			"error": gin.H{
				"code":    http.StatusGone,
				"message": body,
				"status":  "NOT_FOUND",
			},
		})
	default:
		c.JSON(http.StatusGone, gin.H{
			"error": gin.H{
				"message": body,
				"type":    "new_api_error",
				"code":    string(types.ErrorCodeModelRetired),
			},
		})
	}
	c.Abort()
	logger.LogInfo(c, common.LogText("user %d | model_retired model=%q successor=%q", c.GetInt("id"), modelName, retirement.Successor))
}
