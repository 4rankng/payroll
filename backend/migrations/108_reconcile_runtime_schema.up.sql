-- Reconcile schema omissions in older installations without changing balances
-- or rewriting historical migrations. Every step is safe to re-run.
DELIMITER //
DROP PROCEDURE IF EXISTS reconcile_runtime_schema_108//
CREATE PROCEDURE reconcile_runtime_schema_108()
BEGIN
    DECLARE role_type TEXT;

    SELECT COLUMN_TYPE INTO role_type
    FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = 'role';
    IF role_type NOT LIKE '%''accountant''%' THEN
        -- Append the role; do not remove any roles already present locally.
        SET @schema_108_sql = CONCAT('ALTER TABLE users MODIFY COLUMN role ',
            LEFT(role_type, LENGTH(role_type) - 1), ',''accountant'') NOT NULL DEFAULT ''employee''');
        PREPARE schema_108_stmt FROM @schema_108_sql;
        EXECUTE schema_108_stmt;
        DEALLOCATE PREPARE schema_108_stmt;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = 'ledger_entries' AND column_name = 'settlement_id'
    ) THEN
        ALTER TABLE ledger_entries ADD COLUMN settlement_id BIGINT UNSIGNED NULL
            COMMENT 'Settlement linkage for payment clearing entries';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.statistics
        WHERE table_schema = DATABASE() AND table_name = 'ledger_entries' AND column_name = 'settlement_id' AND seq_in_index = 1
    ) THEN
        ALTER TABLE ledger_entries ADD INDEX idx_ledger_entries_settlement_id (settlement_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = 'assets' AND column_name = 'updated_at'
    ) THEN
        ALTER TABLE assets ADD COLUMN updated_at DATETIME(3) NULL;
    END IF;

    -- Retain legacy branch codes. Current bank writes do not populate this
    -- obsolete field, so an old NOT NULL/no-default column blocks all inserts.
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE() AND table_name = 'banks' AND column_name = 'branch_code'
            AND is_nullable = 'NO' AND column_default IS NULL
    ) THEN
        ALTER TABLE banks ALTER COLUMN branch_code SET DEFAULT '';
    END IF;
END//
DELIMITER ;
CALL reconcile_runtime_schema_108();
DROP PROCEDURE reconcile_runtime_schema_108;
