# Persian (fa) locale: session 8 (glossary, batch 8: system settings, batch 9: shared components, RTL fixes)

Temporary review folder. Delete `.fa-review/` before any upstream PR.

Branch: `feat/fa-locale` only (no new upstream branch this session). `upstream/main` was still at `c2b7a9a` (fetched at the start), so no merge was needed. Session 7 is described in the previous version of this file (`git show 839424f:.fa-review/REPORT.md`).

## Decisions applied (session 7's open questions)

| # | Decision | Where it landed |
| --- | --- | --- |
| 1 | Channel and model terms go into the glossary | A (`4d901bd`), plus search and template (`d1e501d`) |
| 2 | Channel-page code bugs: no upstream branch now, keep them listed | Remaining issues 1 (unchanged) |
| 3 | Next batch: system settings, then shared components | B, C |

## Commits (from `839424f`)

```
4d901bd docs(i18n): add the channel, routing and model terms to the Persian glossary
17d9c1b feat(i18n): translate the system settings general in Persian (146 keys)
58de3e3 feat(i18n): translate the system settings auth in Persian (193 keys)
f7ad4f7 feat(i18n): translate the system settings content in Persian (113 keys)
4ba8a4c feat(i18n): translate the system settings maintenance in Persian (117 keys)
3b076b1 feat(i18n): translate the system settings request-limits in Persian (63 keys)
25399dc feat(i18n): translate the system settings request-policies in Persian (103 keys)
86a8c7c feat(i18n): translate the system settings integrations in Persian (266 keys)
97b1b72 feat(i18n): translate the system settings models in Persian (324 keys)
bcc8ea7 feat(i18n): translate the shared system settings components in Persian (10 keys)
0d40127 fix(web): keep the JSON code editor left to right in RTL
f40c978 fix(web): align table headers to the inline start in RTL
10cd251 fix(web): keep inline code, keys and sample output left to right in RTL
53458ed fix(web): right-to-left layout on the system settings pages
3ca2feb feat(i18n): translate the shared components in Persian (59 keys)
d1e501d docs(i18n): settle the Persian spelling of search and the term for template
9f8f102 fix(web): isolate code with plaintext instead of forcing direction ltr
0097c77 feat(i18n): translate the remaining system settings keys in Persian (169 keys)
c2798f3 feat(i18n): translate five more shared component keys in Persian
4c66300 feat(i18n): translate the system update check in Persian (24 keys)
7b04478 fix(web): keep the checked switch thumb inside its track in RTL
daa0076, aaa3325, f0b4d5d, (this hand-off): .fa-review scripts, screenshots, report
```

## Key count

| | Keys |
| --- | --- |
| `fa.json` at the start (`839424f`) | 3530 |
| Batch 8, system settings (new) | **1527**: general 152, auth 193, content 134, maintenance 137, request-limits 63, request-policies 104, integrations 266, models 444, shared settings code 10, plus the update checker (`features/system-update`, shown on the maintenance page) 24 |
| Batch 9, shared components (new) | **64** |
| Existing values changed | 12: 5 glossary alignments (A), `Prefix cache hit rate` (cache «کش»), 6 spellings of search («جستجو» → «جست‌وجو») |
| `fa.json` now | **5121** (`check-fa: 5121 keys, no findings`); `i18n:sync` reports 1674 keys still falling back to English |

## A. Glossary

`docs/i18n/fa.md` gets 27 rows in its existing table format: base URL, tag (and HTML tag «تگ»), priority and weight, mapping, model redirect, override «بازنویسی», pass-through, forward / native forwarding / forwarding route, route and routing, fallback route, endpoint (with its plural), source and target, converter, regex «عبارت باقاعده», catch-all «فراگیر», connection shards «بخش‌های اتصال», multi-key, polling (both senses), credential, metadata, container and replica, snapshot «نمای لحظه‌ای», reset credit, prompt, cache «کش», search «جست‌وجو», template «الگو».

Consistency check of `fa.json` against each term, through the script: `Override` and `System Prompt Override` («جایگزینی» → «بازنویسی»), the inference-status snapshot notice («نمونه» → «نمای لحظه‌ای»), the native forwarding route note («مسیرهای هدایت بومی» → «مسیرهای ارسال بدون تبدیل»), the task usage metadata notice, `Prefix cache hit rate` («حافظۀ نهان» → «کش») and six search strings. Different senses kept apart on purpose: redirecting a person («انتقال»), polling an async task («بررسی دوره‌ای»), a person's sign-in credentials («اطلاعات کاربری»), an HTML tag («تگ»).

