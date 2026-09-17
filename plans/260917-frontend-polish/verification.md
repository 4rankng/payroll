# Payroll UI and regression verification

Base: `85ad38d75ec9d67d6814bb4cff4390da2cefa6a5` on main. `git pull --ff-only origin main` was already up to date. All changes remain uncommitted; no push or deployment occurred.

## Resulting behavior

- Admin and Partner dense mobile dashboards, wallet, ledger, lists and headers reserve more space for useful data; filters/month controls, mobile pagination, long identities/amounts and 44px controls work across responsive variants.
- Mobile dialogs and horizontal sheets fill the viewport below 1024px, including tablet widths. Desktop dialogs remain centered and intentional side drawers retain their widths. The user-reported right gap is fixed at the shared component level, with specific important-width callers corrected.
- Failed queries show failure/retry instead of misleading empty or zero financial data. Cold-loading project assignment waits for a valid project rather than crashing. Nested actions and notifications use independent accessible controls.
- Employee fee previews cannot submit a stale quote; amount changes, errors/retry and successful submission clear old state. Password forms validate and clear secrets appropriately. Fonts cannot hold React startup indefinitely.
- Advance Partner has bank-catalog read access needed by its existing scoped employee editor. Bank administration and global advance file history remain denied; unsupported file-history controls are removed. Accountant tables/dialogs/calendars remain usable on narrow screens.
- Self-account password modals work for all five authenticated roles through a dedicated SelfAccount permission; existing Partner profile links no longer falsely report denial. This does not grant generic user-administration rights.
- TypeScript checks now actually compile application and Vite projects, revealing and correcting previously unchecked contracts.
- Backend fixes preserve transaction visibility during imports, publish employee events after commit, recognize wrapped duplicate-job errors, and use the authenticated export actor for notification history.
- New OnePay exports apply the captured payment percentage and store an explicit transfer amount snapshot. Completion apportions that exact total with integer arithmetic and rejects inconsistent already-paid shares before writes. Unversioned legacy transaction codes keep their established interpretation; no historical repair runs.
- Guarded local schema initialization applies forward migrations only, preserves existing role values/financial rows, and refuses populated database initialization. Migration107 reconciles demonstrated missing schema fields. API tests use current routes, payloads and strict lifecycle/permission assertions.

## Runtime and browser evidence

Local Vite on 5173, Go API on 8080, development MySQL/Redis and local payment sandbox only. Synthetic data covers Admin, Partner, Advance Partner, Accountant, regular Employee and flexible Employee; over 100 projects, long Vietnamese text, payments/attendance and pending/completed requests. Visual sweeps block real API writes; sandbox API integration tests separately exercise synthetic writes.

Viewports: 320,390,768,1023,1024,1280,1440. Initial 37 role/route combinations at 3 widths produced 111 renders. Initial modal variants included duplicate employee query parameters; they are not counted as distinct panels. Final route/control/edge evidence and actual visual inspection are separated in:

- `visual-admin-partner.md`: role routes, month/actions/payrate/cron controls, compact ledger, final modal edge inventory and cold-load regression.
- `employee-visual.md`: Employee/Advance Partner/Accountant nested interactions, fee/bank/notification states, responsive boundary captures and hardware limits.
- `root-visual.md`: shared components, settings/preview, direct transfer edge geometry and corrected density/axe sweeps.
- `backend-verification.md`: financial invariants, schema replay, exact backend checks and skipped fixture reasons.

## Verification gates

- PASS: full backend `go test ./... -v -race -cover`: 1,638 top-level test passes,0 failures,11 explicit external/developer fixture skips; go vet passed. Subsequent narrow authorization and integration-harness race checks passed after their final changes.
- PASS: final root `make api-test`:340 total, 338 passed, 0 failed, 2 calendar-prerequisite skips. It includes successful current disbursement routes, bank-update persistence, denied administrative employee payroll, Advance Partner reference-only bank permission, XLSX/history and full sandbox payment lifecycle.
- PASS: final sandbox OnePay provider amount 168,000 equals timesheet paid amount 168,000; receivable 171,360 equals ledger 171,360. Advance request reaches COMPLETED. Regular-bank and unversioned transaction-code compatibility have targeted regression coverage.
- PASS: fresh/populated local bootstrap, migration replay twice, custom role retention, unchanged financial sentinel rows, legacy bank/asset data and populated-initialization rejection.
- PASS: shared dialog/sheet focused regressions10/10; independent cold-load project assignment regressions14/14.
- PASS: full frontend Vitest suite: 144 files, 635 tests. Subsequent narrow modal-permission and ad-layout changes have focused checks recorded in the role reports.
- PASS: combined 44 Chromium/WebKit browser cases: 28 Employee, 12 Accountant/workspace across 320/390/768/1023/1024/1280, 4 font-startup cases.
- PASS: frontend ESLint, application/Vite TypeScript checks and production build. Final modal permissions add18 focused passing cases with role denials; project-assignment loading/error coverage adds 14 passing cases already included in the full unit run.
- PASS: final authorized modal inventory: 150 page/modal combinations across 320/390/768/1023/1024/1280; 78 mobile edge assertions, 68 preserved desktop/side dialogs, 4 intentional full-page mobile timesheet views. No gap, overflow, uncaught browser error, API failure or write attempt. 30 targeted recaptures additionally check absence of false permission warnings.
- Portable patch verification: clean base apply, byte/executable-mode comparison and reverse dry run; result and SHA256 are recorded in Downloads/payroll-verification.md.

## Limits

- No production database, external funds movement or deployment was accessed. Source patch application does not run migrations or repair data.
- Real payment certificates/signatures/callbacks, external email/Zalo delivery, device camera/location/push, native keyboard/safe-area behavior and production datasets were not verified. Browser emulation and local sandbox behavior are distinct from real-device/provider verification.
- Existing unversioned OnePay in-flight records and historical discrepancies are not automatically corrected; the old shared Amount field also held gross values for regular bank exports, so globally reinterpreting it would be unsafe.
- Browser inspection covers recorded routes and states, not every possible data combination. Remote fonts were blocked in deterministic capture runs; dedicated Chromium/WebKit tests verify startup while fonts remain pending and subsequent font application.
- Legacy unrelated auth/employees/projects/timesheet browser suites contain stale fixtures/selectors and are not counted as passing. The employee, workspace and font suites are the browser regression gates for this patch.
- Optional settings absent from the synthetic database return 404. The isolated email preview retains its sandbox; a Playwright service-worker-block injection warning was checked separately in the in-app browser, where HTML preview worked without console errors.
- Migration107 is forward-only; destination schema state has not been copied or inspected. Review through the normal migration process before production deployment.
- Graphify AST maintenance is performed locally. Understand-Anything maintenance is unavailable because its graph/installed command is absent. Generated indexes are excluded.

## Apply on another machine

Use the recorded base revision, or check a compatible main revision, in a clean repository:

```sh
git apply --check ~/Downloads/payroll.patch
git apply ~/Downloads/payroll.patch
```

New tests and executable bootstrap scripts are included. Credentials, environment files, databases, dependencies, build output and screenshots are excluded. The companion Downloads/payroll-verification.md records the delivered patch checksum and clean-apply evidence. Re-run the documented checks on the destination environment.
