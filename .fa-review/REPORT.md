# Persian (fa) locale for new-api: state and hand-off

Temporary review folder. Delete `.fa-review/` before any upstream PR.

Fork `yaser-k/new-api-fork`, branch `feat/fa-locale`, based on `upstream/main` at `c2b7a9a` (upstream had not moved in session 12; no upstream merge). Upstream issue: QuantumNous/new-api#7198. This file is written so it can serve as the summary for the maintainers; the session details, verification and open items follow the summary.

## 1. What the Persian locale covers

| | |
| --- | --- |
| Keys | **5733** of the 6795 keys in `en.json` are translated in `fa.json`. `fa` is a partial locale: a key that is missing from `fa.json` shows its English text (per key, at run time). |
| Keys still English | 1062: **161** that the code uses (brand and provider names, URLs, example values, identifiers, unit letters glued to numbers, fragments that cannot be ordered in Persian; full list with reasons in section 4) and **901** that no code in `web/src` uses (section 4) |
| Pages | Every page and shared component under `web/src`: auth and sign-up, home, pricing and model details, rankings, dashboard (overview, models, flow, users), API keys, playground and chat, usage, task and drawing logs, audit log, wallet and subscriptions, profile and security, channels, models and deployments, users, redemption codes, subscriptions admin, system info, task plugins, the setup wizard, every system settings section, the legal pages, error pages and the about page |
| Dates | Displayed dates are Solar Hijri in numeric year/month/day order with 24-hour time (`۱۴۰۵/۰۷/۰۳ ۱۵:۰۵:۰۹`); relative times are Persian («۳ دقیقه پیش»). Table and log cells carry the Gregorian date and time as their `title`. Values sent to the API, filters, inputs, exports and copied text stay Gregorian. Dashboard chart axes are Solar Hijri and stay in time order across Nowruz |
| Date pickers | `calendar.tsx` (used by the date and date-time pickers) shows the Solar Hijri calendar through `@daypicker/persian`, loaded lazily only for Persian; the returned `Date` values are unchanged |
| Numbers | Formatted with the interface locale: Persian digits (`۱۲۳`, `۱٫۲۳`), switchable to Latin digits with one constant (`PERSIAN_INTL_LOCALE = 'fa-u-nu-latn'`) |
| Right to left | Selecting Persian sets `dir="rtl"` and `lang="fa"`; a direction provider feeds Base UI and the calendar. Components use logical classes (`ms-*`, `pe-*`, `start-*`, `text-end`), direction-dependent icons are mirrored, code, keys, sample output and `pre` blocks keep their own text direction (`unicode-bidi: plaintext`), left-to-right inputs (URLs, IDs, code) keep `dir="ltr"`, before-and-after arrows and the carousel follow the page direction, and Persian text uses Vazirmatn (also inside monospace text). The config drawer can still force LTR or RTL. Since session 12 no physical direction class is left under `web/src` except 62 kept on purpose, and a test (`styles/physical-direction-classes`) fails on any new one that its allowlist does not list with a reason |

### Rules and tooling

- `docs/i18n/fa.md` (new file): locale status, date rules, a glossary of 93 terms with reasons, the terms that stay English, eleven typography rules (ZWNJ, Persian ی and ک, digits, ۀ, punctuation, spacing around Latin words, FSI … PDI isolates around interpolated left-to-right values, RLI … PDI around Persian placeholders of left-to-right inputs), style and the automated check.
- `bun run i18n:check-fa` (`web/scripts/check-fa.mjs`): exits 1 on any finding. It checks the typography rules, isolate pairing, empty values, stray whitespace, keys missing from `en.json`, and parses `web/src` with `@babel/parser` (now a declared dev dependency, 7.29.7, single copy in `bun.lock`) to check the placeholders of `dir='ltr'` inputs. Today: `check-fa: 5733 keys, no findings`.
- `bun run i18n:sync` reports Persian as partial and never fills it with English. The i18n skill, `AGENTS.md` and `web/AGENTS.md` describe `fa` as the optional partial eighth locale.

### What differs from upstream for other languages

