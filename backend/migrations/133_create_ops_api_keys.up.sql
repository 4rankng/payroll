-- Machine API keys for the payroll-ops-mcp agent surface (pk_ops_*).
-- Kept separate from mcp_api_keys (the chat-integration credential): a
-- different namespace, a different scope vocabulary (read_only | read_write),
-- a write-window bound, and no cross-authentication between the surfaces.
CREATE TABLE IF NOT EXISTS ops_api_keys (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL COMMENT 'agent identity (audit attribution)',
    scope VARCHAR(16) NOT NULL COMMENT 'read_only | read_write',
    key_prefix VARCHAR(20) NOT NULL,
    key_hash CHAR(64) NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    last_used_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    write_window_expires_at DATETIME(3) NULL COMMENT 'read_write auto-reverts to read_only after this instant',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_ops_api_keys_key_hash (key_hash),
    KEY idx_ops_api_keys_revoked_at (revoked_at),
    KEY idx_ops_api_keys_write_window (write_window_expires_at),
    CONSTRAINT fk_ops_api_keys_creator FOREIGN KEY (created_by) REFERENCES users(id)
);
