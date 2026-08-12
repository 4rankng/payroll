-- +migrate Up
--
-- Persist each self-check-out earning's quota-credit deadline. The delayed
-- worker and recovery sweep use this immutable timestamp so later changes to
-- the Admin hold setting never release earnings that were already waiting.
SET @add_quota_credit_eligible_at = IF(
    EXISTS(
        SELECT 1
          FROM information_schema.columns
         WHERE table_schema = DATABASE()
           AND table_name = 'attendances'
           AND column_name = 'quota_credit_eligible_at'
    ),
    'SELECT 1',
    'ALTER TABLE attendances ADD COLUMN quota_credit_eligible_at DATETIME(3) NULL COMMENT ''immutable self-check-out quota-credit deadline'' AFTER quota_credited_at'
);
PREPARE add_quota_credit_eligible_at_stmt FROM @add_quota_credit_eligible_at;
EXECUTE add_quota_credit_eligible_at_stmt;
DEALLOCATE PREPARE add_quota_credit_eligible_at_stmt;

-- Existing pending records were created under the former fixed 24-hour hold.
-- Preserve that contract while making them eligible for the new recovery query.
UPDATE attendances
   SET quota_credit_eligible_at = DATE_ADD(check_out_time, INTERVAL 24 HOUR)
 WHERE quota_credited_at IS NULL
   AND earning_amount > 0
   AND check_out_time IS NOT NULL
   AND quota_credit_eligible_at IS NULL;

SET @add_quota_credit_eligible_index = IF(
    EXISTS(
        SELECT 1
          FROM information_schema.statistics
         WHERE table_schema = DATABASE()
           AND table_name = 'attendances'
           AND index_name = 'idx_attendances_quota_credit_eligible'
    ),
    'SELECT 1',
    'CREATE INDEX idx_attendances_quota_credit_eligible ON attendances (quota_credited_at, quota_credit_eligible_at)'
);
PREPARE add_quota_credit_eligible_index_stmt FROM @add_quota_credit_eligible_index;
EXECUTE add_quota_credit_eligible_index_stmt;
DEALLOCATE PREPARE add_quota_credit_eligible_index_stmt;
