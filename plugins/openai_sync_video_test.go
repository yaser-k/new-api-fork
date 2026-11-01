package plugins_test

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	builtinplugins "github.com/QuantumNous/new-api/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const syncVideoKey = "openai-sync-video"

// The two answers an OpenAI-compatible upstream gave to the same synchronous
// request, cut to their shape. The first links a signed object-store URL
// that expires after a day, the second a URL on the upstream's API host;
// both carry billing and request_id fields of the upstream's own.
var syncVideoAnswers = []struct {
	name, body, url, wantTaskID string
	wantSeconds                 float64
}{
	{
		name:        "signed object-store link, whole seconds",
		body:        `{"id":"6a1f3c9e2b7d4e8f0a5c6b1d","status":"completed","videos":[{"url":"https://objects.upstream.example/videos/a.mp4?Expires=1761566400&Signature=cut","duration_seconds":2}],"usage":{},"request_id":"0f1e2d3c-4b5a-4968-8776-5a4b3c2d1e0f","billing":{"billed_cost_usd":"0.085000"}}`,
		url:         "https://objects.upstream.example/videos/a.mp4?Expires=1761566400&Signature=cut",
		wantTaskID:  "6a1f3c9e2b7d4e8f0a5c6b1d",
		wantSeconds: 2,
	},
	{
		name:        "link on the API host, fractional seconds as a string",
		body:        `{"id":"video_3f9a0c2e5b8d1f4a7c6e9b2d0a3f5c8e1b4d7a0c9e2f5b8d1a4c7e0b3f6a9c2e","status":"completed","videos":[{"url":"https://api.upstream.example/v1/videos/artifacts/opaque-token","duration_seconds":"2.041666666666666667"}],"usage":{},"request_id":"1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d","billing":{"billed_cost_usd":"0.034708"}}`,
		url:         "https://api.upstream.example/v1/videos/artifacts/opaque-token",
		wantTaskID:  "video_3f9a0c2e5b8d1f4a7c6e9b2d0a3f5c8e1b4d7a0c9e2f5b8d1a4c7e0b3f6a9c2e",
		wantSeconds: 2.041666666666666667,
	},
}

func loadSyncVideoPlugin(t *testing.T, source string) *jsplugin.LoadedPlugin {
	t.Helper()
	plugin, err := jsplugin.NewRegistry().RegisterFactory(source, jsplugin.Options{Key: syncVideoKey})
	require.NoError(t, err)
	return plugin
}

func syncVideoSource(t *testing.T) string {
	t.Helper()
	source, err := builtinplugins.Source(syncVideoKey)
	require.NoError(t, err)
	return source
}

// callJSON calls a plugin hook and returns its result as plain JSON values.
func callJSON(t *testing.T, plugin *jsplugin.LoadedPlugin, path []string, args ...any) (map[string]any, error) {
	t.Helper()
	var value any
	var err error
	if len(path) == 1 {
		value, err = plugin.Engine.Call(t.Context(), path[0], args...)
	} else {
		value, err = plugin.Engine.CallPath(t.Context(), path[0], path[1:], args...)
	}
	if err != nil {
		return nil, err
	}
	encoded, marshalErr := common.Marshal(value)
	require.NoError(t, marshalErr)
	var decoded map[string]any
	require.NoError(t, common.Unmarshal(encoded, &decoded))
	return decoded, nil
}

func decodeVideo(t *testing.T, plugin *jsplugin.LoadedPlugin, model string, body map[string]any) (map[string]any, error) {
	t.Helper()
	return callJSON(t, plugin, []string{"protocols", "openai_video", "decodeRequest"}, map[string]any{
		"model": model, "upstreamModel": model, "protocol": "openai_video", "operation": "create",
		"body": map[string]any{"kind": "json", "value": body},
	})
}

func TestOpenAISyncVideoResponsesProtocol(t *testing.T) {
	testVideoResponsesProtocol(t, videoResponsesTestCase{
		pluginKey: syncVideoKey,
		model:     "example-video-model",
		requestBody: map[string]any{
			"model":      "example-video-model",
			"input":      "waves at sunset",
			"resolution": "480p",
			"seconds":    2,
		},
		wantAction: "text_to_video",
		wantRequest: map[string]any{
			"model":      "example-video-model",
			"prompt":     "waves at sunset",
			"resolution": "480P",
			"duration":   float64(2),
		},
		wantUsageKeys:  []string{"resolution", "seconds"},
		wantVendorName: syncVideoKey,
	})
}

