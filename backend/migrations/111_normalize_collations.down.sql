-- Revert 111_normalize_collations.
--
-- Puts exactly the 19 tables that migration converted back on `utf8mb4_unicode_ci`
-- and restores the schema default it changed.
--
-- This is a deliberate return to the broken state: any join between these tables
-- and the 27 that were already on `utf8mb4_0900_ai_ci` raises ERROR 1267 again —
-- `SELECT COUNT(*) FROM users u JOIN employees e ON u.username = e.cccd;` is the
-- shortest example. It exists to abandon the migration, not to be run routinely.
--
-- Why this file can be kept
-- -------------------------
-- The reverse conversion is checked the same way the forward one was: every unique
-- index over a string column in these 19 tables was tested for equality under
-- `utf8mb4_unicode_ci` on the live database and none merges two rows, and the only
-- string foreign key in the schema
-- (`outbox_event_handlers.outbox_event_id -> outbox_events.event_id`) stays on
-- `utf8mb4_0900_ai_ci` on both sides, so no constraint crosses the reverted
-- boundary. The reverse conversion therefore always succeeds on the data this
-- migration was written against.
--
-- Safe to re-run: each conversion is emitted only while the table is still off
-- `utf8mb4_unicode_ci`.

-- Step 1 — restore the schema default. It governs only tables created afterwards;
-- no existing table changes because of this statement.
ALTER DATABASE CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Step 2 — the same 19 tables, back to `utf8mb4_unicode_ci`.

SET @schema_111_api_endpoints = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: api_endpoints already utf8mb4_unicode_ci''',
        'ALTER TABLE `api_endpoints` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'api_endpoints'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_api_endpoints_stmt FROM @schema_111_api_endpoints;
EXECUTE schema_111_api_endpoints_stmt;
DEALLOCATE PREPARE schema_111_api_endpoints_stmt;

SET @schema_111_api_metrics = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: api_metrics already utf8mb4_unicode_ci''',
        'ALTER TABLE `api_metrics` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'api_metrics'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_api_metrics_stmt FROM @schema_111_api_metrics;
EXECUTE schema_111_api_metrics_stmt;
DEALLOCATE PREPARE schema_111_api_metrics_stmt;

SET @schema_111_audit_logs = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: audit_logs already utf8mb4_unicode_ci''',
        'ALTER TABLE `audit_logs` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'audit_logs'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_audit_logs_stmt FROM @schema_111_audit_logs;
EXECUTE schema_111_audit_logs_stmt;
DEALLOCATE PREPARE schema_111_audit_logs_stmt;

SET @schema_111_banks = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: banks already utf8mb4_unicode_ci''',
        'ALTER TABLE `banks` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'banks'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_banks_stmt FROM @schema_111_banks;
EXECUTE schema_111_banks_stmt;
DEALLOCATE PREPARE schema_111_banks_stmt;

SET @schema_111_bulk_transfer_batches = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: bulk_transfer_batches already utf8mb4_unicode_ci''',
        'ALTER TABLE `bulk_transfer_batches` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'bulk_transfer_batches'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_bulk_transfer_batches_stmt FROM @schema_111_bulk_transfer_batches;
EXECUTE schema_111_bulk_transfer_batches_stmt;
DEALLOCATE PREPARE schema_111_bulk_transfer_batches_stmt;

SET @schema_111_bulk_transfer_files = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: bulk_transfer_files already utf8mb4_unicode_ci''',
        'ALTER TABLE `bulk_transfer_files` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'bulk_transfer_files'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_bulk_transfer_files_stmt FROM @schema_111_bulk_transfer_files;
EXECUTE schema_111_bulk_transfer_files_stmt;
DEALLOCATE PREPARE schema_111_bulk_transfer_files_stmt;

SET @schema_111_cron_job_status = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: cron_job_status already utf8mb4_unicode_ci''',
        'ALTER TABLE `cron_job_status` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'cron_job_status'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_cron_job_status_stmt FROM @schema_111_cron_job_status;
EXECUTE schema_111_cron_job_status_stmt;
DEALLOCATE PREPARE schema_111_cron_job_status_stmt;

SET @schema_111_employee_users = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: employee_users already utf8mb4_unicode_ci''',
        'ALTER TABLE `employee_users` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'employee_users'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_employee_users_stmt FROM @schema_111_employee_users;
EXECUTE schema_111_employee_users_stmt;
DEALLOCATE PREPARE schema_111_employee_users_stmt;

