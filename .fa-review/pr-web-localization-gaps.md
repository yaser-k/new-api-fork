Title: fix(web): relative times, dates and labels in the interface language

## Agent

- Tool: Claude Code
- Tool version: 2.1.288
- Model (full id): claude-opus-5-5
- Host: Claude Code on the web
- Date (UTC): 2026-10-03

This code was AI-generated. The git author is not one of the project's historical core developers.

## Links

- Part of #7654: rows 1, 2, 4, 10 (part), 11
- Related: #7653 (conflicts, see Risks)

## User request

<details>
<summary>Verbatim request</summary>

````text
This session prepares a larger generic fix for QuantumNous/new-api in my fork https://github.com/yaser-k/new-api-fork: one pull request that closes several rows of upstream issue https://github.com/QuantumNous/new-api/issues/7654 ("Web console localization gaps"). It is not Persian work. You prepare and push a branch to my fork only; I open the pull request myself.

## Scope (rows of #7654; read the issue first)
In scope, one commit per row, in upstream's commit style (`fix(web): ...`):
- **Row 2, date pickers.** Cherry-pick `897e01d` from `origin/fix/date-picker-chinese-locale` (already checked: both pickers pick `zhCN`/`zhTW`, with its test). Do not change it.
- **Row 1, relative times.** Use the existing `formatTimestampRelative()` (`web/src/lib/format.ts`, `Intl.RelativeTimeFormat`) with the interface locale (`toIntlLocale(i18n.resolvedLanguage || i18n.language)`) for: `features/security/components/login-session-item.tsx` (`dayjs.unix(...).fromNow()`), `features/security/components/passkey-card.tsx` (`dayjs(...).fromNow()`), `features/system-settings/content/announcements-section.tsx` (`getRelativeTime`, hand-built `5m ago` / `h ago` / `d ago`), and `components/notification-popover.tsx` (`getRelativeTime`, `{{count}} … ago` keys). In the popover keep `Just now` under a minute and keep its absolute date for future dates and for dates over two years old; the 28–29 day `0 months ago` must go. `features/models/lib/model-utils.ts` `formatRelativeTime` looks unused: check, and list it rather than change it if nothing calls it.
- **Row 4, dates in two formats.** The three `toLocaleString()` date displays (`features/wallet/components/subscription-plans-card.tsx` around lines 475 and 480, `features/system-settings/integrations/payment-settings-section.tsx` around line 840, `features/channels/components/drawers/channel-mutate-drawer.tsx` around line 338) use the browser locale while the rest of the console uses the `YYYY-MM-DD HH:mm:ss` helpers in `lib/format.ts` (`formatTimestamp`, `formatTimestampToDate`). Use the same helper as the neighbouring displays.
- **Row 10, untranslated text (part).** The carousel's screen-reader labels (`components/ui/carousel.tsx` lines 218 and 248, `Previous slide` / `Next slide`) through `t()`; prefer existing keys (`Previous`, `Next`) if they read well, otherwise add keys to every locale with the i18n skill. `z.string().url()` without a message (`features/system-settings/general/system-info-section.tsx` lines 53 and 109) gets the existing translated message `Must be a valid URL`. Leave `formatUseTime` / `formatTokens` as they are (shared helpers; list them).
- **Row 11, voice input.** `components/ai-elements/prompt-input.tsx` line 1128 sets `speechRecognition.lang = 'en-US'`; use the interface locale instead (a valid BCP 47 tag through `toIntlLocale`, falling back to the browser default when there is none). Keep the change minimal: it is a copied UI component.

