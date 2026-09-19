---
type: architecture
title: Backend Application Services
description: How use cases are orchestrated in internal/app — service packages by capability, the transaction-manager unit-of-work pattern, the after-commit callback discipline, and the asynq worker taxonomy.
tags: [backend, app, services, workers, transaction-manager, asynq, ddd]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-3d0b00c530be91b05c860949
    resource: repo://backend/internal/domain/transaction_manager.go
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Backend Application Services

The `internal/app` tree is where domain rules meet infrastructure: it owns the DI container, the request/response DTOs, the per-capability use-case services, and the asynq worker definitions. Handlers depend on this layer; this layer depends on `domain` ports and `infra` adapters — never the other way around.

## Layer map

```
internal/app/
├── bootstrap/         DI container, route registration, middleware wiring, cron scheduler
├── dto/               Request/response shapes (Vietnamese labels live here, not in handlers)
├── lib/               Temporal date services (payrate effective dates, assignment windows)
├── services/          40+ per-capability service packages
├── utils/             Shared helpers (date ranges, payroll-period math)
├── workers/           asynq task handlers
└── accounting/        App-level accounting helpers
```

Each capability has its own package — `services/timesheet/`, `services/wallet_bulk/`, `services/disbursement/`, etc. A package typically exposes one or more service structs that own the use cases for that capability, plus DTO helpers, request/response types, and unit tests. The DI wiring lives in `bootstrap/services/init.go`; route registration is in `bootstrap/routes.go`.

## Transaction manager (unit of work)

Multi-aggregate writes must be atomic. Every multi-step service uses the `domain.TransactionManager` interface (`backend/internal/domain/transaction_manager.go:10`):

```go
err := s.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
    if err := s.repo.Save(txCtx, entity); err != nil { return err }
    if err := s.otherRepo.Update(txCtx, other); err != nil { return err }
    return nil // auto-commit
})
// After this returns without error, cache invalidation and event publish can safely run
```

The concrete implementation lives in `internal/infra/transaction/` (GORM-backed). Repositories detect whether the context carries a `TransactionContext` and either join the open `*gorm.DB` or use the default connection. Panics inside the closure trigger automatic rollback.

### After-commit callbacks

`RegisterAfterCommit(ctx, fn)` schedules a function to run only after the surrounding transaction commits. If no transaction is active, it runs in a goroutine immediately. `RunAfterCommitCallbacks` is called by the transaction manager after a successful commit and fires every registered callback in its own goroutine. Use this for:

- Cache invalidation (never inside the transaction — see ADR-007).
- Domain event emission to Redis Streams (publish-after-commit, paired with the outbox pattern).
- Web-push fan-out that depends on the persisted state.

`WithoutTransactionContext(ctx)` shadows any inherited transaction, useful when a post-commit hook needs to call back into a service that should not reuse the already-committed TX.

### Cache invalidation discipline

`backend/internal/infra/transaction/gorm_transaction_manager.go` enforces the rule from ADR-007:

```go
err := s.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
    return s.repo.Save(txCtx, entity)
})
if err == nil {
    s.cache.Invalidate(ctx, cacheKey)  // AFTER commit — safe
}
```

Invalidating mid-transaction lets a concurrent reader repopulate the cache with pre-commit data. Code review enforces the rule; integration tests cover the rollback case.

## Service packages by capability

`internal/app/services/` is organized one package per product capability. The list below captures the meaningful clusters; new packages follow the same shape.