1. **Number formatting (split-plan PR 5).** Money, quotas and the numbers fixed so far follow the interface language instead of the browser language or a raw value. For English the output is identical (tested for each change). French, Russian and Vietnamese now show a decimal comma where the interface formats decimals (for example the response-time badge `1,23 s`, chart totals `$22,98`, the subscriptions list price and, since session 11, the plan price on the purchase dialog and wallet plan cards); Chinese and Japanese keep the decimal point. Grouping follows the interface language where it already applied (flow numbers, system info bytes). The compact table page counter and the percent sign (session 11) change only Persian: the counter has no grouping and the sign is `%` in every other language.
2. **The 17 keys from the upstream fix branches** merged into `feat/fa-locale` (9 from `fix/legal-consent-sentence`, 8 from `fix/ui-label-strings`), present in all seven required locales.
3. **Public Sans (branch `fix/public-sans-font`, merged in session 11).** `--font-sans` now names the face that `@fontsource-variable/public-sans` registers (`'Public Sans Variable', 'Public Sans', sans-serif`), so left-to-right languages render in Public Sans instead of the browser's generic sans-serif (Chromium: sample line 820.06 px → 854.92 px; one 26 832-byte Latin woff2 now downloaded). Persian is unchanged: `html:lang(fa)` still sets Vazirmatn, and no Public Sans face loads on a Persian page (checked on the running app).

Nothing else changes for left-to-right languages: every RTL fix uses logical classes or RTL-only rules, and every Persian-only label (audit roles, sign-in methods, the audit ID label and user action names) is guarded by the locale; tests pin the English (and where relevant Chinese) output. Among the locale files only `fa.json` differs from upstream apart from those 17 keys (checked: each of en, fr, ja, ru, vi, zh, zh-TW has 17 added, 0 removed, 0 changed keys).

## 2. This session (session 12): right-to-left layout sweep

Session 11's details (its decisions, the Public Sans branch, the leftovers C1 to C4) are in `git show 7f9b789:.fa-review/REPORT.md`, section 2. This session changes layout only: no locale file changes (all eight identical to `7f9b789`), no new upstream branch.

### A. Dashboard overview: the setup guide (`6dafe80`)

| Part | Before (RTL) | Fix | Measured in Persian, 1440 px (before → after) |
| --- | --- | --- | --- |
| Quick action cards («سکو را آماده نگه دارید») | `QuickActionItem` used `text-left`: title and description sat at the far (left) side, away from their icon | `text-start` | title text x 56–129 → 227–300, icon at 306–342 |
| Setup steps | `StartStepItem` used `text-left` | `text-start` | text now starts next to the step icon |
| Timeline line | `left-4`: ran under the arrows on the left | `start-4` | line x 794 → 1169; step circle 1154–1186 (centre 1170) |
| Decorative backdrop | code texture at `right-0`/`right-3`, `text-right`: it ran behind the heading | mirrored (see decision) | texture 696–1203 → 405–912; heading at 813–1195 |

Decision on the backdrop: **mirror it.** It is composed around the heading: the code texture and the glow sit on the far side from the heading and fade in toward it. Left physical, in RTL the texture lies behind the heading and the description and the glow's bright side sits under the text. The texture wrapper and `pre` use `end-0`/`end-3`; the glow position, the sweep angle and the two fade directions are CSS variables with `rtl:` values (`--setup-glow-x` 78% / 22%, `--setup-sweep` 112deg / 248deg, `--setup-fade` 90deg / 270deg). The code lines keep their own left-to-right direction (the global `unicode-bidi: plaintext` rule for `pre`), so `text-end` alone would stay right-aligned in RTL; `rtl:text-left` keeps them flush with the card edge, as `text-right` does in LTR (checked in Chromium: `text-align` left in RTL, right in LTR). In LTR every computed value is the same as before (78%, 112deg, 90deg, right, right-aligned).

Test: `dashboard/overview/setup-guide` + 3 cases (quick action `text-start`, step link `text-start` and connector `start-4`, backdrop `end-3`/`end-0`/`rtl:text-left` and the glow variables): 3 failed / 6 passed before, 9 passed after.

### B. Sweep of the physical direction classes

Scanner: the guard's own scan (section C) over `web/src` without tests. At `7f9b789`: **476 physical classes in 162 files** (the estimate of about 420 in 147 files counted fewer families). Now: **414 converted, 62 kept in 22 files**, each kept class listed with its reason in `web/src/styles/__tests__/physical-direction-allowlist.json` (table below).

| Folder | Before | Converted | Kept |
| --- | ---: | ---: | ---: |
| components/ui | 161 | 128 | 33 |
| components (other) | 83 | 60 | 23 |
| features/channels | 81 | 81 | 0 |
| features/models | 31 | 31 | 0 |
| features/pricing | 27 | 27 | 0 |
| features/usage-logs | 22 | 22 | 0 |
| features/dashboard | 15 | 15 | 0 |
| features/home | 10 | 4 | 6 |
| features/wallet | 8 | 8 | 0 |
| features/security | 8 | 8 | 0 |
| features/keys | 6 | 6 | 0 |
| features/redemption-codes | 6 | 6 | 0 |
| features/subscriptions | 6 | 6 | 0 |
| features/profile | 5 | 5 | 0 |
| features/users | 4 | 4 | 0 |
| features/system-settings | 3 | 3 | 0 |
| **Total** | **476** | **414** | **62** |

