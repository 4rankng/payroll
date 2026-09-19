---
type: architecture
title: Backend Domain Layer
description: The inner ring of DDD — entities, value objects, domain events, ports, domain services, the wallet aggregate with its state machine and double-entry ledger, and the property-based tests that guard the accounting invariants.
tags: [backend, domain, ddd, wallet, ledger, ports, state-machine]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-9f8ae5750267e989e29ed553
    resource: repo://backend/internal/domain/accounting_rules.go
  - id: openwiki-source-a5c9de3ea4280bad6936ca7f
    resource: repo://backend/internal/domain/AGENTS.md
  - id: openwiki-source-9769e4afb6f2a41f9f5d010e
    resource: repo://backend/internal/domain/ports/infrastructure/audit.go
  - id: openwiki-source-80d81d11a3b80acbb0cff4e4
    resource: repo://backend/internal/domain/ports/services/ledger.go
  - id: openwiki-source-50c7d39e20d2fbd3fc8c2e0c
    resource: repo://backend/internal/domain/transactions/state_machine.go
  - id: openwiki-source-0f104d87e52630bb474cdbe0
    resource: repo://backend/internal/domain/wallet/repository.go
  - id: openwiki-source-f10a0a57b8b00bec9b396088
    resource: repo://docs/decisions/ADR-001-ddd-clean-architecture.md
  - id: openwiki-source-aca61ad26f85a3f6b9e0d210
    resource: repo://docs/decisions/ADR-006-clock-injection-pattern.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Backend Domain Layer

`internal/domain` is the innermost ring. It owns the business rules and the vocabulary the rest of the codebase speaks. It contains entities, value objects, domain events, repository and service interfaces (the *ports*), pure domain services, specification objects, the transaction-manager abstraction, and the wallet aggregate. ADR-001 is the source of truth for why this layer exists and what it cannot import.

## Zero-framework invariant

The domain package has zero framework imports. No Gin, no GORM, no Redis, no asynq, no SMTP client. Only Go's standard library and `internal/pkg/clock`. Every external dependency is hidden behind an interface in `internal/domain/ports/`. This is enforced by the package dependency graph (`transport → app → domain ← infra`) and by code review.

The cost is real — GORM models in `infra/persistence/` must implement `ToDomain()` mappers — but the benefit is that the entire payroll rule set is testable without a database or HTTP server. See `docs/decisions/ADR-001-ddd-clean-architecture.md` for the trade-off analysis.

## Entity groups

Domain entities are flat files under `internal/domain/`. The grouping below tracks how the codebase organizes them by subdomain. New entities pick the closest existing file; if a new subdomain emerges it gets its own file.

- **Identity and access** — `user.go`, `employee.go`, `employee_user.go`, `security.go` (password hashing, token generation), `auth_*` helpers. `error_codes.go` + `errors.go` standardize the typed errors (`NewNotFoundError`, `NewValidationError`, `NewConflictError`, etc.) used across layers.
- **Project structure** — `project.go`, `project_user.go`, `project_employee.go` (assignment lifecycle and constraints), `payrate.go` (temporal validity of rate cards), `assignment_*` domain services.
- **Timesheets** — `timesheet.go`, `timesheet_types.go` (day types, overtime, shifts), `timesheet_edit_request.go`, `timesheet_import_job.go`. Validation lives in `services/timesheet_validation_*.go` (daily hours, daytype, status, zero-rate). See `features/timesheet.md`.
- **Advance payments / FlexPay** — `advance_payment.go`, `advance_payment_request.go`, `advance_payment_fee_schedule.go`, `flexpay_salary_notification.go`. The fee schedule is a tiered structure evaluated against the request amount.
- **Wallet, ledger, accounting** — `wallet/` aggregate (see below), `ledger.go`, `accounting_rules.go`, `transaction.go`, `transaction_code.go`, `settlement*.go`, `loan.go`, `loan_strategy.go`, `bulk_transfer_*.go`. Every financial operation lands in the ledger as paired debit + credit.
- **Attendance** — `attendance.go`, `project_employee_checkin_pending.go`. Concurrency and the advance-hold-hours / quota-credit rules are domain-layer concerns.
- **Audit** — `audit_diff.go`, `audit_templates.go`, plus `audit_log_property_test.go`. Audit messages are standardized through templates so log scrapers can match them.
- **Notification and push** — `notification.go`, `push_subscription.go`, `email.go`, `flexpay_salary_notification.go`. Push subscriptions are stored as PWA-specific records.
- **System / dashboard** — `dashboard.go` (analytics query structures), `search.go` (full-text search types), `settings.go`, `cron_job_status.go`, `cash_forecast_snapshot.go`, `cash_readiness.go`, `api_metric.go`, `api_endpoint.go`, `transaction_manager.go`.

Domain events live in `events.go` (50+ event types) plus per-aggregate factories in `event_factory_*.go`. Events are the source of truth for cross-aggregate side effects; they are published to Redis Streams via the outbox pattern, never emitted directly from handlers.

## Ports

`internal/domain/ports/` is split by role:

- **`ports/infrastructure/`** — adapters the domain needs from infra: `audit.go`, `cache.go`, `disbursement.go`, `notification.go`. Each interface is implemented in `internal/infra/...`. `cache.go` exposes `Invalidate(ctx, key)` and friends; `disbursement.go` is the provider-agnostic surface that OnePay and 9Pay satisfy.
- **`ports/services/`** — services that are not technically domain services but live close to the domain: `ledger.go`, `loan.go`, `payroll.go`, `settlement.go`, `timesheet.go`, `transaction.go`. Application services compose these with ports to deliver use cases.

