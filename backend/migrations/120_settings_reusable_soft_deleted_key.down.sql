-- Restore the bare UNIQUE(key) index on `settings` and drop the generated
-- helper column.
--
-- WARNING: this restores the original trap. After this migration a
-- soft-deleted row once again blocks recreation of its key, and the create path
-- will again report that as a duplicate-key error. Re-applying migration 120 is
-- the correct state for any environment that soft-deletes settings.

ALTER TABLE settings
    DROP INDEX uniq_settings_active_key,
    DROP COLUMN active_key,
    ADD UNIQUE INDEX idx_settings_key (`key`);