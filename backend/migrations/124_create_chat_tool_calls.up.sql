-- Chat assistant (MiMo) — audit trail of every tool call the assistant
-- executor ran (who / what / when / outcome). This is the structured half of
-- the audit contract; the audit_logs row (same action, via the existing
-- audit:log:write pipeline) carries the actor-facing summary.
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
