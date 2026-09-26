# Persian (fa) locale: session 10 (placeholder isolates, response-time badge, batch 10: every remaining page, RTL fixes)

Temporary review folder. Delete `.fa-review/` before any upstream PR.

Branch: `feat/fa-locale` only (no new upstream branch). `upstream/main` was still at `c2b7a9a`, so no merge was needed. The local clone started on a stale older history (`d263b5f`, not an ancestor of the remote head); it was reset to `origin/feat/fa-locale` = `57c75bd` before any work. Session 8 is described in the previous version of this file (`git show 57c75bd:.fa-review/REPORT.md`).

## Decisions applied (session 8's open questions)

| # | Decision | Where it landed |
| --- | --- | --- |
| 1 | Milliseconds «۴۵۶ ms» (Persian digits, Latin unit), seconds «ثانیه» | B: `39a571f` (fa.json), `31f68f8` (the number is now formatted with the interface locale, so the digits really are Persian; see B) |
| 2 | Every remaining page, customer-facing first, task plugins last | C |
| 3 | The two dates that bypass the date helpers stay listed | Remaining issues 2 (unchanged) |

## Commits (from `57c75bd`)

```
67e72b5 fix(i18n): wrap Persian placeholders of left-to-right inputs in an RTL isolate
2242169 feat(i18n): check that Persian placeholders of dir=ltr inputs are isolated
6496416 fix(web): use logical start margins in the channels table cells
39a571f feat(i18n): show milliseconds as «۴۵۶ ms» in Persian
2b721c2 chore: add the parser-based Persian key scanner to .fa-review
5cd6e90 feat(i18n): translate lib, routes, auth and home in Persian (38 keys)
b744e97 feat(i18n): translate pricing, rankings and API keys in Persian (46 keys)
334c3d9 feat(i18n): translate the usage logs in Persian (60 keys)
2adca66 feat(i18n): translate the dashboard in Persian (49 keys)
0d1b17b feat(i18n): translate the playground in Persian (50 keys)
467f231 feat(i18n): translate the setup wizard and system info in Persian (103 keys)
a14be44 feat(i18n): translate the legal pages and static keys in Persian (58 keys)
bdf0202 feat(i18n): translate the task plugins page in Persian (207 keys)
305ee51 docs(i18n): add the terms settled in Persian batch 10 to the glossary
cb80274 fix(web): right-to-left layout on the customer-facing pages
e6f7d76 fix(web): right-to-left layout on the setup, system info and task plugin pages
31f68f8 fix(web): format response-time milliseconds in the interface language
(this hand-off): .fa-review scripts, screenshots, report
```

## Key count

| | Keys |
| --- | --- |
| `fa.json` at the start (`57c75bd`) | 5121 |
| Batch 10 (new) | **611**: lib + routes + auth + home 38 (incl. 9 theme preset names built as `preset.<value>`), pricing + rankings + keys 46, usage-logs 60, dashboard 49, playground 50, setup + system-info 103 (incl. the lowercase task statuses `pending` / `running`, shown through `t(task.status)`), legal + keys only in `i18n/static-keys.ts` 58, task-plugins 207. Channels recheck: 0 (see C) |
| Existing values changed | 17: 16 placeholders wrapped in RLI … PDI (A), `{{value}}ms` (B) |
| `fa.json` now | **5732** (`check-fa: 5732 keys, no findings`); `i18n:sync` reports 1063 en.json keys not in fa.json, of which **161 are used in the code** (list below); the rest are not found by the scanner (unused keys, or keys built at run time) |

## A. Persian placeholders in left-to-right inputs

- Checked first in Chromium 141 (headless, `dir=rtl` page): an empty `<input dir="auto">` has `direction: ltr` and `:dir(ltr)`; it takes its direction from its value, not its placeholder, so `dir=auto` does not help. The same test showed the RLI-wrapped placeholder in order.
- Fix (`67e72b5`): `dir='ltr'` stays; the Persian value of each such placeholder is one RLI (U+2067) … PDI (U+2069) isolate, written through the script. 16 keys on 18 inputs: custom OAuth dialog (`e.g. my-gitlab`, `Icon identifier (e.g. github, gitlab)`, `OAuth Client ID`, `e.g. openid profile email`), OAuth section (GitHub, Discord, OIDC, Telegram, LinuxDO client IDs, `Override auto-discovered endpoint` ×3), bot protection, passkey, chat dialog, Creem product, payment method, performance (temp directory). In Chromium the dialog placeholders now report `<RLI>…<PDI>` with `direction: ltr`, and the icon field reads «شناسۀ آیکون (مثلاً github یا gitlab)» (`161-…-rtl.png`; before: `150-…-rtl.png`). English is unchanged (`161-…-en.png`).
- Rule 11 in `docs/i18n/fa.md`; `check-fa` (`2242169`) now parses the `.tsx`/`.jsx` files under `src/` with `@babel/parser` (already installed through `@tanstack/router-plugin` and `shadcn`; not a direct dependency), collects the `t('…')` keys in the `placeholder` of every element with a literal `dir='ltr'`, and reports a Persian value that is not wrapped (`ltr-placeholder-isolate`). The scan runs when no fa.json path is given (the `i18n:check-fa` script) or with `--src <dir>`.
- Tests: `web/scripts/__tests__/check-fa.test.ts` 4 new cases (unwrapped value reported, wrapped accepted, same placeholder without dir=ltr ignored, project sources clean). The last one failed before the fa.json change (1 failed / 31 passed) and passes after (32 passed).
- Batch 10 added one more such placeholder (`Paste JavaScript source here...` in the left-to-right code editor on the task plugin upload dialog; the editor sets `dir` on a wrapper, so the check does not see it) and wrapped it the same way.

## B. Response-time badge

- `{{value}}ms` in fa.json: `[FSI]{{value}}[PDI] ms` (`39a571f`).
- The badge got the raw number, so the value showed Latin digits («456 ms»). `31f68f8` passes the millisecond value through `formatNumber` with the interface locale in `formatResponseTime` (channels table and channel test dialog, locale passed at each call site) and in the playground message timing. Milliseconds are whole numbers below 1000, so every other language shows the same digits as before; seconds are unchanged (`toFixed(2)`, Latin digits in every language). Test `features/channels/lib/__tests__/response-time-format.test.ts` (fa, en, zhCN, zhTW, fr, ru, ja, vi, an invalid code, seconds): 1 failed / 9 passed before, 10 passed after.
- Measured in the channels table view (Chromium, 1440 wide): «۴۵۶ ms» badge 57 px in a 100 px cell, «1.23 ثانیه» 67 px, «12.35 ثانیه» 75 px; no element overflows (`163-channels-table-response-time-rtl.png`).
- `-ml-1.5` → `-ms-1.5` in `features/channels/components/channels-columns.tsx` (8 places, `6496416`). Test `features/channels/components/__tests__/response-time-direction.test.tsx`: failed before, passes after.

