-- 083: Admin review (accept/reject) audit trail for attendance disputes.
-- When an admin manually reviews a disputed/flexi attendance via
-- POST /admin/attendances/:id/approve|reject, the record is stamped with
-- who/when/why plus the review action. Mirrors the resolution-columns
-- convention established in 082 (attendance_failed_attempts.resolved_*).
-- All columns nullable; no backfill needed — a NULL review_action means
-- "never reviewed" and the row keeps its system-derived status.

ALTER TABLE attendances
    ADD COLUMN review_action VARCHAR(16)     NULL COMMENT 'approved | rejected (manual admin review)',
    ADD COLUMN review_note   TEXT            NULL COMMENT 'admin note / reject reason for the review',
    ADD COLUMN reviewed_by   BIGINT UNSIGNED NULL COMMENT 'admin user id',
    ADD COLUMN reviewed_at   DATETIME(3)     NULL,
    ADD INDEX idx_attendances_review (review_action);
