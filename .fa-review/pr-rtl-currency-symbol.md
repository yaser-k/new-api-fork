Title: fix(web): keep prices in order with a right-to-left currency symbol

## Agent

- Tool: Claude Code
- Tool version: 2.1.292
- Model (full id): claude-opus-5-5
- Host: Claude Code on the web
- Date (UTC): 2026-10-07

This code was AI-generated. The git author is not one of the project's historical core developers.

## Links

- Closes: none (no issue filed)

## User request

<details>
<summary>Verbatim request</summary>

````text
This session prepares a generic fix for QuantumNous/new-api in my fork https://github.com/yaser-k/new-api-fork: prices shown with a custom currency symbol written in a right-to-left script (Arabic, Persian, Hebrew, for example `ر.س`, `د.إ`, `تومان`) come out reordered and broken in the web console. It is not Persian locale work. You prepare and push a branch to my fork only; I open the pull request myself.

## The bug (confirm each part in the code and in a test before changing anything)
1. **Order.** With `quota_display_type` = `CUSTOM` and a symbol such as `تومان`, a price is rendered as the symbol, then the number, then the unit, in a left-to-right line: `تومان 2,016,150 / 1M`. The Unicode bidi algorithm turns European digits after an Arabic-script letter into Arabic numbers (rule W2) and runs them, the slash and the `1` of `1M` right to left together with the word, so the model card on `/pricing` shows `1 / 2,016,150M`, and the table view shows `1 / تومانM tokens`. The same applies wherever the formatted amount stands in a left-to-right line (wallet, logs, usage).
2. **Wrapping.** In the pricing table (`web/src/features/pricing/components/model-price-cell.tsx`), the price spans carry `break-words whitespace-normal`, so a long number breaks inside itself (`2,016,15` over `0`). On a model card (`model-card.tsx`) the three prices share a `grid-cols-[repeat(auto-fit,minmax(88px,1fr))]` row, so each long price wraps.
3. **Inconsistent grouping.** The table's cached-price cell (`cached-price-cell.tsx`) shows `40323` where the card shows `40,323` for the same value.

