package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUpLoggerMasksQueryCredentialsWithoutChangingRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousWriter := gin.DefaultWriter
	t.Cleanup(func() { gin.DefaultWriter = previousWriter })

	testCases := []struct {
		name        string
		method      string
		target      string
		status      int
		rewritePath string
		logged      string
		secrets     []string
	}{
		{
			name: "Gemini key accepted", method: http.MethodPost, status: http.StatusOK,
			target:  "/v1beta/models/gemini-test:generateContent?key=gemini-secret&alt=sse",
			logged:  "/v1beta/models/gemini-test:generateContent?key=***masked***&alt=sse",
			secrets: []string{"gemini-secret"},
		},
		{
			name: "Gemini key rejected", method: http.MethodPost, status: http.StatusUnauthorized,
			target:  "/v1beta/models/gemini-test:generateContent?alt=sse&key=rejected-secret",
			logged:  "/v1beta/models/gemini-test:generateContent?alt=sse&key=***masked***",
			secrets: []string{"rejected-secret"},
		},
		{
			name: "Gemini OpenAI-compatible models", method: http.MethodGet, status: http.StatusOK,
			target:  "/v1beta/openai/models?key=openai-compatible-secret",
			logged:  "/v1beta/openai/models?key=***masked***",
			secrets: []string{"openai-compatible-secret"},
		},
		{
			name: "model directory", method: http.MethodGet, status: http.StatusUnauthorized,
			target:  "/v1/models?key=directory-secret",
			logged:  "/v1/models?key=***masked***",
			secrets: []string{"directory-secret"},
		},
		{
			name: "single model", method: http.MethodGet, status: http.StatusOK,
			target:  "/v1/models/gemini-test?key=model-secret",
			logged:  "/v1/models/gemini-test?key=***masked***",
			secrets: []string{"model-secret"},
		},
		{
			name: "repeated key", method: http.MethodPost, status: http.StatusOK,
			target:  "/v1beta/models/gemini-test:streamGenerateContent?key=first-secret&alt=sse&key=second-secret",
			logged:  "/v1beta/models/gemini-test:streamGenerateContent?key=***masked***&alt=sse&key=***masked***",
			secrets: []string{"first-secret", "second-secret"},
		},
		{
			name: "percent-encoded value", method: http.MethodGet, status: http.StatusOK,
			target:  "/v1/models?key=encoded%2Dvalue%2Dsecret",
			logged:  "/v1/models?key=***masked***",
			secrets: []string{"encoded%2Dvalue%2Dsecret", "encoded-value-secret", "value%2Dsecret"},
		},
		{
			name: "percent-encoded name", method: http.MethodGet, status: http.StatusUnauthorized,
			target:  "/v1/models?%6Bey=encoded-name-secret",
			logged:  "/v1/models?%6Bey=***masked***",
			secrets: []string{"encoded-name-secret"},
		},
		{
			name: "key search", method: http.MethodGet, status: http.StatusOK,
			target:  "/api/token/search?keyword=prod&token=sk-search-secret&p=1",
			logged:  "/api/token/search?keyword=prod&token=***masked***&p=1",
			secrets: []string{"sk-search-secret"},
		},
		{
			name: "Waffo Pancake credential probe", method: http.MethodGet, status: http.StatusOK,
			target:  "/api/option/waffo-pancake/catalog?merchant_id=merchant-1&private_key=pancake-private-secret",
			logged:  "/api/option/waffo-pancake/catalog?merchant_id=merchant-1&private_key=***masked***",
			secrets: []string{"pancake-private-secret"},
		},
		{
			name: "Turnstile response", method: http.MethodGet, status: http.StatusOK,
			target:  "/api/verification?email=user%40example.com&turnstile=turnstile-secret",
			logged:  "/api/verification?email=user%40example.com&turnstile=***masked***",
			secrets: []string{"turnstile-secret"},
		},
		{
			name: "escaped hash in path", method: http.MethodPost, status: http.StatusUnauthorized,
			target:  "/v1beta/models/gemini%23test:generateContent?key=hash-path-secret",
			logged:  "/v1beta/models/gemini#test:generateContent?key=***masked***",
			secrets: []string{"hash-path-secret"},
		},
		{
			name: "escaped percent in path", method: http.MethodPost, status: http.StatusUnauthorized,
			target:  "/v1beta/models/gemini%25test:generateContent?key=percent-path-secret",
			logged:  "/v1beta/models/gemini%test:generateContent?key=***masked***",
			secrets: []string{"percent-path-secret"},
		},
		{
			name: "raw hash before key", method: http.MethodGet, status: http.StatusUnauthorized,
			target:  "/v1/models?alt=#sse&key=after-hash-secret",
			logged:  "/v1/models?alt=#sse&key=***masked***",
			secrets: []string{"after-hash-secret"},
		},
		{
			name: "invalid escape in key value", method: http.MethodGet, status: http.StatusUnauthorized,
			target:  "/v1/models?key=bad-escape-secret%ZZ&alt=sse",
			logged:  "/v1/models?key=***masked***&alt=sse",
			secrets: []string{"bad-escape-secret"},
		},
		{
			name: "invalid escape in parameter name", method: http.MethodGet, status: http.StatusUnauthorized,
			target:  "/v1/models?alt=sse&%ZZkey=bad-name-secret",
			logged:  "/v1/models?alt=sse&%ZZkey=***masked***",
			secrets: []string{"bad-name-secret"},
		},
		{
			name: "semicolon inside a pair", method: http.MethodGet, status: http.StatusUnauthorized,
			target:  "/v1/models?alt=sse;key=semicolon-secret&p=1",
			logged:  "/v1/models?alt=***masked***&p=1",
			secrets: []string{"semicolon-secret"},
		},
		{
			name: "handler rewrites the path", method: http.MethodGet, status: http.StatusOK,
			target:      "/v1/models?key=rewritten-path-secret&alt=sse",
			rewritePath: "/v1/models/rewritten",
			logged:      "/v1/models",
			secrets:     []string{"rewritten-path-secret"},
		},
		{
			name: "ordinary parameters unchanged", method: http.MethodGet, status: http.StatusOK,
			target: "/v1/models?alt=sse&p=1&api-version=2024-02-01",
			logged: "/v1/models?alt=sse&p=1&api-version=2024-02-01",
		},
		{
			name: "escaped hash in path unchanged", method: http.MethodPost, status: http.StatusOK,
			target: "/v1beta/models/gemini%23test:generateContent?alt=sse",
			logged: "/v1beta/models/gemini#test:generateContent?alt=sse",
		},
		{
			name: "escaped percent in path unchanged", method: http.MethodPost, status: http.StatusOK,
			target: "/v1beta/models/gemini%25test:generateContent?alt=sse",
			logged: "/v1beta/models/gemini%test:generateContent?alt=sse",
		},
		{
			name: "escaped question mark in path unchanged", method: http.MethodGet, status: http.StatusOK,
			target: "/v1/models/gemini%3Fkey=path-text?alt=sse",
			logged: "/v1/models/gemini?key=path-text?alt=sse",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			gin.DefaultWriter = &output
			router := gin.New()
			SetUpLogger(router)
			router.Use(RouteTag("relay"))
			request := httptest.NewRequest(tc.method, tc.target, nil)
			originalQuery := request.URL.RawQuery
			handled := false
			router.NoRoute(func(c *gin.Context) {
				handled = true
				assert.Equal(t, originalQuery, c.Request.URL.RawQuery)
				assert.Equal(t, tc.target, c.Request.RequestURI)
				if tc.rewritePath != "" {
					c.Request.URL.Path = tc.rewritePath
				}
				c.Status(tc.status)
			})

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.True(t, handled)
			assert.Equal(t, tc.status, response.Code)
			line := strings.TrimSuffix(output.String(), "\n")
			require.NotContains(t, line, "\n")
			assert.Contains(t, line, "| relay |")
			assert.True(t, strings.HasSuffix(line, " "+tc.method+" "+tc.logged), line)
			for _, secret := range tc.secrets {
				assert.NotContains(t, line, secret)
			}
		})
	}
}
