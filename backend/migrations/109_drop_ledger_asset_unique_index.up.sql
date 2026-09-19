-- Drop the mis-specified unique key `idx_ledger_asset_unique` (asset_id, deleted_at).
--
-- Why it is wrong
-- ---------------
-- MySQL treats NULLs as distinct in unique indexes, so `(asset_id, NULL)` rows are
-- never constrained. The index therefore enforced nothing for live rows — exactly
-- the thing its name implies — and could only ever fire on soft deletes.
--
-- On soft delete it always fires: GORM issues one statement per row-set with a
-- single `deleted_at` value, so any transaction whose ledger block contains two
-- rows sharing a non-null `asset_id` collides. Every bulk-transfer receivable does
-- exactly that (a `receivable` debit and a `revenue` credit, both carrying the file
-- asset). The failure is:
--
--   ERROR 1062 (23000): Duplicate entry '400-2026-09-18 21:06:18.894'
--     for key 'ledger_entries.idx_ledger_asset_unique'
--
-- which rolls the whole delete back and surfaces in the UI as the generic
-- HTTP 500 "Không thể hủy giao dịch" (DELETE /api/v1/transactions/{id}).
--
-- Replacement
-- -----------
-- A *plain* index on `asset_id` is required and is added here: the foreign key
-- `fk_ledger_entries_asset` currently leans on `idx_ledger_asset_unique` as its
-- only `asset_id` index, so dropping the unique key first fails with
--   ERROR 1553: Cannot drop index ... needed in a foreign key constraint.
-- The replacement is deliberately non-unique. "One ledger entry per asset" is not
-- an invariant of this schema: a receivable block legitimately writes both a
-- `receivable` and a `revenue` entry against the same asset (e.g. ledger_entries
-- 995/996 both carry asset_id 403), so a functional unique index over live
-- `asset_id` would reject the application's own writes.
--
-- Reversibility
-- -------------
-- Deliberately no `.down.sql`: re-adding the unique key can only succeed while no
-- duplicate `(asset_id, deleted_at)` pair exists, i.e. only while no cancellation
-- has ever succeeded — precisely the state this migration exists to leave behind.
-- A rollback script that fails in every realistic case is worse than none.
--
-- Safe to re-run: each step is emitted only while it still applies.

-- Step 1 — plain `asset_id` index for the FK (and for asset-scoped lookups).
-- `idx_ledger_asset_unique` is excluded from the check: it also leads with
-- `asset_id`, so counting it would skip the add and step 2 would still fail.
SET @schema_109_add = (
    SELECT IF(
        COUNT(*) = 0,
        'ALTER TABLE ledger_entries ADD INDEX idx_ledger_entries_asset (asset_id)',
        'SELECT ''asset_id index already present'''
    )
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'ledger_entries'
      AND column_name = 'asset_id'
      AND seq_in_index = 1
      AND index_name <> 'idx_ledger_asset_unique'
);

PREPARE schema_109_add_stmt FROM @schema_109_add;
EXECUTE schema_109_add_stmt;
DEALLOCATE PREPARE schema_109_add_stmt;

-- Step 2 — drop the unique key.
SET @schema_109_drop = (
    SELECT IF(
        COUNT(*) > 0,
        'ALTER TABLE ledger_entries DROP INDEX idx_ledger_asset_unique',
        'SELECT ''idx_ledger_asset_unique already absent'''
    )
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'ledger_entries'
      AND index_name = 'idx_ledger_asset_unique'
);

PREPARE schema_109_drop_stmt FROM @schema_109_drop;
EXECUTE schema_109_drop_stmt;
DEALLOCATE PREPARE schema_109_drop_stmt;

-- Verify: unique key gone, an `asset_id` leading index present, and the FK intact.
SELECT
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'ledger_entries'
        AND index_name = 'idx_ledger_asset_unique') AS unique_key_remaining,
    (SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'ledger_entries'
        AND column_name = 'asset_id' AND seq_in_index = 1) AS asset_id_indexes,
    (SELECT COUNT(*) FROM information_schema.referential_constraints
      WHERE constraint_schema = DATABASE() AND constraint_name = 'fk_ledger_entries_asset') AS asset_fk_present;