How it was done: every reported class was read in context first and classified; the rest were swapped by file, line and token with one mapping (`ml/mr/pl/pr` → `ms/me/ps/pe`, `left/right` → `start/end`, `text-left/right` → `text-start/end`, `border-l/r` → `border-s/e`, `rounded-l/r` → `rounded-s/e`, `rounded-tl/tr/bl/br` → `rounded-ss/se/es/ee`, keeping variants, `-` and `!`), then the diff was reviewed per folder and formatted with `oxfmt` (the class sort order changes with the names). One commit per group: shared ui, other shared components, then each feature folder.

Beyond a plain swap:

- **Response renderer tables** (`3b19d75`): the Markdown parser reports an unaligned column as `left`, so an explicit `:---` cannot be told apart from no alignment; alignment is taken relative to the reading direction (`text-start`, `text-end`). Test `ai-elements/response-table-direction` (2 failed → 2 passed).
- **Home page arrows** (`3f8fd7d`): the forward arrows of the hero and closing call-to-action buttons now turn in RTL (`rtl:rotate-180`; the hover nudge moves toward the inline end with `rtl:group-hover:-translate-x-0.5`). Test `home/arrow-direction` (3 failed → 3 passed).
- **Overrides that only worked in LTR**, found while converting: the channel card's `!ml-0` cancelled the columns' `-ms-1.5` badge offset only in LTR (now `!ms-0`, `87f8da9`); the API key group cell's `ml-0` cancelled `BadgeCell`'s `-ms-1.5` only in LTR (now `ms-0`, `3afbf83`). In RTL the negative margin used to leak.
- **tailwind-merge** does not merge a logical class with a later axis shorthand (`cn('ps-2', 'px-3')` keeps both and `ps-2` wins, where `cn('pl-2', 'px-3')` dropped `pl-2`). Every caller that passes `px-`, `mx-`, `inset-x-` or `border-x` to a component whose base class was converted was checked (single-line and multi-line JSX): the only overlaps are variant-scoped base classes (`has-data-[icon=…]:pe-2` on Button, Toggle, Tabs; the alert action padding), which twMerge never merged anyway. The English pixel comparison (section 6) covers the rendered pages.
- Existing components reused: every fix changes classes on the existing components; no new component.

