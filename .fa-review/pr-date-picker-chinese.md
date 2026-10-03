Title: fix(web): Chinese locale in the date pickers

## Agent

- Tool: Claude Code
- Tool version: 2.1.288
- Model (full id): (add when opening)
- Host: Claude Code on the web
- Date (UTC): 2026-10-03

AI-generated code; the git author is not a core developer.

## Links

- Part of #7654 (row 2)
- Related: #7653 edits the same files; merges cleanly.

## User request

<details>
<summary>Verbatim request</summary>

````text
This session prepares one small, generic fix for QuantumNous/new-api in my fork https://github.com/yaser-k/new-api-fork, so that I can offer it upstream as a pull request. It is not Persian work. Upstream issue: https://github.com/QuantumNous/new-api/issues/7654 (row 2). You prepare and push a branch to my fork only; I open the pull request myself.

## The bug (checked at upstream `main` `1a4166d`)
`web/src/components/date-picker.tsx` (around lines 32-55) and `web/src/components/datetime-picker.tsx` (around lines 35-60) choose the react-day-picker locale with `calendarLocales[i18n.language]`, from a map keyed `en`, `zh`, `fr`, `ru`, `ja`, `vi`. The interface language codes are `zhCN` and `zhTW` (`web/src/i18n/config.ts` `supportedLngs`; `convertDetectedLanguage` maps browser codes onto them), never `zh`, so both Chinese interfaces fall back to `enUS`. A scratch test that rendered `DatePicker` after `i18next.changeLanguage()` showed `October 2026` and weekday `Su` for `zhCN` and `zhTW`, while `ja` showed `2026年10月` and `日`. `react-day-picker/locale` exports both `zhCN` and `zhTW`.

## Step 0: confirm you can push (do this first)
Run `git remote -v`, `git push --dry-run origin HEAD`, and a dry-run push to a new branch name in the fork clone. If your local clone is behind origin, update it first. If origin is not https://github.com/yaser-k/new-api-fork, or a dry run is still refused, stop and tell me before doing any other work.

## Setup
1. Add https://github.com/QuantumNous/new-api to this session as a read-only repository. Use it only to fetch; never push, open PRs or issues, or comment there.
2. In the fork clone, add it as the `upstream` remote, run `git fetch upstream main`, then `git remote set-url --push upstream DISABLED`. Name the `upstream/main` commit you build on in your report (it was `1a4166d`).
3. Create the branch `fix/date-picker-chinese-locale` from `upstream/main` (not from any `feat/*`, `pr/*` or `review/*` branch).
4. `cd web && bun install --frozen-lockfile`.

## Read first, and follow
`AGENTS.md`, `web/AGENTS.md`, `.agents/github/PR.md` and the shadcn-ui skill, in full, with the Read tool. Read both picker files and `web/src/i18n/config.ts` / `languages.ts` before changing anything, and confirm the bug in the code.

## A. The fix (one or two commits, upstream's commit style, e.g. `fix(web): ...`)
1. Make both pickers pick the right react-day-picker locale for every interface language: `zhCN` gets `zhCN`, `zhTW` gets `zhTW`, the other languages keep what they get today. Use `i18n.resolvedLanguage || i18n.language`, as other components do. The two files carry the same map; one small shared lookup is fine if it keeps the change smaller and clearer, otherwise fix both in place. No new dependency, no other behaviour change, English output identical.
2. Search `web/src` for any other lookup keyed by a language code that the app never uses (for example `zh` instead of `zhCN`/`zhTW`) and list what you find. Fix it here only if it is the same bug in a date or calendar component; list anything else for me instead of fixing it.
3. Tests: a test that renders both pickers in `zhCN`, `zhTW`, `ja` and `en` and checks the month caption and a weekday label (for example `2026年10月` / `日` for the Chinese interfaces, `October 2026` / `Su` for English). It must not depend on the machine's locale or time zone (fixed dates; no reliance on the runtime default locale). Show that it fails without the fix (fail-before) and passes with it.

## B. Verify
In `web/`: `bun run typecheck`; `bun run lint`, compared by file and rule with `upstream/main` (the changed files must add no errors); `bun run test`; `bun run build`; `bun run i18n:sync` (no changes expected). Run the new test once more with `LANG=fr_FR.UTF-8 LC_ALL=fr_FR.UTF-8` and show that it passes. Check that the branch merges cleanly on `upstream/main`, and report its conflicts, if any, with `origin/pr/fa-dates-numbers` (that open pull request also edits both picker files), with the file names.