## B. Batch 8: system settings

- Scope: every `en.json` key that appears in `web/src/features/system-settings` (t() calls, option lists, `labelKey`/`titleKey`/`descriptionKey` constants, zod messages rendered through `FormMessage`, section titles), plus three billing-expression messages from `features/pricing/lib` shown on these pages and the update checker. Worked subfolder by subfolder in the order asked. 61 keys are left out on purpose: brand and protocol names used as tab labels (GitHub, Discord, OIDC, Telegram, LinuxDO, WeChat, Stripe, Epay, Claude, Gemini, Grok, Uptime Kuma, SSL/TLS, STARTTLS), example values and placeholders (URLs, hosts, IP lists, `80,443,8080`, `price_xxx`, `vip`, `gpt-4`, the JSON examples), identifiers that only match a key by accident (`default`, `field`, `token`, `pending`, `running`, `succeeded`, `Local`) and `e.g. 401, 403, 429, 500-599` (the parser wants Latin commas).
- Method: chunks drafted by helpers from their call sites against `docs/i18n/fa.md`, each run through a checker (the `check-fa` rules plus placeholder and markup equality); I reviewed every value, aligned terms across chunks and wrote them through the skill's `add-missing-keys.mjs` (created, run, deleted per commit; `newKeys.fa` read from a JSON file), then `bun run i18n:sync` and `bun run i18n:check-fa`.
- Scanner fix: the first key scan matched quotes with a regular expression and lost track after an apostrophe in a comment or JSX text, which hid 169 settings keys (`0097c77`). The rescan uses `@babel/parser` (string literals, template literals without substitutions, JSX text). Session 7's channel and model scan used the same regular expression; the parser finds 70 untranslated keys in `features/channels` (most are the brand labels skipped on purpose; not re-checked this session).
- The payment compliance confirmation is typed by the admin in four parts. The Persian parts avoid ZWNJ and ۀ so they can be typed on a standard Persian keyboard, and join into the full sentence: «تذکر انطباق بالا را خواندم و فهمیدم، خطرهای حقوقی مربوط را قبول دارم و مسئولیت حقوقی ناشی از استقرار، اجرا و دریافت هزینه را بر عهده دارم.»
- Fragments: `Domain`/`IP` + `Whitelist`/`Blacklist` are joined in code; the only call site reads «دامنه‌ها: فهرست مجاز». The low-disk warning is joined from four keys with a hard-coded `MB` (below).

### RTL fixes (all no-ops in left-to-right languages)

| Commit | Fix | Test (before → after) |
| --- | --- | --- |
| `0d40127` | Shared `JsonCodeEditor`: the code area sets `dir=ltr` (JSON was right-aligned, brackets mirrored, line numbers on the wrong side, on every JSON setting and in the channel editor); toolbar icon margins `me-1` | `components/json-code-editor/__tests__/json-code-editor-direction.test.tsx`: 1 failed / 1 passed → 2 passed |
| `f40c978` | Shared `TableHead`: `text-left` → `text-start`, checkbox padding `pr-0` → `pe-0` (also `TableCell`) | `components/ui/__tests__/rtl-layout.test.tsx` new case: failed → passed |
| `10cd251` + `9f8f102` | Stylesheet: `[dir='rtl'] :is(code, kbd, samp):not([dir]) { unicode-bidi: plaintext }`, so paths, regexes and commands in settings descriptions and tables keep their punctuation. The first version set `direction: ltr`, which flipped the `end-*` inset of the search box key cap (it covered the placeholder); `plaintext` leaves `direction` alone | `styles/__tests__/code-direction.test.ts` (5 cases, compiles the stylesheet with Tailwind): old rule 3 failed / 2 passed → 5 passed |
| `53458ed` | 36 settings files: physical classes → logical (`me-*`/`ms-*`/`ps-*`/`pe-*`, `start-*`, `border-s`, `text-start`/`text-end`): icons in buttons, search icons inside inputs (they sat under the text), list padding, table action columns, the decision-record timeline. 79 inputs and text areas holding URLs, domains, IP/CIDR lists, ports, paths, client IDs, OAuth field paths, regexes and pricing expressions get `dir=ltr`, as do the log and cache directory paths. The rule editor's `▶` collapse arrow is mirrored with `rtl:-scale-x-100` and hidden from the button name | `general/channel-affinity/__tests__/advanced-toggle-direction.test.tsx`: 2 failed → 2 passed |
| `7b04478` | Shared `Switch`: checked thumb used `translate-x-[calc(100%-2px)]` only, which pushed it out of the track in RTL (every checked switch showed as an empty pill); `rtl:-translate-x-[…]` added for both sizes. Checked in Chromium: every thumb inside its track, 1px from the inline end, both directions | `rtl-layout.test.tsx` 2 new cases: failed → passed |

