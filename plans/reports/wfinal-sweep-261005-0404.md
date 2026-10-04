# W-final — Retirement Sweep Report

Date: 2026-10-05. Branch `main` (uncommitted — no commit per task gates). All gates: tsc green after each batch, scoped vitest green, `pnpm build` green.

## 1. Dead-file deletions (12 files, git rm)

Named by the task (6):
- `frontend/src/components/employees/EmployeeIconFrame.tsx`
- `frontend/src/components/employees/EmployeePayIllustration.tsx`
- `frontend/src/components/employees/InfiniteScrollInfo.tsx`
- `frontend/src/components/employees/EmployeeSummaryCard.tsx`
- `frontend/src/components/employees/EmployeeSummaryCards.tsx`
- `frontend/src/components/transaction/mobile/TransactionPageHeaderMobile.tsx` + `.test.tsx` (test rode its only importer — the component itself)

Found by the systematic zero-importer pass (6):
- `frontend/src/components/employees/details/components/EmployeeViewDetails.tsx`
- `frontend/src/components/employees/details/components/TimesheetRecordCard.tsx`
- `frontend/src/pages/admin/DashboardPage/components/DashboardMasonryCard.tsx`
- `frontend/src/pages/admin/ManualDisbursementPage/RecentTransfers.tsx`
- `frontend/src/pages/admin/ManualDisbursementPage/ReconciliationDownloadDialog.tsx`

Every deletion verified with a repo-wide basename grep (src + tests) before `git rm`: zero references. Doc cleanup: 3 stale rows removed from `frontend/src/components/employees/AGENTS.md` (EmployeeSummaryCard(s), InfiniteScrollInfo).

Sweep scope: `src/components/{ui,shared,employees,advance-payment,admin-dashboard,auth,marketing,foundations}` + `src/pages/**`, excluding barrels/tests/entry points; lazy `import("./pages/X")` strings match by directory name so page modules are covered.

Report-only (gates forbid touching `components/ui/*`) — 8 more zero-importer files:
`ui/animated-hamburger.tsx`, `ui/breadcrumb.tsx`, `ui/input-otp.tsx`, `ui/long-text.tsx`, `ui/modal-link.tsx`, `ui/responsive-container.tsx`, `ui/user-card-row.tsx`, `ui/validated-input.tsx` (deleting `ui/input-otp.tsx` would also free the `input-otp` npm dep).

Alive-but-worth-knowing: `useEmployeeInfiniteScroll`/`usePartnerEmployeeInfiniteScroll` hooks are still imported by `EmployeesPage` (not orphans of InfiniteScrollInfo).

## 2. Tail icon/token swaps (live files)

Icons lucide → verified `@untitledui/icons` (names verified via `search_icons` MCP or in-repo precedent; decorative instances get `aria-hidden="true"`):

| File | Swaps |
|---|---|
| `advance-payment/table-config.tsx` | Wallet/Users were **dead imports** (dropped); ArrowUpDown → ChevronSelectorVertical; MoreHorizontal → DotsHorizontal; RotateCcw → RefreshCw05; ArrowUp/ArrowDown/X 1:1 |
| `attendance/AdminAttendanceTableConfig.tsx` | MoreHorizontal → DotsHorizontal; MapPin → MarkerPin01; Check/X/Zap 1:1 |
| `system-health/BrowserPlatformUsersSheet.tsx` | Users → Users01; Zap/Clock/X 1:1 |
| `system-health/GroupedUsersSheet.tsx` | Users → Users01 (×2 sheets); Zap/Clock/X 1:1 |
| `system-health/FailedLoginDetailDialog.tsx` | ShieldAlert → **Shield03** (UU has no ShieldAlert); Globe → Globe01; Monitor → Monitor01; MapPin → MarkerPin01; Clock/X 1:1 |
| `system-health/ErrorByEndpointTable.tsx` | ChevronsUpDown → ChevronSelectorVertical; ChevronDown/Up 1:1 |
| `system-health/RecentErrorsTable.tsx` | Activity/ChevronDown/ChevronUp 1:1 |
| `system-health/UserInvestigateTable.tsx` | ChevronDown/ChevronRight 1:1 |

