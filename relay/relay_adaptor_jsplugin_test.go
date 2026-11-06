package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	jspluginadaptor "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorMapsMigratedPlatformsToFactoryPlugins(t *testing.T) {
	platforms := []constant.TaskPlatform{
		constant.TaskPlatformSuno,
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeAli)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeGemini)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeMiniMax)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeJimeng)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKling)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVidu)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSora)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeOpenAI)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVertexAi)),
	}
	for _, platform := range platforms {
		_, isJS := GetTaskAdaptor(platform).(*jspluginadaptor.TaskAdaptor)
		assert.True(t, isJS, "platform %s should use its factory plugin", platform)
	}
}

func TestGetTaskAdaptorUsesPlatformAsThirdPartyPluginKey(t *testing.T) {
	t.Cleanup(func() {
		pluginruntime.DefaultRegistry.Unregister("registry-fallback")
	})
	source := `
export const meta = {apiVersion: 1, key: "registry-fallback", name: "Registry Fallback", version: "1.0.0", author: {name: "Test"}, channelTypes: [1999], models: ["fallback-v1"], fetchMode: "per_task"};
export function buildSubmitRequest(ctx) { return {url: ctx.baseUrl + "/submit"}; }
export function parseSubmitResponse(ctx, resp) { return {taskId: "id", taskData: resp.body}; }
export function buildQueryRequest(ctx) { return {url: ctx.baseUrl + "/tasks/" + ctx.taskId}; }
export function parseTaskResult(ctx, body) { return {taskId: body.id, status: "SUCCESS"}; }
`
	_, err := pluginruntime.DefaultRegistry.Register(source, pluginruntime.Options{})
	require.NoError(t, err)

	adaptor := GetTaskAdaptor(constant.TaskPlatform("registry-fallback"))
	require.NotNil(t, adaptor)
	assert.Equal(t, "Registry Fallback", adaptor.GetChannelName())
}

func TestGetTaskAdaptorReturnsNilForUnknownPlatform(t *testing.T) {
	assert.Nil(t, GetTaskAdaptor(constant.TaskPlatform("missing-task-platform")))
}

func TestGetTaskAdaptorForRequestUsesExactPinnedPlugin(t *testing.T) {
	source := `
export const meta = {apiVersion: 1, key: "pinned-request", name: "Pinned Generation", version: "1.0.0", author: {name: "Test"}, models: ["pinned-v1"], fetchMode: "per_task"};
export function buildSubmitRequest(ctx) { return {url: ctx.baseUrl + "/submit"}; }
export function parseSubmitResponse(ctx) { return {taskId: "one"}; }
export function buildQueryRequest(ctx) { return {url: ctx.baseUrl + "/query"}; }
export function parseTaskResult() { return {status: "SUCCESS"}; }
`
	pinned, err := pluginruntime.NewRegistry().Register(source, pluginruntime.Options{})
	require.NoError(t, err)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/vendor/submit", nil)
	c.Set(pluginruntime.ContextKeyPinnedPlugin, pluginruntime.PinnedPlugin{Plugin: pinned})

	platform, adaptor := getTaskAdaptorForRequest(c, constant.TaskPlatform("missing-task-platform"))
	require.NotNil(t, adaptor)
	assert.Equal(t, constant.TaskPlatform("pinned-request"), platform)
	assert.Equal(t, "Pinned Generation", adaptor.GetChannelName())
}

func TestGetTaskAdaptorForRequestPinsLegacyMappedPlugin(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/video_1/remix", nil)
	legacyPlatform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSora))

	platform, adaptor := getTaskAdaptorForRequest(c, legacyPlatform)

	require.NotNil(t, adaptor)
	assert.Equal(t, legacyPlatform, platform)
	pinnedValue, exists := c.Get(pluginruntime.ContextKeyPinnedPlugin)
	require.True(t, exists)
	pinned, ok := pinnedValue.(pluginruntime.PinnedPlugin)
	require.True(t, ok)
	require.NotNil(t, pinned.Generation)
	require.NotNil(t, pinned.Plugin)
	assert.Equal(t, "sora", pinned.Plugin.Meta.Key)
	assert.Same(t, pinned.Generation, pluginruntime.DefaultRegistry.Generation())
}

