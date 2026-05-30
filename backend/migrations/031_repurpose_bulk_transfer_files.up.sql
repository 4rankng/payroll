-- Repurpose bulk_transfer_files for result upload history

-- Make cycle nullable for result upload records (result uploads don't have a cycle)
ALTER TABLE bulk_transfer_files MODIFY COLUMN cycle ENUM('weekly','monthly','flexible') NULL;

-- Rename transfer_result to asset_id
ALTER TABLE bulk_transfer_files CHANGE COLUMN transfer_result asset_id bigint unsigned;

-- Drop unused columns from bulk_transfer_files
ALTER TABLE bulk_transfer_files DROP COLUMN checksum;
ALTER TABLE bulk_transfer_files DROP COLUMN deleted_at;
ALTER TABLE bulk_transfer_files DROP COLUMN transfer_at;

-- Drop unused columns from assets table
ALTER TABLE assets DROP COLUMN metadata;
ALTER TABLE assets DROP COLUMN deleted_at;

-- Backfill transaction counts from data JSON column for records with asset_id but zero counts
-- The data column contains a JSON array of transaction objects with transfer_status field
UPDATE bulk_transfer_files
SET
    transactions_count = JSON_LENGTH(data),
    completed_count = (
        SELECT COUNT(*)
        FROM JSON_TABLE(
            data,
            '$[*]' COLUMNS (transfer_status VARCHAR(50) PATH '$.transfer_status')
        ) AS jt
        WHERE jt.transfer_status = 'completed'
    ),
    failed_count = (
        SELECT COUNT(*)
        FROM JSON_TABLE(
            data,
            '$[*]' COLUMNS (transfer_status VARCHAR(50) PATH '$.transfer_status')
        ) AS jt
        WHERE jt.transfer_status = 'failed'
    ),
    transfer_amount = (
        SELECT COALESCE(SUM(CAST(jt.amount AS SIGNED)), 0)
        FROM JSON_TABLE(
            data,
            '$[*]' COLUMNS (
                transfer_status VARCHAR(50) PATH '$.transfer_status',
                amount VARCHAR(50) PATH '$.amount'
            )
        ) AS jt
        WHERE jt.transfer_status = 'completed'
    )
WHERE asset_id IS NOT NULL
  AND transactions_count = 0
  AND data IS NOT NULL
  AND JSON_LENGTH(data) > 0;

-- Set file_id to NULL in transaction_codes data where it is 0
-- file_id should only be populated when a bulk transfer result is uploaded
UPDATE transaction_codes
SET data = JSON_REMOVE(data, '$.weekly_pay.file_id')
WHERE JSON_EXTRACT(data, '$.weekly_pay.file_id') = 0;

UPDATE transaction_codes
SET data = JSON_REMOVE(data, '$.monthly_pay.file_id')
WHERE JSON_EXTRACT(data, '$.monthly_pay.file_id') = 0;
