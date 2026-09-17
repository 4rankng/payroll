# Frontend polish and production regression review

## Goal
Pull latest main, run locally, audit all roles and responsive routes, implement evidence-backed UI/UX improvements, and deliver every code change as ~/Downloads/payroll.patch.

## Authority and boundaries
- AGENTS.md, CLAUDE.md, frontend/AGENTS.md and nearest module instructions.
- Main directly, no worktrees. Initial checkout clean at 85ad38d7; git pull --ff-only reports up to date.
- Local sandbox only. No deployment, push, production mutation, or production restore.
- Preserve role permissions and payment/accounting semantics. Refactor only to solve demonstrated issues.
- Final patch includes new tests/files and must apply to recorded base. Leave changes uncommitted.

## Scope and active paths
- App.tsx contains Admin, Partner, advance Partner, Accountant and Employee routes; desktop/mobile swaps use ResponsivePage.
- Shared primitives: frontend/src/components/ui; responsive shells: layouts and components/shared.
- Admin/Partner agent owns project workspace, audit and role-specific fixes.
- Employee agent owns self-service Employee* components and pages/employee.
- Runtime agent owns local backend, seed setup and backend checks.
- Root owns shared UI, Accountant, browser sweep, integration review and patch.

## Evidence and decisions
- Graphify and Understand-Anything graph files absent in fresh checkout; source used for navigation. Understand skills not found in installed skills search; record unavailable maintenance honestly.
- Global pnpm 8 fails workspace config; corepack pnpm 10.33.2 in frontend succeeds frozen install.
- Restored root make api-test/db aliases. Fresh local bootstrap now applies forward schema only and refuses populated databases.
- Corrected lint to check the application and Vite TypeScript projects; the original root config silently checked no application files.

## Progress
- PASS: repository refresh, local API/Vite/MySQL/Redis/payment mocks, dependency installation.
- Completed initial 111 authenticated route renders and 102 modal URL renders at 1280/390/320. Duplicate employee URL tab values map to one scrollable details view; they are not separate panel coverage.
- Fixed role-specific project pagination/recovery, advance-partner routing/edit loading, employee payroll failure states, password handling, accountant responsiveness, shared dialog names/focus, touch targets and contrast.
- Deeper dialog checks exposed an undefined metadata-options crash in AddTransactionSheet; restored the existing fallback choices and added regression coverage.
- Corrected Asynq wrapped duplicate-error handling, persisted seed payrates, and API harness pagination/discovery/prerequisites. Later lifecycle testing also proved a OnePay percentage/export defect: new exports now capture the intended transferred amount explicitly and apportion exactly that amount on completion; unversioned legacy codes keep the old interpretation. See verification.md for compatibility limits.
- Independent reviews completed for shared frontend behavior, role permissions, financial amount compatibility and schema/bootstrap safety. Final gate results and clean-apply packaging are recorded in verification.md and the Downloads companion.

## Verification boundaries
- Synthetic local records only; live payment rails, external email/Zalo delivery, real-device camera/geolocation/push and production data were not exercised.
- Read-only browser sweeps block API mutations; local API integration tests separately exercise writes with sandbox providers.
- Existing unrelated legacy auth/employees/projects/timesheet Playwright fixtures no longer match current API contracts. Do not report those suites as passed.
- Runtime artifacts and screenshots are under /tmp/payroll-ui-qa; credentials, .env files, database files and generated output must not enter the patch.

## Completion contract
Report tested routes and actions, actual results, pre-existing failures and untested limitations separately. Do not describe a route sweep as exhaustive verification of every possible data state or hardware integration.

## User steering: deeper visual and interaction review
The user explicitly requested visual browser inspection of every page/subpage/component/role and responsive size before delivery. Continue beyond the automated green checks: inspect actual renders and interact with tabs, menus, filters and nested dialogs; implement demonstrated layout/interaction improvements. Add tablet (768), responsive switch boundaries (1023/1024) and wide desktop (1440+) to the existing320/390/1280 checks. Inventory gaps and external/hardware-only states honestly. Frontend test results above describe the pre-visual-refinement candidate and must be refreshed for subsequent behavior changes. Packaging is pending this expanded pass.

## User steering: compact Admin and Partner mobile
Prioritize data density. Shared dashboard panels now use compact label/value rows, short headers, and a single action rail; Partner uses the shared month chooser. Admin accordions keep concise titles and values. Lists/summaries use compact strips, readable full identities and44px action targets. Verify actual320/390views after changes.

## User-reported right-edge gap
The user caught a real visual defect missed by the earlier overflow-oriented inspection. At390px the transfer dialog was374px wide, anchored at0, leaving16px on the right. Caller calc/max-width classes overrode the mobile primitive; tablet sm/md constraints conflicted with the app's1024px mobile boundary. Shared DialogContent now owns mobile horizontal geometry, and SheetContent does the same only for horizontal mobile sheets. Notification and employee-ad callers no longer use important tablet width overrides. Desktop centered dialogs and side drawers retain their intended dimensions.

Direct browser inspection verifies transfer at320/390/768/1023 with left0/right=viewport, reachable bottom actions at320, and centered580px desktop at1024. Portable regression tests cover conflicting caller dimensions and preserve desktop/side-sheet contracts. Browser tests wait for opening animations before comparing geometry. The final URL modal inventory checks actual edges, not merely absence of overflow, and exposed a cold-load null-project crash that is separately fixed with loading/retry/invalid-link states.

## Final delivery state
635 frontend tests,44 combined Chromium/WebKit cases,1638 backend race tests and338 strict API cases passed, with explicit fixture/calendar skips retained. Final late permission changes passed18 focused cases. The150 authorized modal/page matrix has clean edge geometry and targeted false-denial checks. No deployment or commit. All code and tests are delivered together as ~/Downloads/payroll.patch after clean-apply and file-content verification; generated indexes and runtime fixtures are excluded.
