# Coverage: feat/fa-locale against the three upstream branches

`git diff upstream/main feat/fa-locale` (upstream main `1a4166d`), without `.fa-review/`: 454 files. An `x` means the branch changes the file.
Branch file counts: RTL 341, locale 44, dates/numbers 119.

Check: merging `pr/fa-rtl-layout`, `pr/fa-locale` and `pr/fa-dates-numbers` into upstream main in that order (the first two merge cleanly; the third conflicts in 9 files, resolved with feat's version) gives feat/fa-locale exactly, except the rows marked "left out". A line-level check found every added line of feat's diff in at least one branch, except the left-out lines, 5 calendar lines and 2 import lines that combine two branches' edits of the same line, and the license comment move.

| File | Status | RTL | Locale | Dates/numbers | Note |
| --- | --- | :-: | :-: | :-: | --- |
| `.agents/skills/i18n-translate/SKILL.md` | M |  | x |  |  |
| `AGENTS.md` | M |  | x |  |  |
| `docs/i18n/fa.md` | A |  | x |  |  |
| `web/AGENTS.md` | M |  | x |  |  |
| `web/bun.lock` | M |  | x | x |  |
| `web/package.json` | M |  | x | x |  |
| `web/scripts/__tests__/check-fa.test.ts` | A |  | x |  |  |
| `web/scripts/check-fa.mjs` | A |  | x |  |  |
| `web/scripts/oxlint/__tests__/intl-locale.test.ts` | M |  | x |  |  |
| `web/scripts/sync-i18n.mjs` | M |  | x |  |  |
| `web/src/components/__tests__/activity-time-cell-dates.test.tsx` | A |  |  | x |  |
| `web/src/components/__tests__/date-picker-display.test.tsx` | A |  |  | x |  |
| `web/src/components/__tests__/permission-matrix-direction.test.tsx` | A | x |  |  |  |
| `web/src/components/activity-time-cell.tsx` | M |  |  | x |  |
| `web/src/components/ai-elements/__tests__/code-block-direction.test.tsx` | A | x |  |  |  |
| `web/src/components/ai-elements/__tests__/response-table-direction.test.tsx` | A | x |  |  |  |
| `web/src/components/ai-elements/chain-of-thought.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/code-block.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/context.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/inline-citation.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/open-in-chat.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/prompt-input.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/queue.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/reasoning.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/response-renderer-alert.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/response-renderer-blocks.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/response-renderer-details.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/response-renderer-footnotes.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/response-renderer-table.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/response-renderer.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/sources.tsx` | M | x |  |  |  |
| `web/src/components/ai-elements/task.tsx` | M | x |  |  |  |
| `web/src/components/config-drawer.tsx` | M | x |  |  |  |
| `web/src/components/data-table/core/__tests__/column-pinning.test.ts` | A | x |  |  |  |
| `web/src/components/data-table/core/__tests__/column-resize-direction.test.tsx` | A | x |  |  |  |
| `web/src/components/data-table/core/__tests__/pagination.test.tsx` | M | x |  | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/components/data-table/core/badge-cell.tsx` | M | x |  |  |  |
| `web/src/components/data-table/core/badge-list-cell.tsx` | M | x |  |  |  |
| `web/src/components/data-table/core/column-pinning.ts` | M | x |  |  |  |
| `web/src/components/data-table/core/data-table-header.tsx` | M | x |  |  |  |
| `web/src/components/data-table/core/pagination.tsx` | M | x |  | x |  |
| `web/src/components/data-table/hooks/use-data-table.ts` | M | x |  |  |  |
| `web/src/components/data-table/index.ts` | M | x |  |  |  |
| `web/src/components/data-table/layout/card-row-content.tsx` | M | x |  |  |  |
| `web/src/components/data-table/static/static-data-table-classnames.ts` | M | x |  |  |  |
| `web/src/components/data-table/toolbar/mobile-filter-panel.tsx` | M | x |  |  |  |
| `web/src/components/date-picker.tsx` | M |  |  | x |  |
| `web/src/components/datetime-picker.tsx` | M |  |  | x |  |
| `web/src/components/floating-window.tsx` | M | x |  |  |  |
| `web/src/components/json-code-editor.tsx` | M | x |  |  |  |
| `web/src/components/json-code-editor/__tests__/json-code-editor-direction.test.tsx` | A | x |  |  |  |
| `web/src/components/json-editor.tsx` | M | x |  |  |  |
| `web/src/components/layout/components/__tests__/app-sidebar-direction.test.tsx` | A | x |  |  |  |
| `web/src/components/layout/components/__tests__/app-sidebar-persian.test.tsx` | A |  | x |  |  |
| `web/src/components/layout/components/app-sidebar.tsx` | M | x | x |  |  |
| `web/src/components/layout/components/chat-presets-item.tsx` | M | x |  |  |  |
| `web/src/components/layout/components/footer.tsx` | M | x |  |  |  |
| `web/src/components/layout/components/nav-group.tsx` | M | x |  |  |  |
| `web/src/components/layout/components/public-header.tsx` | M | x |  |  |  |
| `web/src/components/layout/components/sidebar-view-header.tsx` | M | x |  |  |  |
| `web/src/components/masked-value-display.tsx` | M | x |  |  |  |
| `web/src/components/model-group-selector.tsx` | M | x |  |  |  |
| `web/src/components/model-group-selector/layout.ts` | M | x |  |  |  |
| `web/src/components/multi-select.tsx` | M | x |  |  |  |
| `web/src/components/notification-popover.tsx` | M | x |  | x |  |
| `web/src/components/permission-matrix.tsx` | M | x |  |  |  |
| `web/src/components/provider-badge.tsx` | M | x |  |  |  |
| `web/src/components/quota-details-popover.tsx` | M | x |  |  |  |
| `web/src/components/risk-acknowledgement-dialog.tsx` | M | x |  |  |  |
| `web/src/components/tag-input.tsx` | M | x |  |  |  |
| `web/src/components/ui/__tests__/calendar-persian.test.tsx` | A |  |  | x |  |
| `web/src/components/ui/__tests__/carousel-direction.test.tsx` | A | x |  |  |  |
| `web/src/components/ui/__tests__/rtl-layout.test.tsx` | A | x |  |  |  |
| `web/src/components/ui/accordion.tsx` | M | x |  |  |  |
| `web/src/components/ui/alert-dialog.tsx` | M | x |  |  |  |
| `web/src/components/ui/alert.tsx` | M | x |  |  |  |
| `web/src/components/ui/avatar.tsx` | M | x |  |  |  |
| `web/src/components/ui/badge.tsx` | M | x |  |  |  |
| `web/src/components/ui/breadcrumb.tsx` | M | x |  |  |  |
| `web/src/components/ui/button-group.tsx` | M | x |  |  |  |
| `web/src/components/ui/button.tsx` | M | x |  |  |  |
| `web/src/components/ui/calendar.tsx` | M | x |  | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/components/ui/carousel.tsx` | M | x |  |  |  |
| `web/src/components/ui/combobox-input.tsx` | M | x |  |  |  |
| `web/src/components/ui/combobox.tsx` | M | x |  |  |  |
| `web/src/components/ui/command.tsx` | M | x |  |  |  |
| `web/src/components/ui/context-menu.tsx` | M | x |  |  |  |
| `web/src/components/ui/dialog.tsx` | M | x |  |  |  |
| `web/src/components/ui/drawer.tsx` | M | x |  |  |  |
| `web/src/components/ui/dropdown-menu.tsx` | M | x |  |  |  |
| `web/src/components/ui/field.tsx` | M | x |  |  |  |
| `web/src/components/ui/input-group.tsx` | M | x |  |  |  |
| `web/src/components/ui/input-otp.tsx` | M | x |  |  |  |
| `web/src/components/ui/item.tsx` | M | x |  |  |  |
| `web/src/components/ui/markdown.tsx` | M | x |  |  |  |
| `web/src/components/ui/menubar.tsx` | M | x |  |  |  |
| `web/src/components/ui/native-select.tsx` | M | x |  |  |  |
| `web/src/components/ui/navigation-menu.tsx` | M | x |  |  |  |
| `web/src/components/ui/pagination.tsx` | M | x |  |  |  |
| `web/src/components/ui/progress.tsx` | M | x |  |  |  |
| `web/src/components/ui/scroll-area.tsx` | M | x |  |  |  |
| `web/src/components/ui/select.tsx` | M | x |  |  |  |
| `web/src/components/ui/sheet.tsx` | M | x |  |  |  |
| `web/src/components/ui/sidebar.tsx` | M | x |  |  |  |
| `web/src/components/ui/switch.tsx` | M | x |  |  |  |
| `web/src/components/ui/table.tsx` | M | x |  |  |  |
| `web/src/components/ui/tabs.tsx` | M | x |  |  |  |
| `web/src/components/ui/toggle-group.tsx` | M | x |  |  |  |
| `web/src/components/ui/toggle.tsx` | M | x |  |  |  |
| `web/src/components/ui/tooltip.tsx` | M | x |  |  |  |
| `web/src/context/__tests__/direction-provider.test.tsx` | A |  | x |  |  |
| `web/src/context/direction-provider.tsx` | M |  | x |  |  |
| `web/src/features/auth/auth-layout.tsx` | M | x |  |  |  |
| `web/src/features/auth/components/__tests__/legal-consent-sentence.test.tsx` | A |  |  |  | left out: legal-consent fix (overlaps open PR #5998) |
| `web/src/features/auth/components/__tests__/legal-consent.test.tsx` | A | x |  |  | RTL case in RTL branch; Persian sentence case left out |
| `web/src/features/auth/components/__tests__/terms-footer-persian.test.tsx` | A |  |  |  | left out: legal-consent fix (overlaps open PR #5998) |
| `web/src/features/auth/components/__tests__/terms-footer-sentence.test.tsx` | A |  |  |  | left out: legal-consent fix (overlaps open PR #5998) |
| `web/src/features/auth/components/legal-consent.tsx` | M | x |  |  | RTL class in RTL branch; one-sentence consent left out |
| `web/src/features/auth/components/terms-footer.tsx` | M |  |  |  | left out: legal-consent fix (overlaps open PR #5998) |
| `web/src/features/auth/forgot-password/components/__tests__/email-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/auth/forgot-password/components/forgot-password-form.tsx` | M | x |  |  |  |
| `web/src/features/auth/forgot-password/index.tsx` | M | x |  |  |  |
| `web/src/features/auth/otp/index.tsx` | M | x |  |  |  |
| `web/src/features/auth/reset-password-confirm/index.tsx` | M | x |  |  |  |
| `web/src/features/auth/sign-in/components/user-auth-form.tsx` | M | x |  |  |  |
| `web/src/features/auth/sign-in/index.tsx` | M | x |  |  |  |
| `web/src/features/auth/sign-up/components/sign-up-form.tsx` | M | x |  |  |  |
| `web/src/features/auth/sign-up/index.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/__tests__/response-time-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/channels/components/channel-card.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/channels-columns.tsx` | M | x |  | x |  |
| `web/src/features/channels/components/channels-primary-buttons.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/data-table-row-actions.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/data-table-tag-row-actions.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/advanced-custom-editor-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/balance-query-dialog.tsx` | M | x |  | x |  |
| `web/src/features/channels/components/dialogs/channel-test-dialog.tsx` | M | x |  | x |  |
| `web/src/features/channels/components/dialogs/codex-usage-dialog.tsx` | M | x |  | x |  |
| `web/src/features/channels/components/dialogs/copy-channel-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/edit-tag-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/fetch-models-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/multi-key-manage-dialog.tsx` | M | x |  | x |  |
| `web/src/features/channels/components/dialogs/ollama-models-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/param-override-editor-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/passthrough-warning-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/tag-batch-edit-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/dialogs/upstream-update-dialog.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/drawers/channel-mutate-drawer.tsx` | M | x |  | x |  |
| `web/src/features/channels/components/drawers/channel-provider-picker.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/drawers/sections/channel-advanced-section.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/model-mapping-editor.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/numeric-spinner-input.tsx` | M | x |  |  |  |
| `web/src/features/channels/components/upstream-model-selection.tsx` | M | x |  |  |  |
| `web/src/features/channels/lib/__tests__/response-time-format.test.ts` | A |  |  | x |  |
| `web/src/features/channels/lib/channel-utils.ts` | M |  |  | x |  |
| `web/src/features/dashboard/components/flow/flow-charts.tsx` | M | x |  | x |  |
| `web/src/features/dashboard/components/flow/flow-node-filter.tsx` | M | x |  |  |  |
| `web/src/features/dashboard/components/models/__tests__/preferences-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/dashboard/components/models/__tests__/total-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/dashboard/components/models/consumption-distribution-chart.tsx` | M | x |  | x |  |
| `web/src/features/dashboard/components/models/log-stat-cards.tsx` | M |  |  | x |  |
| `web/src/features/dashboard/components/models/model-charts.tsx` | M |  |  | x |  |
| `web/src/features/dashboard/components/models/models-chart-preferences.tsx` | M | x |  |  |  |
| `web/src/features/dashboard/components/models/models-filter-dialog.tsx` | M | x |  |  |  |
| `web/src/features/dashboard/components/overview/__tests__/setup-guide.test.tsx` | M | x |  |  |  |
| `web/src/features/dashboard/components/overview/__tests__/summary-cards-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/dashboard/components/overview/announcement-detail-dialog.tsx` | M | x |  | x |  |
| `web/src/features/dashboard/components/overview/announcements-panel.tsx` | M | x |  | x |  |
| `web/src/features/dashboard/components/overview/overview-dashboard.tsx` | M | x |  |  |  |
| `web/src/features/dashboard/components/overview/performance-health-panel.tsx` | M | x |  |  |  |
| `web/src/features/dashboard/components/overview/summary-cards.tsx` | M | x |  | x |  |
| `web/src/features/dashboard/components/users/user-charts.tsx` | M |  |  | x |  |
| `web/src/features/dashboard/index.tsx` | M | x |  |  |  |
| `web/src/features/dashboard/lib/__tests__/chart-number-locale.test.ts` | A |  |  | x |  |
| `web/src/features/dashboard/lib/__tests__/chart-time-locale.test.ts` | A |  |  | x |  |
| `web/src/features/dashboard/lib/__tests__/flow-number-locale.test.ts` | A |  |  | x |  |
| `web/src/features/dashboard/lib/charts.ts` | M |  |  | x |  |
| `web/src/features/dashboard/lib/flow.ts` | M |  |  | x |  |
| `web/src/features/dashboard/types.ts` | M |  |  | x |  |
| `web/src/features/home/components/__tests__/arrow-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/home/components/__tests__/terminal-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/home/components/hero-buttons.tsx` | M | x |  |  |  |
| `web/src/features/home/components/hero-terminal-demo.tsx` | M | x |  |  |  |
| `web/src/features/home/components/sections/cta.tsx` | M | x |  |  |  |
| `web/src/features/home/components/sections/features.tsx` | M | x |  |  |  |
| `web/src/features/home/components/sections/hero.tsx` | M | x |  |  |  |
| `web/src/features/home/components/sections/how-it-works.tsx` | M | x |  |  |  |
| `web/src/features/keys/components/__tests__/api-key-listing.test.tsx` | M | x |  | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/features/keys/components/api-key-group-cell.tsx` | M | x |  |  |  |
| `web/src/features/keys/components/api-key-quota-cell.tsx` | M | x |  | x |  |
| `web/src/features/keys/components/api-keys-cells.tsx` | M | x |  |  |  |
| `web/src/features/keys/components/api-keys-columns.tsx` | M | x |  |  |  |
| `web/src/features/keys/components/api-keys-mutate-drawer.tsx` | M | x |  |  |  |
| `web/src/features/keys/components/api-keys-primary-buttons.tsx` | M | x |  |  |  |
| `web/src/features/keys/components/auto-group-order-editor.tsx` | M | x |  |  |  |
| `web/src/features/keys/components/data-table-row-actions.tsx` | M | x |  |  |  |
| `web/src/features/models/components/data-table-row-actions.tsx` | M | x |  |  |  |
| `web/src/features/models/components/deployment-access-guard.tsx` | M | x |  |  |  |
| `web/src/features/models/components/deployments-columns.tsx` | M | x |  | x |  |
| `web/src/features/models/components/description-cell.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/description-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/extend-deployment-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/missing-models-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/prefill-group-management-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/rename-deployment-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/sync-wizard-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/update-config-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/vendor-mutate-dialog.tsx` | M | x |  | x |  |
| `web/src/features/models/components/dialogs/view-details-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/dialogs/view-logs-dialog.tsx` | M | x |  |  |  |
| `web/src/features/models/components/drawers/model-mutate-drawer.tsx` | M | x |  |  |  |
| `web/src/features/models/components/drawers/prefill-group-form-drawer.tsx` | M | x |  |  |  |
| `web/src/features/models/components/models-columns.tsx` | M | x |  | x |  |
| `web/src/features/models/components/vendors-table.tsx` | M | x |  | x |  |
| `web/src/features/models/lib/model-utils.ts` | M |  |  | x |  |
| `web/src/features/playground/components/chat/playground-empty-state.tsx` | M | x |  |  |  |
| `web/src/features/playground/components/input/playground-input-tools.tsx` | M | x |  |  |  |
| `web/src/features/playground/components/input/playground-parameter-panel.tsx` | M | x |  |  |  |
| `web/src/features/playground/components/message/__tests__/message-duration-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/playground/components/message/message-error.tsx` | M | x |  |  |  |
| `web/src/features/playground/components/message/message-metadata.tsx` | M |  |  | x |  |
| `web/src/features/playground/lib/__tests__/message-alignment.test.ts` | A | x |  |  |  |
| `web/src/features/playground/lib/message/message-layout-utils.ts` | M | x |  |  |  |
| `web/src/features/playground/lib/message/message-styles.ts` | M | x |  |  |  |
| `web/src/features/playground/lib/parameters/playground-parameters.ts` | M | x |  |  |  |
| `web/src/features/pricing/__tests__/api-tab-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/pricing/__tests__/value-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/pricing/components/dynamic-pricing-breakdown.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/model-details-api.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/model-details-apps.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/model-details-uptime-sparkline.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/model-details.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/model-perf-badge.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/pricing-columns.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/pricing-sidebar.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/pricing-toolbar.tsx` | M | x |  |  |  |
| `web/src/features/pricing/components/search-bar.tsx` | M | x |  |  |  |
| `web/src/features/profile/components/__tests__/profile-header-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/profile/components/__tests__/profile-header-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/profile/components/checkin-calendar-card.tsx` | M | x |  | x |  |
| `web/src/features/profile/components/profile-header.tsx` | M | x |  | x |  |
| `web/src/features/profile/components/sidebar-modules-card.tsx` | M | x |  |  |  |
| `web/src/features/profile/components/tabs/notification-tab.tsx` | M | x |  |  |  |
| `web/src/features/rankings/components/market-share-section.tsx` | M | x |  |  |  |
| `web/src/features/rankings/components/model-leaderboard.tsx` | M | x |  |  |  |
| `web/src/features/rankings/components/models-section.tsx` | M | x |  |  |  |
| `web/src/features/redemption-codes/components/data-table-row-actions.tsx` | M | x |  |  |  |
| `web/src/features/redemption-codes/components/redemptions-columns.tsx` | M | x |  | x |  |
| `web/src/features/redemption-codes/components/redemptions-export-dialog.tsx` | M | x |  |  |  |
| `web/src/features/security/components/__tests__/access-token-dates.test.tsx` | A |  |  | x |  |
| `web/src/features/security/components/__tests__/login-session-dates.test.tsx` | A |  |  | x |  |
| `web/src/features/security/components/access-token-item.tsx` | M |  |  | x |  |
| `web/src/features/security/components/access-tokens-card.tsx` | M | x |  | x |  |
| `web/src/features/security/components/account-bindings.tsx` | M | x |  |  |  |
| `web/src/features/security/components/dialogs/access-token-edit-dialog.tsx` | M | x |  |  |  |
| `web/src/features/security/components/dialogs/delete-account-dialog.tsx` | M | x |  |  |  |
| `web/src/features/security/components/dialogs/two-fa-backup-dialog.tsx` | M | x |  |  |  |
| `web/src/features/security/components/dialogs/two-fa-setup-dialog.tsx` | M | x |  |  |  |
| `web/src/features/security/components/login-session-item.tsx` | M |  |  | x |  |
| `web/src/features/security/components/login-session-utils.ts` | M |  | x |  |  |
| `web/src/features/security/components/passkey-card.tsx` | M | x |  | x |  |
| `web/src/features/security/components/two-fa-card.tsx` | M | x |  |  |  |
| `web/src/features/setup/__tests__/step-number-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/setup/components/complete-step.tsx` | M | x |  |  |  |
| `web/src/features/setup/components/database-step.tsx` | M | x |  |  |  |
| `web/src/features/setup/components/step-navigation.tsx` | M | x |  |  |  |
| `web/src/features/setup/components/usage-mode-step.tsx` | M | x |  |  |  |
| `web/src/features/setup/setup-wizard.tsx` | M | x |  | x |  |
| `web/src/features/subscriptions/components/__tests__/list-number-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/subscriptions/components/__tests__/purchase-price-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/subscriptions/components/data-table-row-actions.tsx` | M | x |  |  |  |
| `web/src/features/subscriptions/components/dialogs/subscription-purchase-dialog.tsx` | M |  |  | x |  |
| `web/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx` | M | x |  | x |  |
| `web/src/features/subscriptions/components/subscriptions-columns.tsx` | M | x |  | x |  |
| `web/src/features/subscriptions/lib/__tests__/format-locale.test.ts` | A |  |  | x |  |
| `web/src/features/subscriptions/lib/format.ts` | M |  |  | x |  |
| `web/src/features/system-info/__tests__/number-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/system-info/__tests__/title-dates.test.tsx` | A |  |  | x |  |
| `web/src/features/system-info/components/system-instances-panel.tsx` | M | x |  | x |  |
| `web/src/features/system-info/components/system-tasks-panel.tsx` | M |  |  | x |  |
| `web/src/features/system-info/components/system-tasks-table.tsx` | M | x |  | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/features/system-settings/auth/basic-auth-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/auth/bot-protection-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/auth/custom-oauth/components/discovery-button.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/auth/custom-oauth/components/preset-selector.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/auth/custom-oauth/components/provider-form-dialog.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/auth/custom-oauth/components/provider-table.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/auth/oauth-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/auth/passkey-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/components/settings-accordion.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/components/settings-form-layout.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/content/announcements-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/content/api-info-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/content/chat-dialog.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/content/chat-settings-visual-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/content/faq-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/content/uptime-kuma-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/general/channel-affinity/__tests__/advanced-toggle-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/system-settings/general/channel-affinity/cache-stats-dialog.tsx` | M | x |  | x |  |
| `web/src/features/system-settings/general/channel-affinity/rule-editor-dialog.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/general/channel-affinity/session-rules-table.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/general/quota-settings-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/general/system-info-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/amount-discount-dialog.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/amount-discount-visual-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/amount-options-visual-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/creem-product-dialog.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/creem-products-visual-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/email-settings-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/ionet-deployment-settings-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/payment-method-dialog.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/payment-methods-visual-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/payment-settings-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/waffo-pancake-settings-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/waffo-settings-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/integrations/worker-settings-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/maintenance/log-settings-section.tsx` | M | x |  | x |  |
| `web/src/features/system-settings/maintenance/performance-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/maintenance/update-checker-section.tsx` | M |  |  | x |  |
| `web/src/features/system-settings/models/channel-selector-dialog.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/group-ratio-form.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/group-ratio-visual-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/group-special-usable-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/model-pricing-sheet.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/model-ratio-form.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/model-ratio-table-columns.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/request-simulation.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/task-pricing-matrix.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/task-usage-pricing-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/tier-price-fields.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/tiered-pricing-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/tool-price-settings.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/upstream-ratio-sync-columns.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/upstream-ratio-sync-table.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/visual-billing-document-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/models/visual-condition-tree.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/request-limits/rate-limit-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/request-limits/rate-limit-visual-editor.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/request-limits/ssrf-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/request-policies/channel-health-section.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/request-policies/decision-record.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/request-policies/related-policy-link.tsx` | M | x |  |  |  |
| `web/src/features/system-settings/request-policies/retry-section.tsx` | M | x |  |  |  |
| `web/src/features/system-update/system-update-dialog.tsx` | M |  |  | x |  |
| `web/src/features/task-plugins/__tests__/change-arrow-direction.test.tsx` | A |  | x |  |  |
| `web/src/features/task-plugins/__tests__/count-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/task-plugins/__tests__/source-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/task-plugins/components/marketplace-install-dialog.tsx` | M | x |  |  |  |
| `web/src/features/task-plugins/components/marketplace-plugin-card.tsx` | M |  | x | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/features/task-plugins/components/marketplace-sources-dialog.tsx` | M | x |  |  |  |
| `web/src/features/task-plugins/components/plugin-card.tsx` | M |  |  | x |  |
| `web/src/features/task-plugins/components/plugin-detail-sheet.tsx` | M | x |  |  |  |
| `web/src/features/task-plugins/components/plugin-endpoints.tsx` | M | x |  |  |  |
| `web/src/features/task-plugins/components/plugin-metadata-card.tsx` | M | x |  | x |  |
| `web/src/features/task-plugins/components/plugin-model-list.tsx` | M |  |  | x |  |
| `web/src/features/task-plugins/components/plugin-url-import-field.tsx` | M | x |  |  |  |
| `web/src/features/task-plugins/components/plugins-table.tsx` | M | x |  | x |  |
| `web/src/features/task-plugins/components/source-diff.tsx` | M | x |  |  |  |
| `web/src/features/task-plugins/components/usage-schema-table.tsx` | M |  | x |  |  |
| `web/src/features/task-plugins/index.tsx` | M | x |  | x |  |
| `web/src/features/usage-logs/audit/__tests__/details-locale.test.ts` | A |  | x |  |  |
| `web/src/features/usage-logs/audit/__tests__/expiry-date-locale.test.ts` | A |  |  | x |  |
| `web/src/features/usage-logs/audit/__tests__/identity-action-locale.test.ts` | A |  | x |  |  |
| `web/src/features/usage-logs/audit/__tests__/viewer.test.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/audit/components/audit-detail-fields.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/audit/components/audit-log-columns.tsx` | M | x |  | x |  |
| `web/src/features/usage-logs/audit/components/audit-log-details-dialog.tsx` | M |  |  | x |  |
| `web/src/features/usage-logs/audit/lib/audit-details.ts` | M |  | x | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/features/usage-logs/components/__tests__/copy-button-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/usage-logs/components/__tests__/cost-display.test.tsx` | M |  |  | x |  |
| `web/src/features/usage-logs/components/__tests__/date-range-dates.test.tsx` | A |  |  | x |  |
| `web/src/features/usage-logs/components/__tests__/date-range-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/usage-logs/components/__tests__/detail-preview.test.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/__tests__/manage-operator-locale.test.tsx` | A |  | x |  |  |
| `web/src/features/usage-logs/components/__tests__/retry-chain-direction.test.tsx` | A |  | x |  |  |
| `web/src/features/usage-logs/components/__tests__/timing-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/usage-logs/components/columns/column-helpers.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/columns/common-logs-columns.tsx` | M | x | x | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/features/usage-logs/components/columns/drawing-logs-columns.tsx` | M | x |  | x |  |
| `web/src/features/usage-logs/components/columns/task-logs-columns.tsx` | M | x |  | x |  |
| `web/src/features/usage-logs/components/common-log-mobile-card.tsx` | M | x |  | x |  |
| `web/src/features/usage-logs/components/common-logs-stats.tsx` | M |  |  | x |  |
| `web/src/features/usage-logs/components/compact-date-time-range-picker.tsx` | M | x |  | x |  |
| `web/src/features/usage-logs/components/dialogs/audio-preview-dialog.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/dialogs/details-dialog.tsx` | M | x | x | x | left out: move of the duplicated upstream license comment (formatter noise); rebases after RTL/locale |
| `web/src/features/usage-logs/components/dialogs/fail-reason-dialog.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/dialogs/image-dialog.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/dialogs/prompt-dialog.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/dialogs/task-details-dialog.tsx` | M | x |  | x |  |
| `web/src/features/usage-logs/components/dialogs/user-info-dialog.tsx` | M |  |  | x |  |
| `web/src/features/usage-logs/components/log-cost-display.tsx` | M |  |  | x |  |
| `web/src/features/usage-logs/components/logs-filter-toolbar.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/model-badge.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/task-artifacts.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/timing-metrics-cell.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/components/usage-logs-mobile-card.tsx` | M | x |  |  |  |
| `web/src/features/usage-logs/lib/__tests__/audit-content-locale.test.ts` | A |  | x |  |  |
| `web/src/features/usage-logs/lib/__tests__/quota-audit-direction.test.ts` | A |  | x |  |  |
| `web/src/features/usage-logs/lib/__tests__/quota-audit-number-locale.test.ts` | A |  |  | x |  |
| `web/src/features/usage-logs/lib/format.ts` | M |  | x |  |  |
| `web/src/features/usage-logs/lib/quota-audit-operation.ts` | M |  | x | x | dates/numbers rebases after RTL/locale (neighbouring lines) |
| `web/src/features/users/components/__tests__/adjust-quota-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/users/components/__tests__/quota-preview-direction.test.tsx` | A |  | x |  |  |
| `web/src/features/users/components/data-table-row-actions.tsx` | M | x | x |  |  |
| `web/src/features/users/components/dialogs/user-binding-dialog.tsx` | M | x |  |  |  |
| `web/src/features/users/components/user-quota-cell.tsx` | M | x |  |  |  |
| `web/src/features/users/components/user-quota-dialog.tsx` | M |  | x |  |  |
| `web/src/features/users/components/users-columns.tsx` | M | x |  |  |  |
| `web/src/features/users/components/users-mutate-drawer.tsx` | M | x |  |  |  |
| `web/src/features/users/lib/__tests__/user-action-name.test.ts` | A |  | x |  |  |
| `web/src/features/users/lib/index.ts` | M |  | x |  |  |
| `web/src/features/users/lib/user-actions.ts` | M |  | x |  |  |
| `web/src/features/wallet/components/__tests__/amount-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/wallet/components/__tests__/plan-price-locale.test.tsx` | A |  |  | x |  |
| `web/src/features/wallet/components/__tests__/spinner-direction.test.tsx` | A | x |  |  |  |
| `web/src/features/wallet/components/affiliate-rewards-card.tsx` | M |  |  | x |  |
| `web/src/features/wallet/components/creem-products-section.tsx` | M | x |  | x |  |
| `web/src/features/wallet/components/dialogs/billing-history-dialog.tsx` | M | x |  | x |  |
| `web/src/features/wallet/components/dialogs/creem-confirm-dialog.tsx` | M | x |  | x |  |
| `web/src/features/wallet/components/dialogs/payment-confirm-dialog.tsx` | M | x |  | x |  |
| `web/src/features/wallet/components/dialogs/transfer-dialog.tsx` | M | x |  | x |  |
| `web/src/features/wallet/components/recharge-form-card.tsx` | M | x |  | x |  |
| `web/src/features/wallet/components/subscription-plans-card.tsx` | M | x |  | x |  |
| `web/src/features/wallet/components/wallet-stats-card.tsx` | M |  |  | x |  |
| `web/src/features/wallet/hooks/use-redemption.ts` | M |  |  | x |  |
| `web/src/features/wallet/lib/__tests__/format-currency-locale.test.ts` | A |  |  | x |  |
| `web/src/features/wallet/lib/billing.ts` | M |  |  | x |  |
| `web/src/features/wallet/lib/format.ts` | M |  |  | x |  |
| `web/src/i18n/__tests__/languages.test.ts` | A |  | x |  |  |
| `web/src/i18n/__tests__/value-change.test.ts` | A |  | x |  |  |
| `web/src/i18n/config.ts` | M |  | x |  |  |
| `web/src/i18n/languages.ts` | M |  | x | x |  |
| `web/src/i18n/locales/en.json` | M |  |  |  | left out: the 9 legal-consent keys only |
| `web/src/i18n/locales/fa.json` | A |  | x |  | locale branch without the 9 legal-consent keys |
| `web/src/i18n/locales/fr.json` | M |  |  |  | left out: the 9 legal-consent keys only |
| `web/src/i18n/locales/ja.json` | M |  |  |  | left out: the 9 legal-consent keys only |
| `web/src/i18n/locales/ru.json` | M |  |  |  | left out: the 9 legal-consent keys only |
| `web/src/i18n/locales/vi.json` | M |  |  |  | left out: the 9 legal-consent keys only |
| `web/src/i18n/locales/zh-TW.json` | M |  |  |  | left out: the 9 legal-consent keys only |
| `web/src/i18n/locales/zh.json` | M |  |  |  | left out: the 9 legal-consent keys only |
| `web/src/i18n/static-keys.ts` | M |  |  |  | left out: legal-consent keys only |
| `web/src/lib/__tests__/display-date-locale.test.ts` | A |  |  | x |  |
| `web/src/lib/__tests__/format-fixed-locale.test.ts` | A |  |  | x |  |
| `web/src/lib/__tests__/format-quota-locale.test.ts` | A |  |  | x |  |
| `web/src/lib/__tests__/percent-sign-locale.test.ts` | A |  |  | x |  |
| `web/src/lib/format.ts` | M |  |  | x |  |
| `web/src/lib/time.ts` | M |  |  | x |  |
| `web/src/styles/__tests__/code-direction.test.ts` | A | x |  |  |  |
| `web/src/styles/__tests__/persian-monospace.test.ts` | A |  | x |  |  |
| `web/src/styles/__tests__/physical-direction-allowlist.json` | A | x |  |  |  |
| `web/src/styles/__tests__/physical-direction-classes.test.ts` | A | x |  |  |  |
| `web/src/styles/index.css` | M | x | x |  |  |
| `web/tsconfig.node.json` | M |  | x |  |  |
| `web/vitest.config.ts` | M |  | x |  |  |
