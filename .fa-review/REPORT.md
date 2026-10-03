# Persian (fa) locale for new-api: state and hand-off

Temporary review folder. Delete `.fa-review/` before any upstream PR.

Fork `yaser-k/new-api-fork`, branch `feat/fa-locale`, merged with `upstream/main` at `1a4166d` in session 13 (merge `d95fb41`). Upstream issue: QuantumNous/new-api#7198. Session 13 also built the three upstream-ready branches `pr/fa-rtl-layout`, `pr/fa-locale` and `pr/fa-dates-numbers` (section 5); their draft PR texts are `pr-rtl-layout.md`, `pr-locale.md` and `pr-dates-numbers.md`, and the file-by-file coverage is `coverage.md`, all in this folder. They are open upstream as QuantumNous/new-api#7651, QuantumNous/new-api#7652 and QuantumNous/new-api#7653. Session 14 fixed CodeRabbit's findings on the last two on new branches `review/fa-locale` and `review/fa-dates-numbers` (section 2), each its PR branch plus new commits, and merged both into this branch; the PR branches are unchanged until they are fast-forwarded.

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

- `.agents/skills/i18n-translate/fa.md` (new file; `docs/i18n/fa.md` until session 14, moved next to the i18n skill because AGENTS.md does not allow new files under `docs/` without a request): locale status, date rules, a glossary of 93 terms with reasons, the terms that stay English, eleven typography rules (ZWNJ, Persian ی and ک, digits, ۀ, punctuation, spacing around Latin words, FSI … PDI isolates around interpolated left-to-right values, RLI … PDI around Persian placeholders of left-to-right inputs), style and the automated check.
- `bun run i18n:check-fa` (`web/scripts/check-fa.mjs`): exits 1 on any finding. It checks the typography rules, isolate pairing, empty values, stray whitespace, keys missing from `en.json`, and parses `web/src` with `@babel/parser` (now a declared dev dependency, 7.29.7, single copy in `bun.lock`) to check the placeholders of `dir='ltr'` inputs. Today: `check-fa: 5817 keys, no findings`.
- `bun run i18n:sync` reports Persian as partial and never fills it with English. The i18n skill, `AGENTS.md` and `web/AGENTS.md` describe `fa` as the optional partial eighth locale.

### What differs from upstream for other languages