## C. Batch 10: every page that is left

- Scanner: `.fa-review/scripts/scan-keys.mjs` (`@babel/parser`: string literals, template literals without substitutions, JSX text; test files skipped). `node ../.fa-review/scripts/scan-keys.mjs [--json out] [--folder features/x] [--all]` from `web/`. Keys built at run time (`t(\`preset.${value}\`)`) are not seen; the theme preset names were found by reading the config drawer.
- Start: 763 used keys missing. Each key was assigned to the first folder in the requested order. Method as in session 8: helpers drafted chunks from the call sites against `docs/i18n/fa.md` and ran `check-fa` (with `--src`) on a merged copy; I reviewed every value, aligned terms, dropped a few (below) and wrote each folder through the skill's `add-missing-keys.mjs` (created, run, deleted per commit; `newKeys.fa` read from a JSON file), then `bun run i18n:sync` and `bun run i18n:check-fa`.
- Review changes to the drafts: `sources`, `Seed`, `s`, `to confirm` left out (reasons in the list below); `Partial Submission` «ارسال بخشی»; task plugins normalised to «بارگذاری» for load and «بین‌مبدأ» for cross-origin; `pending` and `running` added (system tasks table).
- Channels recheck: the parser lists 46 keys owned by `features/channels`; all remain skipped on purpose (channel type brand labels, URLs, example values, the `h`/`m` unit letters glued to numbers, `more mapping` + English `s`). `Total tokens` (system settings) is shown without `t()`, so it is not added.
- New glossary rows (`305ee51`): node «گره», flow «جریان», artifact «خروجی», pop-up «پنجرۀ بازشو», inference «استنتاج», schema «طرح‌واره», marketplace «بازارچه», factory plugin «داخلی», dry run «اجرای آزمایشی», changelog «فهرست تغییرات», integrity hash / check «هش یکپارچگی، بررسی یکپارچگی».

### RTL fixes (no change in left-to-right languages)

| Commit | Fix | Test (before → after) |
| --- | --- | --- |
| `6496416` | channels table cells `-ml-1.5` → `-ms-1.5` | `channels/components/__tests__/response-time-direction.test.tsx`: 1 failed → 1 passed |
| `cb80274` | home hero `text-start`, call-to-action arrows mirror (`rtl:-scale-x-100`), feature badge `-end-1`; config drawer check badges `end-0` + `rtl:-translate-x-1/2`; sign-up and forgot-password email inputs `dir=ltr`; rankings number columns `text-end`; usage logs: cell buttons `text-start`, status badges `-ms-1.5`, dialog scroll gutters `pe-*`, the prompt and fail-reason copy buttons `end-2` with the text `pe-10` (they covered the start of right-to-left text), image URL `dir=ltr`; dashboard node filter badge `pe-1`; playground: icons `me-*`, parameter count badge `-end-1`, parameter list `pe-1`, starter prompts `text-start`, message text `text-end`/`text-start`, user bubble corner `rounded-ee-md` | `usage-logs/components/__tests__/copy-button-direction.test.tsx` (2), `playground/lib/__tests__/message-alignment.test.ts` (2), `auth/forgot-password/components/__tests__/email-direction.test.tsx` (1): 5 failed → 5 passed |
| `e6f7d76` | setup: language switcher `end-*`, step icons `me-2`, summary card `text-start`, usage-mode icon `ms-auto`, data directory path in `<bdi dir=ltr>`; system info tables `pe-4`/`text-end`/`text-start`; copy channel and Codex usage dialogs; deployments columns `-ms-*`; left to right: notification webhook/Bark/Gotify URL inputs, delete-account username confirmation, marketplace index URL, plugin URL import, plugin base URL, endpoint paths, source diff; task plugins lists `ps-5`, badges `ms-*`, detail sheet close button `end-3` + header `pe-*`, action columns `text-end`, install dialog `pe-6`/`ps-12` | `task-plugins/__tests__/source-direction.test.tsx` (2): 2 failed → 2 passed |

Not changed because the file already has lint errors on `upstream/main` (editing it would make it a changed file with errors): `features/pricing/components/model-details-api.tsx:694,701` (`text-right` on RPM/TPM headers) and `features/dashboard/components/models/models-chart-preferences.tsx:75,86` (`mr-2` on icons). Not changed on purpose: rankings trend arrows and chart axes (the chart time axis direction is still an open item), the hero terminal demo (a terminal stays left to right), `space-x-*` (logical in Tailwind 4), the config drawer layout illustration.

Existing components reused: every fix changes classes or `dir` on the existing components; no new components.

## Hard-coded or unkeyed English on these pages (not fixed here; each needs a new key in every locale, or a code change)

