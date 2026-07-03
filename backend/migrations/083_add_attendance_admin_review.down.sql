-- Reverse of 083. Drop the index first (it references review_action), then the columns.
ALTER TABLE attendances DROP INDEX idx_attendances_review;
ALTER TABLE attendances
    DROP COLUMN review_action,
    DROP COLUMN review_note,
    DROP COLUMN reviewed_by,
    DROP COLUMN reviewed_at;