1. **Number formatting (split-plan PR 5).** Money, quotas and the numbers fixed so far follow the interface language instead of the browser language or a raw value. For English the output is identical (tested for each change). French, Russian and Vietnamese now show a decimal comma where the interface formats decimals (for example the response-time badge `1,23 s`, chart totals `$22,98`, the subscriptions list price and, since session 11, the plan price on the purchase dialog and wallet plan cards); Chinese and Japanese keep the decimal point. Grouping follows the interface language where it already applied (flow numbers, system info bytes). The compact table page counter and the percent sign (session 11) change only Persian: the counter has no grouping and the sign is `%` in every other language.
2. **The 9 legal-consent keys** (`fix/legal-consent-sentence`), present in all seven required locales. The other upstream fix branches are now upstream's own code (#7628, section 2).
3. **Public Sans** (now upstream's own fix, #7628; our branch `fix/public-sans-font` is superseded). `--font-sans` now names the face that `@fontsource-variable/public-sans` registers (`'Public Sans Variable', 'Public Sans', sans-serif`), so left-to-right languages render in Public Sans instead of the browser's generic sans-serif (Chromium: sample line 820.06 px → 854.92 px; one 26 832-byte Latin woff2 now downloaded). Persian is unchanged: `html:lang(fa)` still sets Vazirmatn, and no Public Sans face loads on a Persian page (checked on the running app).

Nothing else changes for left-to-right languages: every RTL fix uses logical classes or RTL-only rules, and every Persian-only label (audit roles, sign-in methods, the audit ID label and user action names) is guarded by the locale; tests pin the English (and where relevant Chinese) output. Among the locale files only `fa.json` differs from upstream apart from those 9 keys.

## 2. This session (session 14): CodeRabbit fixes on two review branches

Session 13's details (upstream merge `d95fb41`, how the three branches were split, decisions 1 to 5, merge order) are in `git show 94dc580:.fa-review/REPORT.md`, section 2.

The three PRs are open upstream: QuantumNous/new-api#7651 (`pr/fa-rtl-layout`, `7204ba0`), QuantumNous/new-api#7652 (`pr/fa-locale`, `784f21d`) and QuantumNous/new-api#7653 (`pr/fa-dates-numbers`, `7963a31`). CodeRabbit reviewed the last two (four inline comments and a nitpick on the locale PR; two inline, one outside the diff and a nitpick on the dates/numbers PR). Each finding was checked in the code and fixed, where it holds, on a new branch: the PR branch plus new commits on top, so the PR branch can be fast-forwarded to it (no rebase, amend, merge commit or `.fa-review/`). The PR branches themselves are unchanged. Draft replies, one per comment: `coderabbit-replies.md` in this folder.

### A. `review/fa-locale` (for QuantumNous/new-api#7652), head `4285a18`

| Commit | Finding | Result |
| --- | --- | --- |
| `2255498` docs(i18n): move the Persian guide into the i18n skill | new file under `docs/` (inline) | holds (AGENTS.md forbids new `docs/` files without a request). `git mv docs/i18n/fa.md .agents/skills/i18n-translate/fa.md`; `AGENTS.md`, the skill (three places), the `check-fa.mjs` comment and the guide's own pointer to `SKILL.md` updated. Nothing under `docs/` differs from upstream. The last reference, a comment in `calendar.tsx`, is on the dates branch (B, `2f4df76`) |
| `4d75b96` docs(i18n): let the skill's untranslated-entries sample skip a missing fa.json | Step 3 sample throws `ENOENT` (inline) | holds. Step 3 uses Step 4's `readLocale` guard. Extracted and run on the upstream tree (no `fa.json`): before, `Error: ENOENT … fa.json`, exit 1; after, exit 0 (`fa: all translated`); with `fr.json` missing it still throws |
| `555f22a` test(web): pin the default number locale in the audit quota test | quota assertions depend on the runtime default (inline) | holds (under `fr`: `1 $ · 0 $ → 1 $`). The English/Chinese quota test sets an `en-US` default through the `Intl.NumberFormat` mock the German case already used (now one helper, `mockBrowserNumberLocale`); text checks unchanged |
| `c22a2f4` build(web): declare @tailwindcss/node as a dev dependency | undeclared import (inline) | holds. `"@tailwindcss/node": "^4.3.3"` (the version `bun.lock` resolved through `@rsbuild/plugin-tailwindcss`). `bun install`: the only `bun.lock` change is that line in the workspace `devDependencies`; package entries 1378 before and after; one `@tailwindcss/node@4.3.3`, one `react-day-picker@10.0.1`; `bun install --frozen-lockfile` clean. Package check (web/AGENTS.md 3.15): MIT, Tailwind Labs, already installed at that version, dev only, not in the app bundle |
| `2b2ed43` test(web): explicit return types on the new test helpers | nitpick | `renderUpgradeBadge`/`renderEnumCell`: `void`; `renderOverridePreview`: `Promise<HTMLElement>`; both `renderDialog`: `Promise<BoundFunctions<typeof queries>>` (inline `type` imports; a separate `import type` trips `no-duplicates`); `t`: `string` |
| `4285a18` test(web): pin the default number locale in the quota arrow tests | sweep | see below |

**Sweep (tests that assert a formatted amount, number or date without fixing the locale).** The branch's 16 test files were grepped, and the whole suite was run with `LANG`/`LC_ALL` set to `fr_FR`, `de_DE` and `ar_EG` (Node's default locale follows them; ar-EG also switches to Arabic-Indic digits). Of the branch's own tests, only two more depended on the default: `quota-audit-direction` (English under fr/de/ar; Persian under ar, whose pattern accepted Latin or Persian digits only) and `quota-preview-direction` (English: `\S+` does not match `1 $`). Both now pin `en-US` the same way. Everything else in those runs that failed is upstream's own tests (section 6).

### B. `review/fa-dates-numbers` (for QuantumNous/new-api#7653), head `3a23fa1`

