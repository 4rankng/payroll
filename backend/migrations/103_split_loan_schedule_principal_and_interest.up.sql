-- Persist the allocation within each scheduled loan payment. The historical
-- schedule contract stores a single amount and generated schedules pay all
-- interest before principal, so the backfill follows that same contract.
-- A loan whose existing schedule totals less than its stated principal has no
-- recoverable principal/interest split. Stop before the non-transactional DDL
-- rather than guessing and corrupting its accounting history.
DELIMITER //
DROP PROCEDURE IF EXISTS validate_loan_schedule_component_backfill//
CREATE PROCEDURE validate_loan_schedule_component_backfill()
BEGIN
    DECLARE invalid_loan_count INT DEFAULT 0;
    DECLARE column_exists INT DEFAULT 0;

    SELECT COUNT(*) INTO invalid_loan_count
    FROM (
        SELECT schedules.loan_id
        FROM loan_repayment_schedules AS schedules
        INNER JOIN loans ON loans.id = schedules.loan_id
        GROUP BY schedules.loan_id, loans.principal_amount
        HAVING SUM(schedules.amount) < loans.principal_amount
    ) AS invalid_loans;

    IF invalid_loan_count > 0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Loan schedule total is below principal; repair the affected loan schedules before migration 103';
    END IF;

    SELECT COUNT(*) INTO column_exists
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'loan_repayment_schedules'
      AND column_name = 'principal_amount';
    IF column_exists = 0 THEN
        SET @migration_103_sql = 'ALTER TABLE loan_repayment_schedules ADD COLUMN principal_amount BIGINT NOT NULL DEFAULT 0 AFTER amount';
        PREPARE migration_103_statement FROM @migration_103_sql;
        EXECUTE migration_103_statement;
        DEALLOCATE PREPARE migration_103_statement;
    END IF;

    SELECT COUNT(*) INTO column_exists
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'loan_repayment_schedules'
      AND column_name = 'interest_amount';
    IF column_exists = 0 THEN
        SET @migration_103_sql = 'ALTER TABLE loan_repayment_schedules ADD COLUMN interest_amount BIGINT NOT NULL DEFAULT 0 AFTER principal_amount';
        PREPARE migration_103_statement FROM @migration_103_sql;
        EXECUTE migration_103_statement;
        DEALLOCATE PREPARE migration_103_statement;
    END IF;

    SELECT COUNT(*) INTO column_exists
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'transactions'
      AND column_name = 'loan_principal_amount';
    IF column_exists = 0 THEN
        SET @migration_103_sql = 'ALTER TABLE transactions ADD COLUMN loan_principal_amount BIGINT NOT NULL DEFAULT 0 AFTER amount';
        PREPARE migration_103_statement FROM @migration_103_sql;
        EXECUTE migration_103_statement;
        DEALLOCATE PREPARE migration_103_statement;
    END IF;

    SELECT COUNT(*) INTO column_exists
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'transactions'
      AND column_name = 'loan_interest_amount';
    IF column_exists = 0 THEN
        SET @migration_103_sql = 'ALTER TABLE transactions ADD COLUMN loan_interest_amount BIGINT NOT NULL DEFAULT 0 AFTER loan_principal_amount';
        PREPARE migration_103_statement FROM @migration_103_sql;
        EXECUTE migration_103_statement;
        DEALLOCATE PREPARE migration_103_statement;
    END IF;
END//
DELIMITER ;

CALL validate_loan_schedule_component_backfill();
DROP PROCEDURE validate_loan_schedule_component_backfill;

CREATE TEMPORARY TABLE loan_schedule_components AS
SELECT
    schedule_rows.id,
    LEAST(
        schedule_rows.amount,
        GREATEST(
            schedule_rows.total_repayment - schedule_rows.principal_amount - schedule_rows.prior_repayments,
            0
        )
    ) AS interest_amount
FROM (
    SELECT
        schedules.id,
        schedules.amount,
        loans.principal_amount,
        SUM(schedules.amount) OVER (PARTITION BY schedules.loan_id) AS total_repayment,
        COALESCE(
            SUM(schedules.amount) OVER (
                PARTITION BY schedules.loan_id
                ORDER BY schedules.period, schedules.id
                ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
            ),
            0
        ) AS prior_repayments
    FROM loan_repayment_schedules AS schedules
    INNER JOIN loans ON loans.id = schedules.loan_id
) AS schedule_rows;

UPDATE loan_repayment_schedules AS schedules
INNER JOIN loan_schedule_components AS components ON components.id = schedules.id
SET schedules.interest_amount = components.interest_amount,
    schedules.principal_amount = schedules.amount - components.interest_amount;

DROP TEMPORARY TABLE loan_schedule_components;

