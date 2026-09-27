# Persian (fa) locale for new-api: state and hand-off

Temporary review folder. Delete `.fa-review/` before any upstream PR.

Fork `yaser-k/new-api-fork`, branch `feat/fa-locale`, based on `upstream/main` at `c2b7a9a` (upstream had not moved; no merge this session). Upstream issue: QuantumNous/new-api#7198. This file is written so it can serve as the summary for the maintainers; the session details, verification and open items follow the summary.

## 1. What the Persian locale covers

| | |
| --- | --- |
| Keys | **5733** of the 6795 keys in `en.json` are translated in `fa.json`. `fa` is a partial locale: a key that is missing from `fa.json` shows its English text (per key, at run time). |
| Keys still English | 1062: **161** that the code uses (brand and provider names, URLs, example values, identifiers, unit letters glued to numbers, fragments that cannot be ordered in Persian; full list with reasons in section 4) and **901** that no code in `web/src` uses (section 4) |
| Pages | Every page and shared component under `web/src`: auth and sign-up, home, pricing and model details, rankings, dashboard (overview, models, flow, users), API keys, playground and chat, usage, task and drawing logs, audit log, wallet and subscriptions, profile and security, channels, models and deployments, users, redemption codes, subscriptions admin, system info, task plugins, the setup wizard, every system settings section, the legal pages, error pages and the about page |
| Dates | Displayed dates are Solar Hijri in numeric year/month/day order with 24-hour time (`۱۴۰۵/۰۷/۰۳ ۱۵:۰۵:۰۹`); relative times are Persian («۳ دقیقه پیش»). Table and log cells carry the Gregorian date and time as their `title`. Values sent to the API, filters, inputs, exports and copied text stay Gregorian. Dashboard chart axes are Solar Hijri and stay in time order across Nowruz |
| Date pickers | `calendar.tsx` (used by the date and date-time pickers) shows the Solar Hijri calendar through `@daypicker/persian`, loaded lazily only for Persian; the returned `Date` values are unchanged |
| Numbers | Formatted with the interface locale: Persian digits (`۱۲۳`, `۱٫۲۳`), switchable to Latin digits with one constant (`PERSIAN_INTL_LOCALE = 'fa-u-nu-latn'`) |
| Right to left | Selecting Persian sets `dir="rtl"` and `lang="fa"`; a direction provider feeds Base UI and the calendar. Components use logical classes (`ms-*`, `pe-*`, `start-*`, `text-end`), direction-dependent icons are mirrored, code, keys, sample output and `pre` blocks keep their own text direction (`unicode-bidi: plaintext`), left-to-right inputs (URLs, IDs, code) keep `dir="ltr"`, before-and-after arrows and the carousel follow the page direction, and Persian text uses Vazirmatn (also inside monospace text). The config drawer can still force LTR or RTL |

### Rules and tooling

- `docs/i18n/fa.md` (new file): locale status, date rules, a glossary of 93 terms with reasons, the terms that stay English, eleven typography rules (ZWNJ, Persian ی and ک, digits, ۀ, punctuation, spacing around Latin words, FSI … PDI isolates around interpolated left-to-right values, RLI … PDI around Persian placeholders of left-to-right inputs), style and the automated check.
- `bun run i18n:check-fa` (`web/scripts/check-fa.mjs`): exits 1 on any finding. It checks the typography rules, isolate pairing, empty values, stray whitespace, keys missing from `en.json`, and parses `web/src` with `@babel/parser` (now a declared dev dependency, 7.29.7, single copy in `bun.lock`) to check the placeholders of `dir='ltr'` inputs. Today: `check-fa: 5733 keys, no findings`.
- `bun run i18n:sync` reports Persian as partial and never fills it with English. The i18n skill, `AGENTS.md` and `web/AGENTS.md` describe `fa` as the optional partial eighth locale.

### What differs from upstream for other languages

