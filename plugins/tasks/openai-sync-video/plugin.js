// Video generation from an OpenAI-compatible upstream whose
// POST /v1/videos/generations answers synchronously with the finished video.
//
// The plugin claims no channel type: OpenAI channels (type 1) run the sora
// plugin, and a channel type belongs to one plugin. An administrator selects
// this plugin for a channel by binding it to a Task Plugin channel (type 61)
// whose Base URL and key are the upstream's, and for a model by listing the
// model in MODEL_KEYS below.
//
// The submission is the generation: a finished answer becomes a SUCCESS task
// at once and is never polled. The upstream's video link is kept in private
// plugin state, never in task data, and the video is served only through the
// gateway's content route.

// Each model the upstream serves, with the per-second price keys it has a
// rate for. Administrators replace this table with their upstream's models
// and upload the file as a new version of this plugin; the entry below is an
// example. A key is an output resolution (480P, 720P, 1080P, 2K, 4K), or a
// resolution followed by +TEXT_TO_VIDEO for a model whose rates name that
// mode: every request this plugin sends is text to video. Keys that depend on
// request fields this plugin does not send (STD, PRO, DRAFT, AUDIO,
// VIDEO_INPUT, IMAGE_TO_VIDEO, VOICE_CONTROL) are rejected when the plugin
// loads. The prices themselves are the model's task expression, keyed on
// u("resolution").
const MODEL_KEYS = {
  "example-video-model": ["480P", "720P", "1080P"],
};

const RESOLUTIONS = ["480P", "720P", "1080P", "2K", "4K"];
const TEXT_TO_VIDEO = "+TEXT_TO_VIDEO";
// The host caps second facts at 3600.
const MAX_SECONDS = 3600;
// Request members forwarded upstream. Any other member could select a rate
// the gateway does not bill, so it is rejected before the upstream is called.
const VIDEO_FIELDS = ["model", "prompt", "resolution", "duration", "seconds"];
// Responses envelope members the host owns or that change nothing upstream.
const RESPONSES_FIELDS = VIDEO_FIELDS.concat(["input", "stream", "background", "store", "metadata", "user"]);

function keyResolution(key) {
  return key.endsWith(TEXT_TO_VIDEO) ? key.slice(0, key.length - TEXT_TO_VIDEO.length) : key;
}

for (const model of Object.keys(MODEL_KEYS)) {
  const keys = MODEL_KEYS[model];
  if (!Array.isArray(keys) || !keys.length) throw new Error("model " + model + " lists no price key");
  const seen = [];
  for (const key of keys) {
    const resolution = keyResolution(String(key));
    if (!RESOLUTIONS.includes(resolution)) throw new Error("model " + model + " price key " + key + " is not supported");
    if (seen.includes(resolution)) throw new Error("model " + model + " lists resolution " + resolution + " twice");
    seen.push(resolution);
  }
}

function videoUsageSchema(keys) {
  const labels = {};
  for (const key of keys) {
    const resolution = keyResolution(key);
    labels[key] = key === resolution ? { en: key, zh: key } : { en: resolution + " text to video", zh: resolution + " 文生视频" };
  }
  return {
    // Billable seconds: the duration_seconds of the upstream's answer,
    // fraction included, else the requested duration.
    seconds: {
      type: "number",
      unit: "second",
      description: { en: "Video generation unit price", zh: "视频生成单价" },
    },
    // The model's price key that the request selects.
    resolution: {
      enum: keys,
      enumLabels: labels,
      description: { en: "Output video resolution", zh: "输出视频分辨率" },
    },
  };
}

// One usage profile per distinct key set, so pricing lists only the keys a
// model has a rate for and the host rejects any other key before billing.
function videoUsageProfiles() {
  const profiles = [];
  for (const model of Object.keys(MODEL_KEYS)) {
    const shape = MODEL_KEYS[model].join("/");
    let profile = profiles.find((entry) => entry.shape === shape);
    if (!profile) {
      profile = { shape: shape, models: [], schema: videoUsageSchema(MODEL_KEYS[model]) };
      profiles.push(profile);
    }
    profile.models.push(model);
  }
  return profiles.map((profile) => ({ models: profile.models, schema: profile.schema }));
}

function allKeys() {
  const keys = [];
  for (const model of Object.keys(MODEL_KEYS)) {
    for (const key of MODEL_KEYS[model]) if (!keys.includes(key)) keys.push(key);
  }
  return keys;
}

