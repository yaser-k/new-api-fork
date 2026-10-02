Title: feat(web): Solar Hijri dates and numbers in the interface language
Branch: yaser-k/new-api-fork:pr/fa-dates-numbers (from upstream main 1a4166d)

## Agent

- Tool: Claude Code
- Tool version: Claude Code 2.1.288
- Model (full id): (add when opening; not stored in the repository)
- Host: Claude Code on the web
- Date (UTC): 2026-10-02

The code was AI-generated and reviewed by the submitter; the git author is not one of the project's core developers.

## Links

- Related: #7198
- Part 3 of 3; applies alone on upstream main. Order: right-to-left layout, Persian, then this PR. After the first PR is merged, 9 files need a rebase here (neighbouring lines: a class change next to a locale argument); the combined result is on the fork's `feat/fa-locale`.

## User request

- Verbatim (excerpt): "`pr/fa-dates-numbers` | `feat(web): Solar Hijri dates and numbers in the interface language` | Plan PRs 3 and 5: Solar Hijri dates, chart axes and date pickers in Persian, and money and numbers formatted in the interface language (French, Russian and Vietnamese then show a decimal comma; English must not change)."
- Later constraints: none.

## Out of scope — refuse

- Matched: no

## Open gate — do not open unless all are satisfied

- Out of scope: no · Usage question: no · Issue facts present: yes · Verification is commands and results: yes · Short and factual: yes
- Open: yes (by the submitter)

## Kind

- [x] New feature

## Issue facts

- Actual behavior: many money, quota and count displays use the browser language or a raw value instead of the interface language; dates are Gregorian in every language.
- Impact: a French interface on an English browser shows `1.23`; Persian users see Gregorian dates.
- Applicable types: frontend only.

## Change

1. Dates (Persian only): displayed dates use the Solar Hijri calendar in numeric order with 24-hour time (`۱۴۰۵/۰۷/۰۳ ۱۵:۰۵:۰۹`), relative times are Persian, and table/log cells carry the Gregorian date as `title`. Chart axes are Solar Hijri and stay in time order. The date pickers show the Solar Hijri calendar through `@daypicker/persian` (MIT), loaded only for Persian; `react-day-picker` stays a single copy at 10.0.1. API values, filters, inputs and exports stay Gregorian.
2. Numbers: money, quotas, token counts and percentages go through `@/lib/format` / `@/lib/currency` with `toIntlLocale(...)`.

## Files

| Path | Why |
| --- | --- |
| `web/src/lib/format.ts`, `lib/time.ts`, `features/*/lib/format.ts` | shared helpers take the locale |
| `components/ui/calendar.tsx`, date pickers, `package.json` | Solar Hijri picker |
| `features/**` (dashboard, wallet, usage logs, channels, ...) | call sites |

## Behavior

- Before: browser-locale numbers; Gregorian dates.
- After: numbers follow the interface language. English is unchanged (tested). **French, Russian and Vietnamese show a decimal comma** where decimals are formatted; Chinese and Japanese keep the point. Dates change only in Persian.
- Non-goals: native `<input type="datetime-local">` stays browser-drawn.

## Verification

- `bun run typecheck`: exit 0 (each of the 2 commits).
- `bun run lint`: 165 errors, same as upstream main; no file+rule pair above upstream; the one error in a changed file is upstream's, unchanged.
- `bun run test`: 202 files, 2325 tests passed (first commit alone: 186 files, 2250 passed). `bun run build`: exit 0. `bun run i18n:sync`: no changes.
- Fail-before: with every non-test file reverted to upstream, 89 of the branch's 206 tests fail, in all 32 test files; the passing ones assert unchanged English output.
- Not verified: browsers other than Chromium.

## Risks

- Visible change for French, Russian and Vietnamese (decimal comma); merge when that is agreed.
- Billing / quota / auth impact: display only; no stored value changes.

## Scope check

- Single focused change: yes (display formatting) · Secrets: no · Out of scope: no