// TestOpenAISyncVideoAppliesOnlyWhereSelected keeps OpenAI channels on sora:
// the plugin claims no channel type and runs on the Task Plugin channels and
// models an administrator binds it to.
func TestOpenAISyncVideoAppliesOnlyWhereSelected(t *testing.T) {
	generation := jsplugin.DefaultRegistry.Generation()
	require.NotNil(t, generation)
	plugin, found := generation.Get(syncVideoKey)
	require.True(t, found)
	assert.Empty(t, plugin.Meta.ChannelTypes)
	for _, channelType := range []int{1, 55} {
		owner, owned := generation.GetByChannelType(channelType)
		require.True(t, owned)
		assert.Equal(t, "sora", owner.Meta.Key)
	}
	for _, model := range []string{"sora-2", "sora-2-pro"} {
		binding, claimed := generation.LookupEndpoint("POST", "/v1/videos", model)
		require.True(t, claimed)
		assert.Equal(t, "sora", binding.Plugin.Meta.Key)
	}
	binding, claimed := generation.LookupEndpoint("POST", "/v1/videos", "example-video-model")
	require.True(t, claimed)
	assert.Equal(t, syncVideoKey, binding.Plugin.Meta.Key)
	// Only this plugin's submission is never sent twice; every other
	// built-in plugin keeps the usual retries.
	for _, builtIn := range generation.Plugins() {
		assert.Equal(t, builtIn.Meta.Key != syncVideoKey, builtIn.Meta.IdempotentSubmit(), builtIn.Meta.Key)
	}
}

func TestOpenAISyncVideoSubmitsAndKeepsTheAnswerAsAFinishedTask(t *testing.T) {
	plugin := loadSyncVideoPlugin(t, syncVideoSource(t))
	const model = "example-video-model"
	decoded, err := decodeVideo(t, plugin, model, map[string]any{"model": model, "prompt": "a red kite over the dunes", "resolution": "480P", "seconds": "2"})
	require.NoError(t, err)
	assert.Equal(t, "text_to_video", decoded["action"])
	requestBody := decoded["requestBody"].(map[string]any)
	driver := map[string]any{
		"requestBody": requestBody, "model": model, "upstreamModel": "upstream-video-model", "action": "text_to_video",
		"baseUrl": "https://api.upstream.example", "apiKey": "sk-channel", "authHeader": "Bearer sk-channel", "upstream": map[string]any{"kind": "vendor"},
	}

	descriptor, err := callJSON(t, plugin, []string{"buildSubmitRequest"}, driver)
	require.NoError(t, err)
	assert.Equal(t, "https://api.upstream.example/v1/videos/generations", descriptor["url"])
	assert.Equal(t, "POST", descriptor["method"])
	assert.Equal(t, "Bearer sk-channel", descriptor["headers"].(map[string]any)["Authorization"])
	assert.Equal(t, map[string]any{"model": "upstream-video-model", "prompt": "a red kite over the dunes", "resolution": "480P", "duration": float64(2)}, descriptor["body"])
	assert.NotContains(t, descriptor, "model", "the descriptor never rewrites a mapped model")

	facts, err := callJSON(t, plugin, []string{"extractUsage"}, driver)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"seconds": float64(2), "resolution": "480P"}, facts)

	for _, answer := range syncVideoAnswers {
		t.Run(answer.name, func(t *testing.T) {
			var body map[string]any
			require.NoError(t, common.UnmarshalJsonStr(answer.body, &body))
			parsed, err := callJSON(t, plugin, []string{"parseSubmitResponse"}, driver, map[string]any{"statusCode": 200, "headers": map[string]any{}, "body": body})
			require.NoError(t, err)
			assert.Equal(t, answer.wantTaskID, parsed["taskId"])
			assert.Equal(t, map[string]any{"status": "SUCCESS", "progress": "100%"}, parsed["immediate"])
			assert.Equal(t, map[string]any{"video_url": answer.url}, parsed["state"], "the link stays in private plugin state")
			taskData := parsed["taskData"].(map[string]any)
			assert.Equal(t, map[string]any{"status": "completed", "seconds": answer.wantSeconds, "resolution": "480P"}, taskData)
			encodedData, err := common.Marshal(taskData)
			require.NoError(t, err)
			assert.NotContains(t, string(encodedData), "upstream.example", "task data is shown to the user and must not carry the link")

			completion, err := callJSON(t, plugin, []string{"extractUsageOnComplete"}, map[string]any{"taskId": answer.wantTaskID, "model": model, "upstreamModel": model, "data": taskData}, parsed["immediate"], taskData)
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"seconds": answer.wantSeconds}, completion, "the answer's seconds settle the task")

			task := map[string]any{"task_id": "task_public", "status": "SUCCESS", "progress": "100%", "created_at": 10, "data": taskData, "properties": map[string]any{"origin_model_name": model}}
			video, err := callJSON(t, plugin, []string{"protocols", "openai_video", "render"}, map[string]any{"protocol": "openai_video", "operation": "retrieve"}, task)
			require.NoError(t, err)
			assert.Equal(t, "completed", video["status"])
			assert.Equal(t, common.Interface2String(answer.wantSeconds), video["seconds"])
			encodedVideo, err := common.Marshal(video)
			require.NoError(t, err)
			assert.NotContains(t, string(encodedVideo), "upstream.example")

			artifacts, err := plugin.Engine.Call(t.Context(), "listArtifacts", map[string]any{"taskId": "task_public", "status": "SUCCESS", "action": "text_to_video", "data": taskData, "producerVersion": "1.0.0"})
			require.NoError(t, err)
			encodedArtifacts, err := common.Marshal(artifacts)
			require.NoError(t, err)
			assert.JSONEq(t, `[{"key":"video","type":"video"}]`, string(encodedArtifacts))

			content, err := callJSON(t, plugin, []string{"buildContentRequest"}, map[string]any{
				"artifactKey": "video", "data": taskData, "state": parsed["state"], "upstreamTaskId": answer.wantTaskID,
				"baseUrl": "https://api.upstream.example", "apiKey": "sk-channel", "clientRequest": map[string]any{"method": "GET", "headers": map[string]any{}},
			})
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"url": answer.url, "method": "GET", "credentialless": true}, content, "the gateway proxies the link without the channel key")
		})
	}
}

