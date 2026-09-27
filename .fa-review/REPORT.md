# Persian (fa) locale for new-api: state and hand-off

Temporary review folder. Delete `.fa-review/` before any upstream PR.

Fork `yaser-k/new-api-fork`, branch `feat/fa-locale`, based on `upstream/main` at `c2b7a9a` (upstream had not moved in session 11; no upstream merge). Upstream issue: QuantumNous/new-api#7198. This file is written so it can serve as the summary for the maintainers; the session details, verification and open items follow the summary.

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

1. **Number formatting (split-plan PR 5).** Money, quotas and the numbers fixed so far follow the interface language instead of the browser language or a raw value. For English the output is identical (tested for each change). French, Russian and Vietnamese now show a decimal comma where the interface formats decimals (for example the response-time badge `1,23 s`, chart totals `$22,98`, the subscriptions list price and, since session 11, the plan price on the purchase dialog and wallet plan cards); Chinese and Japanese keep the decimal point. Grouping follows the interface language where it already applied (flow numbers, system info bytes). The compact table page counter and the percent sign (session 11) change only Persian: the counter has no grouping and the sign is `%` in every other language.
2. **The 17 keys from the upstream fix branches** merged into `feat/fa-locale` (9 from `fix/legal-consent-sentence`, 8 from `fix/ui-label-strings`), present in all seven required locales.
3. **Public Sans (branch `fix/public-sans-font`, merged in session 11).** `--font-sans` now names the face that `@fontsource-variable/public-sans` registers (`'Public Sans Variable', 'Public Sans', sans-serif`), so left-to-right languages render in Public Sans instead of the browser's generic sans-serif (Chromium: sample line 820.06 px → 854.92 px; one 26 832-byte Latin woff2 now downloaded). Persian is unchanged: `html:lang(fa)` still sets Vazirmatn, and no Public Sans face loads on a Persian page (checked on the running app).

Nothing else changes for left-to-right languages: every RTL fix uses logical classes or RTL-only rules, and every Persian-only label (audit roles, sign-in methods, the audit ID label and user action names) is guarded by the locale; tests pin the English (and where relevant Chinese) output. Among the locale files only `fa.json` differs from upstream apart from those 17 keys (checked: each of en, fr, ja, ru, vi, zh, zh-TW has 17 added, 0 removed, 0 changed keys).

## 2. This session (session 11)

Session 10's details (its decisions, the decision 4 evidence for the unused keys, the final cleanup pass B1 to B5 and its commit list) are in `git show 50c22b2:.fa-review/REPORT.md`, section 2.

### Decisions applied

| # | Decision | Commit | Result |
| --- | --- | --- | --- |
| 1 | Keep «دانلود» for download; drop «بارگیری» from the glossary | `c7e15d1`* | Glossary: the load row no longer names «بارگیری»; the download row is «دانلود، دانلودشده», with «دریافت» kept for fetch and pull. fa.json (through the script): the two "downloaded source" values in the plugin integrity messages used «دریافت‌شده», now «دانلودشده». Every key whose English says download now uses «دانلود»; «بارگیری» appears nowhere in fa.json or `docs/i18n/fa.md`. *fa.json part to PR 6, fa.md part to PR 2 |
| 2 | Audit label commits `0184320`, `768f817` and their tests move from PR 1 to PR 4 | `bb8feb3` | REPORT.md only (section 5). PR 1 is layout only |
| 3 | Format the shared compact page counter in the interface language | `8f0966d` | `formatFixed(n, 0, locale)` (no grouping): English `1 / 1234` exactly as before, Persian `۱ / ۱۲۳۴`. Test `data-table/core/pagination`: 1 failed / 6 passed → 7 passed (the new English case passes before and after) |
| 4 | Public Sans fixed upstream-style on its own branch | `4a9aaa9` on `fix/public-sans-font`, merged `aa583a7` | See below |

### Upstream fix branch `fix/public-sans-font`

