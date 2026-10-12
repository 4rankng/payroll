-- Recreates the four chat tables exactly as migrations 122-125 created them.
-- The dropped rows are not recoverable: the tables are recreated empty.
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
CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Author (user turns) or conversation owner (assistant turns)',
    role ENUM('user', 'assistant') NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_chat_messages_conversation (conversation_id, id),
    CONSTRAINT fk_chat_messages_conversation
        FOREIGN KEY (conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS chat_tool_calls (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT UNSIGNED NOT NULL,
    message_id BIGINT UNSIGNED NULL COMMENT 'Assistant message the call fed; NULL when the turn aborted before persisting one',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Caller identity — tool scope is enforced against this user at call time',
    tool_name VARCHAR(64) NOT NULL,
    arguments JSON NOT NULL,
    outcome ENUM('ok', 'error', 'unlisted', 'forbidden', 'not_wired') NOT NULL COMMENT 'ok=executed, unlisted=not in manifest, forbidden=not in role manifest, not_wired=port not connected yet, error=failed',
    result_summary VARCHAR(500) NOT NULL DEFAULT '' COMMENT 'Bounded echo of the tool result for audit readers',
    duration_ms INT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_chat_tool_calls_conversation (conversation_id, id),
    KEY idx_chat_tool_calls_user (user_id, created_at),
    CONSTRAINT fk_chat_tool_calls_conversation
        FOREIGN KEY (conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS chat_turn_timings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    role VARCHAR(32) NOT NULL,
    model VARCHAR(64) NOT NULL DEFAULT '',
    round_trips INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Model chat.completions calls in the turn',
    tool_calls INT UNSIGNED NOT NULL DEFAULT 0,
    timed_out TINYINT(1) NOT NULL DEFAULT 0,
    duration_ms INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'End-to-end turn latency: model + tools',
    model_ms INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Sum of model call latencies',
    tool_ms INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Wall time spent in tool execution',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_chat_turn_timings_created (created_at),
    KEY idx_chat_turn_timings_conversation (conversation_id, id)
);
