-- Machine API keys for the MCP agent surface. Kept separate from `api_keys`
-- (the vendor integration keys): a different credential type, a different
-- scope vocabulary (read_only | full), and no cross-authentication between the
-- two surfaces.
CREATE TABLE IF NOT EXISTS mcp_api_keys (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    scope VARCHAR(16) NOT NULL COMMENT 'read_only | full',
    key_prefix VARCHAR(20) NOT NULL,
    key_hash CHAR(64) NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    last_used_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_mcp_api_keys_key_hash (key_hash),
    KEY idx_mcp_api_keys_revoked_at (revoked_at),
    CONSTRAINT fk_mcp_api_keys_creator FOREIGN KEY (created_by) REFERENCES users(id)
);
