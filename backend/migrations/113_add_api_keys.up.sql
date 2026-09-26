-- Machine credentials for the external chatbot integration API. A key is
-- minted once by an admin (Settings → API) and shown in plaintext exactly
-- once; only the SHA-256 hash is persisted. key_prefix keeps the first 12
-- characters so the admin list can identify a key without holding the secret.
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    key_hash CHAR(64) NOT NULL COMMENT 'sha256 hex of the plaintext key',
    created_by BIGINT UNSIGNED NOT NULL,
    last_used_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_api_keys_key_hash (key_hash),
    KEY idx_api_keys_revoked_at (revoked_at),
    CONSTRAINT fk_api_keys_creator FOREIGN KEY (created_by) REFERENCES users(id)
);
