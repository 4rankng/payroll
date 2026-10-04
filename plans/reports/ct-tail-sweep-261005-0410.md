# ct- tail sweep — last daisyUI class retirement (2026-10-05)

Task: retire the last live daisyUI `ct-` class usages in `frontend/src` so the daisyUI plugin can leave the build. Binding rulings from `plans/261004-1812-uu-full-migration/plan.md` applied; token vocabulary per `frontend/src/components/ui/input.tsx` and the W1 bridge.

## Method

- Ground truth for every `ct-` class: a one-off `pnpm exec tailwindcss` compile of a probe file with the repo's live `tailwind.config.ts` (output to `/tmp` only). node_modules is hook-blocked, and the checked-in `dist/assets/index.Bfc59171.css` predates the `ct-` prefix config (it still has `tabs-boxed`), so the probe was the only trustworthy source.
- Key findings from the probe compile (daisyUI 4.12.24):
  - `ct-list`, `ct-list-row`, `ct-list-col-grow`, `ct-tabs-box` emit **zero CSS** — they are daisyUI-5-era names that this version never implemented. Dropping them is a literal zero-change.
  - `ct-card` = `relative flex flex-col` + radius `var(--rounded-box)` (1.25rem admin/partner `congtruong`, 1rem employee); `ct-card-border` = 1px border width; `ct-card-body` = `flex flex-auto flex-col` + padding/gap (always overridden at call sites); `ct-card-title` = `flex items-center gap-2` + type (overridden at call sites).
  - `ct-join` = inline-flex/stretch/rounded-btn (all overridden by `grid`/`rounded-xl` at the one call site) **plus** `.ct-join > :where(*:not(:first-child)) { margin-inline-start: -1px }` — the overlap margin is real layout and was preserved.
  - `ct-tabs` = `display:grid; align-items:flex-end` (grid overridable by utilities); `ct-tab` = inline-flex centering + `select-none` + a `:focus-visible` outline; `ct-tab-active` contributes only `--tw-text-opacity:1` + border-color, both already overridden at call sites.
  - `ct-navbar` = `flex items-center p-2 min-h-16 w-full` + a `> *` inline-flex child rule; padding/min-h/display already overridden, `width:100%` survives; the child rule was fully overridden by the children's own `flex` utilities.
  - daisyUI **color** utilities emit UNPREFIXED (`bg-neutral-content`, `bg-base-100`, …) — they work today and will break when the plugin leaves, independent of the `ct-` grep.

## Per-file swaps