**Lint cleanup first** (`fa1607a`): six touched files already had lint errors on upstream main (`prompt-input.tsx`, `risk-acknowledgement-dialog.tsx`, `tag-input.tsx`, `models-filter-dialog.tsx`, `deployment-access-guard.tsx`, `model-details-apps.tsx`: prefer-spread, prefer-at, optional catch binding, curly, useless spread, useless fragment, nested ternaries, index keys, catch-or-return). Equivalent forms, no behaviour change (the risk dialog computes the same keys in its memo; the prompt input's submit handler returns its promise chain; the speech results list is read by index because its DOM type is not iterable), as session 10 did in `5a2bd38`. **Not converted**: `web-preview.tsx` (1 × `text-left`): the component is unused, and its file's lint errors are the iframe sandbox (`allow-scripts` together with `allow-same-origin`) and an index key, a security decision rather than a layout change; it is in the allowlist with that reason and in section 7.

Kept on purpose (the allowlist, 62 classes in 22 files):

| File (web/src/) | Count | Classes | Reason |
| --- | ---: | --- | --- |
| components/ai-elements/chain-of-thought.tsx | 1 | `left-1/2` | Centred connector line (left-1/2 with -mx-px); symmetric in both directions. |
| components/ai-elements/code-block.tsx | 1 | `right-2` | Overlay on the dir='ltr' code body; it stays at the end of the left-to-right lines instead of covering their start. |
| components/ai-elements/conversation.tsx | 1 | `left-[50%]` | Centred with translate-x-[-50%]; symmetric in both directions. |
| components/ai-elements/web-preview.tsx | 1 | `text-left` | Not converted yet: the component is unused, and its file carries upstream lint errors (the iframe sandbox allows scripts together with same-origin; an index key) that need a security decision, not a layout change. Convert with that fix. |
| components/config-drawer.tsx | 2 | `left-2.5`, `border-l-[1.5px]` | Radius preview that draws a top-left corner together with the inline borderTopLeftRadius style. |
| components/data-table/toolbar/bulk-actions.tsx | 1 | `left-1/2` | Floating bar centred with -translate-x-1/2; symmetric in both directions. |
| components/floating-window.tsx | 13 | `left-2` ×2, `right-2` ×2, `right-0` ×3, `left-0` ×3, `pr-9`, `pl-4`, `rounded-tl-md` | Window geometry is physical: the resize handles are named by compass side (n, e, se ...) and resized from pointer clientX, so the se grip stays bottom-right; the footer padding (pr-9) and the grip corner (rounded-tl-md) follow that grip. |
| components/json-code-editor.tsx | 1 | `pl-2` | On the dir='ltr' editor element itself; code stays left to right. |
| components/layout/components/glow.tsx | 2 | `left-1/2` ×2 | Centred glows (left-1/2 with -translate-x-1/2); symmetric in both directions. |
| components/ui/alert-dialog.tsx | 1 | `left-1/2` | Dialog centred with -translate-x-1/2; symmetric in both directions. |
| components/ui/carousel.tsx | 2 | `left-1/2` ×2 | Vertical carousel buttons centred with -translate-x-1/2; symmetric in both directions. |
| components/ui/dialog.tsx | 1 | `left-1/2` | Dialog centred with -translate-x-1/2; symmetric in both directions. |
| components/ui/drawer.tsx | 6 | `data-[vaul-drawer-direction=left]:left-0`, `data-[vaul-drawer-direction=left]:rounded-r-xl`, `data-[vaul-drawer-direction=left]:border-r`, `data-[vaul-drawer-direction=right]:right-0`, `data-[vaul-drawer-direction=right]:rounded-l-xl`, `data-[vaul-drawer-direction=right]:border-l` | Keyed to vaul's direction prop, which names a physical side (and drives vaul's physical drag gesture). |
| components/ui/navigation-menu.tsx | 3 | `data-[side=bottom]:before:right-0`, `data-[side=bottom]:before:left-0`, `rounded-tl-sm` | The before: pair spans the full width (symmetric); rounded-tl-sm rounds the upward tip of the square rotated 45 degrees. |
| components/ui/radio-group.tsx | 1 | `left-1/2` | Dot centred with -translate-x-1/2; symmetric in both directions. |
| components/ui/resizable.tsx | 2 | `after:left-1/2`, `aria-[orientation=horizontal]:after:left-0` | Centred hit area (with after:-translate-x-1/2), and full-width hit area (with after:w-full); symmetric in both directions. |
| components/ui/sheet.tsx | 4 | `right-0`, `border-l`, `left-0`, `border-r` | Keyed to the side prop, which names a physical side; callers choose the side (the mobile sidebar picks it from the page direction). |
| components/ui/sidebar.tsx | 11 | `data-[side=left]:left-0`, `data-[side=left]:group-data-[collapsible=offcanvas]:left-[calc(var(--sidebar-width)*-1)]`, `data-[side=right]:right-0`, `data-[side=right]:group-data-[collapsible=offcanvas]:right-[calc(var(--sidebar-width)*-1)]`, `group-data-[side=left]:border-r`, `group-data-[side=right]:border-l`, `group-data-[side=left]:-right-4`, `group-data-[side=right]:left-0`, `group-data-[collapsible=offcanvas]:after:left-full`, `[[data-side=left][data-collapsible=offcanvas]_&]:-right-2`, `[[data-side=right][data-collapsible=offcanvas]_&]:-left-2` | Keyed to the side prop, which names a physical side; app-sidebar sets side from the page direction (right in RTL). |
| components/ui/tooltip.tsx | 2 | `data-[side=left]:-right-1`, `data-[side=right]:-left-1` | Arrow for the physical left/right sides that Base UI reports; the inline-start/inline-end sides use logical classes. |
| features/home/components/gateway-card.tsx | 2 | `left-[10%]`, `left-1/2` | Centred decorations (left-[10%] with w-[80%], left-1/2 with -translate-x-1/2); symmetric in both directions. |
| features/home/components/hero-terminal-demo.tsx | 3 | `ml-auto`, `pr-2`, `sm:pr-3` | Inside the dir='ltr' terminal demo; a terminal stays left to right. |
| features/home/components/icon-card.tsx | 1 | `left-1/2` | Centred glow with -translate-x-1/2; symmetric in both directions. |

### C. Guard against new physical classes (`730d5b6`, split-plan PR 1)

`web/src/styles/__tests__/physical-direction-classes.test.ts` scans `web/src` (`.ts`, `.tsx`, `.css`; not `__tests__`, not `*.test.*`, not locales) for physical direction classes: `ml/mr`, `pl/pr`, `left/right` (and negatives), `scroll-m/p l/r`, `text-left/right`, `float-left/right`, `border-l/r` (with width or colour), `rounded-l/r` and the four corners, with any variants and `!`. Classes behind `rtl:` or `ltr:` are direction-specific on purpose and are skipped; comment lines are skipped; spacing and inset values must look like values (number, fraction, `px`, `full`, `auto`, `[…]`, `(…)`), so prose such as "left-to-right" is not a class. Every other finding must be listed in `physical-direction-allowlist.json` (`file`, `classes`, `reason`); the list is counted per file and class, so a new occurrence of an allowed class fails too, and an entry that no longer matches fails as stale. Three cases: the scanner itself (physical, logical, `rtl:`/`ltr:`, comments, prose), no unlisted class, no stale entry and a reason on every entry.

