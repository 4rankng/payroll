# Codebase Summary

High-level map of the payroll monorepo: where things live, how many lines, and what each area does.

## LOC Overview

| Area | Lines (approx) | Notes |
|------|---------------|-------|
| Backend (`internal/`) | 158K Go | 26+ service packages, 83 SQL migrations, ~50 repos |
| Frontend (`src/`) | 159K TS/TSX | 20+ component directories, 35+ hooks |
| Backend tests | 7.4K Go | Integration + unit tests |
| Migrations | 2.7K SQL | 83 `.up.sql` files |

## Backend Layers

All source under `backend/internal/`. Module: `api-server`.

### `domain/` (22K LOC) -- Core Business Rules

Entities, value objects, domain services, and port interfaces. No framework dependencies.

| Subdirectory | Contents |
|-------------|----------|
| `ports/` | Repository interfaces (`repository.go`), service interfaces (`service.go`) |
| `services/` | Pure domain logic: payroll calculation, salary calculation, timesheet validation, assignment conflict resolution, FlexPay reconciliation |
| `specs/` | Business specification objects (`working_days_spec.go`) |
| `transactions/` | Transaction management abstractions, wallet payment records, state machine |
| `wallet/` | Wallet aggregate: balance, payments, top-ups, IPN processing |
| Root | Entity structs: `Employee`, `Timesheet`, `Project`, `AdvancePayment`, `Settlement`, `LedgerEntry`, `User`, `Loan`, `Lender`, `Bank`. Event factory, error codes, accounting rules |

### `app/` (64K LOC) -- Application Services & Orchestration

| Subdirectory | Contents |
|-------------|----------|
| `services/` | 26+ service packages. Key ones: `advance_payment/` (19 files), `attendance/` (geofence, check-in/out, admin review), `timesheet/`, `employee/`, `wallet_service`, `ledger/`, `disbursement/`, `payroll/`, `settlement/`, `project/`, `bcc_import*` (BCC file parsing), `flex_pay/`, `notification/`, `push/`, `scheduler/`, `reporting/`, `config/`, `audit/`, `dashboard/`, `infrastructure/` (audit service, shared infrastructure) |
| `bootstrap/` | DI container (`container.go`), service init, repository init, infrastructure setup, route registration |
| `dto/` | Request/response shapes for API handlers |
| `workers/` | asynq background job handlers (see Async Jobs section) |
| `lib/` | Temporal date services (payrate, assignments) |
| `utils/` | Date range helpers, payroll period calculations |

### `infra/` (32K LOC) -- Infrastructure Adapters

| Subdirectory | Contents |
|-------------|----------|
| `persistence/` | 50+ GORM repository implementations. Grouped by entity (`employee_repository.go`, `timesheet_*.go`, `advance_payment_*.go`, etc.). `common/` has shared query builders, filter utilities |
| `asynq/` | asynq client, server, mux, handler registrations |
| `events/` | `InMemoryEventBus` implementation (publishes to registered handlers, not Redis streams -- see [System Architecture](system-architecture.md)) |
| `disbursement/` | Payment provider adapters: `ninepay/`, `onepay/` |
| `email/` | Email sending via Resend (production) or sandbox |
| `observability/` | `slog`-based structured logging, Prometheus metrics, DB metrics |
| `storage/` | File storage abstraction |
| `transaction/` | GORM transaction manager (unit of work) |

### `transport/http/` (28K LOC) -- API Delivery

| Subdirectory | Contents |
|-------------|----------|
| `handlers/` | Grouped by domain: `admin/`, `advance_payment/`, `attendance/`, `bank/`, `disbursement/`, `employee/`, `lender/`, `loan/`, `project/`, `project_employee/`, `push/`, `settings/`, `settlement/`, `timesheet/`. Also standalone: `auth.go`, `wallet_handler.go`, `ledger.go`, `health.go`, `cron_handler.go`, `metric_handler.go`, `db_export_handler.go` |
| `middleware/` | Auth (JWT), authorization (Casbin RBAC), rate limiting, IP whitelist, security headers, request timeout, audit context, tenant semaphore |
| `helpers/` | Request parsing, pagination, response formatting |
| `response/` | Error translation, response helpers |
| `validation/` | Request validation |

### `pkg/` (7.5K LOC) -- Shared Utility Packages

`bank`, `clock` (Clock interface, Asia/Ho_Chi_Minh), `context`, `db`, `geo`, `httputils`, `ipgeo`, `password`, `retry`, `scopes`, `services`, `tenantqueue`, `timeutil`, `ua`, `utils`, `validation`

### `cmd/` -- Entry Points

| Binary | Purpose |
|--------|---------|
| `api-server` | Main HTTP server + embedded asynq worker |
| `hashpw` | Password hashing utility |
| `seed-temp` | Database seed data |

### `configs/` -- RBAC Policy

`casbin_model.conf` (RBAC model) + `casbin_policy.csv` (~4.8K of role-permission mappings for Admin, Partner, Employee roles).

## Frontend Structure

All source under `frontend/src/`. Package: `vite_react_shadcn_ts`.

### `components/` (92K LOC) -- UI Library

Organized by domain, each with its own subdirectory:

