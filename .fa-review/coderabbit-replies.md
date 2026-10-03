# Replies to CodeRabbit (sessions 14 and 15)

Draft replies for the CodeRabbit review comments, to post by hand after `pr/fa-locale` and `pr/fa-dates-numbers` are fast-forwarded to `review/fa-locale` and `review/fa-dates-numbers`. Commit ids are on those branches.

## QuantumNous/new-api#7652 (`pr/fa-locale`)

### 1. `docs/i18n/fa.md`: get explicit approval for the new `docs/` file (inline)

Moved to `.agents/skills/i18n-translate/fa.md` with `git mv` in 2255498, next to the i18n skill that already points to it. `AGENTS.md`, the skill and the `check-fa.mjs` comment use the new path, and the PR no longer adds anything under `docs/`; the one comment in QuantumNous/new-api#7653 that named the old path is updated there (2f4df76).

### 2. `.agents/skills/i18n-translate/SKILL.md`: handle a missing `fa.json` in the Step 3 sample (inline)

Fixed in 4d75b96: Step 3 now uses the same `readLocale` guard as Step 4, so a missing `fa.json` reads as an empty translation and any other read error is rethrown.

### 3. `web/src/features/usage-logs/lib/__tests__/audit-content-locale.test.ts`: quota assertions depend on the runtime default locale (inline)

Fixed in 555f22a: the English and Chinese quota assertions now run with an `en-US` default set through the same `Intl.NumberFormat` mock as the German case; the text checks are unchanged. Two more tests had the same dependency and are pinned the same way in 4285a18.

### 4. `web/src/styles/__tests__/persian-monospace.test.ts`: declare `@tailwindcss/node` (inline)

Fixed in c22a2f4: `@tailwindcss/node` is now a dev dependency at `^4.3.3`, the version the lockfile already resolved. The only lockfile change is that direct entry.

### 5. Missing return types on the test helpers (nitpick)

Done in 2b2ed43: `renderUpgradeBadge` and `renderEnumCell` return `void`, `renderOverridePreview` returns `Promise<HTMLElement>`, both `renderDialog` helpers return `Promise<BoundFunctions<typeof queries>>`, and `t` returns `string`.

## QuantumNous/new-api#7653 (`pr/fa-dates-numbers`)

### 1. `web/src/features/usage-logs/components/columns/common-logs-columns.tsx`: pass the resolved language to `buildDetailSegments` (inline)

Fixed in 39bc56c: the details cell now passes `i18n.resolvedLanguage || i18n.language`, like the other cells. A new test renders the cost and details cells with Persian resolved and unresolved.

### 2. `web/src/features/usage-logs/lib/__tests__/quota-audit-number-locale.test.ts`: the English assertion depends on the runtime default locale (inline)

Fixed in e0173e8: the English test pins an `en-US` default through an `Intl.NumberFormat` mock, so it passes whatever the machine's locale (checked with fr, de and ar-EG). Four unknown-language fallback tests had the same dependency and are pinned in 7afd1a7, and upstream's flow tooltip test, which this PR made depend on it, in 3a23fa1.

### 3. `web/src/features/subscriptions/components/dialogs/subscription-purchase-dialog.tsx`: pass `locale` to all quota formatters (outside the diff)

Fixed in 35562b8: the three `formatQuota` calls get the interface locale, with a test of the three amounts in English and Persian. A sweep of the files this PR touches found the same omission in the table pager total, the redemption code quota, the admin user subscriptions quota and the channel balance cell, notice and dialog; fixed with tests in c3d2e96.

### 4. `web/src/features/wallet/components/subscription-plans-card.tsx`: Persian branching duplicates `formatDisplayDate` (nitpick)

Left as is on purpose: for every language other than Persian this keeps upstream's exact `toLocaleString()` output, because this PR only changes dates shown in Persian. Moving all languages to `formatTimestampToDate` would change the English date format and belongs in its own change.

## QuantumNous/new-api#7653, second review (session 15)

CodeRabbit's second review of `pr/fa-dates-numbers` (on `3a23fa1`) raised two findings outside the diff. Commit ids are on `review/fa-dates-numbers`; post these after `pr/fa-dates-numbers` is fast-forwarded to it.

### 5. `web/src/components/data-table/core/pagination.tsx`: page buttons and page size options render raw numbers (outside the diff, lines 189 and 141)

Fixed in be09101: the page buttons, their screen-reader text, the page size options and the selected size now use `formatFixed(n, 0, locale)`, the helper of the compact counter, so English stays `247` and `100`. The table state, the select values and the React keys keep the numbers; tests cover English and Persian, and that choosing a Persian option still sets the numeric page size.

Sweep, f48f88f: the billing history dialog's own pager (shown range and page counter) had the same raw numbers and now uses the interface locale too.

### 6. `web/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx`: plan selector price uses `toFixed(2)` (outside the diff, line 264)

Fixed in d8b7d81 with `formatFixed(price, 2, locale)`, as the purchase dialog formats plan prices; English stays `Pro ($9.90)` and the option value is still the plan id. The dialog's test, renamed to `user-subscriptions-number-locale`, adds the English and Persian labels and a Persian add-subscription request with the numeric id.

Sweep, c59e20b: the usage logs had the same kind of raw numbers beside formatted prices and counts: the group ratio and search call counts in the details breakdown, the group ratio in the token and details cells, RPM and TPM, and the audit "Changed / Total". They now use `formatFixed` with the interface locale; English output is unchanged.

Sweep, f48f88f: the same in the wallet and subscription screens: the purchase limit counts, the subscription counts, raw quota tooltip and used share, the invite count, the Creem price and the recharge minimums. `formatCreemPrice` now takes the locale and keeps `$9.90` without one, so the admin product editor is unchanged.

Sweep, c64af49: also the check-in total, the server log file count and the channel affinity cache stats (TTL, hit rate, token counts).
