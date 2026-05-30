-- Recover orphaned settlements and orphan ledger entries.
--
-- Root cause (now fixed in code):
--   1. Atomic revenue_paid + Settlement creation: prevents the orphan state where
--      timesheets are revenue_paid=1 but no Settlement record exists.
--   2. Per-transaction settlement allocation (FlexPay): prevents buggy bulk settlements
--      that left duplicate ledger entries without backing Settlement records.
--
-- This script repairs legacy data:
--   STEP 1. Delete orphan settlement-style ledger entries (cash debit + receivable credit
--           paired entries whose asset has no Settlement record but the same transaction
--           has other proper settlements). These are leftovers from the FlexPay double-credit bug.
--   STEP 2. Create recovery Settlements + double-entry ledger entries for transactions where
--           SUM(revenue_paid timesheets revenue_receivable) > settled_amount.
--
-- Idempotent: running it again after all orphans are fixed is a no-op.

-- ============================================================================
-- STEP 1: Delete orphan settlement-style ledger entries
-- ============================================================================
-- An entry is "settlement-style" if it has an asset_id, account=cash|receivable,
-- and transaction_id. It is an "orphan" if its asset_id has no Settlement record
-- AND a duplicate (same transaction, account, amount) settlement-backed entry exists.
-- MySQL requires a derived-table form to avoid "can't reference target table" error.
DELETE FROM ledger_entries
WHERE id IN (
    SELECT id FROM (
        SELECT le.id
        FROM ledger_entries le
        WHERE le.asset_id IS NOT NULL
          AND le.settlement_id IS NULL
          AND le.transaction_id IS NOT NULL
          AND le.account IN ('cash', 'receivable')
          AND NOT EXISTS (SELECT 1 FROM settlements s WHERE s.proof_asset_id = le.asset_id)
          AND EXISTS (
              SELECT 1 FROM ledger_entries le3
              WHERE le3.transaction_id = le.transaction_id
                AND le3.account = le.account
                AND le3.debit = le.debit AND le3.credit = le.credit
                AND le3.settlement_id IS NOT NULL
          )
    ) AS tmp
);

SELECT ROW_COUNT() AS deleted_orphan_ledger_entries;

-- ============================================================================
-- STEP 2: Create recovery settlements for orphaned timesheets
-- ============================================================================
DELIMITER //

DROP PROCEDURE IF EXISTS RecoverOrphanedSettlements//

CREATE PROCEDURE RecoverOrphanedSettlements()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE v_txn_id BIGINT UNSIGNED;
    DECLARE v_amount BIGINT;
    DECLARE v_orphan BIGINT;
    DECLARE v_party VARCHAR(255);
    DECLARE v_settlement_id BIGINT UNSIGNED;
    DECLARE v_count INT DEFAULT 0;

    DECLARE cur CURSOR FOR
        SELECT
            t.id,
            t.amount,
            COALESCE(SUM(CASE WHEN ts.revenue_paid = 1 THEN ts.revenue_receivable ELSE 0 END), 0) - t.settled_amount AS orphan_amount
        FROM transactions t
        LEFT JOIN timesheets ts ON ts.transaction_id = t.id
        WHERE t.transaction_type = 'revenue'
          AND t.status NOT IN ('settled', 'cancelled')
        GROUP BY t.id, t.amount, t.settled_amount
        HAVING orphan_amount > 0;

    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;

    OPEN cur;
    read_loop: LOOP
        FETCH cur INTO v_txn_id, v_amount, v_orphan;
        IF done THEN LEAVE read_loop; END IF;

        -- Pick the party from the transaction's existing receivable entry (initial creation entry).
        SELECT party INTO v_party
        FROM ledger_entries
        WHERE transaction_id = v_txn_id AND account = 'receivable' AND debit > 0
        ORDER BY id ASC LIMIT 1;

        IF v_party IS NULL THEN SET v_party = 'Recovery'; END IF;

        -- Create recovery settlement
        INSERT INTO settlements (
            transaction_id, amount, settlement_date, payment_method,
            notes, created_by, created_at, updated_at, settlement_uuid
        ) VALUES (
            v_txn_id, v_orphan, CURDATE(), 'bank_transfer',
            'Recovery settlement for orphaned revenue_paid timesheets', 1, NOW(), NOW(), UUID()
        );

        SET v_settlement_id = LAST_INSERT_ID();

        -- Double-entry ledger: debit cash, credit receivable
        INSERT INTO ledger_entries (date, account, party, debit, credit, transaction_id, settlement_id, created_by, created_at, updated_at)
        VALUES
            (CURDATE(), 'cash',        v_party, v_orphan, 0,        v_txn_id, v_settlement_id, 1, NOW(), NOW()),
            (CURDATE(), 'receivable',  v_party, 0,        v_orphan, v_txn_id, v_settlement_id, 1, NOW(), NOW());

        -- Recompute settled_amount from settlements (source of truth) and mark fully settled
        UPDATE transactions
        SET settled_amount = (SELECT COALESCE(SUM(amount), 0) FROM settlements WHERE transaction_id = v_txn_id AND deleted_at IS NULL),
            status = CASE
                WHEN (SELECT COALESCE(SUM(amount), 0) FROM settlements WHERE transaction_id = v_txn_id AND deleted_at IS NULL) >= amount THEN 'settled'
                ELSE 'partially_settled'
            END
        WHERE id = v_txn_id;

        SET v_count = v_count + 1;
    END LOOP;
    CLOSE cur;

    SELECT v_count AS recovered_settlements;
END//

DELIMITER ;

CALL RecoverOrphanedSettlements();
DROP PROCEDURE RecoverOrphanedSettlements;
