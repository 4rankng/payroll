# Test Plan — Admin UI Shell Redesign (wave: users/projects/employees/payment-history/ledger + transactions removal)

Date: 2026-09-06
Scope: frontend only. Visual shell unification with the advance-payments/timesheet
idiom (ambient canvas, glass header card, operations section card with embedded
table). No API, hook, or data-flow changes. `/admin/transactions` route retired
(redirects to `/admin/ledger`); "Giao dịch" nav entries removed.

## A. Static gates (must all pass)

| # | Check | Command | Result |
|---|-------|---------|--------|
| A1 | TypeScript (real gate — `pnpm lint`'s tsc is a no-op) | `npx tsc -p tsconfig.app.json --noEmit` | ✅ 0 new errors (2 hits in touched files verified pre-existing on HEAD: App.tsx:346 router `future` prop, UsersPage:117 role-union — identical code on HEAD; repo baseline 144 errors untouched files) |
| A2 | ESLint | `pnpm lint` | ✅ exit 0 |
| A3 | Production build | `pnpm build` | ✅ exit 0 (PWA 148 precache entries) |
| A4 | No stale `/admin/transactions` links outside App.tsx redirect | `grep -rn "/admin/transactions" src --include="*.ts*"` → only App.tsx route redirect | ✅ only nav-test absence assertion remains |

## B. Unit/component tests (vitest)

| # | Test file | Why | Result |
|---|-----------|-----|--------|
| B1 | `layouts/AdminLayout.navigation.test.ts` | rewritten: ledger present, transactions absent in both navs | ✅ 4 passed |
| B2 | `components/payroll/BankTransferHistoryPageContent.test.tsx` | shell assertions updated for admin canvas + partner unchanged | ✅ 14 passed |
| B3 | `components/ResponsivePage.test.tsx` | guard against regressions from route changes | ✅ 2 passed |
| B4 | Full vitest suite (regression sweep) | `pnpm vitest run` | ✅ 121 files / 504 tests passed |

## C. Behavior preserved (code-level, verified by reading + types)

| # | Invariant | Where | Result |
|---|-----------|-------|--------|
| C1 | Users: role stat click-to-filter + infinite scroll + "Thêm người dùng" | pages/admin/UsersPage | ☐ |
| C2 | Projects: filters (search/status/month), sorting, pagination, create modal | pages/admin/ProjectsPage | ☐ |
| C3 | Employees: stat strip filters, MissingBankDetails, export/add modals, server pagination | pages/admin/EmployeesPage | ☐ |
| C4 | Payment-history: partner variant markup byte-identical; admin variant keeps workspace internals (accordion, sticky header, ct-card hook) | BankTransferHistoryPageContent | ☐ |
| C5 | Ledger: all 9 dialogs/actions wired (sao kê export/email, history, OnePay fee, chốt lương, mô phỏng, thêm bút toán) | pages/admin/TransactionsPage | ☐ |
| C6 | `/admin/transactions` URL → redirects to `/admin/ledger`; mobile dead page deleted | App.tsx | ☐ |

## D. Visual pass (browser, localhost dev — NOT prod login)

Run 2026-09-06 via Playwright (localhost:3000, system Chrome, `auth_token` pre-injected
via addInitScript, service workers blocked; vision-model review of 1440×900 screenshots;
EmailPromptGate modal was open on every shot — pre-existing dev-account behavior, not a defect):
- [x] All 5 pages show ambient canvas + glass header + one operations card (users, projects, employees, payment-history, ledger all confirmed)
- [x] No card-in-card on any page (filters flattened inside ops card — confirmed all 5)
- [x] Sidebar + mobile "more" menu show "Sổ Cái", no "Giao dịch" (data-level: unit test B1 covers both nav arrays)
- [x] `/admin/transactions` redirects to `/admin/ledger` (Playwright URL log)
- [x] Ledger summary cards (Thanh khoản 3.215.067.314 ₫, Công nợ, Vốn chủ 2.000.000.000 ₫) + ops card with legend/filters/status-strip table render on canvas

## Notes / unresolved
- Payment-history partner variant intentionally untouched (has its own shell + QA'd safe-area behavior).
- Projects page has no stats band (no stats endpoint exists; count already in description) — no fake data invented.
