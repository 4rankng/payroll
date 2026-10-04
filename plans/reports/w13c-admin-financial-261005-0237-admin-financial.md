# W13c — Admin Financial + Utility Pages (UU PRO migration)

Wave: W13 sub-wave 3 of `plans/261004-1812-uu-full-migration/plan.md`.
Date: 2026-10-05. Agent: w13c-admin-financial. Uncommitted (rides the combined gate).

## Scope delivered

All 37 page files in the assigned territory rebuilt/resealed on the UU bridge — desktop AND
mobile parity — with 100% behavior/flow/Vietnamese-copy preservation. Routes, hooks, services,
mutations, query keys, payment/disbursement logic, flexible-pay quota logic, and permission
logic untouched.

### Pages (chrome restyle, icon swaps, ct-/raw-palette retirement)

Desktop:
- `pages/admin/TransactionsPage/` — "Bút toán" toolbar label → `text-fg-primary`; header icons
  swapped in `TransactionPageHeader` (below). Table/filter machinery untouched (W13b).
- `pages/admin/LoansPage/` — lucide → UU (Bank, TrendUp01, CoinsHand, Wallet01, Users01);
  KpiHeroCard icons pass UU through the widened slot; toolbar label → `text-fg-primary`.
- `pages/admin/AdvancePaymentsPage/` — all icons UU (BankNote01, Wallet01, Users01, Download01,
  FileDownload01, SwitchHorizontal02, Scan, ClockRewind, DotsVertical, UserCheck01); overflow
  `icon` array type `LucideIcon` → `ComponentType<SVGProps<SVGSVGElement>>`; section borders
  slate-300 → `border-utility-gray-300`; row title/subtitle slate → fg-* ladders.
- `pages/admin/AdvancePaymentsPage/AdvPartnerView.tsx` — icons UU; `typeof Wallet` tab icon type
  widened; status dots (amber/emerald/red/gray-400) → utility ladders; two neutral "paper"
  action buttons → utility-gray outline (kept neutral — NOT promoted to primary).
- `pages/admin/WalletPage/` — icons UU (SearchMd, Upload01, SwitchHorizontal02, Wallet01).
- `pages/admin/PaymentHistoryPage/`, `pages/admin/EmailPage/` — 5-line wrappers; seams fixed in
  their imported components (below).
- `pages/admin/SystemHealthPage/` — icons UU (Activity, AlertTriangle, Speedometer01 for Gauge,
  Eye); HEALTH_META pills/dots → utility-success/warning/error ladders.
- `pages/admin/CronHealthPage/` — Loader2 → `Loading01` (kept `animate-spin`).
- `pages/admin/PayrateEditPage/` — icons UU (CalendarDate for CalendarClock, Lock01, Save01);
  every red/amber error/warning accent → utility-error/utility-warning ladders + `fg-error-*`
  /`fg-warning-*`; flexible badge → utility-success. Validation flow byte-preserved.
- `pages/admin/AuditLogPage/` (index, AuditLogCard, AuditLogFilters, AuditLogDetailSheet,
  MetadataRenderer, utils) — icons UU; card chrome slate → utility-gray; action-chip variants
  green/red/yellow/orange → ladders. **blue and teal variants keep the legacy palette** per the
  W2 ruling (no utility-blue/teal ladder yet). Diff table accents → `utility-error-600` /
  `fg-success-primary`.

Mobile:
- `pages/mobile/admin/LoansPage/` + `LendersPage.tsx` — icons UU; status chips → ladders;
  `text-gray-300` separators → `text-utility-gray-300`.
- `pages/mobile/admin/AdvancePaymentsPage/` (+ EmployeeListPage, EmployeeDetailSheet) — icons
  UU; blue-gray hex chrome (`#D8E2EE`/`#F3F7FB`) and slate → utility-gray bridge; attendance
  card approve/credit/reject accents → fg-success/warning/error; orange/amber quota chips →
  warning ladder. AttendanceMobileCard (exported, test-pinned) byte-preserved semantics.
- `pages/mobile/admin/WalletPage/` — **daisyUI retired**: `bg-neutral text-neutral-content` →
  `bg-brand-section_subtle text-fg-white`; `ct-btn ct-btn-outline/-primary` → equivalent plain
  utility classes (min-h-11 kept — test-pinned); `ct-card(-body) bg-base-100` → plain `bg-white`.
- `pages/mobile/admin/AuditLogPage/` — icons UU; `ct-btn ct-btn-outline` → plain utilities;
  filter-sheet slate → utility-gray; `text-destructive` → `text-fg-error-primary`.
- `pages/mobile/admin/SystemHealthPage/` — icons UU; `ct-badge` → plain pill utilities (border
  kept so the ladder border color paints); own HEALTH_META copy → ladders.
- `pages/mobile/admin/CronHealthPage/` — Loader2 → Loading01; emerald badge → success ladder.
- `pages/mobile/admin/PayrateEditPage/` — same treatment as desktop twin.
- No `pages/mobile/admin/{TransactionsPage,PaymentHistoryPage}` exist — the ledger route's
  mobile half is `LedgerEntriesPageMobile` (NOT in this territory) and payment-history is the
  shared content component (below).

### Components (surgical icon/ct-/token swaps only — importers verified)