From `upstream/main` (`c2b7a9a`), head `4a9aaa9`, one commit, merged into `feat/fa-locale` with a merge commit (`aa583a7`). Nothing Persian on the branch.

| File | Change |
| --- | --- |
| `web/src/styles/theme.css` | `--font-sans: 'Public Sans', sans-serif` → `'Public Sans Variable', 'Public Sans', sans-serif` (the `--font-serif` pattern) |
| `web/src/styles/__tests__/font-sans.test.ts` (new) | Reads the `@fontsource` sans package that `index.css` imports, collects the family its CSS registers (`Public Sans Variable`) and checks it is the first family of `--font-sans` in `theme.css`. Fails on `upstream/main` (`expected 'Public Sans' to be 'Public Sans Variable'`), passes with the fix |

Other places checked: `index.css` (the import, `font-sans` on `html`, `--font-body` on `body`), `theme-presets.css` (font axis: `sans` → `var(--font-sans)`), `lib/theme-customization.ts` (only a comment names "Public Sans"; the font options are `default`/`sans`/`serif`, no family names). None needed a change. No lint errors in the touched files, so no lint commit.

Chromium 141.0.7390.37 headless, 1440×900, light theme, default preset (`--font-body: var(--font-sans)`), English, `/` and `/sign-in`, after `networkidle`, `document.fonts.ready` and 1.5 s; sample line "The quick brown fox jumps over the lazy dog 0123456789" at 32 px (`.fa-review/scripts/font-probe.mjs`):

| | `upstream/main` (`c2b7a9a`) | `fix/public-sans-font` (`4a9aaa9`) |
| --- | --- | --- |
| `--font-sans` / body font | `"Public Sans", sans-serif` | `"Public Sans Variable", "Public Sans", sans-serif` |
| Public Sans faces in `document.fonts` | 3 × `Public Sans Variable`, all `unloaded` | latin face `loaded` (latin-ext and vietnamese `unloaded`, not needed) |
| Width in `var(--font-sans)` | 820.06 px | 854.92 px |
| Width in generic `sans-serif` | 820.06 px (same: fallback font) | 820.06 px |
| Width in `'Public Sans Variable'` (forced load, measured last) | 854.92 px | 854.92 px (same: the font is used) |
| woff2 downloaded by the page | none | `public-sans-latin-wght-normal.035c7fe496.woff2`, 26 832 B |
| Console errors | `/`: `ERR_TUNNEL_CONNECTION_FAILED` (external resource blocked by the sandbox proxy); `/sign-in`: 401 of the session probe | same |

The same numbers on `/` and `/sign-in`, and on `feat/fa-locale` in English after the merge. Screenshots `310`/`311` (home) and `312`/`313` (sign-in), before and after.

### Leftovers fixed (part C)

| Item | Commit | What changed (LTR unchanged except PR 5 formatting, English identical) | Tests (before → after) |
| --- | --- | --- | --- |
| C1 percent sign | `7146052` | `appendPercentSign(formatted, locale)` in `@/lib/format` takes the sign from `Intl` (`percentSign` part): `%` for every language, `٪` for Persian, appended with no space as before. Used where the number is already formatted in the interface language: system info CPU/memory/disk and task progress, API key remaining percentage (cell label and details), dashboard flow share tooltip. Persian `۴٫۴%` → `۴٫۴٪` | `lib/percent-sign-locale` (all languages + invalid) new; `system-info/number-locale` 2 failed → pass; `dashboard/flow-number-locale` 1 failed → pass; `keys/api-key-listing` 1 failed → pass (English cases pass before and after) |
| C2 retry chain | `0c605b3` | `formatValueChain(values, language)` in `@/i18n/languages` (and `formatValueChange` now calls it): LTR `3 → 7 → 12` unchanged; RTL `⁨3⁩ ← ⁨7⁩ ← ⁨12⁩`. Used by the admin "Retry Chain" row in the usage log details | `i18n/value-change` +10 chain cases (failed before, the function did not exist); `usage-logs/retry-chain-direction` Persian 1 failed / English 1 passed → 2 passed |
| C3 plan price | `57a0080` | Purchase dialog amount due and wallet plan card price: `toFixed(2)` → `formatFixed(…, 2, locale)`, as in the subscriptions list. English `$9.90`, Persian `$۹٫۹۰` | `subscriptions/purchase-price-locale`, `wallet/plan-price-locale`: Persian 1 failed / English 1 passed each → 2 passed each |
| C4 `mr-1` | `d264460` | Adjust Quota icon in the user drawer `mr-1` → `me-1` | `users/adjust-quota-direction` 1 failed → pass |