| Commit | Finding | Result |
| --- | --- | --- |
| `35562b8` fix(web): format the purchase dialog quota amounts in the interface locale | `formatQuota` without locale at lines 279, 309, 313 (outside the diff) | holds: in Persian, plan quota `$10`, required `$9.9` and available `$5` were Latin next to `$۹٫۹۰`. All three get `locale`; `purchase-price-locale.test.tsx` gains English and Persian cases (`‎$۱۰`, `‎$۹٫۹`, `‎$۵`; Intl puts an LRM before a Persian currency amount) |
| `c3d2e96` fix(web): pass the interface locale to the remaining quota and number formatters | sweep of the files the branch touches | see below |
| `39bc56c` fix(web): use the resolved language for the usage log details cell | `i18n.language` to `buildDetailSegments` (inline) | holds: when Persian is requested but resolves to English (no Persian translations loaded), the details price was `‎$۰٫۲۵` next to a `$0.01` cost. Now `i18n.resolvedLanguage \|\| i18n.language`, like line 359; new `detail-segments-locale.test.tsx` (resolved and unresolved) |
| `e0173e8` test(web): pin the default number locale in the quota audit amount test | `toMatch(/\$2$/)` depends on the runtime default (inline) | holds (fr: `2 $`; ar-EG: Arabic-Indic digits). The English test pins `en-US`; the `t` stub gets a return type |
| `2f4df76` docs(web): point the calendar comment to the moved Persian guide | follows A1 | the `calendar.tsx` comment on the `@daypicker/persian` version lock named `docs/i18n/fa.md` |
| `7afd1a7` test(web): pin the runtime default locale in the unknown-language tests | test sweep | see below |
| `3a23fa1` test(web): pin the default number locale in the flow tooltip test | test sweep | see below |
| — | nitpick on `wallet/components/subscription-plans-card.tsx` 475-494 | not changed: other languages keep upstream's exact `toLocaleString()` (upstream lines 475 and 480); the PR changes only dates shown in Persian, and the Gregorian `title` rule is for table, list and log cells (this is a card) |

**Formatter sweep (`c3d2e96`).** A parser script listed every `formatQuota`, `formatLogQuota`, `formatNumber`, `formatCompactNumber`, `formatFixed`, `formatCurrencyFromUSD`, `formatQuotaWithCurrency`, `formatBillingCurrencyFromUSD`, `formatLocalCurrencyAmount`, `Intl.*`, `toLocale*String` and `toFixed` call in the 85 non-test source files the branch touches. Fixed, each with an English and a Persian test (the Persian one fails without the fix):

| File | Call |
| --- | --- |
| `components/data-table/core/pagination.tsx` | `Total:` (compact and full pager): `totalRows.toLocaleString()` → `formatNumber(totalRows, locale)` (section 3 listed it) |
| `features/redemption-codes/components/redemptions-columns.tsx` | quota badge `formatQuota(quota)` |
| `features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx` | used/total quota |
| `features/channels/components/channels-columns.tsx` | balance cell: the full used and remaining amounts (only the compact forms had `locale`) and the "Balance updated" notice |
| `features/channels/components/dialogs/balance-query-dialog.tsx` | current balance |

Not changed: `formatBalance` and `formatQuota` in `channels/lib/channel-utils.ts` (exported, no caller); the non-Persian `toLocaleString()` dates in `channel-mutate-drawer.tsx` and `subscription-plans-card.tsx` (upstream output kept, as for B4); `playground/…/message-metadata.tsx` message time `Intl.DateTimeFormat(undefined, …)` (a time of day, not a number formatter; in Persian it follows the browser); `formatTaskUsageUnitPrice` in the usage log details task prices (takes no locale; its file `pricing/lib/dynamic-price.ts` is not in this branch and has 13 callers); raw `toFixed` displays already listed in section 3 / 7.4 (ratios, percentages, the admin plan picker price `$${price.toFixed(2)}`); `formatCurrencyUSD` (a wrapper without a locale parameter).

