-- 079: Capture failed check-in/check-out attempts for admin health monitoring.
-- ValidationError (HTTP 400) responses are persisted so the admin dashboard can
-- surface aggregate failure counts by reason category (geofence, permissions,
-- shift-window, etc.). Rows are bounded by the recent-window queries; a future
-- periodic cleanup job should prune rows older than 90 days.

CREATE TABLE IF NOT EXISTS attendance_failed_attempts (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    employee_id     BIGINT UNSIGNED    NOT NULL,
    attempt_type    VARCHAR(16)        NOT NULL COMMENT 'check_in or check_out',
    reason_category VARCHAR(48)        NOT NULL COMMENT 'classified error code',
    project_id      BIGINT UNSIGNED    NOT NULL DEFAULT 0,
    lat             DECIMAL(10,7),
    lng             DECIMAL(10,7),
    error_message   VARCHAR(500),
    created_at      DATETIME(3)        NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_failed_attempts_created (created_at),
    INDEX idx_failed_attempts_type_category (attempt_type, reason_category, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
