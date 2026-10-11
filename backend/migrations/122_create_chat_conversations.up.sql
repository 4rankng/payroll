-- Chat assistant (MiMo) — conversation headers. One row per chat session.
-- Ownership is enforced at the service layer (user_id = owner); Casbin is
-- role×path only and cannot express per-record ownership.
CREATE TABLE IF NOT EXISTS chat_conversations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Owner — the only non-admin allowed to read this conversation',
    role VARCHAR(32) NOT NULL COMMENT 'JWT role at creation time: admin | partner',
    title VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'First user message, truncated',
    message_count INT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_chat_conversations_user (user_id, updated_at),
    CONSTRAINT fk_chat_conversations_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
