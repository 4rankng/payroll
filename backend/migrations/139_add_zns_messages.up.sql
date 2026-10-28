-- 139: zns_messages — THE single audit table for every ZNS message the system
-- sends. Rows are written at the provider layer (infra/zalo.Provider) right
-- after each send API call, so every feature (salary, reset, self check-in,
-- ...) is recorded automatically. Admin ZNS tracking reads exclusively from
-- this table; flexpay_salary_notifications keeps its internal retry-queue role
-- only (claim/lease) and is NOT an admin data source. Do NOT add per-flow ZNS
-- tracking tables in the future — tag sends with their `feature` instead.
CREATE TABLE zns_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    feature VARCHAR(32) NOT NULL DEFAULT '',
    template_id VARCHAR(64) NOT NULL DEFAULT '',
    phone VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT '',
    provider_msg_id VARCHAR(128) NOT NULL DEFAULT '',
    provider_code INT NOT NULL DEFAULT 0,
    error_msg VARCHAR(512) NOT NULL DEFAULT '',
    duration_ms INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_zns_messages_created_at (created_at),
    KEY idx_zns_messages_feature (feature),
    KEY idx_zns_messages_status (status),
    KEY idx_zns_messages_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Backfill: copy the delivery history from the flexpay salary queue (the
-- pre-existing ZNS flow) so admin tracking is complete from day one. One row
-- per flexpay notification, created_at = send time (sent_at) when sent,
-- otherwise enqueue time. The guard makes the copy idempotent: it only runs
-- against an empty zns_messages table (the INSERT..SELECT is atomic, so a
-- re-run can never double-copy).
INSERT INTO zns_messages (created_at, feature, template_id, phone, status, provider_msg_id, provider_code, error_msg, duration_ms)
SELECT COALESCE(f.sent_at, f.created_at),
       'salary_notification',
       f.template_id,
       f.phone,
       f.status,
       f.provider_msg_id,
       f.provider_code,
       f.last_error,
       0
FROM flexpay_salary_notifications AS f
WHERE NOT EXISTS (SELECT 1 FROM zns_messages LIMIT 1);
