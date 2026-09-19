-- Hard-delete the duplicate orphaned VFIC receivable created by the manual
-- bulk-transfer file PhamThiHoe.xlsx (transaction_code
-- bc039188-867d-4ca4-a3eb-ac859cb04503, local id 204).
--
-- Why it is a duplicate: the batch result file 26081021052552580.xlsx reported
-- employee 1284's transfer as `failed`, so its timesheets were never linked.
-- A manual file (PhamThiHoe.xlsx) was then recorded for her alone and created
-- this receivable. A later batch result (26081022381994008.xlsx) reported the
-- same 1.799.982 VND transfer with a real bank reference, created receivable
-- 205, and re-pointed timesheets 54856/54857/54858 to it. This transaction was
-- left with zero timesheets, zero settlements and zero payment history, so no
-- sao ke upload can ever settle it. 205 is the settled source of truth.
--
-- Safety properties:
--   * targets one transaction_code, never a numeric id (ids differ per env)
--   * refuses anything that is not `pending`
--   * refuses if settlements, timesheets, advance requests, loan schedules or
--     reversed-by references exist
--   * refuses if any outbox event for the transaction is still unpublished
--   * refuses unless the ledger block is exactly the standard bulk-transfer
--     4-entry shape and is internally balanced
--   * children (ledger entries) are hard-deleted before the parent, so the
--     transaction FK is never violated
--   * shifts the running `balance` column for every ledger entry ordered after
--     the deleted block by exactly the block's balance delta, so the running
--     column stays truthful without rewriting unrelated history
--   * writes an audit_logs DELETE row (the app's TransactionDeleted handler
--     cannot run for a direct DML change)
--   * reruns are a verified no-op
--
-- Usage:
--   -- dry run (default), prints what would change and touches nothing
--   CALL fix_duplicate_orphan_receivable('bc039188-...', 0);
--   -- apply
--   CALL fix_duplicate_orphan_receivable('bc039188-...', 1);
--
-- After a successful run, invalidate the app's transaction cache:
--   redis-cli --scan --pattern 'transactions:*' | xargs -r redis-cli DEL

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

DROP PROCEDURE IF EXISTS fix_duplicate_orphan_receivable;
DELIMITER $$