Existing components reused: every fix changes formatting or classes on the existing components; no new component. New shared helpers: `appendPercentSign` (`@/lib/format`), `formatValueChain` (`@/i18n/languages`).

### Commits (from `50c22b2`)

```
c7e15d1 fix(i18n): use «دانلود» for every Persian download string
bb8feb3 chore: move the Persian-only audit label commits to split-plan PR 4
8f0966d fix(web): write the compact table page counter in the interface language
7146052 fix(web): use the interface language's percent sign after formatted numbers
0c605b3 fix(web): make the admin retry chain follow the page direction
57a0080 fix(web): format the plan price on the purchase dialog and wallet cards
d264460 fix(web): logical margin on the Adjust Quota icon
aa583a7 Merge branch 'fix/public-sans-font' into feat/fa-locale   (4a9aaa9)
(this hand-off): .fa-review scripts, screenshots, report
```

## 3. Hard-coded or unkeyed English (not fixed here; each needs a new key in every locale, or a code change)

Rows updated in session 11: the page counter, the retry chain, the two plan prices and `mr-1` are removed; the rows found in session 11 are added at the end.

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


## 5. Upstream split plan (proposal only; nothing built)

Commits that mix concerns (marked *) need their hunks split when the PR branch is built. Upstream-ready branches in the fork (each from `upstream/main`):

| Branch | Head | State |
| --- | --- | --- |
| `fix/dashboard-chart-time-order` | `81b140e` | chart points ordered by timestamp; merged into `feat/fa-locale` (`bc388d4`) |
| `fix/billing-status-label` | `6bf13ab` | not merged |
| `fix/delete-account-confirm-label` | `747769c` | not merged |
| `fix/2fa-setup-step-label` | `6d614c0` | not merged |
| `fix/quota-insufficient-i18n` | `28b7893` | not merged |
| `fix/legal-consent-sentence` | `6a660f7` | merged (`44610c0`, `6f7168b`); overlaps an older upstream PR by another contributor, so not offered as its own PR for now |
| `fix/ui-label-strings` | `3b86126` | merged (`b2d14dc`) |
| `fix/public-sans-font` | `4a9aaa9` | **new in session 11**; `--font-sans` names the loaded face; merged (`aa583a7`). Changes the font of every left-to-right language (to the intended Public Sans) |

