---
type: testing
title: Integration Suite
description: Backend integration test harness (make api-test), real-DB fixtures, the scenarios that cover wallet, salary disbursement, attendance, FlexPay, BCC import, and partner scoping end-to-end.
tags: [testing, integration, api-test, fixtures, wallet, scenarios]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-410ab36600fefdde765cb24e
    resource: repo://backend/AGENTS.md
  - id: openwiki-source-2e5dcb886f75efbe2020e47d
    resource: repo://backend/tests/AGENTS.md
  - id: openwiki-source-bf27fb010957bd7d3b81f9b1
    resource: repo://docs/testing.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Testing: Integration Suite

<!-- openwiki: broken internal link [../docs/testing.md] file "../docs/testing.md" does not exist. Fix the href or restore the target, then delete this comment. -->
The backend integration suite lives under `backend/tests/integration/`. It exercises the full stack — HTTP → handler → application service → domain service → repository → MySQL — against a real test database. The harness is `make api-test`. The broader testing strategy is in [`docs/testing.md`](../docs/testing.md); this page is the harness summary.

## Harness

`make api-test` (defined in the root and `backend/Makefile`):

1. Spins up a clean test DB (separate from the dev DB).
2. Applies migrations.
3. Runs `go test ./tests/integration/... -race -cover`.
4. Reports coverage and exits non-zero on failure.

The test harness uses real GORM, real Redis, and a mock payment provider (the 9Pay mock at `localhost:9001` and the OnePay mock at `localhost:9002`). Mocks are preferred over provider SDK stubs because they exercise the same wire format the provider would send.

## Fixtures

`backend/tests/integration/` carries:

- `setup_test.go` — DB setup, migration runner, container helpers.
- `helpers/` — clock helpers (`SetServerTime`, `AdvanceServerTime`, `ResetServerTime`, `GetServerTime`), payment-mock helpers, assertion helpers.
- Per-domain test files (`flow_auth_user.go`, `flow_timesheet.go`, `flow_wallet.go`, `flow_disbursement.go`, `flow_flexpay.go`, `flow_attendance.go`, `flow_bcc_import.go`, …).
- Per-role scenarios (`admin_*`, `partner_*`, `employee_*`).

The clock helpers exist because `clock.Now()` is the business clock; advancing time deterministically is the only way to test scenarios like "what happens at month-end" or "what does the cash-readiness forecast show 3 days before payday".

## What the scenarios cover

The integration suite is organized by capability. Each scenario exercises the full stack — handler, service, repository, event bus, asynq worker — so a regression in any layer surfaces here.

- **Auth flow** (`flow_auth_user.go`) — login, OTP, password reset, Casbin policy matrix.
- **Timesheet** — bulk approve/reject, edit requests, BCC import, payroll export.
- **Wallet** — top-up, payment, IPN, status inquiry, double-entry invariants.
- **Disbursement** — bulk transfer pipeline, recovery sweepers, IPN signature verification.
- **FlexPay** — request, fee schedule, kill switch, settlement, salary notification.
- **Attendance** — check-in/out, geofence, advance hold, quota credit, auto-reject sweep.
- **BCC import** — pipeline + dispatch into timesheets + lock interaction with bulk approve.

## Conventions

- Use `pageSize=200` when querying timesheets in tests (matches the production default).
- Use `findAvailableDateWithMin` for date selection in fixtures.
- Clock helpers are the only way to advance time; never mock `clock.Now()` directly.
- Pre-test DB cleanup when needed:
  ```bash
  docker exec payroll-mysql mysql -uroot -prootpassword payroll_db \
      -e "DELETE FROM timesheets WHERE ..."
  ```
- New scenarios follow the existing per-capability naming and live alongside the relevant code in `backend/tests/integration/`.

## Coverage

Per-package coverage is reported by `make api-test`. Target coverage by package:

- Domain layer: ≥90% (pure logic, easy to cover).
- Application services: ≥80% (the bulk of business logic).
- Transport / handlers: ≥70% (happy paths + key error paths).
- Infra adapters: ≥60% (mocked provider interactions).

These are guidelines, not hard gates. The integration scenarios are the primary regression surface; unit tests in each package cover the small invariants.

## Running the suite locally

```bash
make api-test
# or, with race + verbose:
cd backend && go test ./tests/integration/... -race -v -cover
```

A focused run against a single file:

```bash
cd backend && go test ./tests/integration/... -run TestWalletIPN
```

## Relationships

<!-- openwiki: broken internal link [../docs/testing.md] file "../docs/testing.md" does not exist. Fix the href or restore the target, then delete this comment. -->
- Testing strategy — [`docs/testing.md`](../docs/testing.md).
- Local dev setup — `operations/local-dev.md`. The test DB is a sibling of the dev DB.
- Frontend E2E — `testing/e2e-playwright.md`.
- Wallet invariants — `features/wallet-ledger.md`.
- Migration discipline — `migrations/schema-evolution.md` and `backend/migrations/AGENTS.md`.