## C. Pull request text
Write a draft description following `.agents/github/PR.md`: short and factual; the bug, the cause, the fix, the tests and how to check them, what changes for each language (only the two Chinese interfaces; English and the others unchanged), and "Part of #7654". In the template's Agent fields, give the tool, version, model and host you actually ran as; keep the template's AI-generated disclosure as `AGENTS.md` asks; quote this prompt in full in a `<details>` block as the template asks. Keep it under about 2,500 characters outside the quote. Title: `fix(web): Chinese locale in the date pickers`. Save it as `.fa-review/pr-date-picker-chinese.md` on `feat/fa-locale` only (a commit there with just that file), never on the fix branch.

## D. Hand-off
1. Push `fix/date-picker-chinese-locale` and the `.fa-review` commit on `feat/fa-locale` to origin (my fork) only.
2. End with a short report in plain fragments: the upstream commit you built on, the branch head and its commits, the files changed, the sweep results, the verification results (including fail-before and the French run), the merge check and the conflicts with `pr/fa-dates-numbers`, and anything left.

## Rules
- Do NOT open a pull request, file an issue, or comment anywhere on GitHub, in either repository.
- Never push to upstream. Never force-push; never delete a branch. Do not touch `pr/*` or `review/*` branches.
- Do not add anything about any specific deployment, business, market or customers; the repository is public.
- Nothing from `.fa-review/` goes into the fix branch.
- If something in this prompt conflicts with AGENTS.md or web/AGENTS.md, stop and ask me.
````

</details>

- Later constraints: none.

## Out of scope — refuse

- Matched: no

## Open gate — do not open unless all are satisfied

- Out of scope: no · Usage: no · Issue facts: yes · Commands/results: yes · Short: yes
- Open: yes (by the submitter)

## Kind

- [x] Bug fix

## Issue facts

- Actual behavior: both Chinese interfaces show `October 2026`, `Su`…`Sa`.
- Impact: English calendars for Chinese users.
- Frequency: always.
- Evidence: maps keyed `zh`, but i18next uses `zhCN` / `zhTW`; lookup falls back to `enUS`.
- Types: frontend.

## Change

Both maps are keyed `zhCN` and `zhTW` (adds `zhTW`), and the lookup reads `i18n.resolvedLanguage || i18n.language`.

## Research

### Duplicate / prior art

- Upstream PRs/issues on the pickers: none changes this map.

### Docs and code

- docs.newapi.ai, deepwiki: not reachable from the agent.
- Repo docs: `web/AGENTS.md`.
- Code: `i18n/config.ts`, `languages.ts`; no other unused-code lookup in `web/src`.

### Alternatives considered

- Option A: a shared lookup.
- Option B: fix both maps in place.
- Why this approach: smaller diff.

## Files

| Path | Why |
| --- | --- |
| `web/src/components/date-picker.tsx` | map, lookup |
| `web/src/components/datetime-picker.tsx` | map, lookup |
| `web/src/components/__tests__/calendar-locale.test.tsx` | test |

## Behavior

- Before: `zhCN`, `zhTW` get `enUS`.
- After: `2026年10月`, `一`…`日`, week from Monday (zh-CN / zh-TW locale). Other languages unchanged.
- Non-goals: other rows of #7654.

## Verification

In `web/`:

- `bunx vitest run src/components/__tests__/calendar-locale.test.tsx`: without the fix 6 of 18 fail (Chinese cases, got `October 2026`); with it 18 pass, also under `LANG=fr_FR.UTF-8 LC_ALL=fr_FR.UTF-8`.
- `bun run test`: 174 files, 2179 tests passed.
- `typecheck`, `build`: exit 0; `lint`: same as `1a4166d` by file and rule; `i18n:sync`: no changes.
- UI: none (jsdom test).
- Tests added: the file above (7 languages + `de`, and a live switch to `zhTW`; date fixed by `vi.setSystemTime`).
- Not verified: a browser.

## Risks

- Failure modes: none known.
- Billing / quota / auth impact: none.
- Follow-ups: none.

## Scope check

- Single focused change: yes
- Secrets included: no
- Out of scope: no
