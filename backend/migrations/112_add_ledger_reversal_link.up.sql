-- Record what a ledger reversal reverses, and why.
--
-- Why
-- ---
-- `POST /ledger/entries/:id/reverse` wrote the mirror of ONE entry through the
-- unguarded `Create` path. That is wrong twice over:
--
--   1. A ledger block must balance. Reversing one leg of a balanced block adds a
--      one-sided entry, so `SUM(debit)` and `SUM(credit)` drift apart by the
--      reversed amount and never come back. Every reversal used to leave the
--      books permanently out of balance.
--   2. The mirror carried no link to its original, so nothing could tell that an
--      entry had already been reversed. Calling reverse twice produced two
--      mirrors and doubled the reversal, and the operator interface had no way
--      to show the state.
--
-- `reversal_of_entry_id` gives each mirror an explicit link to the entry it
-- offsets, which makes "already reversed?" a lookup instead of a guess.
-- `reversal_reason` stores the reason the caller supplies — the API accepted it
-- and dropped it, so the audit trail the operator filled in was discarded.
--
-- Grouping
-- --------
-- Reversals are written for a whole balanced block, and the block is resolved
-- from `transaction_id` when present. Entries created through the manual
-- `/ledger/entries` endpoint carry no transaction id; those are written in a
-- single INSERT, so all legs share `created_at` to the millisecond and are
-- grouped by (`created_at`, `created_by`).
--
-- Existing rows
-- -------------
-- Both columns stay NULL for historical rows. They are the only rows that can be
-- reversed without a link, which is exactly the intended behaviour: an old
-- one-sided entry cannot be reversed (it is not a balanced block), and the
-- endpoint has never been called in production (no `%reverse%` path in
-- api_metrics), so no backfill is required.
--
-- Operational note
-- ----------------
-- Adds two nullable columns and one index to ledger_entries (1.271 rows at the
-- time of writing). MySQL 8 applies the ADD COLUMN operations instantly; the
-- ADD INDEX is an in-place build that briefly blocks writes to the table. Cheap
-- at this size, and the table's writers are the payroll workers.
--
-- Reversibility
-- -------------
-- The down migration drops the index and both columns. Dropping them discards
-- reversal links and reasons, so a reversal written after this migration becomes
-- indistinguishable from a plain entry — the state this migration exists to fix.

ALTER TABLE ledger_entries
    ADD COLUMN reversal_of_entry_id BIGINT UNSIGNED NULL
        COMMENT 'Entry this row reverses (mirror), NULL for originals' AFTER transaction_id,
    ADD COLUMN reversal_reason VARCHAR(255) NULL
        COMMENT 'Operator-supplied reason for the reversal' AFTER reversal_of_entry_id,
    ADD INDEX idx_ledger_entries_reversal_of (reversal_of_entry_id);
