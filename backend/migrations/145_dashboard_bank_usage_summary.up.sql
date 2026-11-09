-- Precomputed bank-usage summary for the dashboard "Ngân hàng nhận lương"
-- cards (overall + per project).
--
-- Why: the live GROUP BY scans over paid timesheets take 90+ CPU-seconds on
-- the 1-vCPU production box — the per-employee LEFT JOIN walks the whole paid
-- prefix of idx_timesheets_paid_date_cover once per employee (~36k entries ×
-- ~1.6k employees ≈ 58M row checks), pushing every dashboard load past the
-- 10s RequestTimeout and spiking the box to 100% CPU.
--
-- project_id = 0 is the overall breakdown; >0 is a per-project card. Rows are
-- replaced as one snapshot by the dashboard:bank_usage_refresh asynq task
-- (@every 2m) and inline on a stale read. computed_at_millis is unix millis —
-- integer compares keep the staleness check hermetic under sqlite tests and
-- free of DATETIME scan quirks.

CREATE TABLE IF NOT EXISTS dashboard_bank_usage_summaries (
  project_id         BIGINT UNSIGNED NOT NULL DEFAULT 0,
  bank_name          VARCHAR(191)    NOT NULL,
  employee_count     INT UNSIGNED    NOT NULL DEFAULT 0,
  total_paid         DECIMAL(20,2)   NOT NULL DEFAULT 0,
  transfer_count     INT UNSIGNED    NOT NULL DEFAULT 0,
  computed_at_millis BIGINT          NOT NULL DEFAULT 0,
  PRIMARY KEY (project_id, bank_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
