---
type: "Reference"
title: "Testing: Integration and Playwright"
openwiki_generated: true
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-494c99eadcb987aa7fbe1fc0
    resource: repo://backend/internal/domain/login_event_property_test.go
  - id: openwiki-source-c1e04b76cefbedca24685f10
    resource: repo://backend/internal/infra/transaction/gorm_transaction_manager_test.go
  - id: openwiki-source-85571e849f256c72027fe515
    resource: repo://backend/tests/integration/client.go
  - id: openwiki-source-8ad78484c9b6439c1868b675
    resource: repo://backend/tests/integration/clock_helper.go
  - id: openwiki-source-e483fd3285d99d05c7b265cf
    resource: repo://frontend/AGENTS.md
  - id: openwiki-source-f18868fac0a3bcf8cf73016e
    resource: repo://frontend/tests/e2e/auth.spec.ts
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---


# Testing: Integration and Playwright

The repository has two distinct test layers with different scopes and tooling. Both run against a live backend — neither mocks the HTTP stack — so the tests catch real regressions in routing, middleware, and serialization.

- **Backend integration tests** live under `backend/tests/integration/`. They drive the live API over HTTP and exercise full flows through database writes, Redis, asynq, and webhook callbacks.
- **Frontend Playwright E2E** lives under `frontend/tests/e2e/`. It drives a real browser against the running frontend to verify role-based routing, responsive layout, and end-to-end workflows.

For test strategy and philosophy, see `docs/testing.md`. For local commands, see the root `AGENTS.md`.

## Backend integration harness

The harness is a `main` package — not a `*_test.go` file — that runs as a CLI against a live backend:

- `backend/tests/integration/client.go` — `APIClient` wraps an `http.Client` with auth token state, multipart upload support, and JSON helpers.
- `backend/tests/integration/clock_helper.go` — `SetServerTime`, `AdvanceServerTime`, `ResetServerTime`, `GetServerTime`. The backend exposes admin-only endpoints that move its internal `clock.Now()` forward or backward without waiting on wall-clock time, so time-dependent validation (pay-period windows, OT rules, settlement dates) can be exercised deterministically.
- `backend/tests/integration/db_helper.go` — direct MySQL access for cleanup between flows.
- `backend/tests/integration/assertions.go` — assertion helpers.
- `backend/tests/integration/flow_*.go` — one file per domain area. Each file is a sequence of steps that builds state, drives the API, and asserts outcomes.

There are 30+ flow files. Examples:

- `flow_bcc_import.go`, `flow_bcc_weekly_import.go`, `flow_bcc_weekly_payment_import.go` — the BCC import pipeline.
- `flow_bulk_transfer.go`, `flow_manual_bulk_transfer.go`, `flow_disbursement.go` — disbursement surface.
- `flow_timesheet_extended.go`, `flow_partner_timesheet.go` — timesheet engine and role scoping.
- `flow_clock_manipulation.go` — exercises the clock helpers themselves.
- `flow_auth_user.go` — login, RBAC denial, role matrix.
- `flow_advance_payment.go`, `flow_flexpay_import.go`, `flow_advance_removal_visibility.go` — FlexPay.
- `flow_ledger.go` — double-entry ledger and accounting rules.

`flow_http/` and `fixtures/` provide lower-level HTTP helpers and seeded data.

### Patterns

- **Use `pageSize=200` when querying timesheets** — keeps test setup fast without paginating.
- **Use `findAvailableDateWithMin`** for date selection in tests — picks a date with enough prior context for assignment/payrate lookups.
- **Clock first, assert later** — call `SetServerTime`/`AdvanceServerTime` before actions that depend on the date, then `ResetServerTime` at the end.
- **Pre-test DB cleanup** — when a flow leaves residue, hit MySQL directly via `docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -e "DELETE FROM ..."`. This is sanctioned in `backend/AGENTS.md` because the test data is fungible.

### Running

`make api-test` runs the integration suite end-to-end. It boots the live backend (or expects it to be running), runs the flows, and exits non-zero on the first failure. The Makefile target also runs `go test ./... -v -race -cover` for unit tests in the backend.

## Frontend Playwright

`frontend/tests/e2e/` holds the Playwright spec files. The pattern:

- `auth.spec.ts` — login, logout, role-based redirects.
- `employees.spec.ts`, `projects.spec.ts`, `timesheet.spec.ts` — Admin desktop flows.
- `employee-portal.spec.ts` — the employee mobile-first surface.

`frontend/tests/page-objects/` provides Page Object Model classes that abstract the route + selectors behind a small API. `frontend/tests/utils/` and `frontend/tests/fixtures/` hold login helpers and seeded data.

Playwright runs in headless Chromium by default. The mobile viewports covered include 390px (standard mobile) and 320px (dense / overflow-prone surfaces). Per `frontend/AGENTS.md`, every Admin and Partner route must be exercised at desktop and mobile widths; tests must fail if a workflow disappears from either view.

### Running

```bash
cd frontend
pnpm test:e2e
```

Playwright artifacts (traces, screenshots on failure, last-run JSON) live under `frontend/playwright-report/` and `frontend/test-results/` — both gitignored.

## Unit tests

Backend unit tests are co-located (`backend/internal/domain/employee_test.go`, `wallet_balance_test.go`, etc.) and run with `go test ./...`. Notable patterns:

- **Property-based tests** in `backend/internal/domain/` use `github.com/leanovate/gopter` to fuzz inputs against invariants. See `login_event_property_test.go`, `audit_log_property_test.go`, `event_factory_property_test.go`.
- **GORM transaction tests** in `backend/internal/infra/transaction/gorm_transaction_manager_test.go` use `gorm.io/driver/sqlite` for an in-memory DB so transaction commit/rollback behavior is hermetic.
- **Cache tests** in `backend/internal/infra/cache/` use `github.com/alicebob/miniredis/v2` for an in-process Redis.

Frontend has no unit tests yet; Vitest is planned. Type safety is the primary guarantee at the frontend layer today.

## Local quality gates

The root `AGENTS.md` mandates the following checks after every change:

| Check | Command |
|---|---|
| Backend lint | `cd backend && make lint` |
| Backend regression | `make api-test` |
| Frontend lint | `cd frontend && pnpm lint` |
| Frontend typecheck | `cd frontend && pnpm type-check` |
| Frontend E2E (when touching flows) | `cd frontend && pnpm test:e2e` |
| Tests, coverage, race | `cd backend && go test ./... -v -race -cover` |

`make deploy` is gated on these passing locally.

## What does not belong in either test layer

- **Mocking the HTTP stack** — both layers drive the real stack. Mock at the provider boundary (OnePay/9Pay) using dedicated stubs under `backend/internal/infra/disbursement/*/stubs.go`, never at the HTTP layer.
- **Sleeping for time to pass** — use `AdvanceServerTime` instead. Sleeping makes tests slow and flaky.
- **Re-implementing production logic in the test** — if a test finds itself duplicating the calculation, the production code needs to expose the calculation as a service so the test can call it.

## Related pages

- [Architecture Overview](../architecture/overview.md) — what the tests exercise.
- [Timesheet Engine](../features/timesheet-engine.md) — the feature with the deepest integration coverage.
- [Deploy, Backup, and Restore](../operations/deploy-backup-restore.md) — the `make api-test` smoke check before deploy.
