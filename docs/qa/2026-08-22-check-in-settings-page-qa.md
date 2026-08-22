# Check-in settings page QA — 2026-08-22

## Scope

- Replace the bulk check-in dialog with a responsive page and role-aware return navigation.
- Show currently enabled employees and current-month `Active` / `Inactive` cohorts.
- Disable the complete server-authoritative current-month `Inactive` cohort in one confirmed action.
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

### CHECKIN-02 — Enabled, Active, and Inactive discovery

- Priority: Critical
- Surface: Check-in configuration page and cohort API
- Expected: `Đang bật` lists enabled employees; `Active` contains enabled employees with at least one check-in in the current Vietnam month; `Inactive` contains enabled employees with none.
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

### CHECKIN-05 — Responsive and accessible presentation

- Priority: High
- Surface: Desktop table and mobile cards
- Expected: No horizontal overflow; readable names and identifiers; keyboard-operable tabs; visible status; controls are at least 44px on mobile.
- Automated evidence: semantic role assertions, production build, lint/typecheck.
- Browser evidence: Pending production deployment.
- Status: NOT RUN

## Automated evidence

- `pnpm vitest run`: PASS — 113 files, 446 tests.
- Focused page/parity/hooks rerun: PASS — 3 files, 9 tests.
- `pnpm lint`: PASS — ESLint and `tsc --noEmit`.
- `pnpm build`: PASS — Vite production build and PWA service worker.
- `go test ./... -v -race -cover`: PASS.
- Focused backend service, repository, authorization, and handler package tests: PASS.
- `make lint`: completed with `go vet` PASS after `golangci-lint` reported two pre-existing unchecked test-file close calls; the feature-specific unused-field finding was fixed.
- `make api-test`: BLOCKED locally — `localhost:8080` was not running, so discovery could not log in. Production API/browser verification remains pending.
- `graphify update .`: PASS — no topology changes requiring output updates.

## Final QA Result

- Result: BLOCKED pending deployment and authenticated production browser verification.
- Human-executable cases: 1/5 contract-complete; 4 require deployed browser evidence.
- Open defects: P0=0, P1=0, P2=0, P3=0.
- Final end-to-end regression: NOT RUN.