SET @schema_111_flexpay_salary_notifications = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: flexpay_salary_notifications already utf8mb4_unicode_ci''',
        'ALTER TABLE `flexpay_salary_notifications` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'flexpay_salary_notifications'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_flexpay_salary_notifications_stmt FROM @schema_111_flexpay_salary_notifications;
EXECUTE schema_111_flexpay_salary_notifications_stmt;
DEALLOCATE PREPARE schema_111_flexpay_salary_notifications_stmt;

SET @schema_111_lenders = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: lenders already utf8mb4_unicode_ci''',
        'ALTER TABLE `lenders` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'lenders'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_lenders_stmt FROM @schema_111_lenders;
EXECUTE schema_111_lenders_stmt;
DEALLOCATE PREPARE schema_111_lenders_stmt;

SET @schema_111_loans = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: loans already utf8mb4_unicode_ci''',
        'ALTER TABLE `loans` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'loans'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_loans_stmt FROM @schema_111_loans;
EXECUTE schema_111_loans_stmt;
DEALLOCATE PREPARE schema_111_loans_stmt;

SET @schema_111_push_subscriptions = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: push_subscriptions already utf8mb4_unicode_ci''',
        'ALTER TABLE `push_subscriptions` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'push_subscriptions'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_push_subscriptions_stmt FROM @schema_111_push_subscriptions;
EXECUTE schema_111_push_subscriptions_stmt;
DEALLOCATE PREPARE schema_111_push_subscriptions_stmt;

SET @schema_111_settlements = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: settlements already utf8mb4_unicode_ci''',
        'ALTER TABLE `settlements` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'settlements'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_settlements_stmt FROM @schema_111_settlements;
EXECUTE schema_111_settlements_stmt;
DEALLOCATE PREPARE schema_111_settlements_stmt;

SET @schema_111_timesheet_edit_requests = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: timesheet_edit_requests already utf8mb4_unicode_ci''',
        'ALTER TABLE `timesheet_edit_requests` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'timesheet_edit_requests'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_timesheet_edit_requests_stmt FROM @schema_111_timesheet_edit_requests;
EXECUTE schema_111_timesheet_edit_requests_stmt;
DEALLOCATE PREPARE schema_111_timesheet_edit_requests_stmt;

SET @schema_111_transactions = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: transactions already utf8mb4_unicode_ci''',
        'ALTER TABLE `transactions` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'transactions'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_transactions_stmt FROM @schema_111_transactions;
EXECUTE schema_111_transactions_stmt;
DEALLOCATE PREPARE schema_111_transactions_stmt;

SET @schema_111_users = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: users already utf8mb4_unicode_ci''',
        'ALTER TABLE `users` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'users'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_users_stmt FROM @schema_111_users;
EXECUTE schema_111_users_stmt;
DEALLOCATE PREPARE schema_111_users_stmt;

SET @schema_111_wallet_ipn = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: wallet_ipn already utf8mb4_unicode_ci''',
        'ALTER TABLE `wallet_ipn` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'wallet_ipn'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_wallet_ipn_stmt FROM @schema_111_wallet_ipn;
EXECUTE schema_111_wallet_ipn_stmt;
DEALLOCATE PREPARE schema_111_wallet_ipn_stmt;

SET @schema_111_wallet_payments = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: wallet_payments already utf8mb4_unicode_ci''',
        'ALTER TABLE `wallet_payments` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'wallet_payments'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_wallet_payments_stmt FROM @schema_111_wallet_payments;
EXECUTE schema_111_wallet_payments_stmt;
DEALLOCATE PREPARE schema_111_wallet_payments_stmt;

SET @schema_111_wallet_topups = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: wallet_topups already utf8mb4_unicode_ci''',
        'ALTER TABLE `wallet_topups` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'wallet_topups'
      AND table_collation <> 'utf8mb4_unicode_ci'
);

PREPARE schema_111_wallet_topups_stmt FROM @schema_111_wallet_topups;
EXECUTE schema_111_wallet_topups_stmt;
DEALLOCATE PREPARE schema_111_wallet_topups_stmt;

-- Verify: on the database 111 was written against, `tables_on_unicode_ci` is back
-- to 19 (the tables above) and `tables_on_0900_ai_ci` to 27 — the tables 111 never
-- owned, which neither direction touches. The schema default must be
-- `utf8mb4_unicode_ci` again.
SELECT
    (SELECT COUNT(*) FROM information_schema.tables
      WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
        AND table_collation = 'utf8mb4_unicode_ci') AS tables_on_unicode_ci,
    (SELECT COUNT(*) FROM information_schema.tables
      WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
        AND table_collation = 'utf8mb4_0900_ai_ci') AS tables_on_0900_ai_ci,
    (SELECT DEFAULT_COLLATION_NAME FROM information_schema.schemata
      WHERE schema_name = DATABASE()) AS schema_default_collation;
