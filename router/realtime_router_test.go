package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/aws"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockruntimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noNetworkHTTPClient struct{ requests *atomic.Int32 }

func (c noNetworkHTTPClient) Do(*http.Request) (*http.Response, error) {
	c.requests.Add(1)
	return nil, errors.New("the test allows no network")
}

type openSonicWriter struct{}

func (openSonicWriter) Send(context.Context, bedrockruntimeTypes.InvokeModelWithBidirectionalStreamInput) error {
	return nil
}
func (openSonicWriter) Close() error { return nil }
func (openSonicWriter) Err() error   { return nil }

type openSonicReader struct {
	events chan bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutput
	once   *atomic.Bool
}

func (r openSonicReader) Events() <-chan bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutput {
	return r.events
}
func (r openSonicReader) Close() error {
	if r.once.CompareAndSwap(false, true) {
		close(r.events)
	}
	return nil
}
func (r openSonicReader) Err() error { return nil }

// A realtime session for Nova Sonic, through the router, its middleware, controller.Relay, WssHelper and
// the AWS adaptor, on an AWS channel of the key's kind. Bedrock is never reached: the stream is opened by
// the SDK with no network, or stands open as a stub.
func TestRealtimeRouteNovaSonicOnAWSChannelKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
		// throughSDK opens the stream with the SDK, as on a real channel, with no network.
		throughSDK bool
		wantType   string
		wantError  string
	}{
		// The SDK fails a bearer-token call before sending it and never returns: the session hung, silent.
		{name: "Bedrock API key", key: "bedrock-api-key|us-east-1", throughSDK: true, wantType: "error",
			wantError: "Nova Sonic on the realtime route needs an AWS channel keyed with an access key (AK|SK|region): Bedrock's bidirectional stream signs every event with SigV4, which a Bedrock API key cannot"},
		{name: "access key", key: "AKIDEXAMPLE|secret|us-east-1", wantType: "session.created"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRelayRouterTestDB(t)
			require.NoError(t, model.DB.AutoMigrate(&model.Channel{}, &model.Log{}, &model.UserSubscription{}))
			saved := ratio_setting.ModelRatio2JSONString()
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"nova-2-5-sonic":1}`))
			t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(saved)) })
			user := model.User{Username: "realtime-user", Status: common.UserStatusEnabled, Group: "private", Quota: 1_000_000}
			require.NoError(t, model.DB.Create(&user).Error)
			require.NoError(t, model.DB.Create(&model.Token{UserId: user.Id, Key: "realtimetestkey", Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true}).Error)
			channel := &model.Channel{Type: constant.ChannelTypeAws, Name: "nova-sonic", Key: tt.key, Status: common.ChannelStatusEnabled, Models: "nova-2-5-sonic", Group: "private"}
			require.NoError(t, model.DB.Create(channel).Error)
			require.NoError(t, channel.AddAbilities(nil))

			var opened, requests atomic.Int32
			savedOpen := aws.OpenNovaSonicStream
			t.Cleanup(func() { aws.OpenNovaSonicStream = savedOpen })
			aws.OpenNovaSonicStream = func(ctx context.Context, client *bedrockruntime.Client, modelID string) (*bedrockruntime.InvokeModelWithBidirectionalStreamEventStream, error) {
				opened.Add(1)
				if !tt.throughSDK {
					return bedrockruntime.NewInvokeModelWithBidirectionalStreamEventStream(func(es *bedrockruntime.InvokeModelWithBidirectionalStreamEventStream) {
						es.Writer = openSonicWriter{}
						es.Reader = openSonicReader{events: make(chan bedrockruntimeTypes.InvokeModelWithBidirectionalStreamOutput), once: &atomic.Bool{}}
					}), nil
				}
				endpoint := "http://127.0.0.1:1"
				output, err := client.InvokeModelWithBidirectionalStream(ctx, &bedrockruntime.InvokeModelWithBidirectionalStreamInput{ModelId: &modelID}, func(o *bedrockruntime.Options) {
					o.BaseEndpoint = &endpoint
					o.HTTPClient = noNetworkHTTPClient{requests: &requests}
				})
				if err != nil {
					return nil, err
				}
				return output.GetStream(), nil
			}

			engine := gin.New()
			SetRelayRouter(engine)
			served := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				engine.ServeHTTP(w, r)
				served <- struct{}{}
			}))
			t.Cleanup(server.Close)
			client, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/v1/realtime?model=nova-2-5-sonic",
				http.Header{"Authorization": {"Bearer sk-realtimetestkey"}})
			require.NoError(t, err)
			assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)

			// The first event comes at once; a session that sends nothing has hung.
			require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))
			_, message, err := client.ReadMessage()
			require.NoError(t, err, "the session sent nothing")
			var event map[string]any
			require.NoError(t, common.Unmarshal(message, &event))
			assert.Equal(t, tt.wantType, event["type"])
			if tt.wantError != "" {
				assert.Contains(t, event["error"].(map[string]any)["message"], tt.wantError)
				assert.Zero(t, opened.Load(), "a Bedrock API key channel opens no stream")
				// The refused session ends: the gateway closes the socket.
				_, _, err = client.ReadMessage()
				assert.Error(t, err)
			} else {
				assert.Equal(t, int32(1), opened.Load())
			}
			assert.Zero(t, requests.Load())

			// Once the client has gone, the session's handler returns.
			_ = client.Close()
			select {
			case <-served:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the session's handler did not return")
			}
		})
	}
}