| # | PR | Commits | Tests | Other languages |
| --- | --- | --- | --- | --- |
| 1 | `fix(web): right-to-left layout in shared components` | `c78c83f`* (without the `formatQuota` locale hunks), `d8ee064`, `cb25020`, `8ccec81`, `936ab28`, `129d598`, the RTL half of `f01c5af`*, the sidebar side of `b2720c9`*, `666c43c` (where PR 1 touches the file), `0d40127`, `f40c978`, `10cd251` + `9f8f102` (squash), `53458ed`, `7b04478`, `6496416`, `cb80274`, `e6f7d76`, **and from this session `5e64f68`, `272cbf9`, `5a2bd38` (lint cleanup, first), `44e9c0b`, `ce52823`, the `bdi` hunk of `aa699fc`***, **and from session 11 `0c605b3` (retry chain), `d264460` (`mr-1`)** | the earlier direction tests, plus **value-change, quota-audit-direction, quota-preview-direction, change-arrow-direction, carousel-direction, api-tab-direction, preferences-direction, code-direction (pre)**, **value-change (chain cases), retry-chain-direction, adjust-quota-direction** | No visible change in LTR. Layout only (decision 2 of session 11: the Persian-only audit labels moved to PR 4) |
| 2 | `feat(i18n): Persian (fa) partial locale, tooling and docs` | `b2720c9`* (without the sidebar), `07e070d`, `759a31e`, `f5296da` + `776c03c`, `bfe01bb`, the script part of `fa96f8b`; AGENTS.md, web/AGENTS.md, the i18n skill; `docs/i18n/fa.md` (with the glossary rows from `4d901bd`, `d1e501d`, `305ee51`, **the fa.md part of `b1c4bad` and of `c7e15d1`**), `2242169`, **`0330995` (`@babel/parser` dev dependency)** | languages, direction-provider, check-fa, persian-monospace, intl-locale lint case | Language switcher lists «فارسی»; `docs/i18n/fa.md` is a new file under `docs/` (ask the maintainers) |
| 3 | `feat(web): Solar Hijri dates, chart axes and date pickers in Persian` | `0ea975e`, `c2369c8` + `42a98a2`, `170ba72`, `9d22fac`, **`d14b92a`** | display-date-locale, activity-time-cell-dates, login-session-dates, date-picker-display, calendar-persian, chart-time-locale, **system-info title-dates** | None |
| 4 | `feat(web): Persian labels in audit text` (roles, sign-in methods, ID label, user action names) | net of `f767402` + `9cd40a8`, **`0184320`, `768f817`** (moved from PR 1, decision 2 of session 11) | audit-content-locale, details-locale, **identity-action-locale, user-action-name, manage-operator-locale** | None: Persian only, English and Chinese pinned (could fold into 3) |
| 5 | `fix(web): format money and numbers in the interface language` | locale half of `f01c5af`*, `formatQuota` hunks of `c78c83f`*, the `locale` argument in `recharge-form-card.tsx`, `31f68f8`, **`aae92d9`, `0153b2d`, `aa699fc`* (without the `bdi` hunk), `d623586`, `172c97e`, `eb96d86`, `089255f`, `5ab137d`**, **session 11: `8f0966d` (page counter), `7146052` (percent sign), `57a0080` (plan price)** | format-quota-locale, format-currency-locale, amount-locale, summary-cards-locale, profile-header-locale, response-time-format, **format-fixed-locale, message-duration-locale, system-info number-locale, flow-number-locale, chart-number-locale, total-direction (bdi half goes with PR 1), step-number-locale, count-locale, subscriptions format-locale and list-number-locale**, **pagination (counter), percent-sign-locale, purchase-price-locale, plan-price-locale** | **Yes**: numbers follow the interface language; French, Russian and Vietnamese show a decimal comma in the formatted decimals; English is unchanged (tested). Its own PR |
| 6 | `feat(i18n): Persian translation batches` | `fa96f8b`, `551a8a3`, `795d296`, `4db67e2`, `3825299`, `aaba1b0`, `8a76a75`, `ada5af2`, `c6aaad5`, `4d901bd` (fa.json part), `17d9c1b`, `58de3e3`, `f7ad4f7`, `4ba8a4c`, `3b076b1`, `25399dc`, `86a8c7c`, `97b1b72`, `bcc8ea7`, `3ca2feb`, `d1e501d` (fa.json part), `0097c77`, `c2798f3`, `4c66300`, `67e72b5`, `39a571f`, `5cd6e90`, `b744e97`, `334c3d9`, `2adca66`, `0d1b17b`, `467f231`, `a14be44`, `bdf0202`, **`b1c4bad` (fa.json part), `4972f14`, `c7e15d1` (fa.json part)** | `bun run i18n:check-fa` | None: only `fa.json` |

Leave out of every PR: `.fa-review/` and all hand-off and script commits, and the merge commits. Order: the upstream fix branches that are offered (`fix/public-sans-font` among them), then 1, 2, 3 (+4), 6; 5 when upstream agrees to the behaviour change. `67e72b5` (fa.json) must land together with or after `2242169`'s check, or the check-fa project-sources test fails; the PR 4 tests for the audit labels read `fa.json`, so PR 4 lands together with or after PR 6. PR 5 test `total-direction` checks the `bdi` from PR 1.