The wallet aggregate is the only place in the codebase where repository interfaces are scoped inside the aggregate (`internal/domain/wallet/repository.go`) rather than the global ports directory. This keeps the wallet a self-contained bounded context.

## Domain services

`internal/domain/services/` holds pure business logic — code that needs no infrastructure but operates on entities, value objects, and ports.

- **`timesheet_domain_service*.go`** — bulk create / update / preview, daily-hours, daytype, status, zero-rate, validation, summary. The split into many files keeps each concern small and named clearly.
- **`payroll_calculation_service.go`** / **`salary_calculation_service.go`** — payroll math, broken into calculator + report-by-project service.
- **`flex_pay_reconciliation_service.go`** — matches 9Pay reconciliation CSV rows to wallet payments.
- **`assignment_lifecycle_manager.go`**, **`assignment_conflict_resolver.go`**, **`employee_assignment_service.go`**, **`employee_domain_service.go`**, **`paytype_construction_service.go`**, **`payroll_report_service.go`** — supporting services for project assignment, payroll period construction, and reporting.

Domain services are constructed by `bootstrap/services/init.go` and depend on ports, never on infra concretes.

## The wallet aggregate

The wallet aggregate (`internal/domain/wallet/`) is the most concentrated piece of business logic in the codebase. It is a self-contained bounded context: it owns its types, its repository interfaces, its service interface, its state machine, and its forecast. Files:

| File | Purpose |
|------|---------|
| `wallet_payment.go` | `WalletPayment` type, filter, response, and stats shapes |
| `wallet_topup.go` | `WalletTopup` type and its filter |
| `wallet_balance.go` | Balance computation (provider + local reconciliation) |
| `wallet_ipn.go` | IPN (Instant Payment Notification) handling |
| `service.go` | `WalletService` interface — the aggregate's use-case surface |
| `repository.go` | `WalletTopupRepository`, `WalletPaymentRepository`, `WalletIPNRepository` interfaces (scoped here, not in `ports/`) |
| `forecast.go` | Cash-demand forecast that drives top-up timing |

The state machine lives at `internal/domain/transactions/state_machine.go` and uses `github.com/qmuntal/stateless`. States are `pending → verified → authorised → completed | failed | reversed`. Triggers are `verify`, `reject`, `authorise`, `ipn_completed`, `ipn_failed`, `ipn_reversed`. `IsTerminal` flags `completed`, `failed`, `reversed`. `Hooks` wires service-layer callbacks into `OnEnterCompleted`, `OnEnterFailed`, `OnEnterReversed` so the aggregate can react to terminal transitions (e.g., write ledger entries on `completed`).

### Double-entry ledger

Every financial operation produces a balanced group of ledger entries: `AccountantRules.ValidateTransactionGroup` (`internal/domain/accounting_rules.go`) sums debits and credits and rejects any group where they do not match. `ValidateBalancedTransaction` enforces the simpler debit/credit pair shape. The accounting helpers live in `internal/app/accounting/` (a `Money` type with safe arithmetic) and the ledger entry shape in `internal/domain/ledger.go`.

The accounting rules are verified by property-based tests (`accounting_rules_test.go` using `gopter`): random inputs are generated, balanced groups must validate and unbalanced groups must fail. This catches off-by-one and rounding regressions that table-driven tests would miss.

### Domain events and outbox

Wallet operations emit events through the per-aggregate event factories (`event_factory_financial.go`, `event_factory_disbursement_fee.go`, `event_factory_weekly_payment_fee.go`, etc.). Events are persisted in the same DB transaction as the state change and pushed to Redis Streams post-commit, so a crash between commit and publish is recoverable.

## Specifications and time abstractions

`internal/domain/specs/` holds business specifications — most notably `working_days_spec.go` (and its property tests), which encodes how to count working days for payroll periods.

All business time goes through `internal/pkg/clock/` per ADR-006. The `Clock` interface exposes `Now`, `NowUTC`, `TodayStart`, `TodayEnd`, `UnixNow`. `RealClock` is used in production; `FakeClock` and `AutoFake` power tests. Admin clock endpoints (`/api/v1/admin/clock/set|advance|reset|time`) are mounted only in non-prod builds. The narrow exceptions (event-bus duration measurements, health-check response time) are documented in the ADR and are not domain logic.

## Error vocabulary

Domain errors are constructed with helpers in `errors.go` (`NewNotFoundError`, `NewValidationError`, `NewConflictError`, `NewUnauthorizedError`, etc.). `error_codes.go` exposes the standardized codes the transport layer maps to HTTP status and Vietnamese user-facing messages. Error constructors return `error`; the transport layer's `response/` package is the only place that translates them.

## What lives where — quick reference

| Concern | Where |
|---------|-------|
| Entities and value objects | `internal/domain/*.go` |
| Repository & service interfaces | `internal/domain/ports/{infrastructure,services}/*.go` (or scoped in `internal/domain/wallet/repository.go` for the wallet) |
| Domain services | `internal/domain/services/*.go` |
| Business specifications | `internal/domain/specs/*.go` |
| Wallet aggregate (types, ports, FSM) | `internal/domain/wallet/*.go` + `internal/domain/transactions/state_machine.go` |
| Domain events | `internal/domain/events.go` + `internal/domain/event_factory_*.go` |
| Property-based accounting tests | `internal/domain/accounting_rules_test.go` (gopter) |
| Transaction manager interface | `internal/domain/transaction_manager.go` |

See `architecture/overview.md` for how this layer sits relative to `app/`, `infra/`, and `transport/`, and `features/wallet-ledger.md` for the consumer-side view.