| File:line (web/src/) | String |
| --- | --- |
| features/system-settings/auth/custom-oauth/components/preset-selector.tsx:115,116 | `t('Select preset')`, key missing from en.json |
| features/system-settings/content/uptime-kuma-section.tsx:192, announcements-section.tsx:243, api-info-section.tsx:215, faq-section.tsx:183 | `{{count}} … deleted. Click "Save Settings" to apply.` toasts: t() keys missing from en.json |
| features/system-settings/general/channel-affinity/cache-stats-dialog.tsx:137-145 | row labels Prompt tokens, Cached tokens, Completion tokens, Total tokens, not passed through t() |
| features/system-settings/general/system-info-section.tsx:53,109 | `z.string().url()` without a message: zod's English "Invalid URL" |
| features/system-settings/content/announcements-section.tsx:100-130 | type labels Default, Ongoing, Success, Warning, Error |
| features/system-settings/content/announcements-section.tsx:300-310 | relative time `5m ago` / `h ago` / `d ago` built in code |
| features/system-settings/content/api-info-section.tsx:94-110 | colour names Blue … Slate |
| features/system-settings/maintenance/performance-section.tsx:688 | `Goroutines:` JSX text |
| features/system-settings/maintenance/performance-section.tsx:143, log-settings-section.tsx:113 | `formatBytes` returns `Bytes` / `0 Bytes` (duplicated, not `@/lib/format`) |
| features/system-settings/maintenance/performance-section.tsx:417 | low-disk warning joined from four keys with a hard-coded ` MB` |
| features/system-settings/request-policies/decision-record.tsx:41 | ` ms` unit |
| features/system-settings/request-policies/channel-health-section.tsx:69 | zod message "Enter a non-negative number or leave empty" (not an en.json key) |
| features/system-settings/hooks/use-form-dirty-guard.ts:52 | default unsaved-changes message for `window.confirm` |
| features/system-settings/request-policies/policy-label.ts:76 | unknown reason codes shown raw |
| features/system-settings/models/tiered-pricing-editor.tsx:704 | pricing preset labels (Flat …) |
| features/system-settings/models/constants.ts:51-54 | endpoint options, including `custom` |
| features/system-settings/models/request-simulation.tsx:199, upstream-ratio-sync.tsx:136 | raw engine and backend error details appended to translated text |
| features/system-settings/integrations/waffo-pancake-settings-section.tsx:522 | `+ Create` + English default pair name |
| features/system-settings/integrations/payment-method-dialog.tsx:90 | `(Epay: alipay)` suffix |
| components/ui/sidebar.tsx:219-220 | mobile sidebar SheetTitle / SheetDescription (the description key is now translated, the JSX does not call t()) |
| lib/oauth.ts:134, lib/passkey.ts:163 | plain `Error` messages (keys exist and are translated now, but the thrown text is English) |
| routes/_authenticated/chat/$chatId.tsx:129; chat2link.tsx:57-61 | English fallback and raw `error.message` |
| routes/_authenticated/chat/$chatId.tsx:101-105 | `{preset.name} {t('opens in an external client…')}`: name joined in front |
| lib/currency.ts:549; features/dashboard/components/overview/summary-cards.tsx:187 | `'Tokens'` currency label |
| features/auth/forgot-password/components/forgot-password-form.tsx:112 | `<FormLabel>Email</FormLabel>` |
| features/home/components/hero-terminal-demo.tsx:297,302; sections/features.tsx:141 | `ms`, `tokens`, `Docs` |
| features/rankings/components/model-leaderboard.tsx:105 | `by` before the vendor link |
| features/keys/components/data-table-bulk-actions.tsx:85; components/data-table/toolbar/bulk-actions.tsx:231-233 | `entityName='API key'`; English plural `s` + `selected` |
| features/pricing/components/model-details-apps.tsx:157,192,209 | app categories and descriptions (demo data) |
| features/pricing/components/model-details-api.tsx:693-707 | `RPM`, `TPM`, `RPD` headers |
| features/keys/components/dialogs/cc-switch-dialog.tsx:36,46,51 | default names `My Claude`, `My Codex`, `My Gemini` |
| features/usage-logs/components/dialogs/image-dialog.tsx:68; details-dialog.tsx:1328; lib/quota-audit-operation.ts:109-110; model-badge.tsx:78 | text joined around `t()` (`Task ID:` + id, `Param Override (n)`, quota summary with `→`, `Model: name`) |
| features/usage-logs/constants.ts (Midjourney submit result) | `Duplicate` shares the key of the verb (fa «تکثیر»), wrong sense here |
| features/dashboard/components/flow/flow-node-filter.tsx:131,207; flow-charts.tsx:251,382,602; lib/flow.ts:832,1112; users/user-charts.tsx:218 | `label: value` joins; `Intl.NumberFormat(undefined)`; share `toFixed(1)+'%'`; raw `Top {{count}}` |
| features/playground/lib/input/input-tool-utils.ts:51 | toast description is the raw action id (`upload-file` …) |
| features/playground/lib/message/message-streaming-utils.ts:245; hooks/use-chat-handler.ts:178-182; lib/streaming/stream-utils.ts:108 | error texts joined with a Latin colon / `HTTP n:` prefix |
| features/playground/components/input/playground-parameter-panel.tsx:130 | numeric values passed through `t()` |
| features/system-info/components/system-instances-panel.tsx:94-95,122,138,450,664; system-tasks-table.tsx:158; system-tasks-panel.tsx:89 | role badge `master`/`worker`; `Intl.NumberFormat(undefined)`; Gregorian title without locale; raw `{{seconds}}` |
| features/setup/setup-wizard.tsx:307; components/complete-step.tsx:50; database-step.tsx:123 | `Initialize` + system name; `Unknown`; `Data directory:` + path |
| features/setup (step numbers 1-4) | Latin digits in the step badges |
| features/system-settings/general/channel-affinity/cache-stats-dialog.tsx:137-145 | `Prompt tokens`, `Completion tokens` missing from en.json; `Cached tokens`, `Total tokens` not through `t()` |
| features/channels/components/dialogs/codex-usage-dialog.tsx:207-217,236 | `h`/`m`/`s` glued to numbers |
| features/security/components/dialogs/delete-account-dialog.tsx:101 | `Type` <name> `to confirm` (upstream fix branch `fix/delete-account-confirm-label`) |
| features/about/index.tsx:68-109 | footer and compliance sentence built from fragments |
| features/users/components/user-quota-dialog.tsx:62-70; features/models/components/deployments-columns.tsx:174,183 | quota preview with `→`; `%` label; `Approx.` + value |
| features/task-plugins/components/upload-dialog.tsx:170,215,247; plugin-card.tsx:103; marketplace-capabilities.tsx:74,84,104,114; index.tsx:187; plugin-metadata-card.tsx:43; marketplace-plugin-card.tsx:179; usage-schema-table.tsx:99 | template descriptions, `Versions tab` (the tab is «Version history»), `label: value` joins, Latin `, ` lists, `v1 → v2`, `value → label` |
| features/task-plugins/components/marketplace-install-dialog.tsx:110,155; marketplace-panel.tsx:79,157; plugin-sandbox.tsx:43 | English `Error` messages shown in the dialog; `t(source.name)` on an admin-entered name |
| features/task-plugins (plugin cards) | plugin names and descriptions come from the plugin metadata (backend), English |

## Keys used in the code and still missing from fa.json (161, final scan of web/src)

Each key is listed once, under the first folder the scanner meets it in.

**assets** (7)

| Key | Reason | First use |
| --- | --- | --- |
| `Discord` | brand/provider name; stays Latin (and not passed through t() at this call site) | assets/brand-icons/icon-discord.tsx:39 |
| `GitHub` | brand/provider name; stays Latin (and not passed through t() at this call site) | assets/brand-icons/icon-github.tsx:39 |
| `LinuxDO` | brand/provider name; stays Latin (and not passed through t() at this call site) | assets/brand-icons/icon-linuxdo.tsx:34 |
| `New API` | product name (DEFAULT_SYSTEM_NAME), stays Latin | assets/logo.tsx:39 |
| `Stripe` | brand name (literal <title> in SVG icon, not a t() call) | assets/brand-icons/icon-stripe.tsx:39 |
| `Telegram` | brand/provider name; stays Latin (and not passed through t() at this call site) | assets/brand-icons/icon-telegram.tsx:39 |
| `WeChat` | brand/provider name; stays Latin (and not passed through t() at this call site) | assets/brand-icons/icon-wechat.tsx:34 |