1. **Number formatting (split-plan PR 5).** Money, quotas and the numbers fixed in this session follow the interface language instead of the browser language or a raw value. For English the output is identical (tested for each change). French, Russian and Vietnamese now show a decimal comma where the interface formats decimals (for example the response-time badge `1,23 s`, chart totals `$22,98`, the subscriptions list price); Chinese and Japanese keep the decimal point. Grouping follows the interface language where it already applied (flow numbers, system info bytes).
2. **The 17 keys from the upstream fix branches** merged into `feat/fa-locale` (9 from `fix/legal-consent-sentence`, 8 from `fix/ui-label-strings`), present in all seven required locales.

Nothing else changes for left-to-right languages: every RTL fix uses logical classes or RTL-only rules, and every Persian-only label (audit roles, sign-in methods, the audit ID label and user action names) is guarded by the locale; tests pin the English (and where relevant Chinese) output. Among the locale files only `fa.json` differs from upstream apart from those 17 keys (checked: each of en, fr, ja, ru, vi, zh, zh-TW has 17 added, 0 removed, 0 changed keys).

## 2. This session

### Decisions applied

| # | Decision | Commit | Result |
| --- | --- | --- | --- |
| 1 | Declare `@babel/parser` as a dev dependency | `0330995` | `^7.29.7` in `web/package.json`; `bun.lock` gains only the workspace entry and still has one `@babel/parser@7.29.7`. Evaluated (web/AGENTS.md 3.15): MIT, Babel team, one dependency (`@babel/types`), 2.0 MB, not in the app bundle, already required by `@babel/core`, `@tanstack/router-plugin` and `shadcn`. Noted next to the check-fa section of `docs/i18n/fa.md` |
| 2 | Seconds in the response-time badge follow the interface language | `aae92d9` | New `formatFixed(value, digits, locale)` in `@/lib/format`: rounds exactly like `toFixed` (so `1005 ms` is still `1.00s`), then only localizes digits and the decimal separator, no grouping. Used by the badge (channels table, channel test dialog) and the playground message timing. English `1.23s`, Persian `۱٫۲۳`, French/Russian/Vietnamese `1,23`. Tests: `response-time-format` 4 failed / 17 passed → 21 passed; `message-duration-locale` 2 failed → 3 passed; `format-fixed-locale` (all languages + invalid) |
| 3 | «بارگذاری» for load/upload, «ناهمگام» for async | `b1c4bad` | 18 «بارگیری» values (all load or reload; none meant download) and 2 «ناهم‌گام» values changed through the script. «ناهمگام» because rule 1 covers only می/نمی, ها and تر/ترین, the prefix نا and the compound همگام are written joined, and 6 of 8 values already used it. Glossary rows: load/loading/reload/upload «بارگذاری», download «دانلود» (what the existing values use), async «ناهمگام» |
| 4 | Of the 902 unused keys, translate only server-sent ones reaching `t()` | `4972f14` | One key (evidence below) |

**Decision 4 evidence.** Go string literals were collected with `go/ast` from every non-test `.go` file outside `web/` (35 516 literals; controller, model, service, relay, middleware, setting, constant, common, pkg …), plus the values of the backend message catalogue `i18n/locales/en.yaml` (embedded with `go:embed`). Of the 902 keys, 8 are Go literals and 0 are catalogue values. The 403 `t()` calls with a non-literal argument were listed with `@babel/parser`; the relevant ones are `t(option.reason)` and `lib/server-error-message.ts:166` (server `message` text passed to `t()`).

| Key | Go source | Reaches `t()`? |
| --- | --- | --- |
| `Passkey authentication is disabled.` | `service/security_verification.go:216` (`option.Reason`, returned by `GET /api/verify/methods`) | **Yes**: `features/auth/secure-verification/components/secure-verification-dialog.tsx:166` `t(option.reason)`. Translated «ورود با کلید عبور غیرفعال است.», like its already translated sibling for passwords |
| `OpenAIMax`, `AILS`, `PaLM`, `API2GPT`, `AIGC2D` | `constant/channel.go:151-158` (channel type names) | No: used lowercased as owner names (`controller/model.go:126-138`) and internally (`controller/channel-test.go:88`); brand names anyway |
| `OpenAI Compatible` | `service/log_info_generate.go:248` (`other.request_conversion`) | No: `usage-logs/components/dialogs/details-dialog.tsx:621-628` joins the chain with ` -> ` |
| `edit_this` | `setting/ratio_setting/group_ratio.go:22` | No: identifier in a default JSON value |