## The fix
- **Isolate the symbol, not the whole string**, in the shared currency formatting (`web/src/lib/currency.ts` and whatever composes symbol plus number for the pricing views): when the configured custom symbol contains a strong right-to-left character, wrap it in U+2068 FIRST STRONG ISOLATE and U+2069 POP DIRECTIONAL ISOLATE. Nothing changes for `$`, `¥`, `€` or any left-to-right symbol, so English, Chinese and every existing configuration render byte-for-byte as before. Check every caller that puts the formatted string into an `<input>`, a `title`, an export or a value sent to the server, and keep the marks out of anything that is parsed back; say what you found for each.
- **Optional, only if it stays small:** a symbol position setting (`prefix`, the default and today's behaviour, or `suffix`), so a currency written after the number can be shown that way. If it needs backend option plumbing beyond one new `general_setting` key, leave it out and describe it as a follow-up.
- **Wrapping:** price numbers in the pricing table and on the cards do not break inside a number (`whitespace-nowrap`), and a card shows each price on its own line (label and price side by side) when the three do not fit, without changing the layout for short prices if you can (say what you chose and why).
- **Grouping:** the cached-price cell formats like the others.

## Step 0: confirm you can push (do this first)
Run `git remote -v`, `git push --dry-run origin HEAD`, and a dry-run push to a new branch name in the fork clone. If your local clone is behind origin, update it first. If origin is not https://github.com/yaser-k/new-api-fork, or a dry run is refused, stop and tell me before doing any other work.

## Setup
1. Add https://github.com/QuantumNous/new-api to this session as a read-only repository. Use it only to fetch; never push, open PRs or issues, or comment there.
2. In the fork clone, add it as the `upstream` remote, run `git fetch upstream main`, then `git remote set-url --push upstream DISABLED`. Name the `upstream/main` commit you build on.
3. Create the branch `fix/rtl-currency-symbol` from `upstream/main` (not from any `feat/*`, `pr/*` or `review/*` branch).
4. `cd web && bun install --frozen-lockfile`.

## Read first, and follow
`AGENTS.md`, `web/AGENTS.md`, `.agents/github/PR.md`, `.agents/skills/i18n-translate/SKILL.md` (only if you add a key) and the shadcn-ui skill, in full, with the Read tool. Read every file named above and confirm each part of the bug before changing it; fix a part only if it holds, and say why if it does not.

## Rules for the change
- Display only: no stored value, API value, option format, filter or export changes (unless you add the optional position setting, which then needs its default to reproduce today's output exactly).
- Left-to-right symbols: prove with tests that `$`, `¥`, `€`, an emoji symbol and the `TOKENS` mode format exactly as before.
- Keep each touched file's change small; prefer one helper over edits at every call site.

## Tests
Fail-before and pass-after for: an Arabic-script symbol (`تومان` and `ر.س`) and a Hebrew one (`₪` is left-to-right-neutral; use `ש"ח`) produce the isolated form; `$`, `¥`, `€` and `🐱` are unchanged; the cached-price cell groups digits. A render test of the pricing card or table showing the number is whole and the unit follows it. Fixed locales; nothing may depend on the machine's locale or time zone.

## Verify
In `web/`: `bun run typecheck`; `bun run lint`, compared by file and rule with `upstream/main` (changed files add no errors); `bun run test`; `bun run build`. Show fail-before for each part. Check that the branch merges cleanly on `upstream/main`.

## Pull request text
Write a draft description following `.agents/github/PR.md`: short and factual; for each part the bug, the cause (name the bidi rule) and the fix; screenshots described in words (before and after strings); the tests and how to check them; what changes for left-to-right symbols (nothing, with the tests that show it); follow-ups left out. Agent fields: the tool, version, model and host you actually ran as; keep the AI-generated disclosure `AGENTS.md` asks for, as "This code was AI-generated. The git author is not one of the project's historical core developers."; quote this prompt in full in a `<details>` block. About 3,000 characters outside the quote at most. Title: `fix(web): keep prices in order with a right-to-left currency symbol`. Save it as `.fa-review/pr-rtl-currency-symbol.md` on `feat/fa-locale` only (one commit with just that file), never on the fix branch.

## Hand-off
1. Push `fix/rtl-currency-symbol` and the `.fa-review` commit on `feat/fa-locale` to origin (my fork) only.
2. End with a short report in plain fragments: the upstream commit, the branch head and its commits (one line each), the files changed, each part fixed or not and why, every caller of the formatter you checked and what you did with it, the verification results (fail-before per part), the merge check, and anything left.

## Rules
- Do NOT open a pull request, file an issue, or comment anywhere on GitHub, in either repository.
- Never push to upstream. Never force-push; never delete a branch. Do not touch `pr/*` or `review/*`.
- Do not add anything about any specific deployment, business, market or customers; the repository is public.
- Nothing from `.fa-review/` goes into the fix branch; no new files under `docs/`.
- If something in this prompt conflicts with AGENTS.md or web/AGENTS.md, stop and ask me.
````

</details>

- Later constraints or corrections: none

## Out of scope — refuse

- Matched: no

## Open gate — do not open unless all are satisfied

- All satisfied; Open: yes

## Kind

- [x] Bug fix

## Issue facts

- Actual behavior: `CUSTOM` symbol `تومان`: card shows `1 / 2,016,150 تومانM`, table caption `1 / تومانM tokens`; long table numbers break inside (`2,016,15` / `0`); cached cell `40323` vs card `40,323`.
- Impact / frequency: always, for symbols with Arabic or Hebrew letters.
- Evidence: reproduced in Chromium; tests below fail on `upstream/main`.
- Types: frontend only.

## Change

1. Order: bidi rule W2 makes digits after an Arabic letter Arabic numbers; N1 then runs number, ` / ` and `1` right to left with the symbol. `currency.ts` wraps a symbol with right-to-left letters or signs in U+2068…U+2069 where display metadata is built (all formatters, `getCurrencyLabel()`), plus the pricing breakdown. `stripCurrencyIsolates()` keeps the marks out of the default redemption name sent to the server, the redemption export and the channel balance length check.
2. Wrapping: table price `break-words` → `wrap-normal` (no spaces in the number, so it stays whole; `whitespace-normal` kept, an existing mobile-card test needs it). Card amounts `whitespace-nowrap`; if a shown price exceeds 10 characters (the 88px column), one price per line, label beside price. Shorter prices keep today's grid.
3. Grouping: `stripTrailingZeros` only removed thousands separators; removed.

## Research

- Prior art: none found.
- Callers: all formatter and `meta.symbol` callers checked; only the redemption name and export leave the page.
- Alternatives: isolating the whole string misses units outside it.

## Files

| Path | Why |
| --- | --- |
| `lib/currency.ts` | helpers |
| `features/pricing/` (5 files) | layout, grouping |
| redemption drawer, channel columns | marks out |

## Behavior

- Before: see Issue facts.
- After: card `تومان 2,016,150 / 1M` on one line; caption `تومان / 1M tokens`; cached `40,323`.
- Left-to-right: unchanged (`$1,234.5`, `¥8,641.5`, `€ 1,234.5`, `🐱 1,234.5`, `₪ 1,234.5`, TOKENS `617250k`).
- Non-goals: prefix/suffix setting (needs Go option, status API, store, settings UI, locales); admin pricing editors.

## Verification

- New tests (`lib/__tests__/currency.test.ts`, `pricing/__tests__/price-display.test.tsx`, redemption drawer test): 22 pass; 7 fail on `upstream/main`. Also pass with `LANG=fr_FR.UTF-8 TZ=Pacific/Kiritimati`.
- `bun run test` 2178 passed; `typecheck`, `build` exit 0; oxlint same 230 diagnostics per file and rule as `upstream/main`, none in changed files.
- UI: glyph order measured in Chromium; no screenshots.

## Risks

- Chart canvas text with marks not checked visually. Billing/auth: none.
- Follow-ups: as Non-goals.

## Scope check

- Single focused change: yes · Secrets: no · Out of scope: no