Shown to fail: putting `text-left` back on `QuickActionItem` (`overview-dashboard.tsx`) gives

```
× finds no physical class under src that the allowlist does not list
- []
+   "features/dashboard/components/overview/overview-dashboard.tsx:437 text-left",
Tests  1 failed | 2 passed (3)
```

and with the class restored: `Tests  3 passed (3)`. The failure message names the logical classes to use or the allowlist to edit.

Not covered by the guard (listed in section 7): `slide-in-from-left/right` animation classes (44 in 11 files, keyed to the physical side a popover opens on), `translate-x-*` (51 in 17 files, mostly centring and animations), `bg-gradient-to-r/l` (7 in 6 files, decorative).

### Commits (from `7f9b789`)

```
fa1607a style(web): clear the upstream lint errors in files the RTL sweep touches
2b6c5ae fix(web): logical direction classes in the shared ui components
3b19d75 fix(web): logical direction classes in the shared components
6dafe80 fix(web): mirror the dashboard setup guide in right-to-left languages
621649f fix(web): logical direction classes in the dashboard
3f8fd7d fix(web): logical classes and forward arrows on the home page
87f8da9 fix(web): logical direction classes in channels
56539fb fix(web): logical direction classes in models and deployments
bdd80f4 fix(web): logical direction classes in pricing and model details
992d825 fix(web): logical direction classes in usage, task and audit logs
651322f fix(web): logical direction classes in wallet (billing history search, recharge options, plan list)
03d29b6 fix(web): logical direction classes in security (access tokens, bindings, passkeys, two-factor)
98caa63 fix(web): logical direction classes in subscriptions
256f5f7 fix(web): logical direction classes in redemption codes
3afbf83 fix(web): logical direction classes in API keys
c547dbe fix(web): logical direction classes in system settings (model pricing editors)
69c73c8 fix(web): logical direction classes in profile
4a27447 fix(web): logical direction classes in users
730d5b6 test(web): guard against new physical direction classes
(this hand-off): .fa-review scripts, screenshots, report
```

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
| 1 | `fix(web): right-to-left layout in shared components` | `c78c83f`* (without the `formatQuota` locale hunks), `d8ee064`, `cb25020`, `8ccec81`, `936ab28`, `129d598`, the RTL half of `f01c5af`*, the sidebar side of `b2720c9`*, `666c43c` (where PR 1 touches the file), `0d40127`, `f40c978`, `10cd251` + `9f8f102` (squash), `53458ed`, `7b04478`, `6496416`, `cb80274`, `e6f7d76`, **and from this session `5e64f68`, `272cbf9`, `5a2bd38` (lint cleanup, first), `44e9c0b`, `ce52823`, the `bdi` hunk of `aa699fc`***, **and from session 11 `0c605b3` (retry chain), `d264460` (`mr-1`)**, **and from session 12 (A to C): `fa1607a` (lint cleanup, first), `2b6c5ae`, `3b19d75`, `6dafe80` (dashboard setup guide), `621649f`, `3f8fd7d`, `87f8da9`, `56539fb`, `bdd80f4`, `992d825`, `651322f`, `03d29b6`, `98caa63`, `256f5f7`, `3afbf83`, `c547dbe`, `69c73c8`, `4a27447`, and last `730d5b6` (the guard and its allowlist)** | the earlier direction tests, plus **value-change, quota-audit-direction, quota-preview-direction, change-arrow-direction, carousel-direction, api-tab-direction, preferences-direction, code-direction (pre)**, **value-change (chain cases), retry-chain-direction, adjust-quota-direction**, **setup-guide (3 direction cases), response-table-direction, arrow-direction, physical-direction-classes (the guard)** | No visible change in LTR. Layout only (decision 2 of session 11: the Persian-only audit labels moved to PR 4) |
| 2 | `feat(i18n): Persian (fa) partial locale, tooling and docs` | `b2720c9`* (without the sidebar), `07e070d`, `759a31e`, `f5296da` + `776c03c`, `bfe01bb`, the script part of `fa96f8b`; AGENTS.md, web/AGENTS.md, the i18n skill; `docs/i18n/fa.md` (with the glossary rows from `4d901bd`, `d1e501d`, `305ee51`, **the fa.md part of `b1c4bad` and of `c7e15d1`**), `2242169`, **`0330995` (`@babel/parser` dev dependency)** | languages, direction-provider, check-fa, persian-monospace, intl-locale lint case | Language switcher lists «فارسی»; `docs/i18n/fa.md` is a new file under `docs/` (ask the maintainers) |
| 3 | `feat(web): Solar Hijri dates, chart axes and date pickers in Persian` | `0ea975e`, `c2369c8` + `42a98a2`, `170ba72`, `9d22fac`, **`d14b92a`** | display-date-locale, activity-time-cell-dates, login-session-dates, date-picker-display, calendar-persian, chart-time-locale, **system-info title-dates** | None |
| 4 | `feat(web): Persian labels in audit text` (roles, sign-in methods, ID label, user action names) | net of `f767402` + `9cd40a8`, **`0184320`, `768f817`** (moved from PR 1, decision 2 of session 11) | audit-content-locale, details-locale, **identity-action-locale, user-action-name, manage-operator-locale** | None: Persian only, English and Chinese pinned (could fold into 3) |
| 5 | `fix(web): format money and numbers in the interface language` | locale half of `f01c5af`*, `formatQuota` hunks of `c78c83f`*, the `locale` argument in `recharge-form-card.tsx`, `31f68f8`, **`aae92d9`, `0153b2d`, `aa699fc`* (without the `bdi` hunk), `d623586`, `172c97e`, `eb96d86`, `089255f`, `5ab137d`**, **session 11: `8f0966d` (page counter), `7146052` (percent sign), `57a0080` (plan price)** | format-quota-locale, format-currency-locale, amount-locale, summary-cards-locale, profile-header-locale, response-time-format, **format-fixed-locale, message-duration-locale, system-info number-locale, flow-number-locale, chart-number-locale, total-direction (bdi half goes with PR 1), step-number-locale, count-locale, subscriptions format-locale and list-number-locale**, **pagination (counter), percent-sign-locale, purchase-price-locale, plan-price-locale** | **Yes**: numbers follow the interface language; French, Russian and Vietnamese show a decimal comma in the formatted decimals; English is unchanged (tested). Its own PR |
| 6 | `feat(i18n): Persian translation batches` | `fa96f8b`, `551a8a3`, `795d296`, `4db67e2`, `3825299`, `aaba1b0`, `8a76a75`, `ada5af2`, `c6aaad5`, `4d901bd` (fa.json part), `17d9c1b`, `58de3e3`, `f7ad4f7`, `4ba8a4c`, `3b076b1`, `25399dc`, `86a8c7c`, `97b1b72`, `bcc8ea7`, `3ca2feb`, `d1e501d` (fa.json part), `0097c77`, `c2798f3`, `4c66300`, `67e72b5`, `39a571f`, `5cd6e90`, `b744e97`, `334c3d9`, `2adca66`, `0d1b17b`, `467f231`, `a14be44`, `bdf0202`, **`b1c4bad` (fa.json part), `4972f14`, `c7e15d1` (fa.json part)** | `bun run i18n:check-fa` | None: only `fa.json` |