The other 901 keys stay untranslated (decision 4).

### Final cleanup pass

| Item | Commits | What changed (Persian unless stated; LTR output unchanged) | Tests (before → after) |
| --- | --- | --- | --- |
| B1 digits | `0153b2d`, `aa699fc`*, `d623586`, `172c97e`, `eb96d86`, `089255f`, `5ab137d` | System info CPU/memory/disk percentages, disk sizes, refresh intervals, task progress and active task count; dashboard flow tooltips, filter values and Top N tabs; user chart Top N tabs; quota distribution header total, chart tooltips and user ranking values (`aa699fc` also isolates the header total in a left-to-right `bdi`, the RTL half*); setup step badges; task plugin model counts, priority, hidden model badge, collapsed count, channel and in-flight counts; subscriptions list price, validity, reset period, priority and plan quota (`formatDuration`/`formatResetPeriod` take an optional locale, also passed by the purchase dialog and the wallet plan cards). Integers use `formatNumber`; decimals written with `toFixed` use `formatFixed`, so English rounding is identical | `system-info/number-locale` 2 failed → pass (+ progress 1 failed, active count 1 failed → pass); `dashboard/flow-number-locale` 3 failed / 2 → 5 passed; `dashboard/chart-number-locale` 4 failed / 1 → 5; `models/total-direction` 1 failed → pass; `setup/step-number-locale` 1 failed / 1 → 2; `task-plugins/count-locale` 3 failed / 1 → 4; `subscriptions` lib + list 2 failed / 2 → 4 |
| B2 dates | `d14b92a` | System info "Last seen" and "Updated" titles pass the locale (Solar Hijri, like the cells). The payment compliance time and the announcements table stay as they are (listed for upstream) | `system-info/title-dates` 2 failed / 2 → 4 |
| B3 arrows | `5e64f68`, `272cbf9` | `formatValueChange(from, to, language)` in `@/i18n/languages`: LTR `from → to` unchanged; RTL `⁨from⁩ ← ⁨to⁩`, the convention the Persian translations already use. Used by the quota audit summary, the user quota override preview, the task plugin upgrade badge and the usage schema enum labels. Carousel: provider direction passed to Embla, logical gutters and button positions, `rtl:-scale-x-100` on the arrows, arrow keys mapped by direction. The chart time axes and the rankings trend icons are left as they are (see remaining issues) | `i18n/value-change` (all languages); `quota-audit-direction` 1 failed / 1 → 2; `quota-preview-direction` 1 failed / 1 → 2; `change-arrow-direction` 2 failed / 2 → 4; `ui/carousel-direction` 4 failed / 1 → 5 |
| B4 lint-blocked RTL fixes | `5a2bd38` (lint only), `44e9c0b` | First the upstream lint errors in `pricing/components/model-details-api.tsx` (`prefer-string-replace-all` ×5, `curly`) and `dashboard/components/models/models-chart-preferences.tsx` (`no-useless-spread` ×4), no behaviour change. Then RPM/TPM/RPD headers `text-right` → `text-end` (their cells were already `text-end`), code language tabs `ml-auto` → `ms-auto`, preference icons `mr-2` → `me-2` | `pricing/api-tab-direction` + `models/preferences-direction`: 3 failed → 3 passed |
| B5 session 6/8 items | `ce52823`, `0184320`, `768f817` | `pre` added to the `unicode-bidi: plaintext` rule (JSON, logs and command output keep their own direction, line by line). Audit details actor/target and the management log operator: Persian ID label «شناسه» with the name isolated; `{{action}}` in `Performed {{action}} on user …` and `Failed to {{action}} user` and the audit action field: the translated button label (`userActionName`), Persian only, like the audit roles; other languages render exactly as upstream (English and Chinese pinned). `user-actions.ts` also got a top-level type import (upstream lint error in a touched file) | `styles/code-direction` +3 (1 failed → pass); `audit/identity-action-locale` 4 failed / 2 → 6 (English and Chinese cases pass before and after); `users/user-action-name` 4 failed → pass; `usage-logs/manage-operator-locale` 1 failed / 1 → 2 |