**Test sweep.** The branch's 36 test files were grepped, and the whole suite was run with `LANG`/`LC_ALL` set to `fr_FR`, `de_DE` and `ar_EG`. Besides `quota-audit-number-locale` (B3), four of the branch's tests depended on the default: the unknown-language fallbacks of `formatFixed` (`1.20`), `appendPercentSign` (`%`) and `formatResponseTime` (`456ms`, `1.23s`) in `format-fixed-locale`, `percent-sign-locale` and `response-time-format` (fr and de: decimal comma; ar-EG also digits and the percent sign). `toIntlLocale('xx-invalid')` returns a well-formed tag that Intl does not support, so a `locales ?? 'en-US'` mock does not reach them; their mock lets en-US stand in for the runtime default when `supportedLocalesOf` finds none of the requested locales. The tests are now named after that fallback. The date tests build both the input and the expected text from local `Date` values, so they do not depend on the locale (and passed in all four runs). Compared with the same fr run on `upstream/main` (75 of its own tests fail under fr), one upstream test failed only on this branch: `dashboard/lib/flow.test.ts` › tooltips expects the share `100.0%`; upstream wrote `toFixed(1)` + `%` whatever the locale, the branch formats it in the given locale, and the test passes none, so under fr/de it became `100,0%`. That test now pins an `en-US` default (`3a23fa1`); its five other ar-EG failures are upstream's own (same on `upstream/main`).

### C. Merge checks

Each review branch merges alone on `upstream/main` (fast-forward). `pr/fa-rtl-layout` + `review/fa-locale`: clean. `review/fa-dates-numbers` on top of those two: 9 conflicted files, the same 9 as session 13 (5 against RTL: `pagination.test.tsx`, `calendar.tsx`, `api-key-listing.test.tsx`, `system-tasks-table.tsx`, `common-logs-columns.tsx`; 4 against locale: `marketplace-plugin-card.tsx`, `audit-details.ts`, `details-dialog.tsx`, `quota-audit-operation.ts`). The new commits add none.

### D. `feat/fa-locale`

Both review branches merged here (`3d86cf0`, `3e2947f`, then `ebab12b` for `3a23fa1`, a clean merge). The PR branches were split from this branch and never merged back, so their own commits conflict (14 and 15 files); each merge was resolved to this branch's content plus only the review delta (`784f21d..4285a18`, `7963a31..7afd1a7`), built with `git apply --3way` and compared line by line with the delta before the merge commit; the merge results equal those trees.

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
| components/data-table/core/pagination.tsx (full pager) | page number buttons written raw (`Total:` follows the interface locale since session 14, `c3d2e96`) |
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


## 5. Upstream branches (built in session 13, reviewed in session 14)

The proposal from session 12 (six PRs) is built as three branches: plan PR 1 → `pr/fa-rtl-layout`; plan PRs 2, 6 and 4 → `pr/fa-locale`; plan PRs 3 and 5 → `pr/fa-dates-numbers`. The session 12 table is in `git show 16bfa21:.fa-review/REPORT.md`, section 5. The PR branches are unchanged in session 14; the review branches add the CodeRabbit fixes on top of them and are meant to be fast-forwarded into them once checked (`git push origin review/fa-locale:pr/fa-locale`, the same for dates/numbers, never forced).

| Branch | PR | Head | Commits since `upstream/main` | Files | Draft text |
| --- | --- | --- | ---: | ---: | --- |
| `pr/fa-rtl-layout` | QuantumNous/new-api#7651 | `7204ba0` | 3: `e75239e` shared components (95 files) · `4b7a931` feature pages (244) · `7204ba0` guard test (2) | 341 | `pr-rtl-layout.md` |
| `pr/fa-locale` | QuantumNous/new-api#7652 | `784f21d` | 3: `2a0384d` (14) · `ffdb275` (23) · `784f21d` (10) | 44 | `pr-locale.md` |
| `review/fa-locale` | for QuantumNous/new-api#7652 | `4285a18` | 9: the 3 above + `2255498` `4d75b96` `555f22a` `c22a2f4` `2b2ed43` `4285a18` (section 2A) | 44 | |
| `pr/fa-dates-numbers` | QuantumNous/new-api#7653 | `7963a31` | 2: `a6f5ca5` (92) · `7963a31` (75) | 119 | `pr-dates-numbers.md` |
| `review/fa-dates-numbers` | for QuantumNous/new-api#7653 | `3a23fa1` | 9: the 2 above + `35562b8` `c3d2e96` `39bc56c` `e0173e8` `2f4df76` `7afd1a7` `3a23fa1` (section 2B) | 124 | |