Leave out of every PR: `.fa-review/` and all hand-off and script commits, and the merge commits. The guard (`730d5b6`) goes in PR 1 after all its conversions: it fails until every folder is converted, and its allowlist must match the files PR 1 ships (if an upstream change adds a physical class before PR 1 lands, convert it or list it). Order: the upstream fix branches that are offered (`fix/public-sans-font` among them), then 1, 2, 3 (+4), 6; 5 when upstream agrees to the behaviour change. `67e72b5` (fa.json) must land together with or after `2242169`'s check, or the check-fa project-sources test fails; the PR 4 tests for the audit labels read `fa.json`, so PR 4 lands together with or after PR 6. PR 5 test `total-direction` checks the `bdi` from PR 1.

## 6. Verification (session 12)

### `feat/fa-locale` (from `web/`, code at `730d5b6`)

| Command | Result |
| --- | --- |
| `git remote -v`, `git push --dry-run origin HEAD`, `git push --dry-run origin HEAD:refs/heads/dryrun-rtl-check` | origin `https://github.com/yaser-k/new-api-fork`; `Everything up-to-date` and `* [new branch] HEAD -> dryrun-rtl-check` (dry run, nothing created). The local branch had an older, different history (`d263b5f`); reset to `origin/feat/fa-locale` = `7f9b789` first |
| `git remote add upstream …`, `git fetch upstream main`, `git remote set-url --push upstream DISABLED` | `upstream/main` = `c2b7a9a` (unchanged, no merge); upstream push URL `DISABLED` |
| `bun install` | 1206 packages |
| `bun run typecheck` | `tsgo -b`, exit 0 |
| `bun run lint` | exit 1: **120 errors, 65 warnings**; `upstream/main` (own worktree): **182 errors, 66 warnings**; error file+rule pairs above upstream: **0**; errors in the 452 files changed against upstream: **0** (session 11: 139 errors; the lint cleanup `fa1607a` removed 19) |
| `bun run test` | `Test Files 244 passed (244)`, `Tests 2510 passed (2510)` (session start: 241 files, 2499 tests) |
| `bun run build` | exit 0, total 66915.8 kB / 20616.7 kB gzip |
| `bun run i18n:sync` | exit 0; fa partial, missing 1062, extras 0; the seven required locales missing 0, extras 0 |
| `bun run i18n:check-fa` | `check-fa: 5733 keys, no findings` |
| `git diff --quiet 7f9b789 -- web/src/i18n/locales` | exit 0: all eight locale files identical to `7f9b789` |
| `bunx oxfmt --check` on every changed file | no findings |
| Guard with `text-left` put back on `QuickActionItem` | `Tests 1 failed | 2 passed (3)`, names `overview-dashboard.tsx:437 text-left`; restored: `3 passed` |
| `go build -o <scratch>/bin/fa .` (branch, fresh `web/dist`) and `go build -o <scratch>/bin/base .` (`7f9b789` worktree, its own `bun run build`) | exit 0 both (Go 1.25.1) |

