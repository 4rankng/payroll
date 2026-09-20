-- Reverses 112_add_ledger_reversal_link.up.sql.
--
-- Drops the reversal link and reason. Applying this after reversals have been
-- written loses their provenance: the mirrors remain (the ledger stays balanced)
-- but nothing records that they offset another entry, and "already reversed?" can
-- no longer be answered.

ALTER TABLE ledger_entries
    DROP INDEX idx_ledger_entries_reversal_of,
    DROP COLUMN reversal_reason,
    DROP COLUMN reversal_of_entry_id;
