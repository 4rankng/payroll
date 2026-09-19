---
type: integration
title: Background Jobs (asynq)
description: asynq task queue runtime, scheduler, retries, worker taxonomy (disbursement, bulk transfer, attendance, BCC import, IPN, payroll email, audit, recovery sweepers), and the queue topology.
tags: [integration, asynq, background-jobs, retry, recovery, idempotency]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-410ab36600fefdde765cb24e
    resource: repo://backend/AGENTS.md
  - id: openwiki-source-dfe3e58a9621ed8e846c8bd7
    resource: repo://backend/internal/app/services/scheduler/scheduler.go
  - id: openwiki-source-021f7f9379129decb4e6e69e
    resource: repo://backend/internal/app/workers/auto_reject_sweep_worker.go
  - id: openwiki-source-0a855c82f0a5d4b5c2d197b8
    resource: repo://backend/internal/app/workers/bcc_import_worker.go
  - id: openwiki-source-80a0f002a8c55cf1eeb8556f
    resource: repo://backend/internal/app/workers/disbursement_execute_worker.go
  - id: openwiki-source-582d96b5d108bf6aae1e2c21
    resource: repo://backend/internal/domain/bulk_transfer_batch.go
  - id: openwiki-source-43c18e617136f11e46e05ed9
    resource: repo://backend/internal/domain/flexpay_salary_notification.go
  - id: openwiki-source-28d4ebbc4a3f54433e091cbe
    resource: repo://docs/decisions/ADR-005-asynq-background-jobs.md
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Integration: Background Jobs (asynq)

Long-running or risky work is enqueued via [asynq](https://github.com/hibiken/asynq) and processed by workers registered in `internal/app/bootstrap/routes.go`. The asynq server is embedded in the api-server process and uses the same Redis instance as caching and the event bus. ADR-005 is the source of truth.

## Runtime

- Client: `internal/infra/asynq/` exposes the typed `asynq.Client` used by services to enqueue tasks.
- Server: asynq server runs in-process; `internal/infra/asynq/` wires the mux and the inspector.
- Tasks: defined in `internal/app/workers/`; the handler functions are registered in the mux at startup.

The task type strings are stable contracts between the enqueue site and the handler. Changing a task type is a breaking change and must be coordinated with the matching handler.

## Worker taxonomy

`internal/app/workers/` is organized by capability. Every worker follows the same shape: parse the task payload, look up the entity, do the work inside a transaction, register after-commit callbacks, emit non-blocking audit events on success/failure.

### Disbursement pipeline

- `disbursement_execute_worker.go` — calls the active provider (OnePay prod / 9Pay sandbox) with the wallet_payment row.
- `disbursement_poller_worker.go` — periodic provider status polling for rows stuck between `authorised` and the next IPN.
- `status_inquiry_worker.go` — single-payment status inquiry for missed-webhook recovery.
- `wallet_settlement_worker.go` — reconciles advance payments against the disbursed salary.
- `ipn_process_worker.go` — IPN handler; verifies the signature, looks up the wallet payment by invoice number, drives the state machine.

### Bulk transfer

- `bulk_transfer_worker.go` — top-level batch orchestration.
- `bulk_transfer_ninepay_execute_worker.go` — 9Pay-specific execute path.
- `bulk_transfer_transaction_worker.go` — revenue-transaction creation.
- `wallet_bulk_transfer_row_worker.go` — per-row processing.
- `bulk_transfer_worker_amount_test.go`, `bulk_transfer_worker_guard_test.go`, `bulk_transfer_worker_test.go` — coverage.

### Recovery sweepers

- `verified_transfer_recovery.go` — re-attempts booking for batches stuck in `completing` (BulkTransferBatch).
- `transfer_paid_amounts.go` — reconciles provider-side paid totals with local wallet payments.
- `verified_transfer_recovery_test.go`, `transfer_paid_amounts_test.go` — coverage.

### Attendance

- `auto_reject_checkout_worker.go` — closes per-gate checkout windows past `K+4h`.
- `auto_reject_sweep_worker.go` — project-wide sweeper for stale `checked_in` rows.
- `credit_quota_worker.go` — banks self-checkout earnings into the advance quota pool using the immutable per-row `quota_credit_eligible_at`.

### BCC import

- `bcc_import_worker.go` — runs the asynq-driven BCC import pipeline (upload → parse → dedup → label-rate → dispatch). See `features/bcc-import.md`.

### Notification & email

- `flexpay_salary_notification_worker.go` — salary-notification delivery (Zalo ZNS) for FlexPay employees.
- `payroll_report_email_worker.go` — weekly payroll report email.

### Audit and import jobs

- `audit_log_write_worker.go` — durable audit-log writer (for events that should never block the request).
- `employee_import_worker.go` — STK bank account import.
- `import_job_worker.go` — generic import job driver.

### Other

- `disbursement_execute_worker_test.go`, `disbursement_poller_worker_test.go` — coverage.
- `status_inquiry_worker_test.go` — coverage.
- `wallet_bulk_transfer_row_worker_test.go` — coverage.

## Retry and idempotency

asynq retries with backoff. Workers must be idempotent because a retry is not distinguishable from a first attempt. The pattern:

- Idempotency keys per task: `bulk_transfer_batches.enqueue_state` (pending → enqueued), `wallet_payments.txn_id`, `attendances.quota_credited_at`, `flexpay_salary_notifications.attempt` + lease model.
- Wrap multi-aggregate writes in `domain.TransactionManager.RunInTransaction`; cache invalidation runs as an after-commit callback per ADR-007.
- After-commit event publishing via Redis Streams is post-commit, so a crash between commit and publish is recoverable (republish from the outbox).

## Queue topology

asynq supports multiple queues with priorities. The current setup uses a small set of queues:

- `default` — short tasks (IPN process, single-row executes).
- `bulk` — batch-level work (bulk_transfer, BCC import).
- `cron` — scheduled tasks (auto-reject, poller, status inquiry, audit log writer).

Queues and priorities are tuned in `internal/infra/asynq/`. New task types pick an existing queue unless they have a distinct SLA.

## Scheduler

`internal/app/services/scheduler/` is the cron-style scheduler. It runs in-process and enqueues scheduled tasks at the configured cadence. The scheduler is the producer; asynq is the consumer. Restarting the api-server reschedules based on the persisted schedule.

## Observability

- Each worker logs at critical junctions via `slog` (never `fmt.Printf`), per `backend/AGENTS.md`.
- Workers emit Prometheus metrics through `internal/infra/observability/`.
- The system-health and cron-health UI surfaces (`frontend/src/components/system-health/`, `cron-health/`) consume these signals.
- Worker failures surface in the cron-health table with the last error and the next retry ETA.

## Tests

Worker tests are co-located with the worker code (`*_test.go`). Coverage:

- Happy path + amount guard tests for the bulk-transfer pipeline.
- Concurrency tests for attendance (two-tap race).
- Settlement and reconciliation tests.
- Poller / status-inquiry tests against mocked provider responses.

## Relationships

- Domain events — `integrations/event-bus.md`. Some workers consume events; some are scheduled; some are HTTP-driven.
- Wallet & ledger — `features/wallet-ledger.md`. IPN-driven state transitions and settlement.
- Salary disbursement — `features/salary-disbursement.md`. The pipeline orchestration.
- Attendance — `features/attendance-geofence.md`. Auto-reject and credit quota.
- BCC import — `features/bcc-import.md`. Import pipeline.