-- Carry each persisted allocation into its associated scheduled-payment
-- transaction so pending and direct settlements post the correct ledger split.
UPDATE transactions AS transactions
INNER JOIN loan_repayment_schedules AS schedules ON schedules.transaction_id = transactions.id
SET transactions.loan_principal_amount = schedules.principal_amount,
    transactions.loan_interest_amount = schedules.interest_amount
WHERE transactions.transaction_type = 'loan_repayment';

-- Rebuild the loan aggregates from settled schedules. This repairs loans whose
-- interest-only installments were previously deducted from principal.
UPDATE loans
INNER JOIN (
    SELECT
        loan_id,
        COALESCE(SUM(CASE WHEN status = 'paid' THEN principal_amount ELSE 0 END), 0) AS paid_principal,
        COALESCE(SUM(CASE WHEN status = 'paid' THEN interest_amount ELSE 0 END), 0) AS paid_interest
    FROM loan_repayment_schedules
    GROUP BY loan_id
) AS paid_schedules ON paid_schedules.loan_id = loans.id
SET loans.outstanding_principal = CASE
        WHEN loans.disbursed_at IS NULL THEN 0
        ELSE GREATEST(loans.principal_amount - paid_schedules.paid_principal, 0)
    END,
    loans.total_interest_paid = paid_schedules.paid_interest,
    loans.status = CASE
        WHEN loans.disbursed_at IS NOT NULL AND paid_schedules.paid_principal >= loans.principal_amount THEN 'closed'
        ELSE 'active'
    END;

-- Correct the opening ledger entry for every scheduled payment. A pure
-- interest payment becomes an expense entry; mixed payments retain the
-- principal debit and receive a separate interest-expense debit. Settlement
-- entries (settlement_id IS NOT NULL) already correctly clear payable to cash.
UPDATE ledger_entries AS entries
INNER JOIN loan_repayment_schedules AS schedules ON schedules.transaction_id = entries.transaction_id
INNER JOIN transactions ON transactions.id = entries.transaction_id
SET entries.debit = schedules.principal_amount
WHERE entries.deleted_at IS NULL
  AND entries.settlement_id IS NULL
  AND entries.account = 'loan'
  AND entries.debit > 0
  AND schedules.principal_amount > 0
  AND transactions.transaction_type = 'loan_repayment';

UPDATE ledger_entries AS entries
INNER JOIN loan_repayment_schedules AS schedules ON schedules.transaction_id = entries.transaction_id
INNER JOIN transactions ON transactions.id = entries.transaction_id
SET entries.account = 'expense'
WHERE entries.deleted_at IS NULL
  AND entries.settlement_id IS NULL
  AND entries.account = 'loan'
  AND entries.debit > 0
  AND schedules.principal_amount = 0
  AND transactions.transaction_type = 'loan_repayment';

INSERT INTO ledger_entries (
    date, account, party, debit, credit, balance, asset_id, transaction_id,
    settlement_id, created_by, created_at, updated_at
)
SELECT
    entries.date, 'expense', entries.party, schedules.interest_amount, 0, 0,
    entries.asset_id, entries.transaction_id, entries.settlement_id,
    entries.created_by, NOW(3), NOW(3)
FROM ledger_entries AS entries
INNER JOIN loan_repayment_schedules AS schedules ON schedules.transaction_id = entries.transaction_id
INNER JOIN transactions ON transactions.id = entries.transaction_id
WHERE entries.deleted_at IS NULL
  AND entries.settlement_id IS NULL
  AND entries.account = 'loan'
  AND entries.debit = schedules.principal_amount
  AND schedules.principal_amount > 0
  AND schedules.interest_amount > 0
  AND transactions.transaction_type = 'loan_repayment'
  AND NOT EXISTS (
      SELECT 1
      FROM ledger_entries AS existing_entries
      WHERE existing_entries.deleted_at IS NULL
        AND existing_entries.transaction_id = entries.transaction_id
        AND existing_entries.settlement_id IS NULL
        AND existing_entries.account = 'expense'
        AND existing_entries.debit = schedules.interest_amount
        AND existing_entries.credit = 0
  );

-- The repair changes historical ledger rows, so refresh the running working-
-- capital balance in the same order used by the ledger repository.
SET @loan_payment_running_balance := 0;
UPDATE ledger_entries AS entries
INNER JOIN (
    SELECT
        ordered_entries.id,
        (@loan_payment_running_balance := @loan_payment_running_balance + CASE
            WHEN ordered_entries.account IN ('cash', 'receivable') THEN ordered_entries.credit - ordered_entries.debit
            WHEN ordered_entries.account IN ('payable', 'loan') THEN ordered_entries.debit - ordered_entries.credit
            ELSE 0
        END) AS balance
    FROM (
        SELECT id, account, debit, credit
        FROM ledger_entries
        WHERE deleted_at IS NULL
        ORDER BY date ASC, id ASC
    ) AS ordered_entries
) AS recalculated_entries ON recalculated_entries.id = entries.id
SET entries.balance = recalculated_entries.balance;