## 6. Verification (session 11)

### `feat/fa-locale` (from `web/`, code at `aa583a7`)

| Command | Result |
| --- | --- |
| `git remote -v`, `git push --dry-run origin HEAD`, `git push --dry-run origin HEAD:refs/heads/dryrun-probe-xyz` | origin `https://github.com/yaser-k/new-api-fork`; `Everything up-to-date` and `* [new branch] HEAD -> dryrun-probe-xyz` (dry run, nothing created). The local branch had an older, different history (`d263b5f`); reset to `origin/feat/fa-locale` = `50c22b2` first |
| `git remote add upstream …`, `git fetch upstream main`, `git remote set-url --push upstream DISABLED` | `upstream/main` = `c2b7a9a` (unchanged, no merge); upstream push URL `DISABLED` |
| `bun install` | 1206 packages |
| `bun run typecheck` | `tsgo -b`, exit 0 |
| `bun run lint` | exit 1: **139 errors, 65 warnings**; `upstream/main` (the `fix/public-sans-font` worktree, which only adds a lint-clean test file): **182 errors, 66 warnings**; error file+rule pairs above upstream: **0**; errors in the 337 files changed against upstream: **0** |
| `bun run test` | `Test Files 241 passed (241)`, `Tests 2499 passed (2499)` (session start: 235 files, 2470 tests) |
| `bun run build` | exit 0, total 66913.4 kB / 20616.5 kB gzip |
| `bun run i18n:sync` | exit 0; fa partial, missing 1062, extras 0; the seven required locales missing 0, extras 0 |
| `bun run i18n:check-fa` | `check-fa: 5733 keys, no findings` (includes the placeholder source scan) |
| Locale files vs `upstream/main` | en, fr, ja, ru, vi, zh, zh-TW: 17 keys added, 0 removed, 0 changed each; `fa.json` +5737 lines. Against `50c22b2`: only `fa.json` (2 values) |
| `go build -o <scratch>/bin/fa .` (Go 1.25.1) | exit 0, embeds the fresh `web/dist` |

### `fix/public-sans-font` (own worktree, from `web/`, head `4a9aaa9`)

| Command | Result |
| --- | --- |
| `bun install` | 1202 packages |
| `bun run typecheck` | exit 0 |
| `oxlint -c .oxlintrc.json src/styles/__tests__/font-sans.test.ts` | no findings (the other changed file is CSS) |
| `vitest run src/styles/__tests__/font-sans.test.ts` | before the fix: 1 failed (`expected 'Public Sans' to be 'Public Sans Variable'`); after: 1 passed |
| `bun run test` | `Test Files 167 passed (167)`, `Tests 2112 passed (2112)` |
| `bun run build` | exit 0, total 66156.9 kB / 20339.0 kB gzip (`upstream/main` build for the "before" binary: 66156.8 kB) |
| `go build` twice, embedding the `upstream/main` and the branch `web/dist` | exit 0 (`ups-base`, `ups-fix`) |

### Running app and screenshots

Three fresh SQLite instances in a scratch directory (`SQLITE_PATH=<scratch>/run-<name>/one-api.db`, rate limits off, `TZ=UTC`): `ups-base` (upstream/main) on 3301, `ups-fix` (`fix/public-sans-font`) on 3302, `fa` (`feat/fa-locale`) on 3303.

```
GET /api/setup -> {"data":{"status":false,"root_init":false,"database_type":"sqlite"},"success":true}   (each)
python3 .fa-review/scripts/seed-s11.py http://127.0.0.1:3301 <db>        # POST /api/setup -> "系统初始化成功", success true
python3 .fa-review/scripts/seed-s11.py http://127.0.0.1:3302 <db>
python3 .fa-review/scripts/seed-s11.py http://127.0.0.1:3303 <db> fa     # + 3 channels, plan $9.90, retried consume log, language fa
node font-probe.mjs http://127.0.0.1:330{1,2} {/,/sign-in} en            # table in section 2
node shots-s11.mjs http://127.0.0.1:3301 .fa-review ups before
node shots-s11.mjs http://127.0.0.1:3302 .fa-review ups after
node font-probe.mjs http://127.0.0.1:3303 / en                          # fa branch in English: Public Sans Variable loaded, 854.92 px
node shots-s11.mjs http://127.0.0.1:3303 .fa-review fa
```

