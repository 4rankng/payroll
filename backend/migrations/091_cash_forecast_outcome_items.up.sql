-- Timesheet-level outcome ledger for cash-forecast calibration. A weekly bank
-- file may be retried or split into several disjoint exports; the unique key
-- makes the measured actual the cumulative sum of distinct timesheets.
CREATE TABLE IF NOT EXISTS cash_forecast_outcome_items (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    target_from_date DATE            NOT NULL,
    target_to_date   DATE            NOT NULL,
    timesheet_id     BIGINT UNSIGNED NOT NULL,
    amount           BIGINT          NOT NULL,
    outcome_source   VARCHAR(64)     NOT NULL,
    created_at       DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_cash_forecast_outcome_item
        (target_from_date, target_to_date, timesheet_id),
    KEY idx_cash_forecast_outcome_range
        (target_from_date, target_to_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
