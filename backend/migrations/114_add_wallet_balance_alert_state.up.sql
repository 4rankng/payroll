-- Single-row state for the wallet low-balance admin alert. Records which
-- side of the configured threshold the wallet balance was on at the last
-- evaluation so the "below threshold" notification fires once per downward
-- crossing and re-arms after the balance recovers. The row id is always 1.
CREATE TABLE IF NOT EXISTS wallet_balance_alert_states (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    is_below TINYINT(1) NOT NULL DEFAULT 0,
    last_notified_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
);

INSERT IGNORE INTO wallet_balance_alert_states (id, is_below) VALUES (1, 0);