Not fixed in B5, with the reason: native date inputs (drawn by the browser), backend content in Chinese or English (Go i18n has en and zh only), `Toggle Sidebar` and other strings that need a new key in every locale (hard-coded list), and **the Public Sans font name mismatch** (see open question 1): the fork's Persian CSS already sets every Persian page, Latin text included, to Vazirmatn, so the mismatch only affects left-to-right languages, and fixing it would change the English font. Confirmed in Chromium on the running app (English): `--font-sans` and `body` ask for `"Public Sans"`, the only registered face is `Public Sans Variable` (status `unloaded`, never used), so English renders in the system sans-serif.

Existing components reused: every fix changes classes, `dir` or formatting on the existing components; no new component. New shared helpers: `formatFixed` (`@/lib/format`), `formatValueChange` (`@/i18n/languages`), `userActionName` (`features/users/lib`), `auditIdentity` (`features/usage-logs/audit/lib`).

### Commits (from `973f3bd`)

```
0330995 build(web): declare @babel/parser as a dev dependency for check-fa
aae92d9 fix(web): format response-time seconds in the interface language
b1c4bad fix(i18n): one Persian term each for load and async
4972f14 feat(i18n): translate the one server-sent key the scanner cannot see
0153b2d fix(web): format system info and dashboard flow numbers in the interface language
aa699fc fix(web): format the dashboard chart totals in the interface language
d623586 fix(web): number the setup steps in the interface language
172c97e fix(web): format task plugin counts in the interface language
eb96d86 fix(web): format the subscriptions list numbers in the interface language
d14b92a fix(web): show the system info date titles in the interface calendar
5e64f68 fix(web): make before-and-after arrows follow the page direction
272cbf9 fix(web): mirror the carousel on right-to-left pages
5a2bd38 style(web): clear the lint errors in the pricing API tab and chart preferences
44e9c0b fix(web): right-to-left layout in the pricing API tab and chart preferences
ce52823 fix(web): isolate preformatted blocks on right-to-left pages
0184320 fix(web): Persian ID label and user action names in audit text
089255f fix(web): format the system task progress in the interface language
768f817 fix(web): Persian action label in the audit action field
5ab137d fix(web): format the active system task count in the interface language
(this hand-off): .fa-review scripts, screenshots, report
```

`089255f`, `768f817` and `5ab137d` came from reviewing the screenshots (a Latin `100%` task progress and `0` task count on system info, the raw `promote` in the audit action field).

## 3. Hard-coded or unkeyed English (not fixed here; each needs a new key in every locale, or a code change)

Rows updated for this session: the fixed digits, dates and arrows are removed; five rows found in this session are added at the end.

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
| components/data-table/core/pagination.tsx:87 | the compact page counter `1 / 1` is written raw (Latin digits); shared by every table |
| features/usage-logs/components/dialogs/details-dialog.tsx:641 | admin channel chain `a → b → c` joined in code (more than two values, so `formatValueChange` does not apply; points the wrong way in RTL) |
| features/subscriptions/components/dialogs/subscription-purchase-dialog.tsx:92; features/wallet/components/subscription-plans-card.tsx:547 | plan price `$` + `toFixed(2)` (the subscriptions list is fixed; these two customer-facing cards are not) |
| features/users/components/users-mutate-drawer.tsx:413 | `mr-1` on the Adjust Quota icon (RTL spacing) |

## 4. Keys still English

