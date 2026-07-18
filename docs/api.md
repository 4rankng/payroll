# API Reference

HTTP API structure, route groups, middleware, and conventions for the payroll backend. All routes are served by the Go API server on port 8080. See [System Architecture](system-architecture.md) for component relationships and [Code Standards](code-standards.md) for coding conventions.

## Base Path

All API routes are under `/api/v1`.

## Route Registration

Routes are registered in `backend/internal/app/bootstrap/routes.go` and split across 9 files:

| File | Group |
|------|-------|
| `routes_auth.go` | Auth, login, OTP, OAuth, captcha |
| `routes_employee.go` | Employees |
| `routes_project.go` | Projects, project-employees |
| `routes_timesheet.go` | Timesheets, payroll |
| `routes_advance_payment.go` | Advance payments (FlexPay) |
| `routes_wallet.go` | Wallet operations |
| `routes_disbursement.go` | Disbursement, bulk transfer, webhooks |
| `routes_admin.go` | Admin, settings, audit, cron, metrics, DB |
| `routes_misc.go` | Notifications, push, assets, loans, email, ledger, transactions, banks, dashboard |

## Route Groups

### Auth (`/api/v1/auth`)

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/login` | Initiate login (step 1: send OTP) |
| POST | `/login/verify` | Verify OTP and get JWT |
| POST | `/login/resend` | Resend OTP |
| POST | `/google` | Google OAuth login (no OTP required) |
| GET | `/captcha` | Get self-hosted image CAPTCHA (after 3 failed logins) |
| GET | `/captcha/required?username=...` | Check whether the account must solve CAPTCHA before login |
| POST | `/logout` | Logout (blacklist token) |
| GET | `/me` | Get current user profile |
| PUT | `/me` | Update profile |
| POST | `/change-password` | Change password |

### Users (`/api/v1/users`)

Full CRUD. Additional: `POST /:id/reset-password`, `GET /summary`, `GET /:id/activities`.

### Employees (`/api/v1/employees`)

CRUD, import/export, payroll per employee, summary, `GET /missing-bank-details`, `POST /init-users`, `GET /unassigned`, payment schedule management.

### Projects (`/api/v1/projects`)

CRUD, payrates per project, timesheets per project, entry table, employee assignment, `POST /activate`, partner summary, `PATCH /:id/employees/:employeeId/checkin-enabled`.

### Project Employees (`/api/v1/project-employees`)

Assignment management, batch operations.

### Timesheets (`/api/v1/timesheets`)

CRUD, bulk approve/reject/reset, approve-all, preview, export, grouped view, payroll report, cash-readiness forecast, edit-requests, BCC import, upload entries, partner import.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/cash-readiness` | Advisory target-Kỳ forecast for the next pay date; excludes outstanding-payment backlog |
| GET | `/summary` | Summary stats (pending payment amount, etc.) |
| POST | `/bulk-approve` | Bulk approve timesheets |
| POST | `/export-entries-template` | Download export template |

`GET /summary` and the shared `status=pending_payment` filter use the same cohort:
approved timesheets whose `payment_status` is `pending` or `failed`. That
cohort also powers the frontend "Chờ thanh toán" filter label and the
bulk-transfer planner/export flow.

`GET /cash-readiness` keeps the legacy percentile fields and adds a transparent
cash decomposition: `observed_approved`, `pending_target_amount`,
`expected_pending_amount`, `expected_future_amount`, `expected_payout`, and
`recommended_reserve`. All amounts use the configured weekly payable percentage,
matching the cash written to the bank-transfer workbook. Reliability fields
(`reliability_state`, `calibration_samples`, WAPE, bias, interval coverage, and
reserve shortfall rate) are measured only from unfiltered company forecasts
resolved against distinct timesheets in generated weekly transfer files.
Filtered forecasts are never mixed into company accuracy. The response remains
advisory and cannot initiate a top-up or payment.

### Payrolls (`/api/v1/payrolls`)

Payroll calculation, disbursement, and the Admin/Partner read-only bank transfer history screen.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/bank-transfer-histories` | Read-only completed bank transfers grouped by employee and fixed weekly cycle; supports `month`, `cycle`, `projectId`, `employeeId`, `search`, `page`, and `pageSize` query filters |

Fixed weekly cycle windows:

| Cycle | Window |
|-------|--------|
| 1 | Days 1-7 |
| 2 | Days 8-14 |
| 3 | Days 15-21 |
| 4 | Days 22-28 |

The history response is intentionally read-only and history-only. It returns completed transfer rows grouped by employee and cycle, including each bank reference number and amount, the cycle window, payment date, total amount, and pagination. Partner access is filtered to accessible projects server-side.

### Payrates (`/api/v1/payrates`)

Payrate CRUD and management.

### Advance Payments / FlexPay (`/api/v1/advance-payments`)

FlexPay lifecycle, fee calculation, import/export, reconciliation, settlement, file download.

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/request` | Create advance payment request |
| POST | `/:id/cancel` | Cancel request |
| GET | `/demand-forecast` | Wallet demand forecast |

### Wallet (`/api/v1/wallet`)