**components** (15)

| Key | Reason | First use |
| --- | --- | --- |
| `...` | pagination/truncation marker pushed into a page-number array in lib/utils.ts, not passed through t(); identica | components/data-table/core/pagination.tsx:63 |
| `Claude` | brand/provider name; stays Latin (and not passed through t() at this call site) | components/ai-elements/open-in-chat.tsx:146 |
| `default` | identifier (enum value for theme preset/radius/font and announcement type), not user-facing text | components/config-drawer.tsx:282 |
| `field` | identifier: `'field' in condition` property check in breakdown-tier-match.ts, not UI text | components/ui/field.tsx:96 |
| `footer.columns.related.links.midjourney` | brand/project name (MjProxy) | components/layout/components/footer.tsx:210 |
| `footer.columns.related.links.newApiKeyTool` | project name (new-api-key-tool) | components/layout/components/footer.tsx:214 |
| `footer.columns.related.links.oneApi` | project name (One API) | components/layout/components/footer.tsx:206 |
| `https://github.com/QuantumNous/new-api` | URL | components/layout/components/footer.tsx:132 |
| `JSON` | format name, stays English | components/json-code-editor.tsx:278 |
| `K` | identifier (tokenUnit enum 'K' in the pricing route search schema); unit that stays Latin | components/search.tsx:55 |
| `log` | identifier: ARIA role value and log-level enum, not a t() call | components/ai-elements/conversation.tsx:36 |
| `OpenAI` | brand/provider name; stays Latin (and not passed through t() at this call site) | components/ai-elements/open-in-chat.tsx:128 |
| `s` | unit glued to a number in codex-usage-dialog.tsx (5m 30s) | components/floating-window.tsx:76 |
| `sources` | joined as t(Used) + count + t(sources) in components/ai-elements/sources.tsx:60; cannot be ordered in Persian | components/ai-elements/sources.tsx:60 |
| `x` | code: string escape char in parser; literal multiplier suffix '{n}x' elsewhere, not passed through t() | components/ui/carousel.tsx:76 |

**features/about** (4)

| Key | Reason | First use |
| --- | --- | --- |
| `JustSong` | person/author name (brand), identical in Persian | features/about/index.tsx:95 |
| `NewAPI` | brand/provider name used as channel type label | features/about/index.tsx:68 |
| `One API` | project/brand name | features/about/index.tsx:86 |
| `QuantumNous` | organization name (protected brand) | features/about/index.tsx:77 |

**features/auth** (2)

| Key | Reason | First use |
| --- | --- | --- |
| `name@example.com` | example value (email placeholder) | features/auth/sign-up/components/sign-up-form.tsx:316 |
| `OIDC` | protocol acronym that stays English (provider label) | features/auth/lib/oauth.ts:62 |

**features/channels** (63)