type pluginQueryTransport func(*http.Request) (*http.Response, error)

func (f pluginQueryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// A redirected task query carries the plugin's credential headers only to the
// channel host and the plugin's allowedHosts. Once a hop leaves them, no later
// hop gets the credentials back.
func TestPluginTaskQueryRedirectKeepsCredentialsWithinAllowedHosts(t *testing.T) {
	service.InitHttpClient()
	client := service.GetHttpClient()
	originalClient := *client
	fetchSetting := system_setting.GetFetchSetting()
	originalSetting := *fetchSetting
	t.Cleanup(func() {
		*client = originalClient
		*fetchSetting = originalSetting
	})
	*fetchSetting = system_setting.FetchSetting{EnableSSRFProtection: true, AllowedPorts: []string{"1-65535"}}
	const source = `
export const meta = {apiVersion:1,key:"query-redirect",name:"Query Redirect",version:"1.0.0",author:{name:"Test"},models:["m"],fetchMode:"per_task"};
export function buildSubmitRequest(ctx) {return {url:ctx.baseUrl+"/submit"};}
export function parseSubmitResponse() {return {taskId:"task"};}
export function buildQueryRequest(ctx) {return {url:ctx.baseUrl+"/query",headers:{"Authorization":"Bearer channel-secret","X-API-Key":"channel-secret","X-Vendor-Credential":"channel-secret","Accept":"application/json"}};}
export function parseTaskResult() {return {status:"SUCCESS"};}
`
	credentialHeaders := []string{"Authorization", "X-API-Key", "X-Vendor-Credential"}
	for _, tc := range []struct {
		name           string
		locations      []string
		allowedHosts   string
		credentialHops int
	}{
		{name: "same host keeps credentials", locations: []string{"/final"}, credentialHops: 2},
		{name: "allowed host keeps credentials", locations: []string{"https://cdn.example/final"}, allowedHosts: `["cdn.example"]`, credentialHops: 2},
		{name: "other host gets no credentials", locations: []string{"https://cdn.example/final?signature=download-token"}, credentialHops: 1},
		{name: "return to the channel host after another host gets no credentials", locations: []string{"https://cdn.example/next", "https://api.example/final"}, credentialHops: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sent []*http.Request
			client.Transport = pluginQueryTransport(func(req *http.Request) (*http.Response, error) {
				sent = append(sent, req.Clone(req.Context()))
				resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"status":"SUCCESS"}`)), Request: req}
				if len(sent) <= len(tc.locations) {
					resp.StatusCode = http.StatusFound
					resp.Header.Set("Location", tc.locations[len(sent)-1])
				}
				return resp, nil
			})
			pluginSource := source
			if tc.allowedHosts != "" {
				pluginSource = strings.Replace(pluginSource, `fetchMode:"per_task"`, `fetchMode:"per_task",allowedHosts:`+tc.allowedHosts, 1)
			}
			plugin, err := pluginruntime.NewRegistry().Register(pluginSource, pluginruntime.Options{})
			require.NoError(t, err)
			task := &model.Task{TaskID: "task", PrivateData: model.TaskPrivateData{UpstreamTaskID: "upstream-task"}}

			resp, err := jspluginadaptor.New(plugin).FetchTask("https://api.example", "channel-secret", task, "")

			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			require.Len(t, sent, len(tc.locations)+1)
			for i, hop := range sent {
				for _, name := range credentialHeaders {
					if i < tc.credentialHops {
						assert.Contains(t, hop.Header.Get(name), "channel-secret", "hop %d, header %s", i, name)
					} else {
						assert.Empty(t, hop.Header.Get(name), "hop %d to %s, header %s", i, hop.URL.Host, name)
					}
				}
				assert.Equal(t, "application/json", hop.Header.Get("Accept"), "hop %d", i)
			}
			if strings.Contains(tc.locations[0], "signature=") {
				assert.Equal(t, "download-token", sent[1].URL.Query().Get("signature"))
			}
		})
	}
}
