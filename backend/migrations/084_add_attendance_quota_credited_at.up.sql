-- 084: 24h holding period on attendance earnings before they enter the advance-
-- payment quota pool. quota_credited_at marks the moment an attendance's earning
-- was banked into advance_payments.salary/max_adv_amount.
--   NULL  = earning not yet credited (pending the 24h hold, or pending a lost task)
--   <ts>  = earning already credited
-- The deferred asynq credit task + the safety-net sweep both gate on
-- quota_credited_at IS NULL, so the column is the idempotency key for crediting.

ALTER TABLE attendances
    ADD COLUMN quota_credited_at DATETIME(3) NULL COMMENT 'when this attendance earning was banked into the advance-payment quota pool; NULL while held/pending' AFTER salary_reject_reason;

-- Backfill: every attendance that already has a checkout was credited
-- synchronously by the pre-084 CheckOut path (AccumulateSalary ran in the same
-- tx). Stamp them as already-credited so the new deferred-credit path and the
-- safety-net sweep never re-bank those earnings. earning_amount=0 / NULL rows
-- have nothing to credit, so the backfill only touches rows with earning>0;
-- either way a non-null value disables the new path for historical rows.
UPDATE attendances
   SET quota_credited_at = IFNULL(check_out_time, NOW())
 WHERE check_out_time IS NOT NULL;

CREATE INDEX idx_attendances_quota_credited
    ON attendances (quota_credited_at, check_out_time);