Existing components reused: the fixes change the shared `JsonCodeEditor`, `Table`, `Switch` and the stylesheet in place; no new components.

## C. Batch 9: shared components

64 keys in `web/src/components` (59 in `3ca2feb`, 5 found by the parser rescan in `c2798f3`): the model edit dialog's icon field (`lobe-icon-field.tsx`: «استفاده از آیکون سازنده»، «آیکون سفارشی مدل»، «آیکون نهایی»، «برگرفته از …»), model and group selectors, multi-select, JSON editors, date pickers, floating window, coming-soon page, AI chat elements, sidebar and footnote labels. 18 left out: identifiers (`default`, `field`, `log`, `pending`, `s`, `x`, `...`), footer link names, the product name, the `K` key cap, `JSON`, and `sources` (below).

## Hard-coded or unkeyed English on these pages (not fixed here; each needs a new key in every locale)

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

## Verification (`feat/fa-locale`, from `web/`, after `7b04478`)

| Command | Result |
| --- | --- |
| `git remote -v`, `git push --dry-run origin HEAD`, dry run to a new branch | origin `https://github.com/yaser-k/new-api-fork`; both accepted after fast-forwarding `d263b5f` → `839424f` |
| `git fetch upstream main` | `upstream/main` = `c2b7a9a`; upstream push URL `DISABLED` |
| `bun install` | 1206 packages |
| `bun run typecheck` | `tsgo -b`, exit 0 |
| `bun run lint` | exit 1: **150 errors, 65 warnings**; `upstream/main` (own worktree and install): **182 errors, 66 warnings**; errors only on this branch (file + rule): **none**; errors in the 241 files changed against upstream: **0** |
| `bun run test` | `Test Files 208 passed (208)`, `Tests 2349 passed (2349)` |
| `bun run build` | exit 0, total 66850.5 kB / 20598.6 kB gzip |
| `bun run i18n:sync` | exit 0; fa partial, missing 1674, extras 0; seven required locales missing 0, extras 0 |
| `bun run i18n:check-fa` | `check-fa: 5121 keys, no findings` |
| `git diff --stat upstream/main -- web/src/i18n/locales/` | `fa.json` +5125; en, fr, ja, ru, vi, zh-TW, zh +17 each (the 9 + 8 keys from the merged fix branches); those seven files are unchanged since `839424f` |
| `go build -o <scratch>/bin/new-api .` (Go 1.25.1) | exit 0, embeds the fresh `web/dist` |

### Running app

```
SQLITE_PATH=<scratch>/run/one-api.db GLOBAL_API_RATE_LIMIT_ENABLE=false GLOBAL_WEB_RATE_LIMIT_ENABLE=false \
  CRITICAL_RATE_LIMIT_ENABLE=false SEARCH_RATE_LIMIT_ENABLE=false TZ=UTC <scratch>/bin/new-api --port 3300
python3 .fa-review/scripts/seed.py  http://127.0.0.1:3300 <db> en   # /api/setup creates the local admin; demo data
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 <db>
python3 .fa-review/scripts/seed7.py http://127.0.0.1:3300
python3 .fa-review/scripts/seed8.py <db>                            # demo response times for the three channels
python3 .fa-review/scripts/seed6.py http://127.0.0.1:3300 lang fa
node .fa-review/scripts/shots8.mjs http://127.0.0.1:3300 .fa-review fa
```

Playwright 1.56.1 (global), Chromium 141.0.7390.37 headless, 1440×900, light theme, UTC, `dir=rtl lang=fa`. 44 captures, none blank (fewest colours 1170; blank threshold 16). Console errors: the 401 from the pre-login session probe and `ERR_CERT_AUTHORITY_INVALID` for an external resource blocked by the sandbox proxy.