**901 keys that no code in `web/src` uses.** Not translated (decision 4): the scanner (`.fa-review/scripts/scan-keys.mjs`, `@babel/parser`: string literals, template literals without substitutions, JSX text) does not find them, and of the 902 found at the start of this session only one reaches `t()` from the server (section 2). Many belong to older pages or the backend. Keys built at run time (such as `preset.<name>`) were checked by reading the code and are translated.

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


## 5. Upstream split plan (proposal only; nothing built)

Commits that mix concerns (marked *) need their hunks split when the PR branch is built. Upstream-ready branches in the fork (each from `upstream/main`), unchanged:

| Branch | Head | State |
| --- | --- | --- |
| `fix/dashboard-chart-time-order` | `81b140e` | chart points ordered by timestamp; merged into `feat/fa-locale` (`bc388d4`) |
| `fix/billing-status-label` | `6bf13ab` | not merged |
| `fix/delete-account-confirm-label` | `747769c` | not merged |
| `fix/2fa-setup-step-label` | `6d614c0` | not merged |
| `fix/quota-insufficient-i18n` | `28b7893` | not merged |
| `fix/legal-consent-sentence` | `6a660f7` | merged (`44610c0`, `6f7168b`); overlaps an older upstream PR by another contributor, so not offered as its own PR for now |
| `fix/ui-label-strings` | `3b86126` | merged (`b2d14dc`) |

| # | PR | Commits | Tests | Other languages |
| --- | --- | --- | --- | --- |
| 1 | `fix(web): right-to-left layout in shared components` | `c78c83f`* (without the `formatQuota` locale hunks), `d8ee064`, `cb25020`, `8ccec81`, `936ab28`, `129d598`, the RTL half of `f01c5af`*, the sidebar side of `b2720c9`*, `666c43c` (where PR 1 touches the file), `0d40127`, `f40c978`, `10cd251` + `9f8f102` (squash), `53458ed`, `7b04478`, `6496416`, `cb80274`, `e6f7d76`, **and from this session `5e64f68`, `272cbf9`, `5a2bd38` (lint cleanup, first), `44e9c0b`, `ce52823`, `0184320`, `768f817`, the `bdi` hunk of `aa699fc`*** | the earlier direction tests, plus **value-change, quota-audit-direction, quota-preview-direction, change-arrow-direction, carousel-direction, api-tab-direction, preferences-direction, code-direction (pre), identity-action-locale, user-action-name, manage-operator-locale** | No visible change in LTR. `0184320` and `768f817` are Persian-only labels in audit text, the same kind as PR 4; they could move to PR 4 if the maintainers prefer |
| 2 | `feat(i18n): Persian (fa) partial locale, tooling and docs` | `b2720c9`* (without the sidebar), `07e070d`, `759a31e`, `f5296da` + `776c03c`, `bfe01bb`, the script part of `fa96f8b`; AGENTS.md, web/AGENTS.md, the i18n skill; `docs/i18n/fa.md` (with the glossary rows from `4d901bd`, `d1e501d`, `305ee51` **and the fa.md part of `b1c4bad`**), `2242169`, **`0330995` (`@babel/parser` dev dependency)** | languages, direction-provider, check-fa, persian-monospace, intl-locale lint case | Language switcher lists «فارسی»; `docs/i18n/fa.md` is a new file under `docs/` (ask the maintainers) |
| 3 | `feat(web): Solar Hijri dates, chart axes and date pickers in Persian` | `0ea975e`, `c2369c8` + `42a98a2`, `170ba72`, `9d22fac`, **`d14b92a`** | display-date-locale, activity-time-cell-dates, login-session-dates, date-picker-display, calendar-persian, chart-time-locale, **system-info title-dates** | None |
| 4 | `feat(web): Persian labels for audit roles and sign-in methods` | net of `f767402` + `9cd40a8` | audit-content-locale, details-locale | None (could fold into 3) |
| 5 | `fix(web): format money and numbers in the interface language` | locale half of `f01c5af`*, `formatQuota` hunks of `c78c83f`*, the `locale` argument in `recharge-form-card.tsx`, `31f68f8`, **`aae92d9`, `0153b2d`, `aa699fc`* (without the `bdi` hunk), `d623586`, `172c97e`, `eb96d86`, `089255f`, `5ab137d`** | format-quota-locale, format-currency-locale, amount-locale, summary-cards-locale, profile-header-locale, response-time-format, **format-fixed-locale, message-duration-locale, system-info number-locale, flow-number-locale, chart-number-locale, total-direction (bdi half goes with PR 1), step-number-locale, count-locale, subscriptions format-locale and list-number-locale** | **Yes**: numbers follow the interface language; French, Russian and Vietnamese show a decimal comma in the formatted decimals; English is unchanged (tested). Its own PR |
| 6 | `feat(i18n): Persian translation batches` | `fa96f8b`, `551a8a3`, `795d296`, `4db67e2`, `3825299`, `aaba1b0`, `8a76a75`, `ada5af2`, `c6aaad5`, `4d901bd` (fa.json part), `17d9c1b`, `58de3e3`, `f7ad4f7`, `4ba8a4c`, `3b076b1`, `25399dc`, `86a8c7c`, `97b1b72`, `bcc8ea7`, `3ca2feb`, `d1e501d` (fa.json part), `0097c77`, `c2798f3`, `4c66300`, `67e72b5`, `39a571f`, `5cd6e90`, `b744e97`, `334c3d9`, `2adca66`, `0d1b17b`, `467f231`, `a14be44`, `bdf0202`, **`b1c4bad` (fa.json part), `4972f14`** | `bun run i18n:check-fa` | None: only `fa.json` |