| Directory | Domain |
|-----------|--------|
| `admin/`, `admin-dashboard/` | Admin pages, dashboard widgets |
| `attendance/` | Check-in/out cards, attendance history |
| `advance-payment/` | FlexPay requests, approvals |
| `dashboard/` | Main dashboard (employee view) |
| `disbursement/` | Disbursement management |
| `employees/` | Employee list, profiles |
| `ledger/` | Ledger entries view |
| `lenders/`, `loans/` | Lender and loan management |
| `partner/`, `partner-dashboard/`, `partner-employees/`, `partner-projects/`, `partner-timesheet/` | Partner-scoped views |
| `payrates/` | Pay rate configuration |
| `payroll/` | Payroll reports, sao ke |
| `projects/`, `project-employees/` | Project management |
| `timesheet/` | Timesheet views and editing |
| `wallet/` | Wallet balance, transactions |
| `system-health/`, `cron-health/` | System monitoring |
| `settings/` | App settings |
| `shared/`, `dialogs/`, `sheets/`, `modals/` | Shared UI primitives |
| `ui/` | shadcn/ui base components |

### `hooks/` (16K LOC) -- Custom Hooks

TanStack Query API wrappers in `hooks/api/` (admin, admin-dashboard, advance-payment, approvals, business, employees, ledger, partner, partner-dashboard, partner-employees, partner-timesheet, projects, settings, shared, timesheet, transactions, users). Plus 20+ utility hooks (useModalSystem, useBreakpoint, useDashboardNavigation, useDisbursementSettings, etc.).

### `pages/` (18K LOC) -- Route Pages

`admin/`, `adv-partner/`, `employee/`, `mobile/`, `partner/` -- each role has its own page set. `Login.tsx`, `NotFound.tsx`, `Index.tsx` at root.

### `contexts/` (1K LOC) -- React Contexts

`AuthContext`, `AppStateContext`, `BottomNavContext`, `CommandPaletteContext`, `MetadataContext`, `ShortcutContext`, `UserPreferencesContext`.

### `config/` & `constants/`

API configuration, dashboard configs, branding, email defaults, modal registry, salary period helpers, timesheet constants.

## Data Stores

| Store | Usage |
|-------|-------|
| **MySQL 8** | Primary database. Tables for employees, timesheets, projects, advance payments, wallet, ledger, settlements, attendance, audit logs, etc. 83 migrations. |
| **Redis** | Caching (cache invalidation after commits), asynq task queue, session store |

## Async Jobs (asynq)

Background workers in `backend/internal/app/workers/` and `backend/internal/infra/asynq/`:

| Worker | Task Type | Purpose |
|--------|-----------|---------|
| `import_job_worker` | `import:job` | BCC file import processing |
| `employee_import_worker` | `employee:import` | Employee data import from STK |
| `ipn_process_worker` | `ipn:process` | Payment provider IPN handling |
| `disbursement_execute_worker` | `disbursement:execute` | Individual disbursement execution |
| `disbursement_poller_worker` | `disbursement:poller` | Periodic disbursement status polling |
| `bulk_transfer_worker` | `bulk_transfer:payment` | Bulk transfer payment status updates |
| `bulk_transfer_ninepay_execute_worker` | `bulk_transfer:ninepay_execute` | 9Pay bulk batch execution |
| `bulk_transfer_transaction_worker` | `bulk_transfer:transaction` | Revenue transaction creation after bulk transfer |
| `wallet_settlement_worker` | - | Wallet settlement processing |
| `status_inquiry_worker` | - | Payment status inquiry |
| `payroll_report_email_worker` | - | Sao ke email delivery |
| `auto_reject_sweep_worker` | - | Sweep auto-rejected attendance |
| `auto_reject_checkout_worker` | - | Auto-reject attendance when checkout window closes |

## Where Do I Find X

| I want to... | Look in... |
|-------------|-----------|
| Add a domain entity | `backend/internal/domain/<entity>.go` |
| Define a repository interface | `backend/internal/domain/ports/repository.go` |
| Implement a repository | `backend/internal/persistence/<entity>_repository.go` |
| Add an API endpoint | `backend/internal/transport/http/handlers/<domain>/`, register in `app/bootstrap/` |
| Add a background job | `backend/internal/app/workers/`, register type in `infra/asynq/mux.go` |
| Add a domain event | `backend/internal/domain/events.go` + `event_factory_*.go` |
| Handle a domain event | `backend/internal/infra/events/` (register handler on EventBus) |
| Add an application service | `backend/internal/app/services/<domain>/` |
| Wire a new service | `backend/internal/app/bootstrap/container.go` |
| Change RBAC permissions | `backend/configs/casbin_policy.csv` |
| Add a frontend page | `frontend/src/pages/<role>/`, route in `src/App.tsx` or router |
| Add a frontend component | `frontend/src/components/<domain>/` |
| Add a TanStack Query hook | `frontend/src/hooks/api/<domain>/` |
| Add a frontend modal | `frontend/src/components/dialogs/` or `sheets/` |
| Change Vietnamese UI text | Directly in `.tsx` files (no i18n layer) |
| Change API config | `frontend/src/config/api.config.ts` |