- `101`–`140-settings-<group>-<section>-rtl.png`: every system settings section (site 4, billing 6, auth 5, content 7, operations 7, security 3, request policies 3, models 5), full page. The script also lists visible text without Persian letters; what remains is brand names, model names, example values, code, the demo legal texts and the hard-coded strings above.
- `150-settings-custom-oauth-dialog-rtl.png`: the add custom OAuth provider dialog.
- `151-model-edit-icon-field-rtl.png`: model edit dialog with the translated icon field.
- `152-channels-table-rtl.png`, `153-channels-table-response-time-rtl.png`: channels table view, scrolled to the response-time column.

Response-time column (table view): 106px wide in Persian (110 set). `۴۵۶ میلی‌ثانیه` needs a 79px label and gets 78, so three-digit millisecond values show «۴۵۶ میلی‌ثا…» (the badge is capped at the cell width); seconds (`1.23 ثانیه`, 67px) fit. English `456ms` is 54px. Not changed: see open question 1.

## Upstream split plan (proposal only; nothing built)

Sizes and grouping as in session 7, with this session's commits added. Commits that mix concerns (marked *) need their hunks split when the PR branch is built.

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
| 1 | `fix(web): right-to-left layout in shared components` | `c78c83f`* (without the `formatQuota` locale hunks; legal-consent hunk superseded), `d8ee064`, `cb25020`, `8ccec81`, `936ab28`, `129d598`, the RTL half of `f01c5af`*, the sidebar side of `b2720c9`*, `666c43c` (header positions, only where PR 1 touches the file), **and from this session `0d40127`, `f40c978`, `10cd251` + `9f8f102` (squash), `53458ed`, `7b04478`** | rtl-layout (now also table header and switch), code-block-direction, column-pinning, column-resize-direction, app-sidebar-direction, date-range-direction, terminal-direction, profile-header-direction, value-direction, timing-direction, spinner-direction, **json-code-editor-direction, code-direction, advanced-toggle-direction** | No visible change in LTR. The stylesheet rule only matches under `[dir=rtl]` |
| 2 | `feat(i18n): Persian (fa) partial locale, tooling and docs` + first batch | `b2720c9`* (without the sidebar), `07e070d`, `759a31e`, `f5296da` + `776c03c`, `bfe01bb`, the script part of `fa96f8b`; AGENTS.md, web/AGENTS.md, the i18n skill; `docs/i18n/fa.md` (**with the glossary rows from `4d901bd` and `d1e501d`**) | languages, direction-provider, check-fa, persian-monospace, intl-locale lint case | Language switcher lists «فارسی»; `docs/i18n/fa.md` is a new file under `docs/` (ask the maintainers) |
| 3 | `feat(web): Solar Hijri dates, chart axes and date pickers in Persian` | `0ea975e`, `c2369c8` + `42a98a2`, `170ba72`, `9d22fac` | display-date-locale, activity-time-cell-dates, login-session-dates, date-picker-display, calendar-persian, chart-time-locale | None |
| 4 | `feat(web): Persian labels for audit roles and sign-in methods` | net of `f767402` + `9cd40a8` | audit-content-locale, details-locale | None (could fold into 3) |
| 5 | `fix(web): format money and numbers in the interface language` | locale half of `f01c5af`*, `formatQuota` hunks of `c78c83f`*, the `locale` argument in `recharge-form-card.tsx` | format-quota-locale, format-currency-locale, amount-locale, summary-cards-locale, profile-header-locale | **Yes** (its own PR, decision 1 of session 6) |
| 6 | `feat(i18n): Persian translation batches` | `fa96f8b`, `551a8a3`, `795d296`, `4db67e2`, `3825299`, `aaba1b0`, `8a76a75`, `ada5af2`, `c6aaad5`, **`4d901bd` (fa.json part), `17d9c1b`, `58de3e3`, `f7ad4f7`, `4ba8a4c`, `3b076b1`, `25399dc`, `86a8c7c`, `97b1b72`, `bcc8ea7`, `3ca2feb`, `d1e501d` (fa.json part), `0097c77`, `c2798f3`, `4c66300`** (fa.json, plus the Persian cases in `legal-consent.test.tsx` and `terms-footer-persian.test.tsx`) | `bun run i18n:check-fa` | None: only `fa.json` |