// TestOpenAISyncVideoRefusesWhatItCannotBill rejects, before the upstream is
// called, every request whose rate the model does not have or whose fields
// could select a rate the gateway does not bill.
func TestOpenAISyncVideoRefusesWhatItCannotBill(t *testing.T) {
	plugin := loadSyncVideoPlugin(t, syncVideoSource(t))
	const model = "example-video-model"
	for _, tc := range []struct {
		name, wantError string
		body            map[string]any
	}{
		{"resolution without a rate", "has no per-second rate for resolution 2K; it has 480P, 720P, 1080P", map[string]any{"model": model, "prompt": "p", "resolution": "2K", "seconds": 2}},
		{"no resolution on a model with several rates", "resolution is required: one of 480P, 720P, 1080P", map[string]any{"model": model, "prompt": "p", "seconds": 2}},
		{"no duration", "duration is required", map[string]any{"model": model, "prompt": "p", "resolution": "480P"}},
		{"duration beyond the host cap", "duration must be a number of seconds", map[string]any{"model": model, "prompt": "p", "resolution": "480P", "duration": 3601}},
		{"zero seconds", "duration must be a number of seconds", map[string]any{"model": model, "prompt": "p", "resolution": "480P", "seconds": "0"}},
		{"hex seconds", "duration must be a number of seconds", map[string]any{"model": model, "prompt": "p", "resolution": "480P", "seconds": "0x10"}},
		{"exponent seconds", "duration must be a number of seconds", map[string]any{"model": model, "prompt": "p", "resolution": "480P", "seconds": "1e3"}},
		{"image input", "field input_reference is not supported by this upstream", map[string]any{"model": model, "prompt": "p", "resolution": "480P", "seconds": 2, "input_reference": "https://example.com/a.png"}},
		{"size instead of resolution", "field size is not supported by this upstream", map[string]any{"model": model, "prompt": "p", "size": "1280x720", "seconds": 2}},
		{"no prompt", "prompt is required", map[string]any{"model": model, "resolution": "480P", "seconds": 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeVideo(t, plugin, model, tc.body)
			require.ErrorContains(t, err, tc.wantError)
		})
	}

	t.Run("multipart file", func(t *testing.T) {
		_, err := callJSON(t, plugin, []string{"protocols", "openai_video", "decodeRequest"}, map[string]any{
			"model": model, "upstreamModel": model,
			"body": map[string]any{"kind": "multipart", "fields": map[string]any{"prompt": []any{"p"}, "resolution": []any{"480P"}, "seconds": []any{"2"}},
				"files": []any{map[string]any{"ref": "request_file:input_reference", "field": "input_reference", "filename": "a.png", "mimeType": "image/png", "size": 1}}},
		})
		require.ErrorContains(t, err, "file inputs are not supported by this upstream")
	})

	t.Run("a duration_seconds that is not a plain decimal settles on the requested duration", func(t *testing.T) {
		driver := map[string]any{"requestBody": map[string]any{"model": model, "prompt": "p", "resolution": "480P", "duration": 2}, "model": model, "upstreamModel": model}
		for _, reported := range []any{"0x10", "1e3", " ", "-2", "2.5s"} {
			body := map[string]any{"id": "job", "status": "completed", "videos": []any{map[string]any{"url": "https://objects.upstream.example/a.mp4", "duration_seconds": reported}}}
			parsed, err := callJSON(t, plugin, []string{"parseSubmitResponse"}, driver, map[string]any{"statusCode": 200, "headers": map[string]any{}, "body": body})
			require.NoError(t, err)
			assert.Equal(t, float64(2), parsed["taskData"].(map[string]any)["seconds"], reported)
		}
	})

	t.Run("an answer without a finished video is a failed task, never an error", func(t *testing.T) {
		driver := map[string]any{"requestBody": map[string]any{"model": model, "prompt": "p", "resolution": "480P", "duration": 2}, "model": model, "upstreamModel": model}
		for _, body := range []map[string]any{
			{"id": "job-1", "status": "processing"},
			{"id": "job-2", "status": "completed", "videos": []any{}},
			{"id": "job-3", "status": "failed", "error": map[string]any{"message": "prompt rejected"}},
		} {
			parsed, err := callJSON(t, plugin, []string{"parseSubmitResponse"}, driver, map[string]any{"statusCode": 200, "headers": map[string]any{}, "body": body})
			require.NoError(t, err, "a parse error would retry the generation on another channel")
			immediate := parsed["immediate"].(map[string]any)
			assert.Equal(t, "FAILURE", immediate["status"])
			assert.Equal(t, body["id"], parsed["taskId"])
			assert.NotContains(t, parsed, "state")
		}
	})
}

