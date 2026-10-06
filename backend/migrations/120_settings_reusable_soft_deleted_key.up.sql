-- Let a soft-deleted settings key be reused, while still allowing only one
-- ACTIVE row per key.
--
-- Why
-- ---
-- `settings.key` carried a UNIQUE index (idx_settings_key) while the table also
-- soft-deletes. That combination is a trap: a soft-deleted row keeps holding
-- its key, so recreating that setting can never succeed. The create path
-- surfaced the duplicate-key as an opaque HTTP 500 ("Không thể tạo cài đặt")
-- instead of a conflict, so the settings tests only failed on the *second* run
-- against the same database — read as flaky, but it was deterministic.
--
-- Why a generated column rather than UNIQUE (key, deleted_at)
-- ---
-- The composite form that `employees` uses (unique_email_deleted_at,
-- unique_cccd_deleted_at) does NOT actually prevent duplicates among active
-- rows: MySQL treats NULLs as distinct inside a unique index, so two rows with
-- the same key and deleted_at = NULL are both permitted. That flaw is why
-- `employees` currently holds 6 active rows sharing a CCCD.
--
-- A generated column that is NULL exactly when the row is soft-deleted gives
-- the real guarantee: every active row produces a non-NULL value (so duplicates
-- are rejected), while every deleted row produces NULL (which never collides).
--
-- Pre-check, run before applying — every key listed here must have at most one
-- active row, otherwise the new index will fail to build:
-- SELECT `key`, COUNT(*) FROM settings
--  WHERE deleted_at IS NULL AND `key` IS NOT NULL
--  GROUP BY `key` HAVING COUNT(*) > 1;

-- Drop the blocking index first so the new index can be added without a window
-- where neither constraint exists.
ALTER TABLE settings
    DROP INDEX idx_settings_key;

ALTER TABLE settings
    ADD COLUMN active_key varchar(100)
        GENERATED ALWAYS AS (IF(`deleted_at` IS NULL, `key`, NULL)) STORED,
    ADD UNIQUE INDEX uniq_settings_active_key (active_key);