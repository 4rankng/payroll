-- 082 down: remove admin-resolution audit columns from attendance_failed_attempts.
ALTER TABLE attendance_failed_attempts
    DROP COLUMN resolved_reason,
    DROP COLUMN resolved_by,
    DROP COLUMN resolved_at;
