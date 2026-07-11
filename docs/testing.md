# Testing Strategy

Testing layers, commands, and conventions for the payroll monorepo. See [Code Standards](code-standards.md) for naming and [Definition of Done](definition-of-done.md) for completion gates.

## Testing Pyramid

```
Unit (Go *_test.go)     →  Domain logic, services, pure functions
       ↓
Integration (Go flows)  →  API flows against live backend
       ↓
Frontend (Playwright)   →  E2E user journeys
       ↓
CI Pipeline             →  Automated gates on every PR/push
```

## Backend Unit Tests

**Location:** Co-located with source as `*_test.go` in the same package.

**Command:**
```bash
cd backend && go test ./... -v -race -cover
# Or with HTML coverage report:
cd backend && make test
# Generates coverage.out + coverage.html
```

### FakeClock Pattern

All business time uses the injected `clock.Clock` interface (see [ADR-006](decisions/ADR-006-clock-injection-pattern.md)). Tests use `FakeClock`:

```go
// Freeze time for deterministic tests
clk := clock.NewFake(time.Date(2026, 7, 10, 9, 0, 0, 0, time.Local))
// AutoFake starts unfrozen like RealClock until Set()/Advance() is called
clk := clock.NewAutoFake()
```

The `Clock` interface: `Now()`, `NowUTC()`, `TodayStart()`, `TodayEnd()`, `UnixNow()`.

### Property-Based Testing

Uses `github.com/leanovate/gopter` for property-based tests:
- `accounting_rules_test.go` — double-entry accounting invariants
- `wallet_demand_forecast_math_test.go` — forecast distribution properties

### Mocks

Generated mocks live in `backend/mocks/` (generated via `github.com/golang/mock`):

| Mock | Interface |
|------|-----------|
| `mock_employee_repo.go` | `EmployeeRepository` |
| `mock_timesheet_repo.go` (largest, 46K) | `TimesheetRepository` |
| `mock_project_employee_repo.go` | `ProjectEmployeeRepository` |
| `mock_payrate_repo.go` | `PayrateRepository` |
| `mock_bank_repo.go` | `BankRepository` |
| `mock_audit_log_repo.go` | `AuditLogRepository` |
| `mock_user_repo.go` | `UserRepository` |

### Key Test Files

| File | Covers |
|------|--------|
| `accounting_rules_test.go` | Double-entry invariants (property-based) |
| `advance_payment_fee_schedule_test.go` | Tiered fee calculation |
| `disbursement_fee_schedule_test.go` | Disbursement fee tiers |
| `cash_readiness_forecast_test.go` | Forecast calculation |
| `cash_readiness_backtest_test.go` | Historical backtest validation |
| `bcc_import_service_test.go` | BCC bank statement import |
| `gorm_transaction_manager_test.go` | Transaction commit/rollback |

## Backend Integration Tests

**Location:** `backend/tests/integration/`

**Command:**
```bash
make api-test    # Runs: cd backend && go run ./tests/integration/
```

Requires a running backend (start with `make dev` or `make db` + `air`).

### Architecture

| File | Purpose |
|------|---------|
| `main.go` | Test runner entry — discovers test data, runs all flows, generates HTML report |
| `client.go` | HTTP test client with auth, request helpers, response parsing |
| `models.go` (34K) | Test data models and response structures |
| `config.go` | Test config (base URL, credentials, timeouts) |
| `reporter.go` | HTML test report with pass/fail/skip statistics |
| `assertions.go` | Custom assertion helpers |
| `clock_helper.go` | Server clock manipulation helpers |

### Clock Helpers

```go
SetServerTime(client, time)      // Freeze server to specific time
AdvanceServerTime(client, dur)   // Advance time by duration
ResetServerTime(client)          // Unfreeze, restore auto-advance
GetServerTime(client)            // Get current server time
```

These call admin endpoints `GET/POST /api/v1/admin/clock/*` (non-prod only).

### Flow Test Files (30)