Shared-component lucide retirement (15 of 16 files): `MobilePageHeader`, `MobileSubPageHeader` (ArrowLeft), `MobilePagination` (ChevronLeft/Right), `SearchBar` + `MobileSearchInput` (Search → SearchLg, X), `MobileSheetHeader` (X; comment updated), `FilterPill` (Check/ChevronDown), `UserAvatarDropdown` (LogOut → LogOut01, UserCircle 1:1, Key → Key01), `MobileOperationsPanel` (ChevronRight), `EarningsPill` (**dead** Banknote import dropped), `QuickActionBar` (HelpCircle/FolderPlus/Zap 1:1, UserPlus → UserPlus01, Download → Download01, Users → Users01, Send → Send01; slot widened LucideIcon → ComponentType<SVGProps<SVGSVGElement>> per W13a pattern), `ContextStrip`/`InlineAlert`/`StatsCards`/`SummaryStatsCards` (type-only slots widened, lucide import deleted).

**SidebarToggle keeps lucide** (`PanelLeftClose`/`PanelLeftOpen`) — the UU catalog has no sidebar collapse/expand glyphs (verified: "sidebar", "collapse", "panel" queries → 0 results). Swapping both states to the same icon would destroy the affordance.

Slate → bridge tokens (`text-fg-primary/secondary/tertiary`, `utility-gray-50/100/200/300` incl. opacity variants; borders/divides → utility-gray ladder):
- `payroll/BankTransferHistoryPageContent.tsx` — all slate classes retired (W13c's "stays until W-final" comment updated); 1 `text-slate-500` → `text-fg-tertiary` in `advance-payment/table-config.tsx`; 2 `text-slate-700` → `text-fg-secondary` in `AdminAttendanceTableConfig.tsx`.
- `shared/AdminPageFrame.tsx` — 2 `border-slate-300` → `border-utility-gray-300` (AdminSectionCard, AdminFilterRow).
- `payroll/BankTransferHistoryPageContent.test.tsx` — pinned class assertions updated to the new bridge tokens (intentional contract change; 14/14 green).

## 3. Lucide audit — remainder (dep stays)

Final count: **366 files** import `lucide-react` (W0 inventory: 476). Count ≠ 0 → `lucide-react` stays in package.json, no pnpm install.

- **13 type-locked seams** (`LucideIcon` in the import): `admin-dashboard/CheckInHealthStrip.tsx`, `HealthDrilldownSheet.tsx`, `LocationMap.tsx`, `loans/LoanSummaryCards.tsx`, `modals/project-detail/ProjectStatusCard.tsx`, `premium/PremiumDashboardHero.tsx`, `PremiumEmptyState.tsx`, `settings/SettingsGeneralPanel.tsx`, `timesheet/TimesheetSummaryCards.tsx`, `timesheet/utils/timesheetStatsConfig.ts`, `users/UserStatsCard.tsx`, `config/actions.ts`, `pages/partner/DashboardPage/index.tsx` — all outside W-final's granted files; widening follows the established W13a pattern when their owner waves touch them.
- **353 plain icon-usage files** — the whole-app icon swap is product-wave surface (admin/partner/employee pages, modals, sheets), not the W-final tail.

## 4. Radix audit (report only, no package.json edits)

| Package | Importers | Verdict |
|---|---|---|
| react-slot | 4 | keep (Button asChild path) |
| react-dialog | 3 | keep (engine-keep) |
| react-label | 2 | keep (engine-keep) |
| react-select | 2 | keep (engine-keep) |
| react-toggle | 2 | keep (ui/toggle.tsx, ui/toggle-group.tsx) |
| react-accordion | 1 | keep (engine-keep) |
| react-alert-dialog | 1 | keep |
| react-avatar | 1 | keep (engine-keep) |
| react-checkbox | 1 | keep (engine-keep) |
| react-collapsible | 1 | keep (ui/collapsible.tsx) |
| react-dropdown-menu | 1 | keep (engine-keep) |
| react-popover | 1 | keep (engine-keep) |
| react-progress | 1 | keep |
| react-radio-group | 1 | keep (engine-keep) |
| react-scroll-area | 1 | keep |
| react-separator | 1 | keep |
| react-slider | 1 | keep (engine-keep) |
| react-switch | 1 | keep (engine-keep) |
| react-tabs | 1 | keep (engine-keep) |
| react-toggle-group | 1 | keep |
| react-tooltip | 1 | keep (engine-keep) |
| **react-aspect-ratio** | **0** | **removable** (primitive deleted W1) |
| **react-context-menu** | **0** | **removable** (primitive deleted W1) |
| **react-hover-card** | **0** | **removable** (primitive deleted W1) |
| **react-menubar** | **0** | **removable** (primitive deleted W1) |
| **react-navigation-menu** | **0** | **removable** (primitive deleted W1) |
| **react-toast** | **0** | **removable** (replaced by sonner) |

6 packages removable on the next dependency-hygiene pass (needs package.json edit = owner approval).

## 5. daisyUI `ct-` proof

Word-boundary grep `(^|[^a-zA-Z0-9_-])ct-[a-z]` across `src` (plain `-0.05ct` loose greps false-positive on "react-"): **16 files**, of which 4 are comments-only (`ui/alert-dialog.tsx` ×3 "intentionally NOT using ct-*", `pages/mobile/admin/{WalletPage,DashboardPage,AuditLogPage}/index.tsx` migration notes). Real daisyUI class usage remains in **12 files** — target 0 NOT reached:

| File | Hits | Owner wave |
|---|---|---|
| `ledger/LedgerEntryDetailsSheet.tsx` | 15 | W13c admin financial (ct-card/ct-stat family) |
| `sheets/UserProfileSheet.tsx` | 8 | W12 partner / settings |
| `timesheet/TimesheetMonthSelector.tsx` | 5 | W13 admin tables |
| `styles/admin-daisy.css` | 3 | W13a (`.ct-menu` override block keyed to daisyUI menu) |
| `system-health/TimeRangeToggle.tsx` | 3 | W-final tail (system-health internals) |
| `admin-dashboard/CheckInHealthStrip.tsx` | 3 | W13a admin dashboard (also a LucideIcon seam) |
| `timesheet/mobile/TimesheetPageHeaderMobile.tsx` | 1 | W13 admin tables |
| `shared/MobilePageHeader.tsx` | 1 (`ct-navbar`) | W13a shell (in a file I touched; class left — daisyUI styling contract unknown-risk to strip blind) |
| `payroll/BankTransferHistoryPageContent.tsx` | 1 (`ct-card`) | W13c |
| `payroll/BankTransferHistoryPageContent.test.tsx` | 1 (test pins `ct-card`) | W13c |
| `notifications/PushNotificationToggle.tsx` | 1 (`ct-toggle`) | W13 utility pages |
| `notifications/NotificationSheet.tsx` | 1 (`ct-tabs`) | W13 utility pages |

Consequence: **the daisyUI plugin + ct- prefix in `tailwind.config.ts` cannot be retired yet** — these classes still resolve from it.

## 6. Bundle report

`pnpm build` green (1m 39s).

| Metric | W0 baseline | W-final | Delta |
|---|---|---|---|
| Main chunk | 1,463.95 kB | 1,496.88 kB | **+32.93 kB (+2.25%)** |
| Main chunk gzip | 412.52 kB | 422.09 kB | **+9.57 kB (+2.32%)** |
| Precache entries | 154 | 187 | +33 |

The migration re-skinned the whole app without exploding the entry chunk; the small growth is the 14 waves' vendored UU surface. pdfmake chunk unchanged by this sweep (not re-measured).

## Verification log

- tsc (`pnpm exec tsc -p tsconfig.app.json --noEmit`): green after each destructive batch (deletions; icon/token batch incl. Shield03 fix; shared batch).
- Scoped vitest: 6 test files green (`AdminAttendanceTableConfig`, `BankTransferHistoryPageContent` 14/14 after pin update, `MobilePageHeader`, `GroupedStatCard`, `AccentStripCard`, `AdvancePaymentHistoryCard`).
- Not run (per gates): full lint, full test:run, Playwright, dev servers, prettier.
- Nothing committed; working tree holds exactly this sweep's edits (verified via git status mid-run — no parallel-session interference).

## Status: DONE_WITH_CONCERNS

Summary: 12 dead files deleted (verified zero importers), 8 more flagged in ui/ (report-only), 8 task-named files + 15 shared components moved off lucide onto verified UU icons / bridge tokens, lucide dep retained (366-file remainder documented: 13 type seams + 353 wave-surface files), Radix audit done (6 packages removable, report-only), ct- proof shows 12 real-usage files remaining (target 0 not reached — named with owner-waves), bundle main chunk +2.25% vs W0, all gates green, nothing committed.

Concerns:
1. ct- retirement is NOT complete — 12 live files still consume daisyUI classes (ledger sheet, user profile sheet, timesheet selector/header, admin-daisy.css menu block, notifications, system-health toggle). daisyUI must stay in tailwind.config until a follow-up sweep retires them.
2. SidebarToggle has no UU equivalent (PanelLeftClose/Open) — permanent lucide consumer unless UU adds sidebar glyphs or a custom SVG is vendored.
3. 8 dead files under `components/ui/*` need a one-line owner approval to delete (gates barred me).
4. Six Radix packages (aspect-ratio, context-menu, hover-card, menubar, navigation-menu, toast) have zero importers — removable with a package.json edit (out of my scope).
5. `ShieldAlert` does not exist in @untitledui/icons; two other files (ApiKeysSection, HealthDrilldownSheet) still import it from lucide — use Shield03 when their waves swap.
