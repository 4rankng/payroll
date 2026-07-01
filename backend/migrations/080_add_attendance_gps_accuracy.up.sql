-- 080: Capture GPS accuracy + device fix timestamp for check-in/out attempts.
-- Both success (attendances) and failure (attendance_failed_attempts) paths gain
-- accuracy and device-fix-time columns so admins can forensically judge whether an
-- out-of-bounds reading was a true location or a poor-signal fix. All columns are
-- nullable for backfill safety and because device-level GPS failures have no fix.

ALTER TABLE attendances
  ADD COLUMN check_in_accuracy  FLOAT       NULL COMMENT 'GPS accuracy in meters (device-reported HDoP)',
  ADD COLUMN check_in_gps_at    DATETIME(3) NULL COMMENT 'device GPS fix timestamp (epoch → UTC instant)',
  ADD COLUMN check_out_accuracy FLOAT       NULL,
  ADD COLUMN check_out_gps_at   DATETIME(3) NULL;

ALTER TABLE attendance_failed_attempts
  ADD COLUMN accuracy FLOAT       NULL COMMENT 'GPS accuracy meters; null when device had no fix',
  ADD COLUMN gps_at   DATETIME(3) NULL COMMENT 'device GPS fix timestamp; null when no fix';
