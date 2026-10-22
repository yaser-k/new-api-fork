package middleware

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const RouteTagKey = "route_tag"

// accessLogCredentialQueryNames are the query parameters, compared without
// case, whose values the access log masks: key is the Gemini-style API key
// (TokenAuth), token the key search term (SearchTokens), private_key the
// Waffo Pancake credential probe and turnstile the Turnstile response; the
// rest are common credential names a client may send on any route.
var accessLogCredentialQueryNames = map[string]struct{}{
	"key":           {},
	"token":         {},
	"private_key":   {},
	"turnstile":     {},
	"api_key":       {},
	"api-key":       {},
	"apikey":        {},
	"access_token":  {},
	"client_secret": {},
	"secret":        {},
	"password":      {},
}

func RouteTag(tag string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(RouteTagKey, tag)
		c.Next()
	}
}

func SetUpLogger(server *gin.Engine) {
	server.Use(redactTaskArtifactAccessQuery())
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var requestID string
		if param.Keys != nil {
			requestID, _ = param.Keys[common.RequestIdKey].(string)
		}
		tag, _ := param.Keys[RouteTagKey].(string)
		if tag == "" {
			tag = "web"
		}
		path := param.Path
		// OAuth callbacks carry one-time codes and state in the query string.
		// Redact the log value only; the handler still needs the original query.
		if strings.HasPrefix(path, "/api/oauth/") || strings.HasPrefix(path, "/oauth/") {
			path, _, _ = strings.Cut(path, "?")
		} else if query, hasQuery := strings.CutPrefix(path, param.Request.URL.Path+"?"); hasQuery {
			path = param.Request.URL.Path + "?" + maskAccessLogQueryCredentials(query)
		} else if path != param.Request.URL.Path {
			// A handler changed the request path, so the logged query can no
			// longer be told apart from the path; drop it rather than guess.
			path, _, _ = strings.Cut(path, "?")
		}
		return fmt.Sprintf("[GIN] %s | %s | %s | %3d | %13v | %15s | %7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			tag,
			requestID,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			path,
		)
	}))
}

// maskAccessLogQueryCredentials replaces credential values in a raw query and
// keeps every other byte, so the remaining parameters read as the client sent
// them. A pair the server's query parser rejects (a bad escape or a semicolon)
// is masked as well, because its name cannot be trusted.
func maskAccessLogQueryCredentials(rawQuery string) string {
	var masked strings.Builder
	for i, pair := range strings.Split(rawQuery, "&") {
		if i > 0 {
			masked.WriteByte('&')
		}
		rawName, rawValue, _ := strings.Cut(pair, "=")
		name, nameErr := url.QueryUnescape(rawName)
		_, valueErr := url.QueryUnescape(rawValue)
		_, isCredential := accessLogCredentialQueryNames[strings.ToLower(name)]
		parsable := nameErr == nil && valueErr == nil && !strings.Contains(pair, ";")
		if rawValue == "" || (parsable && !isCredential) {
			masked.WriteString(pair)
			continue
		}
		masked.WriteString(rawName)
		masked.WriteString("=***masked***")
	}
	return masked.String()
}