Leave out of every PR: `.fa-review/` and all hand-off and script commits, and the merge commits. Order: the upstream fix branches that are offered, then 1, 2, 3 (+4), 6; 5 when upstream agrees to the behaviour change. `67e72b5` (fa.json) must land together with or after `2242169`'s check, or the check-fa project-sources test fails; the tests added in PR 1 for the audit labels read `fa.json`, so they need PR 6 or should move to it. PR 5 test `total-direction` checks the `bdi` from PR 1.

## 6. Verification (`feat/fa-locale`, from `web/`, code at `5ab137d`)

| Command | Result |
| --- | --- |
| `git remote -v`, `git push --dry-run origin HEAD`, dry run to a new branch name | origin `https://github.com/yaser-k/new-api-fork`; `Everything up-to-date` and `* [new branch] HEAD -> dryrun-test-s10` (dry run only, nothing created). Local branch was already at `origin/feat/fa-locale` = `973f3bd` |
| `git fetch upstream main` | `upstream/main` = `c2b7a9a` (unchanged); upstream push URL `DISABLED` |
| `bun install` | 1206 packages |
| `bun run typecheck` | `tsgo -b`, exit 0 |
| `bun run lint` | exit 1: **139 errors, 65 warnings**; `upstream/main` (own worktree and install): **182 errors, 66 warnings**; error file+rule pairs above upstream: **0**; errors in the 324 files changed against upstream: **0** |
| `bun run test` | `Test Files 235 passed (235)`, `Tests 2470 passed (2470)` at `5ab137d` (session start: 214 files, 2371 tests) |
| `bun run build` | exit 0, total 66912.9 kB / 20616.3 kB gzip |
| `bun run i18n:sync` | exit 0; fa partial, missing 1062, extras 0; the seven required locales missing 0, extras 0 |
| `bun run i18n:check-fa` | `check-fa: 5733 keys, no findings` (includes the placeholder source scan) |
| Locale files vs `upstream/main` | `fa.json` +5737 lines; en, fr, ja, ru, vi, zh, zh-TW: 17 keys added, 0 removed, 0 changed each. Against `973f3bd`: only `fa.json` (21 insertions, 20 deletions: 20 terminology values + 1 new key) |
| `go build -o <scratch>/bin/new-api .` (Go 1.25.1) | exit 0, embeds the fresh `web/dist`; rebuilt after `089255f`, `768f817` and `5ab137d` |

### Running app and screenshots