(The Playwright scripts ran from a scratch copy with `node_modules` linked to the global install.) Chromium 141.0.7390.37 headless, 1440×900, light theme, UTC. None blank (fewest colours 853; blank threshold 16). Console errors: the 401 of the pre-login session probe, and `ERR_TUNNEL_CONNECTION_FAILED` / `ERR_CERT_AUTHORITY_INVALID` for external resources blocked by the sandbox proxy.

| File | Shows | Logged values |
| --- | --- | --- |
| `310-home-before-en.png`, `311-home-after-en.png` | English home, `upstream/main` / `fix/public-sans-font` | body `"Public Sans", sans-serif`, no face loaded / body `"Public Sans Variable", …`, loaded `Public Sans Variable` |
| `312-sign-in-before-en.png`, `313-sign-in-after-en.png` | English sign-in, same pair | same |
| `320-system-info-counter-percent-rtl.png` | system info (full page) | task history counter `۱ / ۱`; `۴٫۵٪`, `۴٫۶٪`, `۱۰۰٪` |
| `321-retry-chain-rtl.png` | usage log details, admin retry chain | text `⁨1⁩ ← ⁨2⁩ ← ⁨3⁩`, direction rtl; glyph x: 1 at 821, 2 at 792, 3 at 763 (first channel on the right) |
| `322-wallet-plan-card-rtl.png` | wallet plan card | `$۹٫۹۰` |
| `323-purchase-dialog-rtl.png` | purchase dialog | amount due `$۹٫۹۰` |
| `324-home-vazirmatn-rtl.png` | Persian home | `dir=rtl lang=fa`, body `"Vazirmatn Variable", sans-serif`, loaded faces `Vazirmatn Variable`, `Vazirmatn Persian Script` (no Public Sans) |

## 7. Remaining issues

1. **Code bugs found while translating (English affected too)**, kept listed: the session 7 and 8 items (channels glued labels, `&apos;`, `&mdash;`, `&#10;`, plurals, the Gemini sentence, the compliance separators, OAuth callback base; see `git show 57c75bd:.fa-review/REPORT.md`) and the joins in the hard-coded table.
2. **Dates not in the fork's pattern** (for upstream, decision of session 9): the compliance "Confirmed at" time (`integrations/payment-settings-section.tsx:840`, `toLocaleString()`) and the announcements table (`content/announcements-section.tsx:394`, `dayjs().format(…)`, plus its own relative time).
3. **Left as they are on purpose**: the chart time axes run left to right in RTL (VChart axes; reversing them would also reverse the reading of trends), the rankings trend arrows (they show up and down, not a direction of reading), native date inputs (browser), backend content in Chinese or English (Go i18n has en and zh only), the hero terminal demo (a terminal stays left to right).
4. **Still listed** (hard-coded table): the carousel's screen-reader labels (no key); and, found in session 11, the full pager's page buttons and `Total:`, the raw percentages (`toFixed` + `%`), the discount label's raw number, the request conversion chain ` -> `, and the raw response time and group ratio in the usage log details. Each needs a new key, a shared change or a decision beyond the four items fixed this session.
5. `lib/theme-customization.ts` says in a comment that the `default` preset resolves to serif, while `PRESET_DEFAULT_FONT.default` is `sans` (upstream comment; not touched, the Public Sans branch stays minimal).

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
- Session 11 (this one): download wording, audit labels to PR 4, the page counter, the upstream `fix/public-sans-font` branch (merged), percent sign, retry chain, plan prices, `mr-1`.
