-- 1. Add reconciled_at column to wallet_payments
ALTER TABLE wallet_payments ADD COLUMN reconciled_at DATETIME(3) NULL AFTER settled_at;
ALTER TABLE wallet_payments ADD INDEX idx_wp_reconciled (reconciled_at);

-- 2. wallet_ipn: audit trail for all inbound IPN messages from 9pay
CREATE TABLE wallet_ipn (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    provider          VARCHAR(32) NOT NULL DEFAULT '9pay',
    invoice_no        VARCHAR(128) NOT NULL DEFAULT '',
    request_id        VARCHAR(64) NOT NULL DEFAULT '',
    status            VARCHAR(32) NOT NULL,
    amount            BIGINT NOT NULL DEFAULT 0,
    raw_error_code    VARCHAR(32) NOT NULL DEFAULT '',
    failure_reason    TEXT NULL,
    raw_payload       JSON NULL,
    processing_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    processing_error  TEXT NULL,
    wallet_payment_id BIGINT UNSIGNED NULL,
    created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    processed_at      DATETIME(3) NULL,
    PRIMARY KEY (id),
    INDEX idx_wi_invoice_no (invoice_no),
    INDEX idx_wi_request_id (request_id),
    INDEX idx_wi_status_created (processing_status, created_at),
    INDEX idx_wi_wallet_payment_id (wallet_payment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
