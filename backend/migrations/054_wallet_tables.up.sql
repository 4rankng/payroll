DROP TABLE provider_transactions;

CREATE TABLE wallet_payments (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    txn_id                CHAR(36) NOT NULL,
    request_id            VARCHAR(64) NOT NULL,
    invoice_no            VARCHAR(128) NOT NULL DEFAULT '',
    requested_amount      BIGINT NOT NULL,
    fee                   BIGINT NOT NULL DEFAULT 0,
    recipient_name        VARCHAR(255) NOT NULL DEFAULT '',
    recipient_account_no  VARCHAR(64) NOT NULL DEFAULT '',
    recipient_bank        VARCHAR(64) NOT NULL DEFAULT '',
    description           TEXT NULL,
    status                VARCHAR(32) NOT NULL DEFAULT 'pending',
    error_code            VARCHAR(32) NULL,
    error_message         TEXT NULL,
    entity_id             BIGINT UNSIGNED NULL,
    created_by            BIGINT UNSIGNED NULL,
    version               BIGINT NOT NULL DEFAULT 0,
    created_at            DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    settled_at            DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX idx_wp_txn_id (txn_id),
    UNIQUE INDEX idx_wp_request_id (request_id),
    INDEX idx_wp_invoice_no (invoice_no),
    INDEX idx_wp_status_created (status, created_at),
    INDEX idx_wp_entity_id (entity_id),
    INDEX idx_wp_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE wallet_topups (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    amount          BIGINT NOT NULL,
    bank_ref        VARCHAR(128) NOT NULL,
    occurred_at     DATETIME(3) NOT NULL,
    note            TEXT NULL,
    created_by      BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    version         BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE INDEX idx_wt_bank_ref (bank_ref),
    INDEX idx_wt_occurred_at (occurred_at),
    INDEX idx_wt_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
