-- Migration: 042_backfill_audit_log_actor_for_bulk_transfer_downstream.up.sql
--
-- Backfills audit_logs rows that were written with user_id=0 because the
-- BulkTransferTransactionWorker handed downstream services a context.Background()
-- (the event-bus worker strips the originating ctx). The audit-emit factories
-- for LedgerEntryCreated and TransactionCreated read the actor from ctx only,
-- so the ledger entries and revenue transactions created from a bulk transfer
-- result file were audited as user_id=0.
--
-- Source of truth for the real actor: bulk_transfer_files.created_by, which is
-- the user who uploaded the result file. We join to it through the asset that
-- both the bulk_transfer_files row and the audited entity (ledger_entry /
-- transaction) reference.
--
-- Idempotent: re-running this migration is safe because the WHERE clause
-- filters on user_id=0 and we only set rows where the join finds a non-zero
-- created_by.
--
-- Scope: only ledger_entry and transaction audit rows. Failed-login audit rows
-- (entity_type='user', action='LOGIN') legitimately carry user_id=0 when the
-- attempted username matches no user — those are NOT touched.

-- Backfill ledger_entry audit rows
UPDATE audit_logs a
JOIN ledger_entries le        ON le.id = a.entity_id
JOIN bulk_transfer_files btf  ON btf.asset_id = le.asset_id
SET a.user_id = btf.created_by
WHERE a.user_id = 0
  AND a.entity_type = 'ledger_entry'
  AND btf.created_by > 0;

-- Backfill transaction audit rows
UPDATE audit_logs a
JOIN transactions t           ON t.id = a.entity_id
JOIN bulk_transfer_files btf  ON btf.asset_id = t.asset_id
SET a.user_id = btf.created_by
WHERE a.user_id = 0
  AND a.entity_type = 'transaction'
  AND btf.created_by > 0;
