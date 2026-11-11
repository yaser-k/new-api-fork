package channel_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpstreamRequestEndsWhenTheClientLeaves(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.InitHttpClient()
	for _, tc := range []struct {
		name string
		do   func(channel.Adaptor, *gin.Context, *relaycommon.RelayInfo, io.Reader) (*http.Response, error)
	}{
		{name: "api request", do: channel.DoApiRequest},
		{name: "form request", do: channel.DoFormRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cancelled := make(chan struct{}, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				select {
				case <-r.Context().Done():
					cancelled <- struct{}{}
				case <-time.After(2 * time.Second):
					_, _ = io.WriteString(w, `{}`)
				}
			}))
			defer upstream.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(ctx)
			info := &relaycommon.RelayInfo{
				RelayMode:      relayconstant.RelayModeChatCompletions,
				RequestURLPath: "/v1/chat/completions",
				ChannelMeta:    &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: upstream.URL, ApiKey: "test-key"},
			}
			time.AfterFunc(50*time.Millisecond, cancel)

			resp, err := tc.do(&openai.Adaptor{}, c, info, strings.NewReader(`{"model":"gpt-test"}`))
			if resp != nil {
				_ = resp.Body.Close()
			}
			require.Error(t, err, "the upstream request follows the client's context")
			select {
			case <-cancelled:
			case <-time.After(time.Second):
				t.Fatal("the upstream never saw its request cancelled")
			}
		})
	}
}