Order: 1, 2, 3 (3 conflicts in 9 files with the first two, section 2C; `feat/fa-locale` is the reference resolution). Each merges alone on upstream main. Other languages: RTL none; locale «فارسی» in the language list; dates/numbers the decimal comma in French, Russian and Vietnamese, and amounts that follow the interface language instead of the browser language (section 1). `coverage.md` and the three draft texts describe the session 13 heads: the locale text still names `docs/i18n/fa.md` (Change 3 and the Files table), and the dates/numbers text does not list the files the session 14 sweep added (section 7.8).

## 6. Verification (session 14)

Session 13's verification (all three PR heads, the app and the English pixel comparison) is in `git show 94dc580:.fa-review/REPORT.md`, section 6; its screenshots are listed below. Session 14 changed formatting, tests, the guide's location and one dev dependency, and did not run the app. Bun 1.3.14, Node 22.22.0, `bun install --frozen-lockfile`, then from `web/`; lint compared by file, rule and severity with `upstream/main` (`1a4166d`, 165 errors, 65 warnings, own worktree), from `oxlint -f json`.

| Check | `review/fa-locale` (`4285a18`) | `review/fa-dates-numbers` (`3a23fa1`) | `feat/fa-locale` (`ebab12b`) |
| --- | --- | --- | --- |
| `bun run typecheck` | exit 0 | exit 0 | exit 0 |
| `bun run lint` errors (upstream 165) | 163; 0 pairs above upstream; 0 in changed files | 165; 0; 1 in a changed file, upstream's own (`wallet/components/creem-products-section.tsx`, `no-array-index-key`), unchanged | 119; 0; 0 |
| `bun run test` | 188 files, 2284 passed | 206 files, 2343 passed | 253 files, 2565 passed |
| `bun run build` | exit 0, 66890.1 kB / 20610.3 kB gzip | exit 0, 66330.0 kB / 20385.8 kB gzip | exit 0, 67002.6 kB / 20638.3 kB gzip |
| `bun run i18n:sync` | no changes; fa missing 1056, extras 0; others 0/0 | no changes | no changes; fa missing 1056, extras 0; others 0/0 |
| `bun run i18n:check-fa` | `5808 keys, no findings` | (no `fa.json`) | `5817 keys, no findings` |
| changed tests with a French default (`LANG=LC_ALL=fr_FR.UTF-8`) | 9 files, 56 passed | 11 files, 71 passed | 19 files, 128 passed |
| whole suite with a French default, against `upstream/main` under fr (75 failures of its own) | 75 failed, the same 75 as upstream; none beyond them | 45 failed, all among upstream's 75; none beyond them (30 of upstream's pass: amounts now follow the interface language) | 45 failed, all among upstream's 75; none beyond them |

How the French default was forced: `LANG` and `LC_ALL` set to `fr_FR.UTF-8` for `node` and `vitest`. `node -e "new Intl.NumberFormat().resolvedOptions().locale"` printed `fr-FR`, and a temporary probe test run in the same vitest invocation as the changed tests (not committed) printed `DEFAULT-LOCALE fr-FR` and asserted it. The same variables with `de_DE` and `ar_EG` were used in the sweep.

**Fail-before** (each new or changed test, run with its fix reverted):

| Test | Without the fix | With the fix |
| --- | --- | --- |
| Step 3 sample of the skill (extracted, run on the upstream tree, which has no `fa.json`) | `Error: ENOENT … fa.json`, exit 1 | exit 0; a missing `fr.json` still throws |
| `usage-logs/lib/__tests__/audit-content-locale` | fr: 1 failed (`1 $ · 0 $ → 1 $`) | passes with the default C, fr, de, fa_IR |
| `usage-logs/lib/__tests__/quota-audit-direction`, `users/components/__tests__/quota-preview-direction` | fr: 2 failed; ar-EG: 3 failed | pass with C, fr, de, ar-EG |
| `persian-monospace` (`@tailwindcss/node`) | passed before too (the package was hoisted from `@rsbuild/plugin-tailwindcss`); the finding is about the declaration | lockfile check above |
| helper return types | type-only | `tsgo -b` exit 0 |
| `subscriptions/.../purchase-price-locale` (quota amounts) | default: Persian case failed; fr: English and Persian failed | 4 passed with C, fr, ar-EG |
| `data-table/core/__tests__/pagination` (row total) | default: 2 Persian cases failed; fr: 4 failed | 9 passed |
| `redemption-codes/.../quota-locale` (new) | Persian case failed | 2 passed |
| `subscriptions/.../user-subscriptions-quota-locale` (new) | Persian case failed | 2 passed |
| `channels/.../balance-locale` (new) | 3 Persian cases failed (badges, notice, dialog); with only the notice's `locale` removed, the notice case fails alone | 6 passed |
| `usage-logs/.../detail-segments-locale` (new) | "resolves to English" case failed (details `‎$۰٫۲۵` beside a `$0.01` cost) | 2 passed with C, fr, ar-EG |
| `usage-logs/lib/__tests__/quota-audit-number-locale` | fr, ar-EG: English case failed | passes with C, fr, de, ar-EG |
| `lib/__tests__/format-fixed-locale`, `lib/__tests__/percent-sign-locale`, `channels/lib/__tests__/response-time-format` (unknown language) | ar-EG: 4 failed (fr, de: 2) | 43 passed with C, fr, de, ar-EG |
| `dashboard/lib/flow.test.ts` › tooltips | fr, de: failed (`100,0%`); passes on `upstream/main` under fr | passes with C, fr, de (ar-EG: 5 other cases fail, as on `upstream/main`) |