| File | Swap |
|---|---|
| `src/components/timesheet/TimesheetMonthSelector.tsx` | Dropped `ct-join` (container) — `grid` utility already overrides its display/radius. Dropped 4× `ct-join-item`; added `-ml-px` to the 2nd/3rd/4th buttons to preserve daisyUI's −1px overlap margin exactly (pixel-identical track sizing). |
| `src/components/system-health/TimeRangeToggle.tsx` | Container: `ct-tabs ct-tabs-box` → `items-end` (the surviving `align-items:flex-end`; `ct-tabs-box` inert). Buttons: `ct-tab` → `inline-flex items-center justify-center select-none`; `ct-tab-active` dropped (fully overridden). Focus: daisy's currentColor outline → house UU idiom `focus-visible:outline outline-2 -outline-offset-2 outline-brand` (W13a month-bar idiom). |
| `src/components/shared/MobilePageHeader.tsx` | `ct-navbar` → `w-full` (the only surviving property; the `> *` child rule was overridden by children's own flex utilities). |
| `src/components/sheets/UserProfileSheet.tsx` | `ct-list-row`/`ct-list-col-grow`/2× `ct-list` dropped (inert in 4.12.24 — zero CSS today). `ct-hero` dropped (single `w-full` child; grid/place-items were no-ops). `ct-hero-content` → `z-0 flex items-center gap-4` (padding/max-w/justify already overridden). `ct-avatar` → `relative inline-flex overflow-hidden` (descendant `img`/`> div` rules redundant — `ui/avatar.tsx` already ships `aspect-square h-full w-full object-cover`). `ct-badge ct-badge-sm` → `inline-flex h-4 items-center justify-center rounded-full border text-xs leading-4` (colors/padding were already overridden by call-site utilities). |
| `src/components/ledger/LedgerEntryDetailsSheet.tsx` | 3× `ct-card ct-card-border` → `rounded-2xl border` (W13a card idiom; border-width was the only live part of `ct-card-border`). 3× `ct-card-body` → `flex flex-col` (`flex-auto` basis is inert under a block parent; `flex-col` is load-bearing for the `gap-3`). 3× `ct-stat` dropped (inline-grid vs block stack identical for dt/dd with preflight margin reset; padding/border-color overridden). 3× `ct-stat-title` → `whitespace-nowrap` (the one live property). 3× `ct-stat-value` dropped (size/weight/nowrap all already utility-overridden). |
| `src/components/admin-dashboard/CheckInHealthStrip.tsx` | `ct-card ct-card-border` → `rounded-2xl border`; `ct-card-body` → `flex flex-col` (gap load-bearing); `ct-card-title` → `flex items-center gap-2`. |
| `src/components/timesheet/mobile/TimesheetPageHeaderMobile.tsx` | `ct-card` dropped — explicit `rounded-2xl` already overrode the radius; single-child section renders identically as block. |
| `src/components/payroll/BankTransferHistoryPageContent.tsx` | `ct-card` → `relative flex flex-col` (test-pinned surface; survivors expressed explicitly since `ui/card.tsx` supplies no flex). Test pin updated (below). |
| `src/components/payroll/BankTransferHistoryPageContent.test.tsx` | Pin `toHaveClass('ct-card')` → `toHaveClass('rounded-2xl')`; test renamed "scopes daisyUI card behavior…" → "scopes the admin card treatment…" (intent unchanged). |
| `src/components/notifications/PushNotificationToggle.tsx` | `ct-toggle ct-toggle-success` input → the W7-restyled `ui/switch.tsx` (`Switch`), same `checked`/`disabled`/`aria-label` contract (`onChange` → `onCheckedChange`). The daisy toggle's box-shadow handle art is a daisyUI-only construct, not bridge-expressible on a native input; the UU Switch is the sanctioned equivalent. Also: `border-base-300 bg-base-100` → `border-border bg-card`, `bg-base-200` → `bg-muted`, `text-base-content` ×2 → `text-foreground` (daisy-only colors die with the plugin). |
| `src/components/notifications/NotificationSheet.tsx` | `ct-tabs ct-tabs-box` dropped (display dup + inert + no-op alignment in content-sized grid rows). daisy-only colors → white family, matching the sibling controls already on this dark header: `text-neutral-content` ×4 → `text-white`, `bg-neutral-content/10` ×2 → `bg-white/10`, `bg-neutral-content/15` → `bg-white/15`, `ring-neutral-content/30` ×2 → `ring-white/30` (the "Đọc tất cả" button already used text-white + ring-white/30). |
| `src/styles/admin-daisy.css` | Deleted the `[data-admin-ui] .ct-menu` block (2 rules) — `ct-menu` is referenced by no TSX/TS in src (probe + grep verified). All other rules in the file remain referenced; file kept. |

## Intentional deltas (documented, house-idiom)

1. **Card radius normalization**: `ct-card`'s role-scoped `--rounded-box` (1.25rem admin/partner, 1rem employee) → fixed `rounded-2xl` (1rem) on LedgerEntryDetailsSheet ×3 + CheckInHealthStrip. This is the W13a-approved card idiom (`overflow-hidden rounded-2xl border …`) already live on the admin dashboard; no other property of `ct-card` was lost.
2. **TimeRangeToggle focus ring**: currentColor → `outline-brand` house idiom (keyboard indicator preserved, now on token).
3. **PushNotificationToggle switch**: daisy 3rem×1.5rem success toggle → UU `Switch` (h-7 w-12 pill, brand-solid checked). Vietnamese `aria-label` copy byte-preserved.

## Final grep

`grep -rnE "(^|[^A-Za-z0-9_-])ct-" src --include="*.tsx" --include="*.ts" --include="*.css"` → 11 lines, all non-class:

- Comments only: `ui/alert-dialog.tsx` ×3, `modals/ChangePasswordModal.tsx`, `pages/mobile/admin/DashboardPage/index.tsx`, `WalletPage/index.tsx`, `AuditLogPage/index.tsx`
- Guard/fixture logic, intentional: `ui/alert-dialog.test.tsx` (negative assertion `.not.toContain('ct-')`), `employees/employee-surface-tokens.test.tsx` (portal guard pattern)

`tailwind.config.ts`'s `prefix: "ct-"` daisyUI config untouched (not in scope). **Zero `ct-` class usages remain.**

## Concerns / hand-off

1. **daisyUI-only COLOR utilities remain in 2 files outside my assigned list** — these stop resolving when the daisyUI plugin is removed and must be swapped before W-final retirement:
   - `src/components/notifications/NotificationItem.tsx:34` — `bg-base-100`, `hover:bg-base-200/60`
   - `src/components/wallet/EmployeeAccountLookupDialog.tsx:69,76` — `text-warning-content`, `text-info-content`
   (The same class of swap inside my assigned files — PushNotificationToggle, NotificationSheet — is done.)
2. `dist/` is stale (pre-prefix CSS with `tabs-boxed`) — irrelevant to source, but don't use it as ground truth.
3. Per instructions I did not run build/lint/e2e and did not commit. Verification below.

## Verification

- `pnpm exec tsc -p tsconfig.app.json --noEmit` — exit 0.
- Scoped vitest (`--testTimeout=30000`), 12 files / 68 tests — all passed: BankTransferHistoryPageContent, MobilePageHeader, TimesheetPageHeaderMobile, TimesheetMonthSelector, HealthDrilldownSheet, admin list-error-parity, ProjectsPage workspace-parity, EmployeePage, FlexiblePayEmployeePage, mobile/partner EmployeesPage + TimesheetsPage, partner PaymentHistoryPage.
- Tailwind emission check: one-off CLI compile of a probe containing every newly introduced class (`-ml-px`, `items-end`, `w-full`, `z-0`, `gap-4`, `inline-flex`, `select-none`, `rounded-2xl`, `bg-white/10`, `bg-white/15`, `ring-white/30`, `focus-visible:outline-brand`, `-outline-offset-2`, `leading-4`, …) — all emit; no silent token fallback.

Status: DONE
Summary: All live daisyUI `ct-` class usages retired across 11 source files + admin-daisy.css ct-menu block deleted; zero-change replacements verified against a probe compile of daisyUI 4.12.24; tsc green; 68 scoped tests green; final grep shows only comments + guard tests.
Concerns: 2 files (NotificationItem, EmployeeAccountLookupDialog) still use daisyUI-only color utilities (`bg-base-*`, `*-content`) outside the ct- grep — flagged above as the last blocker for actually removing the daisyUI plugin.