export const meta = {
  apiVersion: 1,
  key: "openai-sync-video",
  name: "OpenAI-compatible synchronous video",
  icon: "text",
  description: {
    en: "Video generation from an OpenAI-compatible upstream that answers with the finished video",
    zh: "通过直接返回成片的 OpenAI 兼容上游生成视频",
  },
  version: "1.0.0",
  author: { name: "QuantumNous" },
  models: Object.keys(MODEL_KEYS),
  fetchMode: "per_task",
  // The submission is the generation, and the upstream bills it once it
  // receives the request: a failed attempt that was sent is never retried.
  submitIdempotent: false,
  usageSchema: videoUsageSchema(allKeys()),
  usageProfiles: videoUsageProfiles(),
  protocols: [{ name: "openai_responses", supports: ["stream", "sync", "background"] }, "openai_video"],
};

function trimmed(value) {
  return String(value === undefined || value === null ? "" : value).trim();
}

function isObject(value) {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

// modelKeys reads the table for the upstream model, else for the client
// model when a channel mapping sends a name the table does not list.
function modelKeys(ctx) {
  for (const name of [trimmed(ctx.upstreamModel), trimmed(ctx.model)]) {
    if (name && Object.prototype.hasOwnProperty.call(MODEL_KEYS, name)) return MODEL_KEYS[name];
  }
  throw new Error("model " + trimmed(ctx.model) + " is not configured for this plugin");
}

function rejectUnknownFields(req, allowed) {
  for (const name of Object.keys(req)) {
    if (!allowed.includes(name)) throw new Error("field " + name + " is not supported by this upstream");
  }
}

// selectKey returns the price key of the requested resolution, or the only
// key of a model with one rate when the request names none.
function selectKey(ctx, resolution) {
  const keys = modelKeys(ctx);
  const requested = trimmed(resolution).toUpperCase();
  if (!requested) {
    if (keys.length === 1) return keys[0];
    throw new Error("resolution is required: one of " + keys.map(keyResolution).join(", "));
  }
  for (const key of keys) if (keyResolution(key) === requested) return key;
  throw new Error("model " + trimmed(ctx.model) + " has no per-second rate for resolution " + requested + "; it has " + keys.map(keyResolution).join(", "));
}

// plainSeconds reads a number of seconds: a finite JSON number, or a string
// holding a plain decimal such as "2" or "2.041666666666666667". Number()
// would also accept hex, exponents and blanks; those are not seconds.
function plainSeconds(value) {
  if (typeof value === "number") return Number.isFinite(value) ? value : NaN;
  if (typeof value === "string" && /^\d+(\.\d+)?$/.test(value.trim())) return Number(value.trim());
  return NaN;
}

function requestSeconds(value) {
  const seconds = plainSeconds(value);
  if (!Number.isFinite(seconds) || seconds <= 0 || seconds > MAX_SECONDS) throw new Error("duration must be a number of seconds between 0 and " + MAX_SECONDS);
  return seconds;
}

// videoRequest validates a client request and returns the body the upstream
// receives, without the model, which buildSubmitRequest sets.
function videoRequest(ctx, req, allowed) {
  rejectUnknownFields(req, allowed);
  const prompt = trimmed(req.prompt);
  if (!prompt) throw new Error("prompt is required");
  const hasSeconds = req.seconds !== undefined && req.seconds !== null;
  const hasDuration = req.duration !== undefined && req.duration !== null;
  if (hasSeconds && hasDuration) throw new Error("send duration or seconds, not both");
  if (!hasSeconds && !hasDuration) throw new Error("duration is required");
  const key = selectKey(ctx, req.resolution);
  return { prompt: prompt, resolution: keyResolution(key), duration: requestSeconds(hasSeconds ? req.seconds : req.duration) };
}

function responsesPrompt(input) {
  if (typeof input === "string") return input;
  if (!Array.isArray(input)) throw new Error("input must be a string or array");
  const texts = [];
  for (const item of input) {
    if (typeof item === "string") {
      texts.push(item);
      continue;
    }
    if (!isObject(item)) continue;
    const content = item.content === undefined ? [item] : Array.isArray(item.content) ? item.content : [item.content];
    for (const part of content) {
      if (typeof part === "string") {
        texts.push(part);
        continue;
      }
      if (!isObject(part)) continue;
      if (["input_text", "text"].includes(part.type) && typeof part.text === "string") {
        texts.push(part.text);
        continue;
      }
      throw new Error("input part " + String(part.type || "") + " is not supported by this upstream");
    }
  }
  return texts.filter((text) => trimmed(text)).join("\n");
}

function videoArtifactText(ctx) {
  const artifact = ctx && ctx.artifacts && ctx.artifacts.video;
  const url = trimmed(artifact && artifact.url);
  if (!url) throw new Error("video artifact is unavailable");
  const escaped = url.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  return '<video controls src="' + escaped + '"></video>';
}

// answerSeconds reads the billed length of the finished video: the upstream
// reports duration_seconds as a number or as a decimal string and bills the
// fraction.
function answerSeconds(video) {
  const seconds = plainSeconds(video && video.duration_seconds);
  if (!Number.isFinite(seconds) || seconds <= 0) return 0;
  return Math.min(seconds, MAX_SECONDS);
}

export function buildSubmitRequest(ctx) {
  const req = isObject(ctx.requestBody) ? ctx.requestBody : {};
  const body = videoRequest(ctx, req, VIDEO_FIELDS);
  return {
    url: ctx.baseUrl + "/v1/videos/generations",
    method: "POST",
    headers: { Authorization: "Bearer " + ctx.apiKey, "Content-Type": "application/json" },
    body: { model: ctx.upstreamModel || ctx.model, prompt: body.prompt, resolution: body.resolution, duration: body.duration },
  };
}

// parseSubmitResponse turns the synchronous answer into a terminal task. A 2xx
// answer without a finished video is a FAILURE, never an error: a failed
// parse would be retried on another channel and generate the video again.
export function parseSubmitResponse(ctx, resp) {
  const body = isObject(resp.body) ? resp.body : {};
  const taskId = trimmed(body.id) || trimmed(body.request_id) || utils.uuid();
  const req = isObject(ctx.requestBody) ? ctx.requestBody : {};
  const key = selectKey(ctx, req.resolution);
  const video = Array.isArray(body.videos) ? body.videos[0] : null;
  const url = trimmed(video && video.url);
  const status = trimmed(body.status).toLowerCase();
  if (status !== "completed" || !url) {
    const reason = isObject(body.error) && trimmed(body.error.message) ? trimmed(body.error.message) : "the upstream returned no finished video";
    return { taskId: taskId, taskData: { status: status || "failed", resolution: key }, immediate: { status: "FAILURE", reason: reason } };
  }
  const seconds = answerSeconds(video) || requestSeconds(req.seconds !== undefined && req.seconds !== null ? req.seconds : req.duration);
  return {
    taskId: taskId,
    taskData: { status: "completed", seconds: seconds, resolution: key },
    state: { video_url: url },
    immediate: { status: "SUCCESS", progress: "100%" },
  };
}

export function extractUsage(ctx) {
  const req = isObject(ctx.requestBody) ? ctx.requestBody : {};
  const body = videoRequest(ctx, req, VIDEO_FIELDS);
  return { seconds: body.duration, resolution: selectKey(ctx, req.resolution) };
}

export function extractUsageOnComplete(task, taskResult, data) {
  const facts = {};
  const seconds = Number((data || {}).seconds);
  if (Number.isFinite(seconds) && seconds > 0) facts.seconds = Math.min(seconds, MAX_SECONDS);
  return facts;
}

// Tasks finish at submission and are never polled.
export function buildQueryRequest() {
  throw new Error("this upstream answers synchronously; there is no task to poll");
}

export function parseTaskResult() {
  return { status: "UNKNOWN", reason: "this upstream answers synchronously; there is no task to poll" };
}

export function listArtifacts(task) {
  return task.status === "SUCCESS" ? [{ key: "video", type: "video" }] : [];
}

// The upstream's link needs no credential: it is either signed or served
// without the key, so the channel key is never sent with it.
export function buildContentRequest(ctx) {
  if (ctx.artifactKey !== "video") throw new Error("artifact_not_found");
  const url = trimmed(ctx.state && ctx.state.video_url);
  if (!url) throw new Error("artifact_not_found");
  return { url: url, method: ctx.clientRequest.method, credentialless: true };
}

function openAIVideo(task) {
  const statuses = { NOT_START: "queued", SUBMITTED: "queued", QUEUED: "queued", IN_PROGRESS: "in_progress", SUCCESS: "completed", FAILURE: "failed" };
  const data = isObject(task.data) ? task.data : {};
  const output = {
    id: task.task_id,
    object: "video",
    model: (task.properties || {}).origin_model_name || "",
    status: statuses[task.status] || "unknown",
    progress: task.status === "SUCCESS" ? 100 : 0,
    created_at: Number(task.created_at || 0),
  };
  if (typeof data.seconds === "number") output.seconds = String(data.seconds);
  if (typeof data.resolution === "string") output.resolution = keyResolution(data.resolution);
  if (task.status === "FAILURE") output.error = { code: "video_generation_failed", message: task.fail_reason || "The video generation task failed." };
  return output;
}

export const protocols = {
  openai_responses: {
    decodeRequest: function (ctx) {
      if (!ctx.body || ctx.body.kind !== "json") throw new Error("JSON body required");
      const req = ctx.body.value;
      if (!isObject(req)) throw new Error("request body must be an object");
      if (!trimmed(req.model)) throw new Error("model is required");
      if (req.input !== undefined && typeof req.input !== "string" && !Array.isArray(req.input)) throw new Error("input must be a string or array");
      rejectUnknownFields(req, RESPONSES_FIELDS);
      const prompt = trimmed(req.input === undefined ? req.prompt : responsesPrompt(req.input));
      const values = { prompt: prompt };
      for (const name of ["resolution", "duration", "seconds"]) if (req[name] !== undefined) values[name] = req[name];
      const body = videoRequest(ctx, values, VIDEO_FIELDS);
      const requestBody = { model: ctx.model, prompt: body.prompt, resolution: body.resolution, duration: body.duration };
      return { kind: "submit", model: ctx.model, action: "text_to_video", requestBody: requestBody };
    },
    renderEvents: function (ctx, task, previousState) {
      const status = String(task.status || "UNKNOWN").toUpperCase();
      const value = Number(String(task.progress || "").replace("%", ""));
      const progress = Number.isFinite(value) && value >= 0 && value <= 100 ? value : null;
      const state = { status: status, progress: progress };
      if (status === "SUCCESS") {
        const events = previousState && previousState.status === status ? [] : [{ type: "output", data: videoArtifactText(ctx) }];
        return { events: events, state: state, done: true };
      }
      if (status === "FAILURE")
        return { events: [{ type: "error", code: "task_failed", message: task.fail_reason || "task failed" }], state: state, done: true };
      if (previousState && previousState.status === status && previousState.progress === progress) return { events: [], state: state, done: false };
      const event = { type: "progress", message: status.toLowerCase() };
      if (progress !== null) event.progress = progress;
      return { events: [event], state: state, done: false };
    },
    renderFinal: function (ctx) {
      return {
        output: [
          {
            type: "message",
            status: "completed",
            role: "assistant",
            content: [{ type: "output_text", text: videoArtifactText(ctx), annotations: [], logprobs: [] }],
          },
        ],
        metadata: { vendor: "openai-sync-video" },
      };
    },
  },
  openai_video: {
    decodeRequest: function (ctx) {
      if (!ctx.body || (ctx.body.kind !== "json" && ctx.body.kind !== "multipart")) throw new Error("JSON or multipart body required");
      let req;
      if (ctx.body.kind === "json") {
        if (!isObject(ctx.body.value)) throw new Error("JSON object required");
        req = ctx.body.value;
      } else {
        if ((ctx.body.files || []).length) throw new Error("file inputs are not supported by this upstream");
        req = {};
        const fields = ctx.body.fields || {};
        for (const name of Object.keys(fields)) {
          if (fields[name].length > 1) throw new Error(name + " must be provided once");
          req[name] = fields[name][0];
        }
      }
      const body = videoRequest(ctx, req, VIDEO_FIELDS);
      const requestBody = { model: ctx.model, prompt: body.prompt, resolution: body.resolution, duration: body.duration };
      return { kind: "submit", model: ctx.model, action: "text_to_video", requestBody: requestBody };
    },
    // The retrieve answer carries no link: the client downloads the video
    // from GET /v1/videos/{id}/content.
    render: function (ctx, task) {
      return openAIVideo(task);
    },
  },
};
