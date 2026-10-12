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
| POST | `/password-reset/request` | Email a single-use password-reset magic link (30-min TTL). Always 200 (anti-enumeration). Rate-limited 3/hr per normalized email. Only registered when `PASSWORD_RESET_ENABLE=true`. |
| POST | `/password-reset/confirm` | Set a new password with a magic-link token. Invalidates all existing sessions for the user. |

### Users (`/api/v1/users`)

Full CRUD. Additional: `POST /:id/reset-password`, `GET /summary`, `GET /:id/activities`.

### Employees (`/api/v1/employees`)

CRUD, import/export, payroll per employee, summary, `GET /missing-bank-details` (bank_account_status='invalid' rows are listed unconditionally while the employee is assigned to an active project; missing-info rows additionally require pending timesheet/advance), `GET /missing-mobile` (employees with no phone on file — the partner "thiếu SĐT" nudge list, scoped to the caller's accessible set), `GET /duplicate-mobiles` (phone numbers carried by more than one employee with the members the caller may see — out-of-scope holders count as `hidden_count`), `POST /init-users`, `GET /unassigned`, payment schedule management.

Mobile rules on create/update: duplicate mobiles are always rejected (one number identifies one employee — `identity.Resolver` fails closed otherwise); when the admin setting `employee.mobile_required` is on, every create path (manual form, Excel import, BCC/FlexPay import) must carry a mobile number.

### Projects (`/api/v1/projects`)

CRUD, payrates per project, timesheets per project, entry table, employee assignment, `POST /activate`, partner summary, `PATCH /:id/employees/:employeeId/checkin-enabled` (enable defers to day 1 of next month; disable is instant), `DELETE /:id/employees/:employeeId/checkin-enabled` (cancel a pending enable), `PATCH /:id/employees/checkin-enabled/bulk`.

### Project Employees (`/api/v1/project-employees`)

Assignment management, batch operations. An omitted `start_date` is computed by the backend: the day after the employee's most recent recorded timesheet (any project), else the 1st of the current month — never "today", so same-month timesheet uploads covering earlier days are not rejected by assignment validation. BCC imports also backdate an existing assignment's start to the file's earliest entry when the upload proves earlier worked days.

### Timesheets (`/api/v1/timesheets`)

CRUD, bulk approve/reject/reset, approve-all, preview, export, grouped view, payroll report, cash-readiness forecast, edit-requests, BCC import, upload entries, partner import.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/cash-readiness` | Advisory target-Kỳ forecast for the next pay date; excludes outstanding-payment backlog |
| GET | `/summary` | Summary stats (pending payment amount, etc.) |
| POST | `/bulk-approve` | Bulk approve timesheets |
| POST | `/export-entries-template` | Download export template |

`GET /summary` reports the operational payment backlog in `pendingPaymentAmount`
and `pendingEmployees`: pending-approval or approved timesheets whose
`payment_status` is `pending` or `failed`. This lets the dashboard show accrued
salary before approval without making it eligible for disbursement.

The `status=pending_payment` list filter and bulk-transfer planner use the
stricter payroll-ready cohort: approved timesheets whose `payment_status` is
`pending` or `failed`. Pending-approval rows never enter payment exports.

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
| 4 | Days 22-end of month |

Each cycle pays shortly after its window closes: cycles 1–3 pay on days 10, 17, and 24 of the same month; cycle 4 pays on day 1 of the next month.

The history response is intentionally read-only and history-only. It returns completed transfer rows grouped by employee and cycle, including the employee name and CCCD (`employee_cccd`), each payment transfer code (`transfer_code`), bank reference number, amount, cycle window, payment date, total amount, and pagination. The `search` filter matches Vietnamese employee names without requiring diacritics and matches transfer codes or bank references case-insensitively. Partner access is filtered to accessible projects server-side.

### Payrates (`/api/v1/payrates`)

Payrate CRUD and management.

### Advance Payments / FlexPay (`/api/v1/advance-payments`)

FlexPay lifecycle, fee calculation, import/export, reconciliation, settlement, file download.

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/request` | Create advance payment request |
| POST | `/:id/cancel` | Cancel request |

### Wallet (`/api/v1/wallet`)

Balance sync/adjust, topups, payments, reconciliation.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/balance` | Get wallet balance |
| POST | `/balance/sync` | Sync balance from provider |
| POST | `/payments/:id/resolve` | Resolve a pending payment |

### Disbursement (`/api/v1/`)

Manual disbursement, bulk transfer, auto-bulk-transfer, fee estimation, reconciliation.

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/auto-bulk-transfer` | Trigger automatic bulk transfer |
| POST | `/check-account` | Pre-flight bank account verification |
| POST | `/admin/manual-disbursement/employee-account-check` | Admin-only, rate-limited read-only verification of an employee's persisted bank account through the active disbursement provider |
| POST | `/webhooks/disbursement/{provider}` | Payment provider IPN webhook |

`POST /admin/manual-disbursement/employee-account-check` accepts `{ "employee_id": <number> }`.
The server loads the employee's persisted bank, SWIFT code, account number, and
account name; callers cannot provide or override those bank details. The lookup
does not create a payment, transfer, transaction code, wallet entry, or employee
update. The response reports `outcome` as `valid`, `invalid`, `name_mismatch`,
or `unverified`, alongside the stored bank tuple and provider result. Common
errors are `employee_not_found`, `employee_bank_data_incomplete`,
`account_verifier_unavailable`, and `account_verification_provider_error`.

The Admin wallet lookup dialog also supports one-off manual input through
`POST /admin/manual-disbursement/check-account`. That request accepts
`bank_code`, `account_no`, `account_name`, and `account_type`; it is read-only
and does not persist the entered values or create a transfer.

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

CRUD. Partner role has read-only access. Notable keys: `employee.mobile_required` (boolean, missing row = off) — when on, every employee create path requires a phone number.

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

### Machine-to-Machine Integrations (`X-API-Key`)

Two API-key surfaces exist outside the JWT/Casbin chain. Both are mounted on `v1` (so they inherit `APIRateLimit` + `APIMetrics`) but are **not** on the protected group, so neither JWT nor Casbin applies. Every key carries a `scope` column that binds it to exactly one surface.

| Group | Scope | Purpose |
|-------|-------|---------|
| `/api/v1/integration` | `chatbot` | Zalo chatbot: password reset, employee lookup, self check-in, Zalo OA token |
| `/api/v1/vendor` | `timesheet_read` | Read-only vendor timesheet surface (sister company) |

`RequireAPIKeyScope` refuses a valid-but-under-scoped key with **403**; an absent/invalid key is **401** from `Authenticate()` before the guard runs. That ordering keeps the two failure modes distinguishable to a caller holding no valid credential.

Admin management lives at `/api/v1/admin/api-keys` (`POST`, `GET`, `DELETE /:id`). `scope` defaults to `chatbot` when omitted, so pre-existing admin clients and keys keep working unchanged (migration 127 backfills the column).

### Vendor Timesheet Read API (`/api/v1/vendor`, scope `timesheet_read`)

**Read-only by construction.** Only `GET` is registered under `/vendor/*` — there is no mutating handler for the router to dispatch to, so `POST`/`PUT`/`PATCH`/`DELETE` return 404/405. Every write route in the system lives on the JWT-protected group, which an API key cannot satisfy. The payload is deliberately full-fidelity (money + PII) because the caller is a sister company that reconciles against it.

Records are addressed by **business codes, never internal IDs**: `projects.code` (unique) and `employees.cccd` (unique). Internal `id`s are withheld from responses so they cannot be cached as stable identifiers.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/vendor/projects` | Project vocabulary (`search`, `page`, `page_size`≤100) |
| GET | `/vendor/employees` | Employee vocabulary keyed by CCCD (`project_code`, `search`, `page`, `page_size`≤100) |
| GET | `/vendor/timesheets` | Timesheets by `project_code` / `cccd` + date range |

`/vendor/timesheets` accepts `project_code` and `cccd` (each repeatable **and** comma-separated), `from_date`/`to_date` (`YYYY-MM-DD`, Vietnam timezone), `status`, `payment_status`, `page`, `page_size`≤200. **At least one of `project_code` / `cccd` / `from_date` is required** — an unfiltered call returns 400 rather than dumping the whole table.

Timesheet rows carry `payrate`, `amount`, `paid_amount`, `payment_reference`, and `paid_at`; employee rows carry mobile, email, address, date of birth, and bank fields. `RevenueReceivable` / `RevenuePaid` are deliberately excluded — those are client-billing figures, not worker payouts.

Error semantics: unknown `project_code` / `cccd` returns **400, not 404** (the request was well-formed, the referenced code simply does not exist). Rate limit is 60/hour per key via `IntegrationRateLimit`.

## Middleware Chain

**Execution order:** recovery → security headers → request timeout → logging/metrics → auth → RBAC → rate limit → handler

| Middleware | File | Purpose |
|-----------|------|---------|
| `ErrorRecovery()` | `error_middleware.go` | Catches panics, logs stack traces, returns 500 |
| `SecurityHeaders()` | `security_headers.go` | X-Content-Type-Options, X-Frame-Options, CSP |
| `RequestTimeout(30s)` | `request_timeout.go` | Maximum request duration |
| `APIMetrics()` | `api_metrics.go` | Request latency, status codes, endpoint metrics |
| `Auth(jwtSecret)` | `auth.go` | JWT extraction and validation, sets user context |
| `APIKeyAuth.Authenticate()` | `api_key_auth.go` | `X-API-Key` → key id/name/scope in context (machine callers only) |
| `APIKeyAuth.RequireAPIKeyScope()` | `api_key_auth.go` | 403 when the authenticated key's scope does not cover the route |
| `IntegrationRateLimit` | `rate_limit.go` | 60/hour per API key id for `/integration/*` and `/vendor/*` |
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

Machine API keys do **not** go through Casbin at all. They authenticate via `X-API-Key` and are bounded by the `api_keys.scope` column instead (`chatbot` or `timesheet_read`), enforced by `RequireAPIKeyScope`.

## OpenAPI / Swagger

No OpenAPI/Swagger generation is currently in place. Route definitions in the `routes_*.go` files are the source of truth.
