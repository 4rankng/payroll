-- Backfill the missing VFIC receivable ledger for the single completed
-- wallet_bulk batch uploaded on 2026-07-19.
--
-- Safety properties:
--   * targets one batch ID and verifies the exact filename
--   * reads the active fee schedule used by the application (2% fallback)
--   * derives the transfer total from successful wallet_payments only
--   * requires every successful VFIC code to resolve to timesheets
--   * refuses to overwrite an existing timesheet transaction link
--   * links the transaction and initial ledger pair to the uploaded asset
--   * validates the complete accounting shape before COMMIT
--   * reruns are a verified no-op
--
-- After a successful production run, rebuild running balances through the
-- authenticated POST /api/v1/ledger/balance/recalculate endpoint.

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

DROP PROCEDURE IF EXISTS backfill_wallet_bulk_receivable_proc;
DELIMITER $$

CREATE PROCEDURE backfill_wallet_bulk_receivable_proc()
main: BEGIN
    DECLARE v_target_batch_id BIGINT UNSIGNED DEFAULT 1;
    DECLARE v_expected_filename VARCHAR(255)
        DEFAULT 'Yeu_cau_chuyen_tien_weekly_20260719_140904.xlsx';

    DECLARE v_filename VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    DECLARE v_source VARCHAR(16);
    DECLARE v_status VARCHAR(32);
    DECLARE v_success_count INT;
    DECLARE v_created_by BIGINT UNSIGNED;
    DECLARE v_asset_id BIGINT UNSIGNED;
    DECLARE v_booked_at DATETIME(3);
    DECLARE v_partner_company VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    DECLARE v_fee_percentage DECIMAL(12,6);
    DECLARE v_transfer BIGINT;
    DECLARE v_receivable BIGINT;
    DECLARE v_desc TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    DECLARE v_new_txn_id BIGINT UNSIGNED;
    DECLARE v_existing_txn_id BIGINT UNSIGNED;
    DECLARE v_existing_txn_count INT DEFAULT 0;
    DECLARE v_success_payment_count INT DEFAULT 0;
    DECLARE v_mapped_payment_count INT DEFAULT 0;
    DECLARE v_target_timesheet_count INT DEFAULT 0;
    DECLARE v_existing_timesheet_count INT DEFAULT 0;
    DECLARE v_linked_timesheet_count INT DEFAULT 0;
    DECLARE v_paid_timesheet_count INT DEFAULT 0;
    DECLARE v_ledger_entry_count INT DEFAULT 0;
    DECLARE v_ledger_shape_count INT DEFAULT 0;
    DECLARE v_total_debit BIGINT DEFAULT 0;
    DECLARE v_total_credit BIGINT DEFAULT 0;
    DECLARE v_error_message TEXT;

    DECLARE EXIT HANDLER FOR SQLEXCEPTION
    BEGIN
        ROLLBACK;
        DROP TEMPORARY TABLE IF EXISTS backfill_target_timesheets;
        RESIGNAL;
    END;

    START TRANSACTION;

    SELECT b.filename,
           b.source,
           b.status,
           b.success_count,
           b.created_by,
           b.asset_id,
           COALESCE(b.completed_at, b.created_at)
      INTO v_filename,
           v_source,
           v_status,
           v_success_count,
           v_created_by,
           v_asset_id,
           v_booked_at
      FROM bulk_transfer_batches b
     WHERE b.id = v_target_batch_id
     FOR UPDATE;

    IF v_filename <> v_expected_filename THEN
        SET v_error_message = CONCAT('Batch #', v_target_batch_id,
            ' filename mismatch: ', v_filename);
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = v_error_message;
    END IF;
    IF v_source <> 'wallet_upload' OR v_status <> 'completed' OR v_success_count <= 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Target batch is not a completed wallet upload with successful transfers';
    END IF;
    IF v_asset_id IS NULL THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Target batch has no uploaded asset';
    END IF;

    -- Match SettingsConfigService behavior: partner_company defaults to
    -- "VFIC Manpower" and the retired flat fee key is ignored. The headline
    -- fee is the first tier of the latest effective schedule; if no schedule
    -- resolves, the application falls back to 2%.
    SELECT COALESCE(NULLIF(TRIM(MAX(s.value)), ''), 'VFIC Manpower')
      INTO v_partner_company
      FROM settings s
     WHERE s.`key` = 'partner_company'
       AND s.deleted_at IS NULL;

    SELECT COALESCE(
               CAST(JSON_UNQUOTE(JSON_EXTRACT(schedule.tiers, '$[0].percentage')) AS DECIMAL(12,6)) / 100,
               0.02
           )
      INTO v_fee_percentage
      FROM (
          SELECT jt.tiers
            FROM settings s
            JOIN JSON_TABLE(
                s.value,
                '$[*]' COLUMNS (
                    effective_date DATE PATH '$.effective_date',
                    tiers JSON PATH '$.tiers'
                )
            ) jt
           WHERE s.`key` = 'advance_payment_fee_schedules'
             AND s.deleted_at IS NULL
             AND jt.effective_date <= DATE(v_booked_at)
           ORDER BY jt.effective_date DESC
           LIMIT 1
      ) schedule;

    SET v_fee_percentage = COALESCE(v_fee_percentage, 0.02);
    IF v_fee_percentage < 0 OR v_fee_percentage > 1 THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'advance_cash_fee_percentage setting is invalid';
    END IF;

    SELECT COUNT(*), COALESCE(SUM(wp.requested_amount), 0)
      INTO v_success_payment_count, v_transfer
      FROM wallet_payments wp
     WHERE wp.bulk_transfer_batch_id = v_target_batch_id
       AND wp.status = 'completed';

    IF v_success_payment_count <> v_success_count OR v_transfer <= 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Successful wallet payment count/amount does not match the batch';
    END IF;

    DROP TEMPORARY TABLE IF EXISTS backfill_target_timesheets;
    CREATE TEMPORARY TABLE backfill_target_timesheets (
        timesheet_id BIGINT UNSIGNED NOT NULL PRIMARY KEY
    );

    INSERT IGNORE INTO backfill_target_timesheets (timesheet_id)
    SELECT jt.timesheet_id
      FROM wallet_payments wp
      JOIN transaction_codes tc
        ON tc.code COLLATE utf8mb4_unicode_ci = wp.request_id
      JOIN JSON_TABLE(
            COALESCE(
                JSON_EXTRACT(tc.data, '$.weekly_pay.timesheet_ids'),
                JSON_EXTRACT(tc.data, '$.monthly_pay.timesheet_ids')
            ),
            '$[*]' COLUMNS (timesheet_id BIGINT UNSIGNED PATH '$')
      ) jt
     WHERE wp.bulk_transfer_batch_id = v_target_batch_id
       AND wp.status = 'completed';

    SELECT COUNT(DISTINCT wp.id)
      INTO v_mapped_payment_count
      FROM wallet_payments wp
      JOIN transaction_codes tc
        ON tc.code COLLATE utf8mb4_unicode_ci = wp.request_id
      JOIN JSON_TABLE(
            COALESCE(
                JSON_EXTRACT(tc.data, '$.weekly_pay.timesheet_ids'),
                JSON_EXTRACT(tc.data, '$.monthly_pay.timesheet_ids')
            ),
            '$[*]' COLUMNS (timesheet_id BIGINT UNSIGNED PATH '$')
      ) jt
     WHERE wp.bulk_transfer_batch_id = v_target_batch_id
       AND wp.status = 'completed';

    SELECT COUNT(*) INTO v_target_timesheet_count
      FROM backfill_target_timesheets;

    IF v_mapped_payment_count <> v_success_payment_count OR v_target_timesheet_count = 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'One or more successful VFIC payments have no timesheet mapping';
    END IF;

    SELECT COUNT(*),
           COALESCE(SUM(t.payment_status = 'paid'), 0)
      INTO v_existing_timesheet_count, v_paid_timesheet_count
      FROM timesheets t
      JOIN backfill_target_timesheets target ON target.timesheet_id = t.id
     FOR UPDATE;

    IF v_existing_timesheet_count <> v_target_timesheet_count THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'One or more mapped timesheets do not exist';
    END IF;
    IF v_paid_timesheet_count <> v_target_timesheet_count THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'One or more mapped timesheets are not paid';
    END IF;

    SET v_receivable = ROUND(v_transfer * (1 + v_fee_percentage));
    SET v_desc = CONCAT('Trả lương cho ', v_partner_company, ', file: ', v_filename);

    SELECT COUNT(*), MIN(t.id)
      INTO v_existing_txn_count, v_existing_txn_id
      FROM transactions t
     WHERE t.deleted_at IS NULL
       AND (
            t.description COLLATE utf8mb4_unicode_ci = v_desc
            OR (t.asset_id = v_asset_id
                AND t.transaction_type COLLATE utf8mb4_unicode_ci = 'revenue')
       );

    IF v_existing_txn_count > 1 THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Multiple candidate receivable transactions already exist';
    END IF;

    IF v_existing_txn_count = 1 THEN
        SELECT COUNT(*) INTO v_existing_txn_count
          FROM transactions t
         WHERE t.id = v_existing_txn_id
           AND t.transaction_type COLLATE utf8mb4_unicode_ci = 'revenue'
           AND t.amount = v_receivable
           AND t.party COLLATE utf8mb4_unicode_ci = v_partner_company
           AND t.status COLLATE utf8mb4_unicode_ci = 'pending'
           AND t.asset_id = v_asset_id
           AND t.created_by = v_created_by
           AND t.description COLLATE utf8mb4_unicode_ci = v_desc
           AND t.deleted_at IS NULL;

        SELECT COUNT(*),
               COALESCE(SUM(le.debit), 0),
               COALESCE(SUM(le.credit), 0),
               COALESCE(SUM(
                   (le.account COLLATE utf8mb4_unicode_ci = 'receivable'
                    AND le.party COLLATE utf8mb4_unicode_ci = v_partner_company
                    AND le.debit = v_receivable AND le.credit = 0 AND le.asset_id = v_asset_id)
                 + (le.account COLLATE utf8mb4_unicode_ci = 'revenue'
                    AND le.party COLLATE utf8mb4_unicode_ci = v_partner_company
                    AND le.debit = 0 AND le.credit = v_receivable AND le.asset_id = v_asset_id)
                 + (le.account COLLATE utf8mb4_unicode_ci = 'cash'
                    AND le.party COLLATE utf8mb4_unicode_ci = 'Nhân viên'
                    AND le.debit = 0 AND le.credit = v_transfer AND le.asset_id IS NULL)
                 + (le.account COLLATE utf8mb4_unicode_ci = 'revenue'
                    AND le.party COLLATE utf8mb4_unicode_ci = v_partner_company
                    AND le.debit = v_transfer AND le.credit = 0 AND le.asset_id IS NULL)
               ), 0)
          INTO v_ledger_entry_count, v_total_debit, v_total_credit, v_ledger_shape_count
          FROM ledger_entries le
         WHERE le.transaction_id = v_existing_txn_id
           AND le.deleted_at IS NULL;

        SELECT COUNT(*) INTO v_linked_timesheet_count
          FROM timesheets t
          JOIN backfill_target_timesheets target ON target.timesheet_id = t.id
         WHERE t.transaction_id = v_existing_txn_id;

        IF v_existing_txn_count <> 1
           OR v_ledger_entry_count <> 4
           OR v_ledger_shape_count <> 4
           OR v_total_debit <> v_receivable + v_transfer
           OR v_total_credit <> v_receivable + v_transfer
           OR v_linked_timesheet_count <> v_target_timesheet_count THEN
            SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT = 'Existing receivable transaction is incomplete or inconsistent';
        END IF;

        COMMIT;
        DROP TEMPORARY TABLE IF EXISTS backfill_target_timesheets;
        SELECT CONCAT('Batch #', v_target_batch_id, ' SKIP - verified transaction #',
                      v_existing_txn_id, ' is already complete') AS result;
        LEAVE main;
    END IF;

    SELECT COUNT(*) INTO v_linked_timesheet_count
      FROM timesheets t
      JOIN backfill_target_timesheets target ON target.timesheet_id = t.id
     WHERE t.transaction_id IS NOT NULL;

    IF v_linked_timesheet_count <> 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Refusing to overwrite an existing timesheet transaction link';
    END IF;

    INSERT INTO transactions
        (description, transaction_type, amount, party, status, asset_id,
         created_by, created_at, updated_at, settled_amount, transaction_code)
    VALUES
        (v_desc, 'revenue', v_receivable, v_partner_company, 'pending', v_asset_id,
         v_created_by, v_booked_at, v_booked_at, 0, UUID());
    SET v_new_txn_id = LAST_INSERT_ID();

    INSERT INTO ledger_entries
        (date, account, party, debit, credit, balance, asset_id,
         transaction_id, created_by, created_at, updated_at)
    VALUES
        (DATE(v_booked_at), 'receivable', v_partner_company, v_receivable, 0, 0,
         v_asset_id, v_new_txn_id, v_created_by, v_booked_at, v_booked_at),
        (DATE(v_booked_at), 'revenue', v_partner_company, 0, v_receivable, 0,
         v_asset_id, v_new_txn_id, v_created_by, v_booked_at, v_booked_at),
        (DATE(v_booked_at), 'cash', 'Nhân viên', 0, v_transfer, 0,
         NULL, v_new_txn_id, v_created_by, v_booked_at, v_booked_at),
        (DATE(v_booked_at), 'revenue', v_partner_company, v_transfer, 0, 0,
         NULL, v_new_txn_id, v_created_by, v_booked_at, v_booked_at);

    UPDATE timesheets t
    JOIN backfill_target_timesheets target ON target.timesheet_id = t.id
       SET t.transaction_id = v_new_txn_id,
           t.updated_at = v_booked_at
     WHERE t.transaction_id IS NULL;

    IF ROW_COUNT() <> v_target_timesheet_count THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Not all target timesheets were linked';
    END IF;

    SELECT COUNT(*),
           COALESCE(SUM(le.debit), 0),
           COALESCE(SUM(le.credit), 0)
      INTO v_ledger_entry_count, v_total_debit, v_total_credit
      FROM ledger_entries le
     WHERE le.transaction_id = v_new_txn_id
       AND le.deleted_at IS NULL;

    SELECT COUNT(*) INTO v_linked_timesheet_count
      FROM timesheets t
      JOIN backfill_target_timesheets target ON target.timesheet_id = t.id
     WHERE t.transaction_id = v_new_txn_id;

    IF v_ledger_entry_count <> 4
       OR v_total_debit <> v_receivable + v_transfer
       OR v_total_credit <> v_receivable + v_transfer
       OR v_linked_timesheet_count <> v_target_timesheet_count THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Post-insert accounting validation failed';
    END IF;

    COMMIT;
    DROP TEMPORARY TABLE IF EXISTS backfill_target_timesheets;

    SELECT CONCAT('Batch #', v_target_batch_id, ' OK - transaction #', v_new_txn_id,
                  ', transfer=', v_transfer,
                  ', receivable=', v_receivable,
                  ', timesheets=', v_target_timesheet_count) AS result;
END$$

DELIMITER ;

CALL backfill_wallet_bulk_receivable_proc();
DROP PROCEDURE backfill_wallet_bulk_receivable_proc;