```
SQLITE_PATH=<scratch>/run/one-api.db GLOBAL_API_RATE_LIMIT_ENABLE=false GLOBAL_WEB_RATE_LIMIT_ENABLE=false \
  CRITICAL_RATE_LIMIT_ENABLE=false SEARCH_RATE_LIMIT_ENABLE=false TZ=UTC <scratch>/bin/new-api --port 3300
GET /api/setup -> {"data":{"status":false,"root_init":false,"database_type":"sqlite"},"success":true}
node .fa-review/scripts/shots-final.mjs http://127.0.0.1:3300 .fa-review setup fa     # before the admin exists
python3 .fa-review/scripts/seed.py  http://127.0.0.1:3300 <db> en   # POST /api/setup -> "系统初始化成功", success true
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 <db>
python3 .fa-review/scripts/seed7.py http://127.0.0.1:3300
python3 .fa-review/scripts/seed8.py <db>                            # response times 1234, 456, 12345 ms
python3 .fa-review/scripts/seed10.py <db>
python3 .fa-review/scripts/seed-final.py http://127.0.0.1:3300      # promote, demote, quota override of demo-user
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 lang fa
node .fa-review/scripts/shots-final.mjs http://127.0.0.1:3300 .fa-review main fa
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 lang en && node .fa-review/scripts/shots-final.mjs ... rt en
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 lang fr && node .fa-review/scripts/shots-final.mjs ... rt fr
node .fa-review/scripts/carousel-shot.mjs http://127.0.0.1:3399/index.html .fa-review/213-carousel-rtl.png http://127.0.0.1:3300
```