Page-chrome companions (all importer-checked): `transaction/TransactionPageHeader` (incl.
BookOpen01 for Sổ Cái), `disbursement/WalletBalanceCard` (on-dark accents → utility ladders;
reconciliation dialog → ladders), `advance-payment/AdvPartnerStatusOverview` (CheckCircle),
`advance-payment/actions/MobileOverflowAction` (slot widened), `admin-dashboard/KpiHeroCard`
(icon slot widened per approved W13a pattern; `strokeWidth={2}` dropped — equals lucide's
default, zero visual change for lucide callers), `system-health/SectionLabel` (slot widened),
`loans/loan-table-config` (Landmark → Bank), `lenders/LenderCard`, `cron-health/*` (constants,
CronJobCard, CronStatusBadge, CronSummaryCards), `wallet/WalletTransactionsList` +
`wallet/BulkTransferBatchList` (Loading01 spinner, SlashCircle01 for Ban, Copy01, XClose,
FilterLines, status chips → ladders), `email/AdminEmailComposer` (error banner → error ladder),
`payroll/BankTransferHistoryPageContent` (icon swap ONLY — shared with partner; slate neutrals
deliberately left for the W-final sweep, same call W13a made on AdminPageFrame),
`system-health/SummaryPills`, `MobileSummaryPills`, `FailedLoginBadges`, `StatusPill` (slot
widened), `AllClear`, `BrowserPlatformStats` (Smartphone → Monitor05; RANK_COLORS repointed to
4.5:1-compliant pairs — warning-700 on warning-200 ≈ 5.1:1), `ErrorByEndpointCards`,
`MobileErrorByEndpointCards`.

Shared components — completed the approved W13a icon-slot sweep (these two slots were missed by
the original 8-component pass): `shared/TabBarWithBadges`, `shared/MobileSheetHeader`.
One component outside `components/shared`: `shared` note — `advance-payment/actions/MobileOverflowAction`
widened as page-needed (my territory).

## Icon vocabulary

All imports verified against `@untitledui/icons@0.0.22` dist types (1179 exports). Notable
mappings: Loader2 → Loading01 (+`animate-spin`), ArrowRightLeft → SwitchHorizontal02, Gauge →
Speedometer01, CalendarClock → CalendarDate (effective-date) / ClockStopwatch (duration),
ScanFace → Scan, History → ClockRewind, Ban → SlashCircle01, Landmark → Bank, HandCoins →
CoinsHand, EllipsisVertical → DotsVertical, Smartphone → Monitor05, BookOpen → BookOpen01,
Send → Send01. Decorative icons `aria-hidden="true"`; interactive labeled controls unchanged.

## Verification (allowed gates only)

- `pnpm exec tsc -p tsconfig.app.json --noEmit` (full): **GREEN, zero errors** — final run on
  the shared tree (includes W13b's in-flight edits; two earlier red runs were my own missed
  prop-style icon refs + `CalendarCheck` JSX, fixed).
- Scoped vitest: **12 files / 39 tests passed** — WalletPage (desktop+mobile), AuditLogPage
  (detail sheet + metadata renderer), AttendanceMobileCard, partner PaymentHistoryPage
  regression, list-error-parity (cron), WalletBalanceCard, AdvPartnerStatusOverview,
  KpiHeroCard, TransactionPageHeader (+mobile variant). Run with `--testTimeout=30000`.
- Not run (per instructions): build, lint, test:run, Playwright, dev servers, prettier. Not
  committed.

## Boundaries / deferred (for the lead)

1. **W13b seams left untouched by agreement:** `advance-payment/table-config.tsx`,
   `attendance/AdminAttendanceTableConfig.tsx`, `ui/responsive-table` surface, system-health
   tables/sheets (`RecentErrorsTable`, `UserInvestigateTable`, `ErrorByEndpointTable`,
   `GroupedUsersSheet`, `BrowserPlatformUsersSheet`, `FailedLoginDetailDialog`) — these are
   table/dialog machinery; icons inside them are still lucide.
2. `transaction/mobile/TransactionPageHeaderMobile.tsx` has no importer outside its own test —
   likely dead; candidate for the W-final dead-code sweep rather than a restyle.
3. `BankTransferHistoryPageContent` slate neutrals + `AdminPageFrame` slate borders → W-final
   palette sweep (consistent with W13a's call).
4. Audit action chips blue/teal keep legacy palette until the utility-blue ladder ruling.
5. Visual baselines: these pages were mostly not in the certified suite; lead re-baselines
   after commit (deltas blessed).

Status: DONE
Summary: 37 assigned page files (desktop+mobile) rebuilt on the UU bridge with verified
@untitledui/icons, daisyUI fully retired from the territory, raw slate/hex chrome moved to the
utility ladders; 24 imported components surgically icon/token-fixed with importer checks; two
shared icon slots widened completing the approved W13a sweep. Full tsc green; 39 scoped tests
green; zero behavior/copy changes.
Concerns: (1) full tsc ran on the shared tree with W13b edits in flight — final combined gate
should re-run once W13b lands; (2) TabBarWithBadges/MobileSheetHeader widening touches
components/shared (W13a territory) — done to unblock my pages using the exact approved pattern,
flagging for the lead; (3) table-configs and system-health tables intentionally left for W13b.
