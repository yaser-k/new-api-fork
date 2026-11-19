package types

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasUpstreamErrorCodeByConstructor(t *testing.T) {
	upstream := WithOpenAIError(OpenAIError{Code: "rate_limit_exceeded", Message: "Slow down"}, http.StatusTooManyRequests)
	for _, tc := range []struct {
		name     string
		err      *NewAPIError
		wantCode ErrorCode
		want     bool
	}{
		{"upstream string code", upstream, "rate_limit_exceeded", true},
		{"upstream numeric code", WithOpenAIError(OpenAIError{Code: 20001, Message: "Quota exhausted"}, http.StatusForbidden), "20001", true},
		{"missing upstream code falls back to unknown_error", WithOpenAIError(OpenAIError{Message: "Failed"}, http.StatusBadGateway), "unknown_error", false},
		{"empty upstream code", WithOpenAIError(OpenAIError{Code: "", Message: "Failed"}, http.StatusBadGateway), "", false},
		{"Claude error type", WithClaudeError(ClaudeError{Type: "overloaded_error", Message: "Overloaded"}, http.StatusInternalServerError), "overloaded_error", true},
		{"Claude error without type falls back to upstream_error", WithClaudeError(ClaudeError{Message: "Failed"}, http.StatusInternalServerError), "upstream_error", false},
		{"InitOpenAIError", InitOpenAIError(ErrorCodeBadResponseStatusCode, http.StatusBadGateway), ErrorCodeBadResponseStatusCode, false},
		{"NewOpenAIError", NewOpenAIError(errors.New("Failed"), ErrorCodeBadResponseStatusCode, http.StatusBadGateway), ErrorCodeBadResponseStatusCode, false},
		{"NewOpenAIError keeps a wrapped upstream code", NewOpenAIError(fmt.Errorf("stream: %w", upstream), ErrorCodeBadResponseBody, http.StatusBadGateway), "rate_limit_exceeded", true},
		{"NewErrorWithStatusCode", NewErrorWithStatusCode(errors.New("Failed"), ErrorCodeInsufficientUserQuota, http.StatusForbidden), ErrorCodeInsufficientUserQuota, false},
		{"local code option", WithOpenAIError(OpenAIError{Code: "blocked", Message: "Blocked"}, http.StatusForbidden, ErrOptionWithLocalErrorCode()), "blocked", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantCode, tc.err.GetErrorCode())
			assert.Equal(t, tc.want, tc.err.HasUpstreamErrorCode())
		})
	}
}

// A Claude-shaped answer (/v1/messages) carries an Anthropic error type even
// when the relay error behind it is OpenAI-shaped.
func TestToClaudeErrorAnswersAnAnthropicErrorType(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  *NewAPIError
		want ClaudeError
	}{
		{
			name: "upstream error without code or type, 400",
			err:  WithOpenAIError(OpenAIError{Message: "temperature: range: 0..1"}, http.StatusBadRequest),
			want: ClaudeError{Type: "invalid_request_error", Message: "temperature: range: 0..1"},
		},
		{
			name: "upstream Anthropic error parsed as OpenAI-shaped keeps its type",
			err:  WithOpenAIError(OpenAIError{Message: "temperature: range: 0..1", Type: "invalid_request_error"}, http.StatusBadRequest),
			want: ClaudeError{Type: "invalid_request_error", Message: "temperature: range: 0..1"},
		},
		{
			name: "own Anthropic type wins over the status",
			err:  WithOpenAIError(OpenAIError{Message: "Overloaded", Type: "overloaded_error"}, http.StatusServiceUnavailable),
			want: ClaudeError{Type: "overloaded_error", Message: "Overloaded"},
		},
		{
			name: "non-Anthropic string code, 401",
			err:  WithOpenAIError(OpenAIError{Message: "Incorrect API key provided", Code: "invalid_api_key"}, http.StatusUnauthorized),
			want: ClaudeError{Type: "authentication_error", Message: "Incorrect API key provided"},
		},
		{
			name: "numeric code, 402",
			err:  WithOpenAIError(OpenAIError{Message: "insufficient credits", Code: float64(402)}, http.StatusPaymentRequired),
			want: ClaudeError{Type: "billing_error", Message: "insufficient credits"},
		},
		{
			name: "403",
			err:  WithOpenAIError(OpenAIError{Message: "forbidden"}, http.StatusForbidden),
			want: ClaudeError{Type: "permission_error", Message: "forbidden"},
		},
		{
			name: "404",
			err:  WithOpenAIError(OpenAIError{Message: "no such model"}, http.StatusNotFound),
			want: ClaudeError{Type: "not_found_error", Message: "no such model"},
		},
		{
			name: "413",
			err:  WithOpenAIError(OpenAIError{Message: "too large"}, http.StatusRequestEntityTooLarge),
			want: ClaudeError{Type: "request_too_large", Message: "too large"},
		},
		{
			name: "429",
			err:  WithOpenAIError(OpenAIError{Message: "slow down", Code: "rate_limit_exceeded"}, http.StatusTooManyRequests),
			want: ClaudeError{Type: "rate_limit_error", Message: "slow down"},
		},
		{
			name: "529",
			err:  WithOpenAIError(OpenAIError{Message: "busy"}, 529),
			want: ClaudeError{Type: "overloaded_error", Message: "busy"},
		},
		{
			name: "503",
			err:  WithOpenAIError(OpenAIError{Message: "unavailable"}, http.StatusServiceUnavailable),
			want: ClaudeError{Type: "api_error", Message: "unavailable"},
		},
		{
			name: "error new-api builds itself, 500",
			err:  NewOpenAIError(errors.New("dial tcp: connection refused"), ErrorCodeDoRequestFailed, http.StatusInternalServerError),
			want: ClaudeError{Type: "api_error", Message: "dial tcp: connection refused"},
		},
		{
			name: "Claude-shaped error passes as it is",
			err:  WithClaudeError(ClaudeError{Type: "overloaded_error", Message: "Overloaded"}, 529),
			want: ClaudeError{Type: "overloaded_error", Message: "Overloaded"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.ToClaudeError()
			assert.Equal(t, tc.want, got)
			assert.NotEqual(t, "<nil>", got.Type)
			assert.NotEmpty(t, got.Type)
		})
	}
}

func TestToOpenAIErrorKeepsTheRelayError(t *testing.T) {
	relayError := OpenAIError{Message: "temperature: range: 0..1", Type: "invalid_request_error"}
	err := WithOpenAIError(relayError, http.StatusBadRequest)

	assert.Equal(t, relayError, err.ToOpenAIError())
}