CREATE PROCEDURE fix_duplicate_orphan_receivable(
    IN p_code  VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    IN p_apply TINYINT
)
main: BEGIN
    DECLARE v_id            BIGINT UNSIGNED;
    DECLARE v_description   TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    DECLARE v_amount        BIGINT;
    DECLARE v_status        VARCHAR(50);
    DECLARE v_created_by    BIGINT UNSIGNED;
    DECLARE v_asset_id      BIGINT UNSIGNED;

    DECLARE v_entries       INT DEFAULT 0;
    DECLARE v_settlements   INT DEFAULT 0;
    DECLARE v_timesheets    INT DEFAULT 0;
    DECLARE v_advance_reqs  INT DEFAULT 0;
    DECLARE v_loan_scheds   INT DEFAULT 0;
    DECLARE v_reversed_by   INT DEFAULT 0;
    DECLARE v_pending_evt   INT DEFAULT 0;

    DECLARE v_total_debit   BIGINT DEFAULT 0;
    DECLARE v_total_credit  BIGINT DEFAULT 0;
    DECLARE v_recv_debits   INT DEFAULT 0;
    DECLARE v_cash_credits  INT DEFAULT 0;
    DECLARE v_orphan_rows   INT DEFAULT 0;
    DECLARE v_shifted       INT DEFAULT 0;

    DECLARE v_delta         BIGINT DEFAULT 0;
    DECLARE v_last_id       BIGINT UNSIGNED;
    DECLARE v_last_date     DATE;
    DECLARE v_error         TEXT;

    DECLARE EXIT HANDLER FOR SQLEXCEPTION
    BEGIN
        ROLLBACK;
        RESIGNAL;
    END;

    START TRANSACTION;

    SELECT t.id, t.description, t.amount, t.status, t.created_by, t.asset_id
      INTO v_id, v_description, v_amount, v_status, v_created_by, v_asset_id
      FROM transactions t
     WHERE t.transaction_code = p_code
     FOR UPDATE;

    IF v_id IS NULL THEN
        COMMIT;
        SELECT 'SKIP' AS result,
               CONCAT('No transaction with code ', p_code,
                      ' - already removed or never existed') AS detail;
        LEAVE main;
    END IF;

    -- ── guards ────────────────────────────────────────────────────────────
    IF v_status COLLATE utf8mb4_unicode_ci <> 'pending' THEN
        SET v_error = CONCAT('Refusing: transaction #', v_id, ' status is "', v_status,
                             '", only pending transactions may be deleted');
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = v_error;
    END IF;

    SELECT COUNT(*) INTO v_settlements  FROM settlements  WHERE transaction_id = v_id;
    SELECT COUNT(*) INTO v_timesheets   FROM timesheets   WHERE transaction_id = v_id;
    SELECT COUNT(*) INTO v_advance_reqs FROM advance_payment_requests WHERE settlement_transaction_id = v_id;
    SELECT COUNT(*) INTO v_loan_scheds  FROM loan_repayment_schedules  WHERE transaction_id = v_id;
    SELECT COUNT(*) INTO v_reversed_by  FROM transactions WHERE reversed_transaction_id = v_id;
    SELECT COUNT(*) INTO v_pending_evt
      FROM outbox_events
     WHERE aggregate_type LIKE '%ransaction%'
       AND aggregate_id = v_id
       AND status COLLATE utf8mb4_unicode_ci <> 'published';

    IF v_settlements > 0 THEN
        SET v_error = CONCAT('Refusing: transaction #', v_id, ' has ', v_settlements,
                             ' settlement(s) - settled money must be reversed, not deleted');
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = v_error;
    END IF;
    IF v_timesheets > 0 THEN
        SET v_error = CONCAT('Refusing: transaction #', v_id, ' has ', v_timesheets,
                             ' linked timesheet(s) - not an orphan');
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = v_error;
    END IF;
    IF v_advance_reqs > 0 OR v_loan_scheds > 0 OR v_reversed_by > 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Refusing: transaction is referenced by advance requests, loan schedules or a reversal';
    END IF;
    IF v_pending_evt > 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Refusing: transaction still has unpublished outbox events';
    END IF;

    -- ── ledger block geometry ─────────────────────────────────────────────
    SELECT COUNT(*),
           COALESCE(SUM(debit), 0),
           COALESCE(SUM(credit), 0),
           COALESCE(SUM(account COLLATE utf8mb4_unicode_ci = 'receivable' AND debit = v_amount AND credit = 0), 0),
           COALESCE(SUM(account COLLATE utf8mb4_unicode_ci = 'cash' AND credit > 0 AND debit = 0), 0),
           COALESCE(SUM(CASE
                            WHEN account IN ('cash', 'receivable') THEN credit - debit
                            WHEN account IN ('payable', 'loan')    THEN debit - credit
                            ELSE 0
                        END), 0),
           MAX(id)
      INTO v_entries, v_total_debit, v_total_credit,
           v_recv_debits, v_cash_credits, v_delta, v_last_id
      FROM ledger_entries
     WHERE transaction_id = v_id
       AND deleted_at IS NULL;

    IF v_entries = 0 THEN
        SET v_error = CONCAT('Refusing: transaction #', v_id, ' has no live ledger entries to delete');
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = v_error;
    END IF;
    IF v_entries <> 4
       OR v_total_debit <> v_total_credit
       OR v_recv_debits <> 1
       OR v_cash_credits <> 1 THEN
        SET v_error = CONCAT('Refusing: ledger block for #', v_id,
                             ' is not the standard balanced bulk-transfer shape (entries=',
                             v_entries, ', debit=', v_total_debit, ', credit=', v_total_credit, ')');
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = v_error;
    END IF;

    SELECT date INTO v_last_date FROM ledger_entries WHERE id = v_last_id;

    SELECT COUNT(*) INTO v_shifted
      FROM ledger_entries
     WHERE deleted_at IS NULL
       AND (date > v_last_date OR (date = v_last_date AND id > v_last_id));

    -- ── dry run report ────────────────────────────────────────────────────
    IF COALESCE(p_apply, 0) = 0 THEN
        COMMIT;
        SELECT 'DRY RUN - nothing changed' AS result;
        SELECT v_id            AS transaction_id,
               v_amount        AS amount,
               v_status        AS status,
               v_asset_id      AS asset_id,
               v_entries       AS ledger_entries_to_delete,
               v_delta         AS ledger_delta,
               v_last_id       AS block_last_ledger_id,
               v_last_date     AS block_last_ledger_date,
               v_shifted       AS rows_to_shift,
               (v_total_debit = v_total_credit) AS block_balanced;
        LEAVE main;
    END IF;

    -- ── apply ─────────────────────────────────────────────────────────────

    -- Keep the running `balance` column truthful: removing the block changes the
    -- cumulative balance of every entry ordered after it (canonical order is
    -- date ASC, id ASC - see LedgerEntryRepository.RecalculateAllBalances) by
    -- exactly v_delta. Pre-existing drift elsewhere is deliberately untouched.
    UPDATE ledger_entries
       SET balance = balance - v_delta
     WHERE deleted_at IS NULL
       AND (date > v_last_date OR (date = v_last_date AND id > v_last_id));

    INSERT INTO audit_logs (user_id, message, action, entity_type, entity_id, metadata, created_at)
    VALUES (v_created_by,
            CONCAT('Đã xóa giao dịch ', v_description, ' (DML: duplicate orphan receivable)'),
            'DELETE', 'transaction', v_id,
            JSON_OBJECT('amount', v_amount,
                        'transaction_code', p_code,
                        'reason', 'duplicate orphan receivable - money already settled under another transaction',
                        'ledger_delta', v_delta),
            NOW(3));

    -- Children first: ledger_entries.transaction_id has an FK to transactions.id.
    DELETE FROM ledger_entries WHERE transaction_id = v_id;
    DELETE FROM transactions    WHERE id = v_id;

    -- ── post-conditions ───────────────────────────────────────────────────
    IF EXISTS (SELECT 1 FROM ledger_entries WHERE transaction_id = v_id)
       OR EXISTS (SELECT 1 FROM transactions    WHERE id = v_id) THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Post-condition failed: transaction or ledger entries survived the delete';
    END IF;

    SELECT COUNT(*) INTO v_orphan_rows
      FROM ledger_entries le
      LEFT JOIN transactions t ON t.id = le.transaction_id
     WHERE le.transaction_id IS NOT NULL
       AND t.id IS NULL;

    COMMIT;

    SELECT 'APPLIED' AS result;
    SELECT v_id      AS deleted_transaction_id,
           v_amount  AS deleted_amount,
           v_delta   AS ledger_delta,
           v_shifted AS balances_shifted,
           v_orphan_rows AS orphan_ledger_rows_remaining;
END$$

DELIMITER ;