Balance sync/adjust, topups, payments, demand forecast, reconciliation.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/balance` | Get wallet balance |
| POST | `/balance/sync` | Sync balance from provider |
| POST | `/payments/:id/resolve` | Resolve a pending payment |
| GET | `/demand-forecast` | Wallet demand forecast |

### Disbursement (`/api/v1/`)

Manual disbursement, bulk transfer, auto-bulk-transfer, fee estimation, reconciliation.

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/auto-bulk-transfer` | Trigger automatic bulk transfer |
| POST | `/check-account` | Pre-flight bank account verification |
| POST | `/webhooks/disbursement/{provider}` | Payment provider IPN webhook |

### Dashboard (`/api/v1/dashboard`)

20+ endpoints for admin analytics: financial overview, bank usage, employee activity, historical data, check-in health, project profitability, weekly profit, quota anomalies, salary distribution, cash flow, top-paid employees, new employees, recent activities, system notifications, partner dashboard.

### Ledger (`/api/v1/ledger`)

Double-entry queries, entries CRUD, `POST /entries/:id/reverse`, reconciliation.

### Transactions (`/api/v1/transactions`)

CRUD, `POST /:id/settle`, `POST /:id/reverse`.

### Attendance (`/api/v1/mobile/attendance`)

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/check-in` | Employee check-in (geofence validated) |
| POST | `/check-out` | Employee check-out |
| POST | `/cancel-current` | Cancel current attendance |
| GET | `/today` | Today's attendance |
| GET | `/history` | Attendance history |
| POST | `/attempt-log` | Log failed attempt |

### Loans (`/api/v1/loans`)

CRUD, `POST /:id/disburse`, `POST /:id/repay`.

### Settings (`/api/v1/settings`)

CRUD. Partner role has read-only access.

### Notifications (`/api/v1/notifications`)

List, unread count, mark read.

### Push (`/api/v1/push`)

Web push subscription management (VAPID).

### Audit (`/api/v1/audit`)

List audit logs.

### Cron (`/api/v1/cron`)

List jobs, run manually, toggle.

### Metrics (`/api/v1/metrics`)

API performance: latency trends, slowest endpoints, error breakdown, browser stats, OS stats, top endpoints, failed logins.

### Admin Clock (`/api/v1/admin/clock`)

**Non-prod only.** Time manipulation for testing.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/time` | Get current server time |
| POST | `/set` | Set server to specific time |
| POST | `/advance` | Advance time by duration |
| POST | `/reset` | Reset to real time |

### Health

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/ping` | Liveness check |
| GET | `/healthz` | Health check |
| GET | `/ready` | Readiness check |

## Middleware Chain

**Execution order:** recovery → security headers → request timeout → logging/metrics → auth → RBAC → rate limit → handler

| Middleware | File | Purpose |
|-----------|------|---------|
| `ErrorRecovery()` | `error_middleware.go` | Catches panics, logs stack traces, returns 500 |
| `SecurityHeaders()` | `security_headers.go` | X-Content-Type-Options, X-Frame-Options, CSP |
| `RequestTimeout(30s)` | `request_timeout.go` | Maximum request duration |
| `APIMetrics()` | `api_metrics.go` | Request latency, status codes, endpoint metrics |
| `Auth(jwtSecret)` | `auth.go` | JWT extraction and validation, sets user context |
| `Authorization(casbinEnforcer)` | `authorization.go` | Casbin RBAC enforcement |
| `APIRateLimit` | `rate_limit.go` | Per-IP and per-account throttling (Redis-backed) |
| `LoginRateLimit` | (per-login-route) | Account-specific login rate limiting |
| `IPWhitelist(allowedIPs)` | `ip_whitelist.go` | Webhook endpoint IP restriction |
| `AuditContext()` | `audit_context.go` | Propagates audit metadata through context |
| `TenantSemaphore()` | `tenant_semaphore.go` | Limits concurrent requests per tenant |

## DTOs

All Data Transfer Objects (request/response shapes) live in `backend/internal/app/dto/`. 25+ DTO files cover every domain: employee, project, timesheet, payroll, dashboard (25.5K — largest), advance_payment, transaction, ledger, payrate, notification, user, settings, loan, email, export, auth, bank, lender, asset, attendance, adv_partner, flexpay_reconciliation, disbursement_fee_schedule.

## Error Responses

Domain errors use constructors from `internal/domain/errors.go`:

| Constructor | HTTP Status | Usage |
|-------------|-------------|-------|
| `NewNotFoundError(msg)` | 404 | Resource not found |
| `NewValidationError(msg)` | 400 | Invalid input |
| `NewForbiddenError(msg)` | 403 | Action not allowed |
| `NewConflictError(msg)` | 409 | State conflict (e.g., already exists) |
| `NewInternalError(msg, err)` | 500 | Database/infra error |

Sentry-style context can be attached: `.WithContext("key", value)`. The error translator at `transport/http/response/error_translator.go` maps provider errors to Vietnamese messages.

## RBAC Roles

Four roles defined in Casbin policy (`configs/casbin_policy.csv`):

| Role | Access |
|------|--------|
| `admin` | Full wildcard access (`/api/*, *, allow`) |
| `partner` | Scoped access (projects, employees, timesheets, payrates). Explicit deny on bulk-approve/reject/reset |
| `adv_partner` | External partner for advance payment pipeline (import, view/export, no financial actions) |
| `employee` | Self-service only (`/api/v1/me/*`) |

All authenticated users can access notifications and push subscriptions. See [ADR-008](decisions/ADR-008-casbin-rbac-authorization.md).

## OpenAPI / Swagger

No OpenAPI/Swagger generation is currently in place. Route definitions in the `routes_*.go` files are the source of truth.