Leave out of every PR: `.fa-review/`, all hand-off and script commits, the merge commits. Order: the upstream fix branches that are offered (not `fix/legal-consent-sentence` for now), then 1, 2, 3 (+4), 6; 5 when upstream agrees to the behaviour change. `4d901bd` and `d1e501d` mix `docs/i18n/fa.md` (PR 2) and `fa.json` (PR 6).

## Remaining issues

1. **Code bugs found while translating (English affected too), kept listed per decision 2:**
   - Channels (session 7): `Set a tag for` / `Are you sure you want to delete` + count with no space ("for5"); `Update balance for:`, `Edit Tag:`, `Create a copy of:`, `Edit all channels with tag:` glued to the name; `more mapping` + English `s`; `h`/`m`/`s` glued to numbers; `{{field}} updated to {{value}}` gets the raw English field name (Priority, Weight); `Concatenate channel system prompt with user&apos;s prompt` shows a literal `&apos;`.
   - New in settings: `UI granularity only &mdash; …` shows a literal `&mdash;` (content/dashboard-section.tsx:184); the placeholders `example.com&#10;company.com` (auth/basic-auth-section.tsx:252) and the two SSRF list placeholders (request-limits/ssrf-section.tsx:315,380) show a literal `&#10;`; `{{count}} override` (models/group-ratio-visual-editor.tsx:968), `${tierCount} ${t('tiers')}` (models/model-pricing-snapshots.ts:106) and `{n} {t('rules')}` (models/group-special-usable-editor.tsx:212) print plurals for one; the Gemini `-thinking-{{budget}}` sentence is split over three keys with a backtick pair across two, so `{{budget}}` shows literally (models/gemini-settings-card.tsx:340); the compliance separators are Chinese punctuation keys (`，`, `，and `, `、`) joined without spaces, so English shows "reminder，acknowledge" (integrations/payment-settings-section.tsx:310-322); the OAuth callback URLs use `t('Site URL')` as the base when the server address is empty, so the copied URL contains a translated label (auth/oauth-section.tsx:249-258).
   - Shared components: `{t('Used')} {count} {t('sources')}` (components/ai-elements/sources.tsx:60) cannot be ordered in Persian; left English.
2. **Dates not in the fork's pattern on these pages:** the compliance "Confirmed at" time uses `new Date(…).toLocaleString()` (integrations/payment-settings-section.tsx:840), and the announcements table uses `dayjs().format('YYYY-MM-DD HH:mm:ss')` (content/announcements-section.tsx:394) and its own relative time. Changing them to `formatDisplayDate`/`formatFromNow` with the interface locale also changes the English output, so they are listed, not fixed.
3. **Response-time badge** truncates three-digit values in Persian (above; open question 1). The same column's badge still has a physical `-ml-1.5` (features/channels/components/channels-columns.tsx:1205; also at 809, 960, 970, 1155), outside this batch's pages.
4. **Channels scan**: session 7 used the same quote-matching scan; the parser now lists 70 untranslated keys in `features/channels` (mostly the deliberate brand skips). Remaining untranslated by folder: task-plugins 213, channels 70, usage-logs 66, system-settings 61 (deliberate skips), setup 57, playground 54, dashboard 54, system-info 45, lib 34, rankings 31, pricing 21, keys 20, components 18 (skips), routes 16, home 16, auth 12, others ≤ 9 each.
5. Session 6 items still open: dashboard chart header total, subscriptions list digits and `-ml-1.5`, audit details `ID`, native date inputs, chart time axis direction, backend content in Chinese or English, `{{action}}` raw in two audit strings, carousel arrows, other `<pre>` blocks, the Public Sans font name mismatch. Model edit dialog: the sidebar toggle label `Toggle Sidebar` is not an i18n key.

## New open questions

1. **Response-time badge**: three-digit values truncate by one pixel in Persian. Options: (a) leave `{{value}}ms` out of `fa.json`, so Persian shows `456ms` next to «۱٫۲۳ ثانیه»; (b) widen the column for every language upstream (changes the LTR layout); (c) keep the full word and accept the ellipsis. Which one?
2. **Next batch**: task plugins (213 keys), or the remaining channel keys found by the parser (70, mostly brand labels to confirm), usage logs (66), setup (57), playground and dashboard?
3. **The date items in remaining issue 2**: fix them in the fork with the shared helpers, accepting the English format change (Gregorian `YYYY-MM-DD HH:mm:ss` instead of the browser's `toLocaleString`), or keep them listed for a later upstream PR?