### Running app, screenshots and the English pixel comparison

One database for every instance, so both builds render the same data: `seed-s12.py` ran once against the branch binary (`GET /api/setup` → `status false, database_type sqlite`; `POST /api/setup` → `系统初始化成功`, success true; 3 channels, an API key, a plan, a redemption code, 3 usage logs with fixed timestamps on 2026-09-25 UTC); that instance was stopped and its SQLite file copied. Then `base` (`7f9b789`) on 3401 and `fa` (branch) on 3402, `TZ=UTC`, rate limits off. `shots-s12.mjs` signs in as admin, fixes the browser clock at 2026-09-25 14:00 UTC, sets reduced motion, expands the setup guide and captures full pages; `compare-s12.mjs` compares two sets pixel by pixel in Chromium (canvas `getImageData`, any RGBA difference counts). Chromium 141.0.7390.37 headless, 1440×900, light theme. None blank (fewest colours 1320; blank threshold 16). Console errors: the 401 of the pre-login session probe and `ERR_CERT_AUTHORITY_INVALID` for an external resource blocked by the sandbox proxy.

```
python3 seed-s12.py http://127.0.0.1:3400 <scratch>/run-seed/one-api.db      # SEED_DONE
node shots-s12.mjs http://127.0.0.1:3402 <shots>/fa-after  fa rtl           # branch, Persian
node shots-s12.mjs http://127.0.0.1:3401 <shots>/fa-before fa rtl-before    # 7f9b789, Persian
node shots-s12.mjs http://127.0.0.1:3401 <shots>/en-base   en en-base       # 7f9b789, English
node shots-s12.mjs http://127.0.0.1:3401 <shots>/en-base2  en en-base2      # 7f9b789 again (noise floor)
node shots-s12.mjs http://127.0.0.1:3402 <shots>/en-after  en en            # branch, English
# branch binary on port 3401 with a copy of the base database (same port, same affiliate code):
node shots-s12.mjs http://127.0.0.1:3401 <shots>/en-after2 en en
node compare-s12.mjs <shots>/en-base en-base <shots>/en-base2 en-base2
node compare-s12.mjs <shots>/en-base en-base <shots>/en-after  en
node compare-s12.mjs <shots>/en-base en-base <shots>/en-after2 en
```

English, `7f9b789` against the branch (differing pixels):

| Page | base vs base (noise) | base vs branch, port 3402 | base vs branch, same port and database |
| --- | ---: | ---: | ---: |
| 410 dashboard, setup guide expanded | 0 | 38 (the port `3401`/`3402` in the curl example) | **0** |
| 411 usage logs | 0 | 0 | **0** |
| 412 pricing | 0 | 0 | **0** |
| 413 model details (`/pricing/gpt-4o`) | 0 | 0 | **0** |
| 414 channels | 0 | 0 | 53 (x 309–414, y 83–102) |
| 415 models | 0 | 0 | **0** |
| 416 profile | 0 | 0 | **0** |
| 417 wallet | 0 | 360 (the random affiliate code in the invite link, created per database) | **0** |
| 418 subscriptions | 0 | 0 | **0** |
| 419 channels, row menu open | 53 (x 309–414, y 83–102) | 53 (same) | **0** |