(The Playwright scripts ran from a scratch copy with `node_modules/playwright` linked to the global install. The carousel is not rendered by any page, its only user `ai-elements/inline-citation.tsx` being unused, so `carousel-harness.tsx` bundles the real `components/ui/carousel.tsx` with `bun build` and loads the app's built stylesheet, served on port 3399.)

Chromium 141.0.7390.37 headless, 1440×900, light theme, UTC. Persian: `dir=rtl lang=fa`. 17 captures, none blank (fewest colours 332, the carousel harness; blank threshold 16). Console errors: the 401 of the pre-login session probe and `ERR_CERT_AUTHORITY_INVALID` for external resources blocked by the sandbox proxy.

| File | Shows | Logged values |
| --- | --- | --- |
| `200-setup-steps-rtl.png` | setup wizard before the admin exists | step badges `۱ ۲ ۳ ۴` |
| `201-channels-response-time-rtl.png` | channels table, response-time column | `⁨۴۵۶⁩ ms`, `⁨۱٫۲۳⁩ ثانیه`, `⁨۱۲٫۳۵⁩ ثانیه` |
| `201-channels-response-time-en.png` | same, English | `456ms`, `1.23s`, `12.35s` (unchanged) |
| `201-channels-response-time-fr.png` | same, French | `456 ms`, `1,23 s`, `12,35 s` (decimal comma) |
| `202-system-info-rtl.png` | system info (full page, disk tooltip open) | `هر ۳۰ ثانیه`, `۳۷٫۳%`, `۴٫۶%`, `۴٫۴%`, task progress `۱۰۰%`, disk `۱۱٫۱ GB` / `۲۵٫۹ GB` / `۲۵۲ GB`, date titles `۱۴۰۵/۰۷/۰۵ …` |
| `203-dashboard-flow-rtl.png` | dashboard flow view | tabs `۱۰ مورد برتر` … `۱۰۰ مورد برتر` |
| `204-task-plugins-rtl.png` | task plugins | model counts in Persian digits |
| `205-audit-log-rtl.png` | audit log list | `عملیات ⁨ارتقا به مدیر⁩ روی کاربر ⁨demo-user⁩ انجام شد (شناسه: 2)`; quota summary `⁨$۵⁩ ← ⁨$۸⁩` |
| `206-audit-entry-action-rtl.png` | audit entry with an action | actor `⁨admin⁩ (شناسه: 1)`, target `⁨demo-user⁩ (شناسه: 2)`, action field `ارتقا به مدیر` |
| `207-quota-audit-summary-rtl.png` | quota override audit entry | `سهمیۀ درخواستی: $۸ · $۵ ← $۸` |
| `208-user-quota-preview-rtl.png` | user quota override preview | `سهمیۀ فعلی: ⁨$۸⁩ ← ⁨$۵⁩` |
| `209-pricing-api-tab-rtl.png` | pricing model details, API tab (full page) | RPM/TPM/RPD headers `text-align: end`; language tabs at the inline end |
| `210-dashboard-models-rtl.png` | dashboard models, header total | `مجموع: $۴٫۹۰` |
| `211-chart-preferences-rtl.png` | chart preferences dialog | icons on the inline start |
| `212-subscriptions-rtl.png` | subscriptions list | `$۹٫۹۰`, `۱ ماه`, priority `۱۰` |
| `213-carousel-rtl.png` | carousel harness, RTL above LTR | RTL: previous button on the right (x 824) pointing right, next on the left (x 236), first slide on the right; LTR unchanged |

Visible Latin text left on these pages: model, user, key, node and channel names, `master`, `linux/amd64`, `v0.0.0`, `RPM`/`TPM`/`RPD`, `POST`, routes and IDs, the channel priority and weight inputs, chart axis numbers, and the compact pager `1 / 1` (hard-coded list).

## 7. Remaining issues

1. **Code bugs found while translating (English affected too)**, kept listed: the session 7 and 8 items (channels glued labels, `&apos;`, `&mdash;`, `&#10;`, plurals, the Gemini sentence, the compliance separators, OAuth callback base; see `git show 57c75bd:.fa-review/REPORT.md`) and the joins in the hard-coded table.
2. **Dates not in the fork's pattern** (for upstream, decision of session 9): the compliance "Confirmed at" time (`integrations/payment-settings-section.tsx:840`, `toLocaleString()`) and the announcements table (`content/announcements-section.tsx:394`, `dayjs().format(…)`, plus its own relative time).
3. **Left as they are on purpose**: the chart time axes run left to right in RTL (VChart axes; reversing them would also reverse the reading of trends), the rankings trend arrows (they show up and down, not a direction of reading), native date inputs (browser), backend content in Chinese or English (Go i18n has en and zh only), the hero terminal demo (a terminal stays left to right).
4. **Found in this session, not fixed** (outside the requested items or needing a shared change): the compact pager `1 / 1` in the shared data table; the admin channel chain `a → b → c` (three or more values); the price with `$` + `toFixed(2)` on the purchase dialog and the wallet plan cards; `mr-1` on the Adjust Quota icon; the carousel's screen-reader labels (no key). All in the hard-coded table.
5. **Public Sans** (open question 1).

## 8. Open questions

1. **Public Sans font name**: `--font-sans: 'Public Sans', sans-serif` never matches the loaded face `Public Sans Variable`, so every left-to-right language renders in the system sans-serif (confirmed in Chromium). Persian is not affected (Vazirmatn). The fix (`'Public Sans Variable', 'Public Sans', sans-serif`) changes the English font, which conflicts with "no change in left-to-right languages". Apply it in the fork as its own commit and offer it upstream as a separate fix, or leave it listed?
2. **Download wording**: the glossary now reserves «بارگیری» for download, but the existing download values use «دانلود» (3 keys). Keep «دانلود» (current), or switch them to «بارگیری»?
3. **PR 1 vs PR 4 for the audit labels**: `0184320` and `768f817` (Persian ID label and action names in audit text) are split into PR 1 as instructed; they are the same kind of change as PR 4 (audit roles). Keep them in PR 1, or move them to PR 4?
4. **Shared pager digits**: format the compact `1 / 1` page counter of the shared data table with the interface locale (touches every table; PR 5)?

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
- Session 10 (this one): the four settled decisions, the final cleanup pass (digits, dates, arrows, lint-blocked RTL fixes, audit labels, `pre` blocks), this summary.