| Key | Reason | First use |
| --- | --- | --- |
| `360` | brand/provider name used as channel type label | features/channels/constants.ts:56 |
| `_copy` | identifier: default name suffix also used as the input's initial value and placeholder | features/channels/components/dialogs/copy-channel-dialog.tsx:45 |
| `"default": "us-central1", "claude-3-5-sonnet-20240620": "europe-west1"` | example value (JSON code) | features/channels/components/drawers/channel-mutate-drawer.tsx:3982 |
| `AccessKey / SecretAccessKey` | API field names | features/channels/components/drawers/channel-mutate-drawer.tsx:3708 |
| `Ali` | brand name (en value 'Alibaba Bailian'), channel type label; Persian would be identical | features/channels/constants.ts:54 |
| `Anthropic` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:51 |
| `AWS` | brand/provider name used as channel type label | features/channels/constants.ts:66 |
| `Azure` | brand/provider name used as channel type label | features/channels/constants.ts:40 |
| `AZURE_OPENAI_ENDPOINT` | environment variable name | features/channels/components/drawers/channel-mutate-drawer.tsx:3558 |
| `Baidu` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:52 |
| `Baidu V2` | brand/provider name used as channel type label | features/channels/constants.ts:79 |
| `Cloudflare` | brand/provider name used as channel type label | features/channels/constants.ts:72 |
| `Cohere` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:67 |
| `Coze` | brand/provider name used as channel type label | features/channels/constants.ts:82 |
| `DeepSeek` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:76 |
| `Dify` | brand name (mock app list entry) | features/channels/constants.ts:70 |
| `Doubao` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:87 |
| `e.g., gpt-4.1-nano,regex:^claude-.*$,regex:^sora-.*$` | example value (code placeholder, session 8 skip kept) | features/channels/components/drawers/channel-mutate-drawer.tsx:2349 |
| `FastGPT` | brand name (mock app list entry) | features/channels/constants.ts:59 |
| `Gemini` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:61 |
| `gpt-3.5-turbo` | model name (example value) | features/channels/components/model-mapping-editor.tsx:333 |
| `gpt-3.5-turbo-0125` | model name (example value) | features/channels/components/model-mapping-editor.tsx:333 |
| `h` | unit suffix glued to a number with no space (`${hours}${t('h')}`); a Persian word would render '3ساعت'; fix co | features/channels/components/dialogs/codex-usage-dialog.tsx:207 |
| `HTTP/1.1` | protocol name | features/channels/components/drawers/channel-mutate-drawer.tsx:2078 |
| `https://ark.ap-southeast.bytepluses.com` | URL | features/channels/components/drawers/channel-mutate-drawer.tsx:4015 |
| `https://ark.cn-beijing.volces.com` | URL | features/channels/components/drawers/channel-mutate-drawer.tsx:1093 |
| `https://cloud.siliconflow.cn/i/hij0YNTZ` | URL | features/channels/components/drawers/channel-mutate-drawer.tsx:3845 |
| `Invalid items format` | not user-facing: message of an Error thrown and caught; the catch shows t('Failed to parse group items') inste | features/channels/components/drawers/channel-mutate-drawer.tsx:1312 |
| `Jimeng` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:84 |
| `Jina` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:71 |
| `Kling` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:83 |
| `LingYiWanWu` | brand/provider name used as channel type label | features/channels/constants.ts:65 |
| `m` | unit suffix glued to a number with no space (`${minutes}${t('m')}`); fix code first (see bugs) | features/channels/components/dialogs/codex-usage-dialog.tsx:207 |
| `MiniMax` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:68 |
| `Mistral` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:75 |
| `MjProxy` | brand/provider name used as channel type label | features/channels/constants.ts:39 |
| `MjProxyPlus` | brand/provider name used as channel type label | features/channels/constants.ts:42 |
| `MokaAI` | brand/provider name used as channel type label | features/channels/constants.ts:77 |
| `Moonshot` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:62 |
| `more mapping` | fragment with English plural 's' appended by code (`{t('more mapping')}{n > 1 ? 's' : ''}`); any Persian would | features/channels/components/drawers/channel-mutate-drawer.tsx:2452 |
| `OhMyGPT` | brand/provider name used as channel type label | features/channels/constants.ts:44 |
| `Ollama` | brand/provider name used as channel type label | features/channels/constants.ts:41 |
| `OpenRouter` | brand/provider name used as channel type label | features/channels/constants.ts:57 |
| `org-...` | example value (OpenAI organization ID prefix) | features/channels/components/drawers/channel-mutate-drawer.tsx:2613 |
| `Perplexity` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:64 |
| `Replicate` | brand/provider name used as channel type label | features/channels/constants.ts:89 |
| `Responses WebSocket` | feature/protocol name; existing fa.json keeps 'Responses WebSocket' in Latin (e.g. «فعال‌سازی Responses WebSoc | features/channels/components/channel-quick-options.tsx:169 |
| `SGLang` | brand/provider name used as channel type label | features/channels/constants.ts:96 |
| `SiliconFlow` | brand/provider name used as channel type label | features/channels/constants.ts:73 |
| `socks5://user:pass@host:port` | example value (proxy URL) | features/channels/components/drawers/channel-mutate-drawer.tsx:1811 |
| `Sora` | brand/provider name used as channel type label | features/channels/constants.ts:88 |
| `Submodel` | brand/provider name used as channel type label | features/channels/constants.ts:86 |
| `SunoAPI` | brand/provider name used as channel type label | features/channels/constants.ts:69 |
| `Tencent` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:60 |
| `Vertex AI` | brand/provider name used as channel type label | features/channels/constants.ts:74 |
| `Vidu` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:85 |
| `vLLM` | brand/provider name used as channel type label | features/channels/constants.ts:95 |
| `VolcEngine` | brand/provider name used as channel type label | features/channels/constants.ts:78 |
| `xAI` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:81 |
| `Xinference` | brand/provider name used as channel type label | features/channels/constants.ts:80 |
| `Xunfei` | brand/provider name used as channel type label | features/channels/constants.ts:55 |
| `Zhipu` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/channels/constants.ts:53 |
| `Zhipu GLM` | brand/provider name used as channel type label | features/channels/constants.ts:63 |

**features/dashboard** (3)

| Key | Reason | First use |
| --- | --- | --- |
| `ms` | unit that stays Latin in a monospace terminal demo; hard-coded, not passed through t() | features/dashboard/components/overview/api-info-item.tsx:82 |
| `token` | identifier: enum/unit value 'token' (QUOTA_TYPES, billing unit), never passed to t() in these call sites | features/dashboard/types.ts:60 |
| `tokens` | identifier (currency display kind enum 'tokens' in lib/currency.ts and lib/format.ts), not user-facing | features/dashboard/index.tsx:66 |

**features/home** (2)

| Key | Reason | First use |
| --- | --- | --- |
| `API` | acronym that stays English; array literal in features.tsx:141, not passed through t() | features/home/components/sections/features.tsx:141 |
| `CC Switch` | product/brand name (alt text and label of an external app link) | features/home/components/sections/hero.tsx:208 |

**features/keys** (1)

| Key | Reason | First use |
| --- | --- | --- |
| `Codex` | product/brand name | features/keys/components/dialogs/cc-switch-dialog.tsx:45 |

**features/playground** (2)

| Key | Reason | First use |
| --- | --- | --- |
| `Seed` | sampling parameter name, kept like Top P | features/playground/lib/parameters/playground-parameters.ts:83 |
| `Top P` | API parameter name (top_p); the conventional label stays Latin in Persian AI tools | features/playground/lib/parameters/playground-parameters.ts:47 |

**features/pricing** (6)

| Key | Reason | First use |
| --- | --- | --- |
| `https://api.example.com` | example value / URL fallback returned as server address, not UI text | features/pricing/components/model-details-api.tsx:460 |
| `Local` | identifier: Go timezone name compared in billing-expression runtime, not UI text | features/pricing/lib/billing-expression/runtime.ts:128 |
| `RPM` | acronym that stays English; also the column header is a literal, not passed through t() | features/pricing/components/model-details-api.tsx:693 |
| `stream` | API field name (request parameter name in the API parameter table) | features/pricing/lib/mock-stats.ts:568 |
| `TPM` | acronym that stays English; also the column header is a literal, not passed through t() | features/pricing/components/model-details-api.tsx:700 |
| `uptime` | identifier: chart series id / yField, not UI text | features/pricing/components/model-details-charts.tsx:222 |

**features/profile** (3)

| Key | Reason | First use |
| --- | --- | --- |
| `https://api.day.app/yourkey/{{title}}/{{content}}` | example URL value in placeholder | features/profile/components/tabs/notification-tab.tsx:204 |
| `https://example.com/webhook` | example URL value in placeholder | features/profile/components/tabs/notification-tab.tsx:178 |
| `https://gotify.example.com` | example URL value in placeholder | features/profile/components/tabs/notification-tab.tsx:224 |

**features/security** (2)

| Key | Reason | First use |
| --- | --- | --- |
| `OAuth` | protocol name that stays English | features/security/components/login-session-utils.ts:63 |
| `to confirm` | joined after t(Type), which is the noun «نوع» elsewhere (delete-account-dialog.tsx:101) | features/security/components/dialogs/delete-account-dialog.tsx:101 |

**features/system-info** (1)

| Key | Reason | First use |
| --- | --- | --- |
| `CPU` | Acronym kept in Latin; existing fa.json already writes CPU (e.g. "CPU Threshold (%)" -> «آستانۀ CPU (درصد)»),  | features/system-info/components/system-instances-panel.tsx:257 |

**features/system-settings** (41)

| Key | Reason | First use |
| --- | --- | --- |
| `1000` | example value (numeric input placeholder) | features/system-settings/general/checkin-settings-section.tsx:160 |
| `10000` | example value (numeric input placeholder) | features/system-settings/general/checkin-settings-section.tsx:182 |
| `[{"ChatGPT":"https://chat.openai.com"},{"Lobe Chat":"https://chat-prev` | example value (JSON code) | features/system-settings/content/chat-settings-section.tsx:182 |
| `[{"name":"支付宝","type":"alipay","icon":"SiAlipay"}]` | example value (JSON code) | features/system-settings/integrations/payment-settings-section.tsx:996 |
| `/status/` | URL path fragment | features/system-settings/content/uptime-kuma-section.tsx:426 |
| `/your/endpoint` | example value (endpoint path) | features/system-settings/models/channel-selector-dialog.tsx:269 |
| `192.168.1.1&#10;10.0.0.0/8` | example value (IP list) | features/system-settings/request-limits/ssrf-section.tsx:380 |
| `80,443,8080` | example value (port list) | features/system-settings/request-limits/ssrf-section.tsx:400 |
| `Alipay` | brand name; also used as the payment method's stored name (`name: t('Alipay')`), so translating would save Per | features/system-settings/integrations/payment-method-dialog.tsx:90 |
| `checkout.session.completed` | identifier (Stripe webhook event name) | features/system-settings/integrations/payment-settings-section.tsx:1284 |
| `checkout.session.expired` | identifier (Stripe webhook event name) | features/system-settings/integrations/payment-settings-section.tsx:1288 |
| `e.g. 401, 403, 429, 500-599` | example value (status code list, session 8 skip kept) | features/system-settings/request-policies/channel-health-section.tsx:553 |
| `example.com&#10;blocked-site.com` | example value (domain list) | features/system-settings/request-limits/ssrf-section.tsx:315 |
| `example.com&#10;company.com` | example value (domain list) | features/system-settings/auth/basic-auth-section.tsx:252 |
| `gpt-4` | model name (example value) | features/system-settings/models/model-pricing-sheet.tsx:895 |
| `Grok` | brand/provider name; stays Latin (and not passed through t() at this call site) | features/system-settings/models/section-registry.tsx:111 |
| `https://docs.example.com` | URL | features/system-settings/general/system-info-section.tsx:252 |
| `https://example.com/logo.png` | URL | features/system-settings/general/system-info-section.tsx:231 |
| `https://example.com/qr-code.png` | URL | features/system-settings/auth/oauth-section.tsx:1127 |
| `https://example.com/topup` | URL | features/system-settings/general/quota-settings-section.tsx:317 |
| `https://gateway.example.com` | URL | features/system-settings/integrations/payment-settings-section.tsx:1192 |
| `https://pay.example.com` | URL | features/system-settings/integrations/payment-settings-section.tsx:1168 |
| `https://provider.com/.well-known/openid-configuration` | URL | features/system-settings/auth/oauth-section.tsx:729 |
| `https://status.example.com` | URL | features/system-settings/content/uptime-kuma-section.tsx:404 |
| `https://wechat-server.example.com` | URL | features/system-settings/auth/oauth-section.tsx:1077 |
| `https://worker.example.workers.dev` | URL | features/system-settings/integrations/worker-settings-section.tsx:129 |
| `https://your-server.example.com` | URL | features/system-settings/auth/custom-oauth/components/preset-selector.tsx:124 |
| `my-status` | example value (slug) | features/system-settings/content/uptime-kuma-section.tsx:422 |
| `New API &lt;noreply@example.com&gt;` | example value (email sender) | features/system-settings/integrations/email-settings-section.tsx:376 |
| `noreply@example.com` | example value (email) | features/system-settings/integrations/email-settings-section.tsx:353 |
| `price_xxx` | example value (Stripe price ID) | features/system-settings/integrations/payment-settings-section.tsx:1367 |
| `smtp.example.com` | example value (host) | features/system-settings/integrations/email-settings-section.tsx:200 |
| `SSL/TLS` | protocol name | features/system-settings/integrations/email-settings-section.tsx:274 |
| `STARTTLS` | protocol name | features/system-settings/integrations/email-settings-section.tsx:286 |
| `Total tokens` | shown without t() in cache-stats-dialog.tsx:145 | features/system-settings/general/channel-affinity/cache-stats-dialog.tsx:145 |
| `TTL` | abbreviation kept in Latin; existing fa.json keeps TTL («TTL (ثانیه)») | features/system-settings/general/channel-affinity/session-rules-table.tsx:200 |
| `Uptime Kuma` | product name (tab label) | features/system-settings/content/section-registry.tsx:88 |
| `vip` | example group name; call site is a literal <td>vip</td>, not t() | features/system-settings/models/group-ratio-form.tsx:594 |
| `Waffo` | brand name (payment provider) | features/system-settings/integrations/payment-settings-section.tsx:888 |
| `WeChat Pay` | brand name; also used as the payment method's stored name (`name: t('WeChat Pay')`) | features/system-settings/integrations/payment-method-dialog.tsx:96 |
| `whsec_xxx` | example value (Stripe webhook secret) | features/system-settings/integrations/payment-settings-section.tsx:1340 |

**features/task-plugins** (1)

| Key | Reason | First use |
| --- | --- | --- |
| `override` | identifier: QuotaAdjustMode enum value, not passed to t() | features/task-plugins/index.tsx:184 |

**features/usage-logs** (1)

| Key | Reason | First use |
| --- | --- | --- |
| `{{method}} {{route}}` | Placeholders only (HTTP method and route of the generic audit fallback); the Persian value would equal the Eng | features/usage-logs/lib/format.ts:564 |

**lib** (7)

| Key | Reason | First use |
| --- | --- | --- |
| `Forest Whisper` | theme preset `name` field, not passed through t(); the UI shows t(`preset.forest-whisper`) instead | lib/theme-customization.ts:67 |
| `Lake View` | theme preset `name` field, not passed through t(); the UI shows t(`preset.lake-view`) instead | lib/theme-customization.ts:57 |
| `Lavender Dream` | theme preset `name` field, not passed through t(); the UI shows t(`preset.lavender-dream`) instead | lib/theme-customization.ts:77 |
| `Ocean Breeze` | theme preset `name` field, not passed through t(); the UI shows t(`preset.ocean-breeze`) instead | lib/theme-customization.ts:72 |
| `Rose Garden` | theme preset `name` field, not passed through t(); the UI shows t(`preset.rose-garden`) instead | lib/theme-customization.ts:52 |
| `Sunset Glow` | theme preset `name` field, not passed through t(); the UI shows t(`preset.sunset-glow`) instead | lib/theme-customization.ts:62 |
| `Underground` | theme preset `name` field, not passed through t(); the UI shows t(`preset.underground`) instead | lib/theme-customization.ts:47 |


## Verification (`feat/fa-locale`, from `web/`, at `31f68f8`)

| Command | Result |
| --- | --- |
| `git remote -v`, `git push --dry-run origin HEAD`, dry run to a new branch | origin `https://github.com/yaser-k/new-api-fork`; both accepted after resetting the stale local branch to `origin/feat/fa-locale` (`57c75bd`) |
| `git fetch upstream main` | `upstream/main` = `c2b7a9a`; upstream push URL `DISABLED` |
| `bun install` | 1206 packages |
| `bun run typecheck` | `tsgo -b`, exit 0 |
| `bun run lint` | exit 1: **150 errors, 65 warnings**; `upstream/main` (own worktree and install): **182 errors, 66 warnings**; errors only on this branch (file + rule): **none**; errors in the 290 files changed against upstream: **0** |
| `bun run test` | `Test Files 214 passed (214)`, `Tests 2371 passed (2371)` |
| `bun run build` | exit 0 (total 66909.6 kB / 20615.2 kB gzip at `e6f7d76`; rebuilt at `31f68f8` for the Go binary, exit 0) |
| `bun run i18n:sync` | exit 0; fa partial, missing 1063, extras 0; seven required locales missing 0, extras 0 |
| `bun run i18n:check-fa` | `check-fa: 5732 keys, no findings` (includes the new source scan) |
| `git diff --stat upstream/main -- web/src/i18n/locales/` | `fa.json` +5736; en, fr, ja, ru, vi, zh-TW, zh +17 each (the 9 + 8 keys from the merged fix branches); `git diff --stat 57c75bd -- web/src/i18n/locales/`: only `fa.json` |
| `go build -o <scratch>/bin/new-api .` (Go 1.25.1) | exit 0, embeds the fresh `web/dist` (rebuilt after `31f68f8`) |

### Running app

```
SQLITE_PATH=<scratch>/run/one-api.db GLOBAL_API_RATE_LIMIT_ENABLE=false GLOBAL_WEB_RATE_LIMIT_ENABLE=false \
  CRITICAL_RATE_LIMIT_ENABLE=false SEARCH_RATE_LIMIT_ENABLE=false TZ=UTC <scratch>/bin/new-api --port 3300
GET /api/setup -> {"status":false,"root_init":false,"database_type":"sqlite"}
node .fa-review/scripts/shots10.mjs http://127.0.0.1:3300 .fa-review setup fa   # before the admin exists
python3 .fa-review/scripts/seed.py  http://127.0.0.1:3300 <db> en   # POST /api/setup -> "系统初始化成功", success true; demo data
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 <db>
python3 .fa-review/scripts/seed7.py http://127.0.0.1:3300
python3 .fa-review/scripts/seed8.py <db>
python3 .fa-review/scripts/seed10.py <db>                           # last 7 days of hourly usage, two demo nodes
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 lang fa
node .fa-review/scripts/shots10.mjs http://127.0.0.1:3300 .fa-review main fa
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 lang en
node .fa-review/scripts/shots10.mjs http://127.0.0.1:3300 .fa-review main en   # custom OAuth dialog in English
```

(`shots10.mjs` imports `playwright`; it was run from a scratch copy with `node_modules/playwright` linked to the global install.)

Playwright (global), Chromium 141.0.7390.37 headless, 1440×900, light theme, UTC. Persian: `dir=rtl lang=fa`; English: `dir=ltr lang=en`. 15 captures, none blank (fewest colours 1205; blank threshold 16). Console errors: the 401 from the pre-login session probe, `ERR_CERT_AUTHORITY_INVALID` / `ERR_TUNNEL_CONNECTION_FAILED` for external resources blocked by the sandbox proxy.

- `160-setup-rtl.png`: setup wizard before the admin exists (visible Latin: `New API`, `SQLite`).
- `161-settings-custom-oauth-dialog-rtl.png` / `-en.png`: the fixed placeholders; English unchanged.
- `162-settings-auth-oauth-rtl.png`: OAuth section (GitHub client ID placeholder `<RLI>…<PDI>`, `direction: ltr`).
- `163-channels-table-response-time-rtl.png`: «۴۵۶ ms», «1.23 ثانیه», «12.35 ثانیه».
- `164-usage-logs-rtl.png`, `165-dashboard-rtl.png`, `166-dashboard-flow-rtl.png`, `167-playground-rtl.png`, `168-rankings-rtl.png` (full page), `169-api-keys-rtl.png`, `170-system-info-rtl.png` (full page), `171-task-plugins-rtl.png`, `173-home-rtl.png`. Visible text without Persian letters: model, user, key and channel names, `RPM`/`TPM`, `CPU`, `master`, `linux/amd64`, task plugin names and descriptions (plugin metadata), the home page brand names and the English footer line.

## Upstream split plan (proposal only; nothing built)

Sizes and grouping as in session 8, with this session's commits added. Commits that mix concerns (marked *) need their hunks split when the PR branch is built.

Upstream-ready branches in the fork (each from `upstream/main`), unchanged this session:

| Branch | Head | Content |
| --- | --- | --- |
| `fix/dashboard-chart-time-order` | `81b140e` | chart points ordered by timestamp; **merged** (`bc388d4`); `04b86d9` belongs to no upstream branch (leave out, or fold into PR 1 where it touches PR 1 files) |
| `fix/billing-status-label` | `6bf13ab` | not merged |
| `fix/delete-account-confirm-label` | `747769c` | not merged |
| `fix/2fa-setup-step-label` | `6d614c0` | not merged |
| `fix/quota-insufficient-i18n` | `28b7893` | not merged |
| `fix/legal-consent-sentence` | `6a660f7` | consent line + terms footer; **merged** (`44610c0`, `6f7168b`). The terms footer change overlaps an older upstream PR by another contributor, so this branch is **not offered as its own PR for now**; kept as it is |
| `fix/ui-label-strings` | `3b86126` | **merged** (`b2d14dc`) |

| # | PR | Commits / files | Tests | Other languages |
| --- | --- | --- | --- | --- |
| 1 | `fix(web): right-to-left layout in shared components` | `c78c83f`* (without the `formatQuota` locale hunks; legal-consent hunk superseded), `d8ee064`, `cb25020`, `8ccec81`, `936ab28`, `129d598`, the RTL half of `f01c5af`*, the sidebar side of `b2720c9`*, `666c43c` (header positions, only where PR 1 touches the file), `0d40127`, `f40c978`, `10cd251` + `9f8f102` (squash), `53458ed`, `7b04478`, **and from this session `6496416`, `cb80274`, `e6f7d76`** | rtl-layout, code-block-direction, column-pinning, column-resize-direction, app-sidebar-direction, date-range-direction, terminal-direction, profile-header-direction, value-direction, timing-direction, spinner-direction, json-code-editor-direction, code-direction, advanced-toggle-direction, **response-time-direction, copy-button-direction, message-alignment, email-direction, source-direction** | No visible change in LTR |
| 2 | `feat(i18n): Persian (fa) partial locale, tooling and docs` + first batch | `b2720c9`* (without the sidebar), `07e070d`, `759a31e`, `f5296da` + `776c03c`, `bfe01bb`, the script part of `fa96f8b`; AGENTS.md, web/AGENTS.md, the i18n skill; `docs/i18n/fa.md` (with the glossary rows from `4d901bd`, `d1e501d` **and `305ee51`**), **`2242169`** (check-fa placeholder scan, rule 11, tests) | languages, direction-provider, check-fa (now with the placeholder cases), persian-monospace, intl-locale lint case | Language switcher lists «فارسی»; `docs/i18n/fa.md` is a new file under `docs/` (ask the maintainers); check-fa relies on `@babel/parser` from the existing dependency tree (ask whether to declare it) |
| 3 | `feat(web): Solar Hijri dates, chart axes and date pickers in Persian` | `0ea975e`, `c2369c8` + `42a98a2`, `170ba72`, `9d22fac` | display-date-locale, activity-time-cell-dates, login-session-dates, date-picker-display, calendar-persian, chart-time-locale | None |
| 4 | `feat(web): Persian labels for audit roles and sign-in methods` | net of `f767402` + `9cd40a8` | audit-content-locale, details-locale | None (could fold into 3) |
| 5 | `fix(web): format money and numbers in the interface language` | locale half of `f01c5af`*, `formatQuota` hunks of `c78c83f`*, the `locale` argument in `recharge-form-card.tsx`, **`31f68f8`** (response-time milliseconds; no change for other languages) | format-quota-locale, format-currency-locale, amount-locale, summary-cards-locale, profile-header-locale, **response-time-format** | **Yes** for the older commits (its own PR, decision 1 of session 6); `31f68f8` alone changes nothing outside Persian and could go with PR 6 if 5 waits |
| 6 | `feat(i18n): Persian translation batches` | `fa96f8b`, `551a8a3`, `795d296`, `4db67e2`, `3825299`, `aaba1b0`, `8a76a75`, `ada5af2`, `c6aaad5`, `4d901bd` (fa.json part), `17d9c1b`, `58de3e3`, `f7ad4f7`, `4ba8a4c`, `3b076b1`, `25399dc`, `86a8c7c`, `97b1b72`, `bcc8ea7`, `3ca2feb`, `d1e501d` (fa.json part), `0097c77`, `c2798f3`, `4c66300`, **`67e72b5`, `39a571f`, `5cd6e90`, `b744e97`, `334c3d9`, `2adca66`, `0d1b17b`, `467f231`, `a14be44`, `bdf0202`** (fa.json, plus the Persian cases in `legal-consent.test.tsx` and `terms-footer-persian.test.tsx`) | `bun run i18n:check-fa` | None: only `fa.json` |

Leave out of every PR: `.fa-review/` (including `2b721c2`), all hand-off and script commits, the merge commits. Order: the upstream fix branches that are offered (not `fix/legal-consent-sentence` for now), then 1, 2, 3 (+4), 6; 5 when upstream agrees to the behaviour change. `67e72b5` (fa.json) must land together with or after `2242169`'s check, or the check-fa test fails; they sit in PR 6 and PR 2, so PR 2's project-sources test case needs PR 6 or should move to PR 6.

## Remaining issues

1. **Code bugs found while translating (English affected too), kept listed:** the session 7 and 8 items in `git show 57c75bd:.fa-review/REPORT.md` (channels glued labels, `&apos;`, `&mdash;`, `&#10;`, plurals, the Gemini sentence, the compliance separators, OAuth callback base), plus the joins and raw numbers in the hard-coded table above.
2. **Dates not in the fork's pattern:** the compliance "Confirmed at" time (`integrations/payment-settings-section.tsx:840`, `toLocaleString()`) and the announcements table (`content/announcements-section.tsx:394`, `dayjs().format(…)`, plus its own relative time), kept for a later upstream PR (decision 3). New in this batch: the system info "Last seen" and "Updated" titles use `formatTimestampToDate` without the locale (`system-instances-panel.tsx:450`, `system-tasks-table.tsx:158`).
3. **Digits:** several values are interpolated raw and show Latin digits in Persian: seconds in the response-time badge, `Top {{count}}`, `{{seconds}}`, the system info percentages and bytes (`Intl.NumberFormat(undefined)`), the dashboard flow numbers, the setup step numbers, task plugin counts.
4. **Arrows:** `→` joined in code (quota audit summary, user quota preview, task plugin `v1 → v2` and `value → label`) points the wrong way in right-to-left text; the rankings trend icons and chart axes are not mirrored.
5. **Lint-blocked RTL fixes:** `pricing/components/model-details-api.tsx:694,701` and `dashboard/components/models/models-chart-preferences.tsx:75,86` (above).
6. Session 6 and 8 items still open: dashboard chart header total, subscriptions list digits, audit details `ID`, native date inputs, chart time axis direction, backend content in Chinese or English, `{{action}}` raw in two audit strings, carousel arrows, other `<pre>` blocks, the Public Sans font name mismatch, `Toggle Sidebar`.
7. «بارگیری» (18 older values) and «بارگذاری» are both used for "load" in fa.json; this batch uses «بارگذاری». «ناهم‌گام» and «ناهمگام» are both used for "async".

## New open questions

1. **`@babel/parser` in check-fa**: it is installed only as a transitive dependency. Declare it as a devDependency in the fork (changes `package.json` and `bun.lock`), or keep it transitive and note it in PR 2?
2. **Seconds in the response-time badge** still show Latin digits («1.23 ثانیه» next to «۴۵۶ ms»). Formatting them with the locale changes the decimal separator in French, Russian and Vietnamese («1,23 s»). Leave as is, or format them for every language upstream (PR 5)?
3. **«بارگیری» vs «بارگذاری»** for "load" (18 older values): align the older values to «بارگذاری» in a small fa.json commit?
4. **The 902 en.json keys the scanner does not find in the code**: many look unused (old pages, backend-only). Translate them anyway, or leave them until the code uses them?