Out of scope (list them in the report and the PR text as follow-ups, do not change): row 3 numbers (#7653), rows 5 and 6 plural forms and numbers inside translations (they need a decision on i18next plural keys in every locale), rows 7 to 9 (#7651, #7652), `formatUseTime` / `formatTokens`.

## Step 0: confirm you can push (do this first)
Run `git remote -v`, `git push --dry-run origin HEAD`, and a dry-run push to a new branch name in the fork clone. If your local clone is behind origin, update it first. If origin is not https://github.com/yaser-k/new-api-fork, or a dry run is still refused, stop and tell me before doing any other work.

## Setup
1. Add https://github.com/QuantumNous/new-api to this session as a read-only repository. Use it only to fetch; never push, open PRs or issues, or comment there.
2. In the fork clone, add it as the `upstream` remote, run `git fetch upstream main`, then `git remote set-url --push upstream DISABLED`. Name the `upstream/main` commit you build on (it was `1a4166d`).
3. Create the branch `fix/web-localization-gaps` from `upstream/main` (not from any `feat/*`, `pr/*` or `review/*` branch). Leave `fix/date-picker-chinese-locale` as it is.
4. `cd web && bun install --frozen-lockfile`.

## Read first, and follow
`AGENTS.md`, `web/AGENTS.md`, `.agents/skills/i18n-translate/SKILL.md`, `.agents/github/PR.md` and the shadcn-ui skill, in full, with the Read tool. Read every file named above and confirm each problem in the code before changing it; fix a row only if it holds, and say why if it does not.

## Rules for the change
- Display only; no stored value, API value, filter or export changes.
- English: say in the report and the PR text exactly what English output changes (for example Day.js `an hour ago` becomes `1 hour ago`, `5m ago` becomes `5 minutes ago`, a browser-locale date becomes `YYYY-MM-DD HH:mm:ss`); nothing else may change in English.
- Locale files only through `bun run i18n:sync` and the i18n skill; every new key translated in every required locale. Keys that become unused may be dropped by the sync; say which.
- Keep each touched file's change small: the open PR #7653 (`origin/pr/fa-dates-numbers`) edits several of the same files.

## Tests
For each row, a test that fails without its fix (fail-before) and passes with it: at least English plus Simplified Chinese and Russian for row 1 (including `2 часа назад`, `21 минуту назад`, and the 28–29 day case), English and one other language for rows 4, 10 and 11. Fixed dates and clock (`vi.setSystemTime`), explicit locales; nothing may depend on the machine's locale or time zone.

## Verify
In `web/`: `bun run typecheck`; `bun run lint`, compared by file and rule with `upstream/main` (changed files add no errors); `bun run test`; `bun run build`; `bun run i18n:sync` (no further changes). Show fail-before per row. Run the new tests again with `LANG=fr_FR.UTF-8 LC_ALL=fr_FR.UTF-8` (show that Node reports `fr`) and with `TZ=Pacific/Kiritimati`. Check that the branch merges cleanly on `upstream/main`, and list its conflicts with `origin/pr/fa-dates-numbers` file by file.

## Pull request text
Write a draft description following `.agents/github/PR.md`: short and factual; for each row the bug, the cause and the fix; the tests and how to check them; what changes per language (the English changes exactly); the follow-ups left out; "Part of #7654" and the rows it covers. Agent fields: the tool, version, model and host you actually ran as (write the model id if you know it); keep the AI-generated disclosure `AGENTS.md` asks for, as "This code was AI-generated. The git author is not one of the project's historical core developers."; quote this prompt in full in a `<details>` block. About 3,500 characters outside the quote at most. Title: `fix(web): relative times, dates and labels in the interface language`. Save it as `.fa-review/pr-web-localization-gaps.md` on `feat/fa-locale` only (one commit with just that file), never on the fix branch.

## Hand-off
1. Push `fix/web-localization-gaps` and the `.fa-review` commit on `feat/fa-locale` to origin (my fork) only.
2. End with a short report in plain fragments: the upstream commit, the branch head and its commits (one line each), the files changed, each row fixed / not fixed and why, keys added or dropped, the English changes, the verification results (fail-before per row, French and time-zone runs), the merge check and the conflicts with `pr/fa-dates-numbers`, and anything left.

## Rules
- Do NOT open a pull request, file an issue, or comment anywhere on GitHub, in either repository.
- Never push to upstream. Never force-push; never delete a branch. Do not touch `pr/*`, `review/*` or `fix/date-picker-chinese-locale`.
- Do not add anything about any specific deployment, business, market or customers; the repository is public.
- Nothing from `.fa-review/` goes into the fix branch; no new files under `docs/`.
- If something in this prompt conflicts with AGENTS.md or web/AGENTS.md, stop and ask me.
````

</details>

- Later corrections: none

## Out of scope — refuse

- Matched: no

## Open gate — do not open unless all are satisfied

- Out of scope: no. Facts, verification, short body: yes. Open: yes (by the user)

## Kind

- [x] Bug fix

## Issue facts

- #7654 rows 1, 2, 4, 10, 11; every render; frontend only.

## Change

- Row 2: calendar map keyed `zh`, codes are `zhCN`/`zhTW`, so Chinese calendars were English; maps the codes.
- Row 1: Day.js `fromNow()` without a locale (`最后活跃于 2 hours ago`), hand-built `5m ago`, `{{count}}` keys (`2 часов назад`, `0 months ago` at 28–29 days). All four now use `formatTimestampRelative()` with `toIntlLocale(i18n.resolvedLanguage || i18n.language)`; the popover keeps `Just now` and its absolute dates.
- Row 4: three `toLocaleString()` dates now use `formatTimestampToDate()`.
- Row 10: carousel labels through `t()`, new keys `Previous slide`/`Next slide` (`Previous`/`Next` read as wizard steps in zh, ru); logo URL error `Must be a valid URL`, not zod's `Invalid URL` (line 53 is type-only).
- Row 11: `SpeechRecognition.lang` from `toIntlLocale(...)`, not `en-US`; unset without a valid tag.

## Research

### Duplicate / prior art

- Read #7654; no search. #7653 is numbers only.

### Docs and code

- docs.newapi.ai, deepwiki: blocked. Code: `lib/format.ts`, `toIntlLocale`.

### Alternatives considered

- Day.js locales or plural keys (rows 5–6); chose existing helpers.

## Files

| Path | Why |
| --- | --- |
| 12 source files | one commit per row |
| 10 tests (9 new) | fail-before |
| 7 locales | 2 keys |

## Behavior

- English, exactly:
  - Sessions, passkey: `a few seconds ago` → `30 seconds ago`, `an hour ago` → `1 hour ago` (also minute, day, month, year); units change at 60 s/60 min/24 h/30 d/365 d, not 45 s/45 min/22 h/26 d/11 mo.
  - Announcements: `5m ago` → `5 minutes ago`; months/years after 30 days; future `-120m ago` → `in 2 hours`.
  - Popover: no weeks (`14 days ago`); `0 months ago` → `28 days ago`; rounds, not floors (90 s `2 minutes ago`, 345–364 d `12 months ago`, 548–729 d `2 years ago`).
  - `11/1/2026, 9:30:00 AM` → `2026-11-01 09:30:00`; no end time → `-`; `Invalid URL` → `Must be a valid URL`; voice `en-US` → `en`.
- Other languages: Intl grammar (`2 часа назад`), Chinese calendars, translated labels, voice.
- Left out: row 3 (#7653); rows 5–6 (plural keys); rows 7–9 (#7651, #7652); `formatUseTime`/`formatTokens`; unused `formatRelativeTime` in `models/lib/model-utils.ts`; 11 popover keys now unused (sync keeps them).

## Verification

- `web/`: `typecheck` 0; `lint` 230 (165 errors) on base and branch, same per file and rule; `test` 2222/2222; `build` ok; `i18n:sync` no diff.
- Fail-before (fix reverted): row 2 6/18, row 1 21/27, row 4 6/92, row 10 3/4, row 11 6/6.
- New tests with `LANG=fr_FR.UTF-8` (Node `fr-FR`), `TZ=Pacific/Kiritimati`: 147/147.
- Not verified: real browsers.

## Risks

- Display only. #7653 conflicts: popover (4 hunks), passkey (3), subscription card (3), login session (2), channel drawer (1).

## Scope check

- Single focused change: yes. Secrets: no. Out of scope: no
