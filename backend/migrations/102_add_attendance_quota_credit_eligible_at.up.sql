-- +migrate Up
--
-- Persist each self-check-out earning's quota-credit deadline. The delayed
-- worker and recovery sweep use this immutable timestamp so later changes to
-- the Admin hold setting never release earnings that were already waiting.
ALTER TABLE attendances
    ADD COLUMN IF NOT EXISTS quota_credit_eligible_at DATETIME(3) NULL COMMENT 'immutable self-check-out quota-credit deadline' AFTER quota_credited_at;

-- Existing pending records were created under the former fixed 24-hour hold.
-- Preserve that contract while making them eligible for the new recovery query.
UPDATE attendances
   SET quota_credit_eligible_at = DATE_ADD(check_out_time, INTERVAL 24 HOUR)
 WHERE quota_credited_at IS NULL
   AND earning_amount > 0
   AND check_out_time IS NOT NULL
   AND quota_credit_eligible_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_attendances_quota_credit_eligible
    ON attendances (quota_credited_at, quota_credit_eligible_at);
