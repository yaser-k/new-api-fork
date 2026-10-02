# Persian (fa) locale for new-api: state and hand-off

Temporary review folder. Delete `.fa-review/` before any upstream PR.

Fork `yaser-k/new-api-fork`, branch `feat/fa-locale`, merged with `upstream/main` at `1a4166d` in session 13 (merge `d95fb41`). Upstream issue: QuantumNous/new-api#7198. Session 13 also built the three upstream-ready branches `pr/fa-rtl-layout`, `pr/fa-locale` and `pr/fa-dates-numbers` (section 5); their draft PR texts are `pr-rtl-layout.md`, `pr-locale.md` and `pr-dates-numbers.md`, and the file-by-file coverage is `coverage.md`, all in this folder.

## 1. What the Persian locale covers

| | |
| --- | --- |
| Keys | **5817** of the 6873 keys in `en.json` are translated in `fa.json`. `fa` is a partial locale: a key that is missing from `fa.json` shows its English text (per key, at run time). |
| Keys still English | 1056 (session 12's lists in section 4 still apply; session 13 translated the 95 keys upstream added and `Logs` and `Failed to save`, which upstream's permission catalog now uses, and dropped 13 keys upstream removed): **161** that the code uses (brand and provider names, URLs, example values, identifiers, unit letters glued to numbers, fragments that cannot be ordered in Persian; full list with reasons in section 4) and **901** that no code in `web/src` uses (section 4) |
| Pages | Every page and shared component under `web/src`: auth and sign-up, home, pricing and model details, rankings, dashboard (overview, models, flow, users), API keys, playground and chat, usage, task and drawing logs, audit log, wallet and subscriptions, profile and security, channels, models and deployments, users, redemption codes, subscriptions admin, system info, task plugins, the setup wizard, every system settings section, the legal pages, error pages and the about page |
| Dates | Displayed dates are Solar Hijri in numeric year/month/day order with 24-hour time (`۱۴۰۵/۰۷/۰۳ ۱۵:۰۵:۰۹`); relative times are Persian («۳ دقیقه پیش»). Table and log cells carry the Gregorian date and time as their `title`. Values sent to the API, filters, inputs, exports and copied text stay Gregorian. Dashboard chart axes are Solar Hijri and stay in time order across Nowruz |
| Date pickers | `calendar.tsx` (used by the date and date-time pickers) shows the Solar Hijri calendar through `@daypicker/persian`, loaded lazily only for Persian; the returned `Date` values are unchanged |
| Numbers | Formatted with the interface locale: Persian digits (`۱۲۳`, `۱٫۲۳`), switchable to Latin digits with one constant (`PERSIAN_INTL_LOCALE = 'fa-u-nu-latn'`) |
| Right to left | Selecting Persian sets `dir="rtl"` and `lang="fa"`; a direction provider feeds Base UI and the calendar. Components use logical classes (`ms-*`, `pe-*`, `start-*`, `text-end`), direction-dependent icons are mirrored, code, keys, sample output and `pre` blocks keep their own text direction (`unicode-bidi: plaintext`), left-to-right inputs (URLs, IDs, code) keep `dir="ltr"`, before-and-after arrows and the carousel follow the page direction, and Persian text uses Vazirmatn (also inside monospace text). The config drawer can still force LTR or RTL. Since session 12 no physical direction class is left under `web/src` except 62 kept on purpose, and a test (`styles/physical-direction-classes`) fails on any new one that its allowlist does not list with a reason |

### Rules and tooling

- `docs/i18n/fa.md` (new file): locale status, date rules, a glossary of 93 terms with reasons, the terms that stay English, eleven typography rules (ZWNJ, Persian ی and ک, digits, ۀ, punctuation, spacing around Latin words, FSI … PDI isolates around interpolated left-to-right values, RLI … PDI around Persian placeholders of left-to-right inputs), style and the automated check.
- `bun run i18n:check-fa` (`web/scripts/check-fa.mjs`): exits 1 on any finding. It checks the typography rules, isolate pairing, empty values, stray whitespace, keys missing from `en.json`, and parses `web/src` with `@babel/parser` (now a declared dev dependency, 7.29.7, single copy in `bun.lock`) to check the placeholders of `dir='ltr'` inputs. Today: `check-fa: 5817 keys, no findings`.
- `bun run i18n:sync` reports Persian as partial and never fills it with English. The i18n skill, `AGENTS.md` and `web/AGENTS.md` describe `fa` as the optional partial eighth locale.

