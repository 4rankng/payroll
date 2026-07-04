-- Deduplication / audit table for FlexPay reconciliation settlement uploads.
-- A row is written only after a settlement run completes with no per-iteration
-- failures, keyed by a hash of the settled request-ID set so a re-upload of the
-- same recon content is a no-op. Partial-failure runs are NOT recorded, so a
-- re-upload reprocesses only the records whose transaction write rolled back
-- (already-settled transactions are skipped by the status precondition).
CREATE TABLE IF NOT EXISTS settlement_uploads (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    file_hash        CHAR(64)        NOT NULL,
    uploaded_at      DATETIME(3)     NOT NULL,
    request_ids_json JSON            NULL,
    settled_count    BIGINT          NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uq_settlement_uploads_file_hash (file_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
