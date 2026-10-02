Title: feat(i18n): add Persian (fa)
Branch: yaser-k/new-api-fork:pr/fa-locale (from upstream main 1a4166d)

## Agent

- Tool: Claude Code
- Tool version: Claude Code 2.1.288
- Model (full id): (add when opening; not stored in the repository)
- Host: Claude Code on the web
- Date (UTC): 2026-10-02

The code was AI-generated and reviewed by the submitter; the git author is not one of the project's core developers.

## Links

- Closes #7198
- Part 2 of 3; merges alone. Order: `fix(web): right-to-left layout in shared components`, this PR, then `feat(web): Solar Hijri dates and numbers in the interface language`. Alone, Persian works but some components keep LTR geometry.

## User request

- Verbatim (excerpt): "`pr/fa-locale` | `feat(i18n): add Persian (fa)` | Plan PRs 2, 6 and 4: `fa.json`, the language registration (switcher, direction provider), the check-fa script and its tests, the `@babel/parser` dev dependency, the AGENTS.md, web/AGENTS.md and i18n-skill changes, `docs/i18n/fa.md`, and the Persian-only audit labels."
- Later constraints: none.

## Out of scope — refuse

- Matched: no

## Open gate — do not open unless all are satisfied

- Out of scope: no · Usage question: no · Issue facts present: yes (#7198) · Verification is commands and results: yes · Short and factual: yes
- Open: yes (by the submitter)

## Kind

- [x] New feature

## Issue facts

- Actual behavior: no Persian interface; see #7198.
- Applicable types: frontend only.

## Change

1. Registration: `fa` («فارسی», `dir: 'rtl'`) in `INTERFACE_LANGUAGE_OPTIONS`, browser detection, `toIntlLocale` (`PERSIAN_INTL_LOCALE`). `fa` is a partial locale: a missing key falls back to English per key, and `i18n:sync` reports missing Persian keys without filling them. The direction provider follows the interface language and sets `dir`/`lang` on `<html>` (the config drawer can still force LTR/RTL); the sidebar docks on the reading side. Vazirmatn for Persian text. `fa.json`: 5808 keys.
2. Direction-aware JS: before/after arrows and the admin retry chain read in the page direction (`formatValueChange`/`formatValueChain`); in Persian, audit text shows translated roles, sign-in methods, the ID label and user action names. Other languages keep their current output (pinned by tests).
3. Tooling: `bun run i18n:check-fa` (typography rules, isolate pairing, `dir='ltr'` placeholders parsed with `@babel/parser`, MIT dev dependency); `docs/i18n/fa.md` (glossary, typography, date rules); AGENTS.md, web/AGENTS.md and the i18n skill name `fa` as the optional partial locale. The date rules in `fa.md` take effect with the third PR.

## Files

| Path | Why |
| --- | --- |
| `web/src/i18n/*`, `locales/fa.json`, `scripts/sync-i18n.mjs` | registration, translations, partial locale |
| `web/src/context/direction-provider.tsx`, `styles/index.css`, `app-sidebar.tsx` | direction, font, sidebar |
| `web/src/features/{usage-logs,users,task-plugins,security}/**` | arrows, Persian audit labels |
| `web/scripts/check-fa.mjs`, `docs/i18n/fa.md`, agent rules | tooling and guide |

## Behavior

- Before: no Persian.
- After: the language list gains «فارسی». No other language changes.
- Non-goals: Solar Hijri dates and number formatting (third PR).

## Verification

- `bun run typecheck`: exit 0 (each of the 3 commits).
- `bun run lint`: 163 errors vs 165 on upstream main; no file+rule pair above upstream; 0 errors in changed files.
- `bun run test`: 188 files, 2284 tests passed. `bun run build`: exit 0.
- `bun run i18n:sync`: no changes. `bun run i18n:check-fa`: `5808 keys, no findings`.
- Fail-before: with every non-test file reverted to upstream, 30 of the branch's 63 tests fail in 14 of 15 files; the passing file adds `fa` cases to the existing `intl-locale` lint test.
- Screenshots (Persian and English) of the touched pages are in the fork's hand-off.

## Risks

- A missing Persian key shows English (by design).
- Billing / quota / auth impact: none (labels only).

## Scope check

- Single focused change: yes (one language) · Secrets: no · Out of scope: no