### What differs from upstream for other languages

1. **Number formatting (split-plan PR 5).** Money, quotas and the numbers fixed so far follow the interface language instead of the browser language or a raw value. For English the output is identical (tested for each change). French, Russian and Vietnamese now show a decimal comma where the interface formats decimals (for example the response-time badge `1,23 s`, chart totals `$22,98`, the subscriptions list price and, since session 11, the plan price on the purchase dialog and wallet plan cards); Chinese and Japanese keep the decimal point. Grouping follows the interface language where it already applied (flow numbers, system info bytes). The compact table page counter and the percent sign (session 11) change only Persian: the counter has no grouping and the sign is `%` in every other language.
2. **The 9 legal-consent keys** (`fix/legal-consent-sentence`), present in all seven required locales. The other upstream fix branches are now upstream's own code (#7628, section 2).
3. **Public Sans** (now upstream's own fix, #7628; our branch `fix/public-sans-font` is superseded). `--font-sans` now names the face that `@fontsource-variable/public-sans` registers (`'Public Sans Variable', 'Public Sans', sans-serif`), so left-to-right languages render in Public Sans instead of the browser's generic sans-serif (Chromium: sample line 820.06 px → 854.92 px; one 26 832-byte Latin woff2 now downloaded). Persian is unchanged: `html:lang(fa)` still sets Vazirmatn, and no Public Sans face loads on a Persian page (checked on the running app).

Nothing else changes for left-to-right languages: every RTL fix uses logical classes or RTL-only rules, and every Persian-only label (audit roles, sign-in methods, the audit ID label and user action names) is guarded by the locale; tests pin the English (and where relevant Chinese) output. Among the locale files only `fa.json` differs from upstream apart from those 9 keys.

## 2. This session (session 13): upstream merge and the three upstream branches

Session 12's details (dashboard setup guide, the sweep of physical classes, the guard) are in `git show 16bfa21:.fa-review/REPORT.md`, section 2.

### A. Merge of `upstream/main` (`1a4166d`, 19 commits since `789c970`) into `feat/fa-locale`: `d95fb41`

| Conflict | Resolution |
| --- | --- |
| `dashboard/lib/charts.ts` | upstream's ordering code (#7628, chart order across a year boundary); our `locale` arguments kept on it. Our `chart-time-order` test dropped (upstream's `charts.test.ts` covers it) |
| `pricing/model-details.tsx`, `usage-logs/.../common-logs-columns.tsx` | upstream's code (#7628 fixed the same labels; our comment dropped) |
| `wallet/recharge-form-card.tsx` | upstream's `Pay … <savings>• Save …</savings>` sentence, with our `locale` on the amounts |
| `security/access-token-card.tsx` (modify/delete) | upstream's deletion accepted; our work moved to the new access-token components (below) |
| `usage-logs/audit/audit-log-details-dialog.tsx`, `audit/lib/audit-details.ts` | both sides: upstream's options object (`scopeResources`) gains `locale`; callers and tests pass `{ locale }` |
| `users/data-table-row-actions.tsx` | both sides (upstream's step-up verification, our Persian action name) |
| `i18n/static-keys.ts` | upstream's access-token keys + our legal-consent keys; our redemption key dropped (upstream's) |
| seven locale files | upstream's files, then the 9 legal-consent keys re-added through the i18n skill's script, then `bun run i18n:sync` (no hand edits) |

Where #7628 fixed the same bug as our merged fix branches (2FA step label, delete-account label, chart order, the six hard-coded labels, Public Sans, billing status), upstream's code, keys and tests are kept, and our tests for them are dropped: `billing-status-label`, `chart-time-order`, `font-sans`, `price-unit-note`, `delete-invalid-sentence`, `plan-status-label`, `tokens-header`, `inviter-label`, `preset-labels`.