- **`timesheet/`** — `creator.go`, `validator.go`, `workflow.go`, `bulk_operations.go`, `edit_request_service.go`, `reject_unpaid.go`, `transaction_orchestrator.go`, `response_service.go`. The orchestrator owns multi-step writes (approve → ledger → event) and is the canonical user of the transaction manager.
- **`advance_payment/`** and **`flex_pay/`** — request lifecycle, fee schedule application, settlement service, reconciliation exporter, salary-notification hooks. See `features/flexpay.md`.
- **`wallet_bulk/`** — `booking.go`, `parser.go`, `kq_generator.go`, `reversal.go`, `queue.go`, `read_methods.go`. Owns the file → batch → row pipeline that feeds `disbursement/`. See `features/salary-disbursement.md`.
- **`disbursement/`** — `registry.go`, `registry_selector.go`, `fee_schedule_service.go`, `provider_transaction_service.go`, `wallet_payment_service.go`. Picks the active payment provider at runtime (OnePay in prod, 9Pay in sandbox).
- **`attendance/`** — `attendance_checkin.go`, `attendance_checkout.go`, `attendance_geofence.go`, `attendance_query.go`, `attempt_classifier.go`. Enforces concurrency on check-in/out and the advance-hold-hours rule.
- **`bcc_import_*`** — split across many files for backdate, dedup, dispatch, label-rate, multi-position, weekly-rates, weekly payment, weekly BCC, weekly shared, weekly payrate, weekly BCC processing. Pipeline: parse → dedup → label rate → dispatch into timesheets. See `features/bcc-import.md`.
- **`cash_readiness_*`** — `forecast.go`, `forecast_v2_math.go`, plus backtest and forecast-cache tests. The statistical demand forecast that drives wallet top-up timing (ADR-010).
- **`ledger/`, `loan/`, `settlement/`** — financial-services layer. Ledger entries are written inside transactions by other services; these packages own reads, reconciliation, and reporting.
- **`audit/`, `reporting/`, `dashboard/`, `db_export/`** — read-side and admin services; emit non-blocking audit events for file imports/exports.
- **`auth/`, `otp/`, `passwordreset/`, `zaloconnect/`, `zaloreset/`** — authentication and recovery flows; the Zalo services sit on top of the `internal/infra/zalo/` stateless client.
- **`push/`, `notification/`** — push-subscription lifecycle and notification fan-out; web-push delivery runs through the `internal/infra/events/` bus.
- **`payroll/`, `scheduler/`, `cleanup/`** — payroll-period helpers, cron-scheduler wiring, post-run cleanup.
- **`employee/`, `project/`, `asset/`, `ad_banner/`, `lender/` (via `loan/`)** — supporting entity services.

Every service follows the same struct shape: it holds ports (`repo`, `otherRepo`), collaborators (`txManager`, `cache`, `eventBus`, `clock`), and the `auditService` used for non-blocking import/export audit logs.

## DTO conventions

`internal/app/dto/` holds request/response shapes, separated from domain types so handlers can change wire contracts without touching entities. Conventions:

- One file per capability (`employee.go`, `advance_payment.go`, `cash_readiness.go`).
- Vietnamese label strings live in DTOs and translation tables — not in handlers — so a label change stays in one file.
- DTOs use the same validation tags consumed by `internal/transport/http/validation/`.
- Pagination, sorting, and filter types live in a small `helpers/` package and are reused.

## Handlers / services split

Handlers in `internal/transport/http/handlers/` translate HTTP into service calls and translate domain errors into response envelopes. They:

- Do not import `infra/persistence` or GORM models.
- Receive the actor (from middleware) and pass it as `context.Context` to the service.
- Never construct a transaction; they call the service, which calls `txManager.RunInTransaction`.
- Emit non-blocking audit events for imports/exports via the injected `auditService`.

This split keeps handlers thin and lets the same service be reused by tests, future RPC/gRPC surfaces, or background workers that need the same business workflow.

## Workers (asynq tasks)

`internal/app/workers/` defines asynq task handlers. They are registered in `bootstrap/routes.go` and run against the Redis queue. The taxonomy:

- **Disbursement pipeline** — `disbursement_execute_worker.go`, `disbursement_poller_worker.go`, `status_inquiry_worker.go`, `wallet_settlement_worker.go`. Drive the bulk-transfer → provider execute → IPN → settlement loop. See `features/salary-disbursement.md` and `integrations/payment-providers.md`.
- **Bulk transfer** — `bulk_transfer_worker.go`, `bulk_transfer_ninepay_execute_worker.go`, `bulk_transfer_transaction_worker.go`, `wallet_bulk_transfer_row_worker.go`. Row-level fan-out.
- **Recovery sweepers** — `verified_transfer_recovery.go`, `transfer_paid_amounts.go`. Detect stuck payments and recover them.
- **BCC import** — `bcc_import_worker.go`. Drives the weekly import pipeline asynchronously after upload.
- **Attendance** — `auto_reject_checkout_worker.go`, `auto_reject_sweep_worker.go`, `credit_quota_worker.go`. Sweep stale pending check-ins and credit attendance quota.
- **Notification** — `flexpay_salary_notification_worker.go`, `payroll_report_email_worker.go`. Deliver FlexPay deductions and weekly payroll emails.
- **Audit and import jobs** — `audit_log_write_worker.go`, `employee_import_worker.go`, `import_job_worker.go`.
- **IPN processing** — `ipn_process_worker.go`. Validates payment-provider callbacks and applies wallet state transitions.

Workers follow the same rules as services: depend on ports, run inside transactions, register after-commit callbacks, and emit non-blocking audit events.

## Bootstrap and wiring

`internal/app/bootstrap/container.go` is the root DI container. `bootstrap/services/init.go` constructs every service struct (holding ports, transactions, clock, audit, cache, event bus); `bootstrap/routes.go` mounts the Gin router and registers asynq handlers. Anything new — domain port, repo, service, handler, route — is wired here exactly once.

See `architecture/transport-http.md` for the handler-side view and `integrations/background-jobs.md` for the asynq runtime details.
