Title: fix(web): right-to-left layout in shared components
Branch: yaser-k/new-api-fork:pr/fa-rtl-layout (from upstream main 1a4166d)

## Agent

- Tool: Claude Code
- Tool version: Claude Code 2.1.288
- Model (full id): (add when opening; not stored in the repository)
- Host: Claude Code on the web
- Date (UTC): 2026-10-02

The code was AI-generated and reviewed by the submitter; the git author is not one of the project's core developers.

## Links

- Related: #7198 (Persian locale). Part 1 of 3; merges alone. Order: this PR, then `feat(i18n): add Persian (fa)`, then `feat(web): Solar Hijri dates and numbers in the interface language`.

## User request

- Verbatim (excerpt): "`pr/fa-rtl-layout` | `fix(web): right-to-left layout in shared components` | Plan PR 1: every RTL conversion, its tests and, last, the physical-class guard with an allowlist that matches what this branch ships. No visible change in left-to-right languages."
- Later constraints: none.

## Out of scope — refuse

- Matched: no

## Open gate — do not open unless all are satisfied

- Out of scope: no · Usage question: no · Issue facts present: yes · Verification is commands and results: yes · Short and factual: yes
- Open: yes (by the submitter)

## Kind

- [x] Bug fix

## Issue facts

- Actual behavior: on a right-to-left page (`dir="rtl"`), components that use physical classes (`ml-*`, `pr-*`, `left-*`, `text-left`, `border-l`, `rounded-l`) keep their left-to-right geometry: icons, paddings, badges and alignment sit on the wrong side.
- Impact: any right-to-left language (#7198); left-to-right languages are unaffected.
- Frequency: every page.
- Evidence: the classes are in `web/src`; the config drawer's RTL switch shows it today.
- Applicable types: frontend only.

## Change

- Shared components, then feature pages: physical direction classes become logical ones (`ms/me`, `ps/pe`, `start/end`, `text-start/end`, `border-s/e`, `rounded-s/e`); direction icons get `rtl:` mirroring; code, keys and `pre` blocks keep their own direction (`unicode-bidi: plaintext` on RTL pages); the sidebar docks on the reading side; a few `<bdi>`/`dir` isolates. Lint errors upstream already had in touched files are fixed without behaviour change.
- Last commit: a test that fails on any new physical class not in `physical-direction-allowlist.json` (62 kept on purpose, each with a reason: centred elements, physical-side props, LTR code areas).
- Existing components are reused; no new component.

## Files

| Path | Why |
| --- | --- |
| `web/src/components/**`, `styles/index.css` | shared components (95 files) |
| `web/src/features/**` | feature pages (244 files) |
| `web/src/styles/__tests__/physical-direction-*` | guard and allowlist |

## Behavior

- Before: RTL pages keep LTR geometry.
- After: RTL pages mirror. LTR: no visible change.
- Non-goals: no language is added here.

## Verification

- `bun run typecheck`: exit 0 (each of the 3 commits).
- `bun run lint`: 121 errors vs 165 on upstream main; no file+rule pair above upstream; 0 errors in changed files.
- `bun run test`: 202 files, 2236 tests passed.
- `bun run build`: exit 0. `bun run i18n:sync`: no changes.
- Fail-before: with every non-test file reverted to upstream, the branch's 34 test files: 66 tests fail, 96 pass (the passing ones assert that LTR output is unchanged).
- English pixel comparison, 16 pages, Chromium 141, 1440×900, same database and port: differences only where upstream differs from itself between runs (sign-in time in the audit log, sessions on the security page, vendor order on pricing, anti-aliasing). Table in the fork's hand-off.
- Not verified: browsers other than Chromium.

## Risks

- tailwind-merge does not merge `ps-*` with a later `px-*`; callers were checked.
- Billing / quota / auth impact: none.

## Scope check

- Single focused change: yes · Secrets: no · Out of scope: no