**New access-token components** (upstream `caca52f`, `4924361`): `access-tokens-card.tsx` (`pe-20`, `end-3` in the access records sheet), `access-token-edit-dialog.tsx` (`sm:me-auto`); in Persian the token list shows numeric Solar Hijri dates through `formatTimestampToDate` (other languages keep upstream's `Intl` medium date) with the Gregorian date as `title`. The audit "Expiration" field uses the shared date helper instead of `dayjs().format`. Upstream's `permission-matrix.tsx`: logical classes and a collapsed-group chevron that points to the inline end in RTL (`rtl:rotate-90`, open state `rotate-0`; checked in the built CSS order). Tests: `access-token-dates` (2), `permission-matrix-direction` (2), `expiry-date-locale` (2).

**Keys**: upstream added 102 keys to `en.json` and removed 16. All 95 new keys without a Persian value are translated (none skipped), `Save {{amount}}` moved to the new savings sentence, 13 Persian keys of removed keys dropped by the sync; then `Logs` and `Failed to save` (older keys that upstream's permission catalog now uses) were translated after the screenshots showed «Logs» in English. Guard: the 4 physical classes upstream added (permission matrix) are converted; the allowlist is unchanged.

### B. Preparing the split (commits on `feat/fa-locale`)

- `89af00d`: `languages.ts` reordered so each upstream branch adds its own block (direction helper written to type-check with or without an RTL language; arrow helpers after `convertDetectedLanguage`; `isPersianIntlLocale` at the end); the Solar Hijri half of the date-range test in its own file; tests that skip Persian compare the code as a string.
- `16360b5`: the sidebar docking test sets the direction through the provider (the Persian switch case in its own file); the quota-audit arrow test accepts either digit set; the Persian quota amounts get their own test.
- `020b9fc`: the code-isolation CSS rule at the end of `index.css`.
- `a589e09`: `Logs`, `Failed to save` in Persian.

### C. How the branches were built

Each is a new branch from `upstream/main` (`1a4166d`), with no merge commits, no `.fa-review/` and nothing upstream already fixed. The diff of `feat/fa-locale` against upstream was split by hunk: every changed line was attributed with `git blame` to the commit that wrote it and mapped to the split plan (section 5); the commits that mixed concerns (`c78c83f`, `f01c5af`, `aa699fc`, `b2720c9`) were split line by line (number formatting → dates/numbers, classes → RTL, the sidebar side → RTL, the rest of the foundation → locale); a dozen hunks were split by hand. Decisions:

1. The JS-level direction features (`formatValueChange`/`formatValueChain`: before/after arrows, the retry chain; `5e64f68`, `0c605b3`) go with the locale branch: they need a registered right-to-left language, which the RTL branch does not add. The RTL branch is CSS and markup only and is inert in left-to-right languages.
2. Shared code needed by two branches is added identically in both, at the same place, so git merges it cleanly: `TextDirection`/`getInterfaceLanguageDirection` and the sidebar docking (RTL and locale), `isPersianIntlLocale` and the audit `locale` option (locale and dates/numbers).
3. `docs/i18n/fa.md` stays whole in the locale branch (its date rules take effect with the dates branch; said in that PR text).
4. `quota-audit-operation.ts`: the arrow to the locale branch, the Persian amounts to the dates/numbers branch; `calendar.tsx`: the class swaps to RTL, the Solar Hijri rewrite (with upstream's physical classes) to dates/numbers.
5. Dependencies: each branch's `bun.lock` is seeded from `feat/fa-locale`'s, so `@daypicker/persian` stays at 10.0.1 with a single `react-day-picker` (a fresh resolve picked 10.0.2 and a second copy, as `docs/i18n/fa.md` warns).

**Left out on purpose**: `.fa-review/`; `fix/legal-consent-sentence` (overlaps open PR #5998): `legal-consent.tsx`/`terms-footer.tsx` sentences, their 4 tests, 9 keys in each locale, static keys (the RTL class on the consent label and its test are in the RTL branch); the move of the duplicated upstream license comment in `usage-logs/components/dialogs/details-dialog.tsx` (formatter noise).

**Merge order** (checked in a scratch branch): `pr/fa-rtl-layout` then `pr/fa-locale` merge cleanly; `pr/fa-dates-numbers` then conflicts in 9 files (`pagination.test.tsx`, `calendar.tsx`, `api-key-listing.test.tsx`, `system-tasks-table.tsx`, `marketplace-plugin-card.tsx`, `audit-details.ts`, `common-logs-columns.tsx`, `details-dialog.tsx`, `quota-audit-operation.ts`), all neighbouring-line edits (a class next to a locale argument, two names in one import). With feat's version of those 9 files the result equals `feat/fa-locale` except the left-out items. So the dates/numbers PR needs one rebase after the others land; `feat/fa-locale` is the reference resolution.

## 3. Hard-coded or unkeyed English (not fixed here; each needs a new key in every locale, or a code change)

Unchanged in session 12 (layout only). Rows updated in session 11: the page counter, the retry chain, the two plan prices and `mr-1` are removed; the rows found in session 11 are added at the end.

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
| features/usage-logs/components/dialogs/image-dialog.tsx:68; details-dialog.tsx:1328; model-badge.tsx:78 | text joined around `t()` (`Task ID:` + id, `Param Override (n)`, `Model: name`) |
| features/usage-logs/constants.ts (Midjourney submit result) | `Duplicate` shares the key of the verb (fa «تکثیر»), wrong sense here |
| features/dashboard/components/flow/flow-node-filter.tsx:131,207; users/user-charts.tsx | `label: value` joins |
| features/playground/lib/input/input-tool-utils.ts:51 | toast description is the raw action id (`upload-file` …) |
| features/playground/lib/message/message-streaming-utils.ts:245; hooks/use-chat-handler.ts:178-182; lib/streaming/stream-utils.ts:108 | error texts joined with a Latin colon / `HTTP n:` prefix |
| features/playground/components/input/playground-parameter-panel.tsx:130 | numeric values passed through `t()` |
| features/system-info/components/system-instances-panel.tsx:94-95 | role badge `master`/`worker` |
| features/setup/setup-wizard.tsx:307; components/complete-step.tsx:50; database-step.tsx:123 | `Initialize` + system name; `Unknown`; `Data directory:` + path |
| features/system-settings/general/channel-affinity/cache-stats-dialog.tsx:137-145 | `Prompt tokens`, `Completion tokens` missing from en.json; `Cached tokens`, `Total tokens` not through `t()` |
| features/channels/components/dialogs/codex-usage-dialog.tsx:207-217,236 | `h`/`m`/`s` glued to numbers |
| features/security/components/dialogs/delete-account-dialog.tsx:101 | `Type` <name> `to confirm` (upstream fix branch `fix/delete-account-confirm-label`) |
| features/about/index.tsx:68-109 | footer and compliance sentence built from fragments |
| features/models/components/deployments-columns.tsx:174,183 | `%` label; `Approx.` + value |
| features/task-plugins/components/upload-dialog.tsx:170,215,247; plugin-card.tsx:103; marketplace-capabilities.tsx:74,84,104,114; index.tsx:187; plugin-metadata-card.tsx:43 | template descriptions, `Versions tab` (the tab is «Version history»), `label: value` joins, Latin `, ` lists |
| features/task-plugins/components/marketplace-install-dialog.tsx:110,155; marketplace-panel.tsx:79,157; plugin-sandbox.tsx:43 | English `Error` messages shown in the dialog; `t(source.name)` on an admin-entered name |
| features/task-plugins (plugin cards) | plugin names and descriptions come from the plugin metadata (backend), English |
| components/ui/carousel.tsx:229,263 | screen-reader labels `Previous slide` / `Next slide` are literals (no key) |
| components/data-table/core/pagination.tsx (full pager) | page number buttons written raw, and `Total:` uses `toLocaleString()` (browser language, not the interface language); only the compact counter was in decision 3 |
| channels/…/codex-usage-dialog.tsx:488; wallet/…/subscription-plans-card.tsx:519; system-settings/…/log-settings-section.tsx:417; …/cache-stats-dialog.tsx:33; dashboard/…/uptime-panel.tsx:166; rankings (market share, growth); pricing (uptime, success rate) | percentages from a raw number or `toFixed` + `%`: Latin digits and `%` in Persian (C1 covered only numbers already formatted in the interface language) |
| wallet/lib/format.ts:79 | `{{percent}}% OFF`: a raw integer interpolated, so Persian shows Latin digits; the fa value keeps `%` to match them |
| usage-logs/…/details-dialog.tsx:631 | request conversion chain joined with ` -> ` (ASCII arrow, not direction-aware) |
| usage-logs/…/details-dialog.tsx (details) | response time `3.0s (FRT: 0.8s)` and group ratio `1.0000x` written raw |

## 4. Keys still English

**901 keys that no code in `web/src` uses.** Not translated (session 10, decision 4): the scanner (`.fa-review/scripts/scan-keys.mjs`, `@babel/parser`: string literals, template literals without substitutions, JSX text) does not find them, and of the 902 found at the start of this session only one reaches `t()` from the server (session 10 report, section 2). Many belong to older pages or the backend. Keys built at run time (such as `preset.<name>`) were checked by reading the code and are translated.

**161 keys the code uses**, listed once under the first folder the scanner meets them in (unchanged this session):

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


## 5. Upstream branches (built in session 13)

The proposal from session 12 (six PRs) is built as three branches, as asked: plan PR 1 → `pr/fa-rtl-layout`; plan PRs 2, 6 and 4 → `pr/fa-locale`; plan PRs 3 and 5 → `pr/fa-dates-numbers`. The session 12 table is in `git show 16bfa21:.fa-review/REPORT.md`, section 5.

| Branch | Head | Commits (files) | Files | Draft text |
| --- | --- | --- | ---: | --- |
| `pr/fa-rtl-layout` | `7204ba0` | `e75239e` fix(web): logical direction classes in the shared components (95) · `4b7a931` fix(web): logical direction classes on the feature pages (244) · `7204ba0` test(web): guard against new physical direction classes (2) | 341 | `pr-rtl-layout.md` |
| `pr/fa-locale` | `784f21d` | `2a0384d` feat(i18n): add Persian (fa) as a partial right-to-left locale (14) · `ffdb275` feat(web): reading-direction arrows and Persian audit labels (23) · `784f21d` feat(i18n): Persian typography check, translation guide and rules (10) | 44 | `pr-locale.md` |
| `pr/fa-dates-numbers` | `7963a31` | `a6f5ca5` feat(web): Solar Hijri dates and date pickers in Persian (92) · `7963a31` fix(web): format money and numbers in the interface language (75) | 119 | `pr-dates-numbers.md` |

Order: 1, 2, 3 (3 needs one rebase, section 2C). Each merges alone on upstream main. Other languages: RTL none; locale «فارسی» in the language list; dates/numbers the decimal comma in French, Russian and Vietnamese. The older `fix/*` branches stay as they are; the ones #7628 fixed are superseded and not offered.

## 6. Verification (session 13)

Session 12's verification is in `git show 16bfa21:.fa-review/REPORT.md`, section 6. Bun install, then from `web/`; lint compared by file and rule with `upstream/main` (165 errors, 65 warnings, in its own worktree).

| Check | `feat/fa-locale` (`a589e09`) | `pr/fa-rtl-layout` | `pr/fa-locale` | `pr/fa-dates-numbers` |
| --- | --- | --- | --- | --- |
| `bun run typecheck` | exit 0 | exit 0 at each commit | exit 0 at each commit | exit 0 at each commit |
| `bun run lint` errors (upstream 165) | 119; 0 pairs above upstream; 0 in changed files | 121; 0; 0 | 163; 0; 0 | 165; 0; 1 in a changed file, upstream's own, unchanged |
| `bun run test` | 249 files, 2547 passed | 202 files, 2236 passed (commit 1: 2196, commit 2: 2233) | 188 files, 2284 passed (commit 1: 2197, commit 2: 2252) | 202 files, 2325 passed (commit 1: 2250) |
| `bun run build` | exit 0, 67002.5 kB / 20638.3 kB gzip | exit 0, 66259.9 kB | exit 0, 66890.1 kB | exit 0, 66330.0 kB |
| `bun run i18n:sync` | no changes; fa missing 1056, extras 0; others 0/0 | no changes | no changes | no changes |
| `bun run i18n:check-fa` | `5817 keys, no findings` | (no fa.json) | `5808 keys, no findings` | (no fa.json) |
| fail-before: branch tests with every other changed file reverted to upstream | | 34 files: 66 failed, 96 passed | 15 files: 30 failed, 33 passed; 14 of 15 files fail | 32 files: 89 failed, 117 passed |

Fail-before notes: the passing tests assert that left-to-right / English output stays as upstream renders it; the only locale-branch file that passes is `scripts/oxlint/__tests__/intl-locale.test.ts`, which adds `fa` cases to the existing lint rule; `check-fa.test.ts` is not collected before (its `vitest.config.ts` include belongs to the branch, and `check-fa.mjs` does not exist there). Intermediate commits were tested before two last-minute changes (the CSS rule position in RTL commit 1, two Persian keys in locale commit 1); the final heads were tested after them.

### Running app and screenshots

`go build` of `feat/fa-locale` (fresh `web/dist`), `pr/fa-rtl-layout` and `upstream/main` (each with its own `bun run build`), Go 1.25.1. `scripts/seed-s13.py` ran once against the feat binary on a fresh SQLite database in a scratch directory: `GET /api/setup` → `status false, database_type sqlite`; `POST /api/setup` → `系统初始化成功`, success; then three channels, an API key, a plan, a redemption code, a demo Epay configuration (top-up presets, 20% off the 100 preset), a second user, two scoped access tokens created through password verification (`POST /api/verify` → proof → `POST /api/user/access_tokens`), and three usage logs on 2026-09-25 UTC. `scripts/shots-s13.mjs` (Chromium 141, 1440×900, light, clock fixed at 2026-09-25 14:00 UTC, reduced motion) signs in and captures; no capture is blank (fewest colours 1834; blank threshold 16). Console errors: the 401 of the pre-login session probe and `ERR_CERT_AUTHORITY_INVALID` for an external resource.

| Files | Shows |
| --- | --- |
| `501-security-access-tokens-{rtl,en}.png` | new access tokens: «ساخته‌شده در ۱۴۰۵/۰۷/۰۲ ۱۲:۰۰:۰۰», «انقضا: ۱۴۰۵/۰۸/۰۳ ۱۲:۰۰:۰۰», «آخرین استفاده: ۱۴۰۵/۰۷/۰۳ ۱۳:۰۰:۰۰ · 203.0.113.7», «بدون انقضا»; English «Created Sep 24, 2026, …» |
| `502-access-token-edit-{rtl,en}.png` | edit dialog with the permission matrix (chevrons, «گزارش‌ها») |
| `503-usage-logs`, `504-audit-log`, `505-audit-access-token-details` | usage and audit logs; access token audit entry («توکن دسترسی ساخته شد», «مجوزها», «انقضا») |
| `506-users`, `507-wallet-topup`, `508-model-details`, `509-dashboard-charts` | users; top-up presets «پرداخت ۵۸۴ • صرفه‌جویی ۱۴۶» / «Pay 584 • Save 146»; model details; dashboard charts |
| `600`–`615-*-en-rtl-branch.png` | English main pages on `pr/fa-rtl-layout` |
| `616-pricing-vendor-order-…png`, `617-audit-time-…png` | the two differences explained below (upstream left, RTL branch right) |

**English pixel comparison, `pr/fa-rtl-layout` against `upstream/main`** (`scripts/compare-s12.mjs`; every run on a fresh copy of the seeded database, same port 3410, since each sign-in adds sessions and audit rows). Differing pixels against the first upstream run:

| Page | upstream run 2 | upstream run 3 | RTL run 1 | RTL run 2 | RTL run 3 |
| --- | ---: | ---: | ---: | ---: | ---: |
| 600 dashboard, 602 usage logs, 605 model details, 607 models, 608 keys, 609 profile, 611 wallet, 613 users, 614 redemption codes | 0 | 0 | 0 | 0 | 0 |
| 601 dashboard models | 72 | 0 | 0 | 0 | 0 |
| 603 audit log (sign-in time of the newest row, `617`) | 155 | 150 | 232 | 218 | 150 |
| 604 pricing (vendor filter order varies between runs, `616`) | 0 | 2143 | 2143 | 0 | 0 |
| 606 channels (anti-aliasing of the "Max Retries" badge) | 53 | 0 | 0 | 53 | 53 |
| 610 security (session times) | 113 | 195 | 57 | 197 | 60 |
| 612 subscriptions | 0 | 30 | 0 | 0 | 0 |
| 615 system settings | 0 | 0 | 0 | 0 | 5 |

Every difference of the RTL branch also occurs between two upstream runs, in the same region. **English renders the same.**

## 7. Remaining issues

1. **Code bugs found while translating (English affected too)**, kept listed: the session 7 and 8 items (channels glued labels, `&apos;`, `&mdash;`, `&#10;`, plurals, the Gemini sentence, the compliance separators, OAuth callback base; see `git show 57c75bd:.fa-review/REPORT.md`) and the joins in the hard-coded table.
2. **Dates not in the fork's pattern** (for upstream, decision of session 9): the compliance "Confirmed at" time (`integrations/payment-settings-section.tsx:840`, `toLocaleString()`) and the announcements table (`content/announcements-section.tsx:394`, `dayjs().format(…)`, plus its own relative time).
3. **Left as they are on purpose**: the chart time axes run left to right in RTL (VChart axes; reversing them would also reverse the reading of trends), the rankings trend arrows (they show up and down, not a direction of reading), native date inputs (browser), backend content in Chinese or English (Go i18n has en and zh only), the hero terminal demo (a terminal stays left to right).
4. **Still listed** (hard-coded table): the carousel's screen-reader labels (no key); and, found in session 11, the full pager's page buttons and `Total:`, the raw percentages (`toFixed` + `%`), the discount label's raw number, the request conversion chain ` -> `, and the raw response time and group ratio in the usage log details. Each needs a new key, a shared change or a decision beyond the four items fixed this session.
5. `lib/theme-customization.ts` says in a comment that the `default` preset resolves to serif, while `PRESET_DEFAULT_FONT.default` is `sans` (upstream comment; not touched, the Public Sans branch stays minimal).
6. **Session 12, left for later.** None of items 1 to 5 was about direction classes, so none is removed. New: (a) `components/ai-elements/web-preview.tsx` keeps one `text-left` until its iframe sandbox (`allow-scripts` with `allow-same-origin`) and index key are decided upstream; the component is unused. (b) Outside the guard: `slide-in-from-left/right` animation classes (44 in 11 files; popovers keyed to the physical side they open on), `translate-x-*` (51 in 17 files; mostly centring and motion), `bg-gradient-to-r/l` (7 in 6 files; decorative). (c) tailwind-merge treats `ps/pe`, `ms/me`, `start/end` and `border-s/e` as separate from `px`, `mx`, `inset-x` and `border-x`, unlike their physical forms; callers were checked (section 2), but a future caller that passes `px-*` to a component whose base has an unscoped `ps-*`/`pe-*` would not override it. Extending the tailwind-merge config is possible, but it would also change merges that upstream code already relies on, so it was not done.

7. **Session 13.** (a) `pr/fa-dates-numbers` needs one rebase once the first two branches land (9 files, section 2C). (b) The audit "Expiration" field and the access token list now use Solar Hijri in Persian; the access token sessions list («آخرین فعالیت: ۷ روز دیگر» in the screenshots) only reflects the fixed browser clock. (c) Not offered upstream: `fix/legal-consent-sentence` (PR #5998 overlap) and `fix/quota-insufficient-i18n` (backend, separate). (d) PR template fields left for the submitter: tool version and model id.

## 8. Open questions

None. Session 10's four questions were settled (session 11 decisions 1 to 4, section 2).

## 9. Session history (short)

Taken from the titles of the earlier versions of this file (`git log -- .fa-review/REPORT.md`).

- Session 1: foundation (partial `fa` locale, right-to-left direction, tooling and docs) and the first batch.
- Session 2: RTL fixes and the second batch.
- Session 3: RTL fixes, numbers in the interface locale, the third batch.
- Session 4: audit batch, Persian dates, the FSI/PDI rule, an upstream label fix.
- Session 5: chart order, Solar Hijri charts and date pickers, audit values, batch 5.
- Session 6: Persian-only audit labels, RTL polish, billing status fix, batch 6, the upstream split plan.
- Session 7: legal consent and UI label fixes in upstream style, merges of the fix branches, batch 7 (channels and models).
- Session 8: glossary, batch 8 (system settings), batch 9 (shared components), RTL fixes.
- Session 9 (its report called itself session 10): placeholder isolates and their check, response-time milliseconds, batch 10 (every remaining page), RTL fixes.
- Session 10: the four settled decisions, the final cleanup pass (digits, dates, arrows, lint-blocked RTL fixes, audit labels, `pre` blocks), this summary.
- Session 11: download wording, audit labels to PR 4, the page counter, the upstream `fix/public-sans-font` branch (merged), percent sign, retry chain, plan prices, `mr-1`.
- Session 12: the dashboard setup guide in RTL, the sweep of every physical direction class (414 converted, 62 kept with reasons), the guard test, the lint cleanup of the touched files.
- Session 13 (this one): merge of upstream `1a4166d` (#7628 replaces six of our fixes; access tokens rebuilt upstream), 97 new keys in Persian, and the three upstream branches `pr/fa-rtl-layout`, `pr/fa-locale`, `pr/fa-dates-numbers` with coverage, fail-before and the English pixel comparison.
