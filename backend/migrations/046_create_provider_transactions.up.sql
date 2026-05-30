-- +migrate Up
--
-- provider_transactions: persistent record of every money-sending interaction
-- with a disbursement provider (currently 9pay). Drives a deterministic
-- finite-state-machine lifecycle (pending → completed | failed | reversed)
-- with optimistic-locking concurrency control via the version column.
--
-- See:
--   internal/domain/transactions/provider_transaction.go
--   internal/domain/transactions/state_machine.go
--   internal/infra/persistence/provider_transaction_repository.go
CREATE TABLE IF NOT EXISTS provider_transactions (
    id                    BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    txn_id                CHAR(36)         NOT NULL,
    request_id            VARCHAR(64)      NOT NULL,
    payment_no            VARCHAR(128)     DEFAULT NULL,

    requested_amount      BIGINT           NOT NULL,
    charged_amount        BIGINT           DEFAULT NULL,
    fee                   BIGINT           DEFAULT NULL,

    recipient_name        VARCHAR(255)     NOT NULL,
    recipient_account_no  VARCHAR(64)      NOT NULL,
    recipient_bank        VARCHAR(64)      NOT NULL,

    metadata              JSON             DEFAULT NULL,

    status                VARCHAR(32)      NOT NULL DEFAULT 'pending',
    error_code            VARCHAR(32)      DEFAULT NULL,
    error_message         TEXT             DEFAULT NULL,

    entity_id             BIGINT UNSIGNED  DEFAULT NULL,

    version               BIGINT           NOT NULL DEFAULT 0,

    created_at            DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    settled_at            DATETIME(3)      DEFAULT NULL,

    PRIMARY KEY (id),
    UNIQUE INDEX idx_pt_txn_id (txn_id),
    UNIQUE INDEX idx_pt_request_id (request_id),
    INDEX idx_pt_status_created (status, created_at),
    INDEX idx_pt_error_code_created (error_code, created_at),
    INDEX idx_pt_payment_no (payment_no),
    INDEX idx_pt_entity_id (entity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