The only remaining difference is 53 pixels of anti-aliasing in the "Max Retries: 0" badge of the channels header, which also differs between two runs of the same `7f9b789` build (cropped and compared: same text, same position). **English renders the same on every captured page.** The dashboard layout probe gives identical English positions on both builds (code texture 528–1035, text-align right/end; step line x 270 at circle 254–286; quick action title 1140–1198 after its icon 1098–1134).

Persian, `7f9b789` against the branch: every page changed except the profile (0 pixels: its converted classes do not move anything with this data); dashboard 268 343 pixels, row menu 14 785, models 6 973, pricing 4 947, channels 4 922, wallet 4 622 (includes the affiliate code), model details 3 031, usage logs 1 748, subscriptions 1 110.

| File | Shows | Logged values |
| --- | --- | --- |
| `410-dashboard-setup-guide-rtl-before.png`, `410-dashboard-setup-guide-rtl.png` | Persian overview, setup guide expanded, `7f9b789` / branch | code texture x 696–1203 (behind the heading at 813–1195), `text-align: right` → x 405–912, `left`; step line x 794 → 1169 (circle 1154–1186); quick action title 56–129 → 227–300 (icon 306–342) |
| `410-dashboard-setup-guide-en.png` | English overview, branch | same positions as `7f9b789` (above); 0 pixels different |
| `411-usage-logs-rtl.png`, `412-pricing-rtl.png`, `413-model-details-rtl.png`, `414-channels-rtl.png`, `415-models-rtl.png`, `416-profile-rtl.png`, `417-wallet-rtl.png`, `418-subscriptions-rtl.png` | Persian pages whose classes changed | numeric pricing columns end on the left; channel card badges without the leaked `-ms-1.5` |
| `419-channels-row-menu-rtl.png` | channel card menu (shared dropdown) | item icons at the inline end (left), as they sit on the right in English |
| `411-usage-logs-en.png`, `412-pricing-en.png`, `414-channels-en.png`, `417-wallet-en.png` | English, branch | 0 pixels different from `7f9b789` |

## 7. Remaining issues

1. **Code bugs found while translating (English affected too)**, kept listed: the session 7 and 8 items (channels glued labels, `&apos;`, `&mdash;`, `&#10;`, plurals, the Gemini sentence, the compliance separators, OAuth callback base; see `git show 57c75bd:.fa-review/REPORT.md`) and the joins in the hard-coded table.
2. **Dates not in the fork's pattern** (for upstream, decision of session 9): the compliance "Confirmed at" time (`integrations/payment-settings-section.tsx:840`, `toLocaleString()`) and the announcements table (`content/announcements-section.tsx:394`, `dayjs().format(…)`, plus its own relative time).
3. **Left as they are on purpose**: the chart time axes run left to right in RTL (VChart axes; reversing them would also reverse the reading of trends), the rankings trend arrows (they show up and down, not a direction of reading), native date inputs (browser), backend content in Chinese or English (Go i18n has en and zh only), the hero terminal demo (a terminal stays left to right).
4. **Still listed** (hard-coded table): the carousel's screen-reader labels (no key); and, found in session 11, the full pager's page buttons and `Total:`, the raw percentages (`toFixed` + `%`), the discount label's raw number, the request conversion chain ` -> `, and the raw response time and group ratio in the usage log details. Each needs a new key, a shared change or a decision beyond the four items fixed this session.
5. `lib/theme-customization.ts` says in a comment that the `default` preset resolves to serif, while `PRESET_DEFAULT_FONT.default` is `sans` (upstream comment; not touched, the Public Sans branch stays minimal).
6. **Session 12, left for later.** None of items 1 to 5 was about direction classes, so none is removed. New: (a) `components/ai-elements/web-preview.tsx` keeps one `text-left` until its iframe sandbox (`allow-scripts` with `allow-same-origin`) and index key are decided upstream; the component is unused. (b) Outside the guard: `slide-in-from-left/right` animation classes (44 in 11 files; popovers keyed to the physical side they open on), `translate-x-*` (51 in 17 files; mostly centring and motion), `bg-gradient-to-r/l` (7 in 6 files; decorative). (c) tailwind-merge treats `ps/pe`, `ms/me`, `start/end` and `border-s/e` as separate from `px`, `mx`, `inset-x` and `border-x`, unlike their physical forms; callers were checked (section 2), but a future caller that passes `px-*` to a component whose base has an unscoped `ps-*`/`pe-*` would not override it. Extending the tailwind-merge config is possible, but it would also change merges that upstream code already relies on, so it was not done.

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
- Session 12 (this one): the dashboard setup guide in RTL, the sweep of every physical direction class (414 converted, 62 kept with reasons), the guard test, the lint cleanup of the touched files.
