# Phase 1: Measurement foundation

**Status:** Completed

## Files

- `backend/migrations/090_cash_forecast_snapshots.up.sql`
- `backend/migrations/090_cash_forecast_snapshots.down.sql`
- `backend/internal/domain/cash_forecast_snapshot.go`
- `backend/internal/infra/persistence/cash_forecast_snapshot_repository.go`
- repository bootstrap wiring
- `backend/internal/infra/events/cash_forecast_accuracy_handler.go`

## Implementation

1. Add a snapshot table keyed by company scope, target cycle, model version, and
   cycle day. Store input decomposition, point/reserve/interval outputs,
   generated time, and nullable actual outcome fields.
2. Implement GORM upsert, resolved-residual lookup, accuracy aggregation, and
   idempotent outcome resolution.
3. Subscribe an event handler to company-wide weekly
   `BulkTransferFileExportedEvent` events emitted only after workbook generation.
   Resolve the exact work-date range through a distinct timesheet outcome ledger:
   split files accumulate, retries deduplicate, and forced backlog outside the
   declared range is excluded.
4. Unit-test upsert identity, company-only measurement, date parsing, non-weekly
   event rejection, and duplicate outcome handling.

## Resolution notes

Migration 091 implements the timesheet-ID-deduplicated outcome source. Outcome
amounts are allocated from the exact post-percentage employee/project workbook
total, including its integer rounding remainder.
