# Check-in settings page QA — 2026-08-22

## Scope

- Replace the bulk check-in dialog with a responsive page and role-aware return navigation.
- Present compact Vietnamese status filters and correctly sized desktop/mobile controls.
- Show currently enabled employees and current-month active / not-yet-checked-in cohorts.
- Paginate employees at the API and SQL layers with a default page size of 20.
- Disable the complete server-authoritative current-month `Inactive` cohort in one confirmed action.
- Cancel the complete server-authoritative `Chờ kích hoạt` cohort in one confirmed action.
- Preserve Admin and Advance Partner access across desktop and mobile.

## Cases

### CHECKIN-01 — Dedicated page and return navigation

- Priority: High
- Surface: Authenticated web application
- Roles: Admin, Advance Partner
- Viewports: 1280px, 390px, 320px
- Expected: Each active advance-payment surface opens a page, not a dialog; `Quay lại` returns to the matching role route.
- Automated evidence: `CheckInSettingsPage.test.tsx`, `CheckInSettingsRouteParity.test.ts`
- Browser evidence: Pending production deployment.
- Status: NOT RUN

### CHECKIN-02 — Coherent Vietnamese cohort discovery

- Priority: Critical
- Surface: Check-in configuration page and cohort API
- Expected: `Đang bật` lists enabled employees; `Đã điểm danh` contains enabled employees with at least one check-in in the current Vietnam month; `Chưa điểm danh` contains enabled employees with none; `Chờ kích hoạt` contains deferred employees.
- Automated evidence: repository classification tests and exact-label component test.
- Browser evidence: Pending production deployment.
- Status: NOT RUN

### CHECKIN-03 — Disable the complete Inactive cohort

- Priority: Critical
- Surface: Confirmed bulk action and API
- Expected: Confirmation identifies project, month, and count. The server locks current assignments, re-queries the full cohort independent of search or pagination, disables every still-Inactive employee atomically (including duplicate current assignments), and zeroes current-month quota.
- Automated evidence: service lock/full-cohort/duplicate tests, repository inactive/deduplication tests, confirmation component test.
- Production mutation: Must not be executed with live employee data during QA; confirmation and endpoint are verified without submitting a destructive production action.
- Status: PASS (automated contract); production mutation NOT APPLICABLE for safety.

### CHECKIN-04 — Role and project authorization

- Priority: Critical
- Surface: Casbin policy, project scoping, handler authorization
- Expected: Admin is allowed. Advance Partner receives a minimal selector projection of only creator/shared modifiable projects and must have project modification access for configuration reads and writes. Standard Partner is denied.
- Automated evidence: Casbin allow/deny tests, handler role/project-scope tests, minimal-response contract test, and backend compile/race suite.
- Browser evidence: Pending production deployment.
- Status: NOT RUN

### CHECKIN-05 — True server pagination

- Priority: Critical
- Surface: Cohort API, repository query, and responsive page controls
- Expected: Page changes send `page` and `pageSize` to the API; SQL applies matching `LIMIT` and `OFFSET`; the response returns authoritative totals; the client never loads a deliberately huge page for local slicing.
- Automated evidence: repository pagination test and page-2 component request assertion with `pageSize=20`.
- Status: PASS (automated contract).

### CHECKIN-06 — Cancel all pending activations

- Priority: Critical
- Surface: Confirmed bulk action and API
- Expected: The server re-queries the complete pending cohort for the selected project, independent of the visible search/page, locks current assignments, clears deferred activation state, and leaves already-active or unrelated employees unchanged.
- Automated evidence: service full-cohort/lock test, Casbin route test, integration route contract, and component confirmation test.
- Production mutation: Not executed; the user-facing action requires explicit project-scoped confirmation.
- Status: PASS (automated contract); production mutation NOT RUN.

### CHECKIN-07 — Responsive and accessible presentation

- Priority: High
- Surface: Desktop table and mobile cards
- Expected: The table is used only at wide desktop widths; narrower layouts use cards without horizontal overflow. Desktop row actions are compact while mobile controls remain at least 44px.
- Automated evidence: semantic role assertions, production build, lint/typecheck.
- Browser evidence: Pending production deployment.
- Status: NOT RUN

## Automated evidence

- `pnpm vitest run`: PASS — 115 files, 454 tests.
- Focused page/parity rerun: PASS — 2 files, 11 tests (9 page + 2 route parity).
- `pnpm lint`: PASS — ESLint and `tsc --noEmit`.
- `pnpm build`: PASS — Vite production build and PWA service worker.
- `go test ./... -v -race -cover`: PASS.
- Focused backend service, repository, authorization, and handler package tests: PASS.
- `make lint` (backend): completed with `go vet` PASS after `golangci-lint` reported two pre-existing unchecked test-file close calls.
- `make api-test` (root): unavailable because the root Makefile has no such target. The actual target is in `backend/Makefile`.
- `make api-test` (backend): BLOCKED locally — `localhost:8080` was not running, so discovery could not log in. Production API/browser verification remains pending.
- `graphify update .`: PASS — 26,729 nodes and 55,785 edges rebuilt.

## Final QA Result

- Result: BLOCKED pending deployment and authenticated production browser verification.
- Human-executable cases: 3/7 contract-complete; 4 require authenticated browser evidence.
- Open defects: P0=0, P1=0, P2=0, P3=0.
- Final end-to-end regression: NOT RUN.