| Flow | Coverage |
|------|----------|
| `flow_advance_payment.go` | FlexPay lifecycle, fee schedules, file import/export |
| `flow_advance_removal_visibility.go` | Advance payment removal visibility |
| `flow_assets.go` | Asset management |
| `flow_audit.go` | Audit log verification |
| `flow_auth_user.go` | Login, JWT, RBAC, unauthorized access |
| `flow_bank_crud.go` | Bank CRUD |
| `flow_bcc_import.go` | BCC bank statement import |
| `flow_bcc_weekly_import.go` | Weekly BCC import |
| `flow_bulk_transfer.go` | Bulk payment transfer |
| `flow_clock_manipulation.go` | Fake clock set/advance/reset |
| `flow_cron.go` | Cron job management |
| `flow_dashboard.go` | Dashboard analytics |
| `flow_disbursement.go` | Payment disbursement |
| `flow_employee_crud.go` | Employee CRUD |
| `flow_employee_self_service.go` | Employee self-service |
| `flow_flexpay_import.go` | FlexPay file import |
| `flow_infra.go` | Infrastructure checks |
| `flow_ledger.go` | Double-entry ledger |
| `flow_loan.go` | Loan lifecycle |
| `flow_manual_bulk_transfer.go` | Manual bulk transfer |
| `flow_metrics.go` | API performance metrics |
| `flow_notification.go` | Notification delivery |
| `flow_partner_timesheet.go` | Partner timesheet access |
| `flow_payrate_crud.go` | Payrate management |
| `flow_payrate_partner_visibility.go` | Payrate partner visibility |
| `flow_project_crud.go` | Project CRUD |
| `flow_saoke.go` | Bank statement import/export |
| `flow_settings.go` | System settings |
| `flow_timesheet_extended.go` | Extended timesheet operations |
| `flow_transaction.go` | Financial transactions |
| `flow_wallet.go` | Wallet operations |

## Frontend Tests

### Playwright E2E

**Location:** `frontend/tests/`

**Commands:**
```bash
cd frontend
pnpm test:e2e          # Run E2E tests
pnpm test:e2e:headed   # Run with browser visible
pnpm test:e2e:debug    # Debug mode
pnpm test:e2e:ui       # Interactive UI mode
pnpm test:e2e:report   # Show last report
```

### Vitest (Unit)

```bash
cd frontend
pnpm test              # Watch mode
pnpm test:run          # Single run
pnpm test:ui           # UI mode
```

### Lint & Type-Check

```bash
cd frontend
pnpm lint              # ESLint + tsc --noEmit
pnpm type-check        # TypeScript compiler check
pnpm build             # Production build (includes type-check)
```

## CI Pipeline

**Config:** `.github/workflows/ci-cd.yml`

**Triggers:** Push to `main`, PR to `main`.

| Job | What it does | Blocks deploy? |
|-----|-------------|----------------|
| `backend-lint` | `gofmt -l`, `go vet`, `golangci-lint v2.12.2` | Yes |
| `frontend-lint` | `yarn install --frozen-lockfile`, `eslint --max-warnings=0`, `tsc -b`, `vite build` | Yes |
| `backend-test` | `go test` with coverage, uploads `coverage.out` artifact | Yes |
| `frontend-test` | `vitest run --coverage`, uploads `coverage/` artifact | Yes |
| `security` | `gosec` (SARIF), `npm audit --audit-level=high` | Yes |
| `docker-backend` | Build + push `linux/amd64` backend image to DockerHub | Only on push to main |
| `docker-frontend` | Build + push `linux/amd64` frontend image to DockerHub | Only on push to main |
| `deploy` | SSH to `tingting.vip`, pull images, recreate containers | Only on push to main |

**Concurrency:** `deploy-${{ github.ref }}` — cancels in-progress PR runs.

## Known Test Issues

1. **Test data pollution** — Integration tests can exhaust available dates. Clean up with:
   ```bash
   docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -e "DELETE FROM timesheets WHERE ..."
   ```
2. **Always use `pageSize=200`** when querying timesheets in tests to avoid pagination false failures.
3. **Monthly timesheet tests skip gracefully** when all dates are occupied by paid records.
4. **Clock manipulation tests must call `Reset()` afterward** to avoid polluting other tests.
5. **OnePay tests removed** — production provider, tested in staging only. 9Pay sandbox + mocks used in CI.
6. **Dual lockfile pitfall** — Frontend has both `yarn.lock` (Docker build) and `pnpm-lock.yaml` (local dev). Running `pnpm add` locally updates `pnpm-lock.yaml` but leaves `yarn.lock` stale, breaking `make deploy`. See [Troubleshooting](troubleshooting.md).

## Coverage

- Backend: `make test` generates `coverage.out` (2.2MB) + `coverage.html`. Target: >70%.
- Frontend: `vitest run --coverage` generates `frontend/coverage/`.
- CI uploads both as artifacts (7-day retention).
