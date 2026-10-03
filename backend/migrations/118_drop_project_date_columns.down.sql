-- Restore the columns dropped by the up migration, schema only.
--
-- The historical date values are gone once the up migration ran; re-adding
-- the columns brings back the shape (nullable date bounds, as in migration
-- 001) with NULL for every row. Data recovery requires the pre-migration
-- backup.

ALTER TABLE projects
    ADD COLUMN start_date date DEFAULT NULL,
    ADD COLUMN end_date date DEFAULT NULL;