// TestOpenAISyncVideoAdministratorModelTable loads the plugin the way an
// administrator uploads it, with the upstream's models and their keys.
func TestOpenAISyncVideoAdministratorModelTable(t *testing.T) {
	const example = `"example-video-model": ["480P", "720P", "1080P"],`
	source := syncVideoSource(t)
	require.Contains(t, source, example)

	t.Run("each model keeps its own keys", func(t *testing.T) {
		plugin := loadSyncVideoPlugin(t, strings.Replace(source, example, `"video-a": ["480P", "720P", "1080P"], "video-b": ["720P+TEXT_TO_VIDEO"],`, 1))
		assert.Equal(t, []string{"video-a", "video-b"}, plugin.Meta.Models)
		schema, _ := plugin.Meta.UsageForModels("video-b")
		assert.Equal(t, []string{"720P+TEXT_TO_VIDEO"}, schema["resolution"].Enum)

		decoded, err := decodeVideo(t, plugin, "video-b", map[string]any{"model": "video-b", "prompt": "p", "seconds": 4})
		require.NoError(t, err, "a model with one rate needs no resolution")
		requestBody := decoded["requestBody"].(map[string]any)
		assert.Equal(t, "720P", requestBody["resolution"], "the upstream receives the resolution alone")
		facts, err := callJSON(t, plugin, []string{"extractUsage"}, map[string]any{"requestBody": requestBody, "model": "video-b", "upstreamModel": "video-b"})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"seconds": float64(4), "resolution": "720P+TEXT_TO_VIDEO"}, facts)

		_, err = decodeVideo(t, plugin, "video-b", map[string]any{"model": "video-b", "prompt": "p", "resolution": "1080P", "seconds": 4})
		require.ErrorContains(t, err, "has no per-second rate for resolution 1080P; it has 720P")
	})

	t.Run("rejects a submitIdempotent that is not a boolean", func(t *testing.T) {
		require.Contains(t, source, "submitIdempotent: false,")
		_, err := jsplugin.NewRegistry().RegisterFactory(strings.Replace(source, "submitIdempotent: false,", `submitIdempotent: "false",`, 1), jsplugin.Options{Key: syncVideoKey})
		require.ErrorContains(t, err, "submitIdempotent must be a boolean")
	})

	for _, table := range []string{`"video-a": ["720P+AUDIO"],`, `"video-a": ["PRO"],`, `"video-a": ["720P", "720P+TEXT_TO_VIDEO"],`} {
		t.Run("rejects "+table, func(t *testing.T) {
			_, err := jsplugin.NewRegistry().RegisterFactory(strings.Replace(source, example, table, 1), jsplugin.Options{Key: syncVideoKey})
			require.Error(t, err)
		})
	}
}
