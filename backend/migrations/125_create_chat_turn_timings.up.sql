-- Chat assistant (MiMo) — per-turn latency instrumentation. The fast contract
-- (p95 ≤2s direct / ≤5s tool turns) is measured from these rows in production,
-- not from a spike: one row per message turn including failed/timed-out turns.
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
