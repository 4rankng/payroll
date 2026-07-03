-- 082: Audit admin resolution of failed check-in attempts.
-- When an admin overrides a device-GPS failure (reason_category gps_*) and
-- records the check-in manually, the source attendance_failed_attempts row is
-- stamped with who/when/why. This keeps a full audit trail and makes the
-- override idempotent (IsResolved). All columns nullable; no backfill needed.
ALTER TABLE attendance_failed_attempts
    ADD COLUMN resolved_at     DATETIME(3)     NULL,
    ADD COLUMN resolved_by     BIGINT UNSIGNED NULL,
    ADD COLUMN resolved_reason VARCHAR(255)    NULL;
