-- Durable, company-wide point-in-time measurements for the advisory cash
-- readiness forecast. The unique key makes repeated reads on the same cycle
-- day/model an update, while retaining one row per decision horizon.
CREATE TABLE IF NOT EXISTS cash_forecast_snapshots (
    id                         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    scope_key                  VARCHAR(64)     NOT NULL,
    cycle_key                  VARCHAR(32)     NOT NULL,
    cycle_day                  SMALLINT        NOT NULL,
    model_version              VARCHAR(64)     NOT NULL,
    target_from_date           DATE            NOT NULL,
    target_to_date             DATE            NOT NULL,
    horizon_days               SMALLINT        NOT NULL,
    observed_approved_amount   BIGINT          NOT NULL DEFAULT 0,
    pending_amount             BIGINT          NOT NULL DEFAULT 0,
    expected_pending_amount    BIGINT          NOT NULL DEFAULT 0,
    expected_future_amount     BIGINT          NOT NULL DEFAULT 0,
    expected_payout_amount     BIGINT          NOT NULL DEFAULT 0,
    recommended_reserve_amount BIGINT          NOT NULL DEFAULT 0,
    interval_lower_amount      BIGINT          NOT NULL DEFAULT 0,
    interval_upper_amount      BIGINT          NOT NULL DEFAULT 0,
    generated_at               DATETIME(3)     NOT NULL,
    actual_payout_amount       BIGINT          NULL,
    outcome_source             VARCHAR(64)     NULL,
    resolved_at                DATETIME(3)     NULL,
    created_at                 DATETIME(3)     NOT NULL,
    updated_at                 DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_cash_forecast_snapshot_identity
        (scope_key, cycle_key, cycle_day, model_version),
    KEY idx_cash_forecast_resolved_horizon
        (scope_key, model_version, horizon_days, resolved_at),
    KEY idx_cash_forecast_target_range
        (scope_key, target_from_date, target_to_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