### Running app and screenshots (session 13)

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

7. **Session 13.** (a) `pr/fa-dates-numbers` needs one rebase once the first two branches land (9 files; still the same 9 with the session 14 commits, section 2C). (b) The audit "Expiration" field and the access token list now use Solar Hijri in Persian; the access token sessions list («آخرین فعالیت: ۷ روز دیگر» in the screenshots) only reflects the fixed browser clock. (c) Not offered upstream: `fix/legal-consent-sentence` (PR #5998 overlap) and `fix/quota-insufficient-i18n` (backend, separate). (d) PR template fields left for the submitter: tool version and model id.

8. **Session 14.** (a) The open PR descriptions (QuantumNous/new-api#7652 and QuantumNous/new-api#7653) and the drafts in this folder still describe the session 13 heads: the locale one names `docs/i18n/fa.md`, the dates/numbers one lacks the sweep files (`pagination.tsx`, `redemptions-columns.tsx`, `user-subscriptions-dialog.tsx`, `channels-columns.tsx`, `balance-query-dialog.tsx`, `flow.test.ts` and four new test files) and both quote session 13's test counts; update them when the PR branches are fast-forwarded. Replies to CodeRabbit are drafted in `coderabbit-replies.md`, not posted. (b) Formatters without the interface locale in files the dates/numbers branch does not touch, left for a follow-up: `formatQuota` in `redemption-codes/components/redemptions-mobile-list.tsx:167` (the mobile twin of the fixed column), `redemptions-mutate-drawer.tsx:233` (default name), `system-settings/general/quota-settings-section.tsx:79`, `users/components/user-quota-dialog.tsx:61-66` (preview), `users/components/users-columns.tsx:261`, `users/components/users-mutate-drawer.tsx:473`. (c) In files it touches, not changed: `formatTaskUsageUnitPrice` (`pricing/lib/dynamic-price.ts`, no locale parameter, 13 callers; the usage log details task prices follow the browser), the playground message time (`Intl.DateTimeFormat(undefined, …)`), the admin plan picker price (`toFixed(2)`), the non-Persian `toLocaleString()` dates kept as upstream, and the unused `formatBalance`/`formatQuota` exports in `channels/lib/channel-utils.ts`. (d) Upstream's own tests depend on the machine's locale: 75 fail on `upstream/main` with a French default; the branches leave those as they are.

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
- Session 13: merge of upstream `1a4166d` (#7628 replaces six of our fixes; access tokens rebuilt upstream), 97 new keys in Persian, and the three upstream branches `pr/fa-rtl-layout`, `pr/fa-locale`, `pr/fa-dates-numbers` with coverage, fail-before and the English pixel comparison.
- Session 14 (this one): CodeRabbit's findings on QuantumNous/new-api#7652 and QuantumNous/new-api#7653 fixed on `review/fa-locale` and `review/fa-dates-numbers` (guide moved out of `docs/`, skill Step 3 guard, `@tailwindcss/node`, return types, purchase dialog and swept formatters in the interface locale, resolved language in the details cell), tests made independent of the machine's locale (checked under fr, de, ar-EG), both merged here; replies drafted in `coderabbit-replies.md`.
