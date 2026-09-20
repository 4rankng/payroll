-- Normalize every table onto one collation: `utf8mb4_0900_ai_ci`.
--
-- The problem
-- -----------
-- `payroll_db` is split across two collations — 27 tables on `utf8mb4_0900_ai_ci`
-- and 19 on `utf8mb4_unicode_ci`. MySQL refuses to compare a column from one side
-- with a column from the other:
--
--   ERROR 1267 (HY000): Illegal mix of collations (utf8mb4_unicode_ci,IMPLICIT)
--     and (utf8mb4_0900_ai_ci,IMPLICIT) for operation '='
--
-- The failure is not theoretical: it killed a real investigation query, because
-- `users` (`utf8mb4_unicode_ci`, created by 001_init_db at a time when the server
-- default was `utf8mb4_unicode_ci` — see `--collation-server` in
-- `docker-compose.dev.yml`) cannot be joined to `employees`
-- (`utf8mb4_0900_ai_ci`):
--
--   SELECT COUNT(*) FROM users u JOIN employees e ON u.username = e.cccd;
--
-- Every ad-hoc, reporting or maintenance query that spans the two families fails
-- the same way, and the split silently widens each time a new table is created
-- without an explicit COLLATE.
--
-- Why `utf8mb4_0900_ai_ci` is the target
-- --------------------------------------
-- It already owns the majority of the schema (27 of 46 tables) and is the only
-- collation the application pins explicitly
-- (`internal/infra/persistence/asset_repository.go` compares
-- `JSON_UNQUOTE(...) COLLATE utf8mb4_0900_ai_ci`). Converting the 19 stragglers is
-- the smaller and the application-consistent move.
--
-- Foreign keys
-- ------------
-- MySQL requires the character set and collation of a foreign key pair to match
-- only when the key columns are string-typed. Exactly one foreign key in this
-- schema is:
--
--   outbox_event_handlers.outbox_event_id -> outbox_events.event_id
--
-- and both of those tables already sit on `utf8mb4_0900_ai_ci`, so they are not in
-- the list below. Every other foreign key is numeric, and a numeric key is
-- unaffected by a table rebuild. No conversion here can therefore orphan a
-- constraint or fail on one, and the statements below are order-independent.
--
-- Uniqueness
-- ----------
-- `utf8mb4_0900_ai_ci` weighs characters with UCA 9.0.0 where
-- `utf8mb4_unicode_ci` uses UCA 4.0.0, so a conversion can in principle collide
-- two rows that were distinct before it. Every unique index over a string column
-- in the 19 tables was checked against 0900 equality on the live database before
-- this migration was written, and none has a collision. If a future database does,
-- the ALTER fails loudly with ERROR 1062 rather than dropping a row; the guards
-- make the file safe to re-run once the offending rows are reconciled.
--
-- Reversibility
-- -------------
-- `111_normalize_collations.down.sql` puts the same 19 tables back on
-- `utf8mb4_unicode_ci` and restores the previous schema default.
--
-- Safe to re-run: every step is emitted only while the target still differs.
--
-- Ordering note: the table list is alphabetical. Because no string foreign key
-- crosses the boundary (see above), alphabetical order is also a valid execution
-- order.

-- Step 1 — schema default.
-- The server default (`--collation-server=utf8mb4_unicode_ci`) still applies to
-- anything created without an explicit COLLATE, which is how the split started.
-- Point the schema itself at the canonical collation so new tables stay consistent.
-- Naturally idempotent: re-running sets the same value.
ALTER DATABASE CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Step 2 — the 19 tables still on `utf8mb4_unicode_ci`.
-- This is the exact snapshot of
--   SELECT table_name FROM information_schema.tables
--    WHERE table_schema = DATABASE()
--      AND table_collation = 'utf8mb4_unicode_ci' ORDER BY table_name;
-- taken when the file was written. Each conversion re-checks that condition for
-- its own table before emitting anything, so a table that has already moved (or
-- that does not exist yet) is skipped untouched.
--
-- `CONVERT TO CHARACTER SET` rewrites every string column of the table *and* adopts
-- the new collation as its table default, which is what makes columns added later
-- land correctly. The guard also means the file can be re-run, and is safe on a
-- database created after this migration.

-- Parent of `api_metrics.endpoint_id` (numeric foreign key, so execution order is free).
SET @schema_111_api_endpoints = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: api_endpoints already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `api_endpoints` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'api_endpoints'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_api_endpoints_stmt FROM @schema_111_api_endpoints;
EXECUTE schema_111_api_endpoints_stmt;
DEALLOCATE PREPARE schema_111_api_endpoints_stmt;

-- Largest table in this file (~26 MB / 152k rows); its rebuild dominates the runtime.
SET @schema_111_api_metrics = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: api_metrics already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `api_metrics` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'api_metrics'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_api_metrics_stmt FROM @schema_111_api_metrics;
EXECUTE schema_111_api_metrics_stmt;
DEALLOCATE PREPARE schema_111_api_metrics_stmt;

SET @schema_111_audit_logs = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: audit_logs already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `audit_logs` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'audit_logs'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_audit_logs_stmt FROM @schema_111_audit_logs;
EXECUTE schema_111_audit_logs_stmt;
DEALLOCATE PREPARE schema_111_audit_logs_stmt;

SET @schema_111_banks = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: banks already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `banks` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'banks'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_banks_stmt FROM @schema_111_banks;
EXECUTE schema_111_banks_stmt;
DEALLOCATE PREPARE schema_111_banks_stmt;

SET @schema_111_bulk_transfer_batches = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: bulk_transfer_batches already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `bulk_transfer_batches` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'bulk_transfer_batches'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_bulk_transfer_batches_stmt FROM @schema_111_bulk_transfer_batches;
EXECUTE schema_111_bulk_transfer_batches_stmt;
DEALLOCATE PREPARE schema_111_bulk_transfer_batches_stmt;

SET @schema_111_bulk_transfer_files = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: bulk_transfer_files already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `bulk_transfer_files` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'bulk_transfer_files'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_bulk_transfer_files_stmt FROM @schema_111_bulk_transfer_files;
EXECUTE schema_111_bulk_transfer_files_stmt;
DEALLOCATE PREPARE schema_111_bulk_transfer_files_stmt;

SET @schema_111_cron_job_status = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: cron_job_status already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `cron_job_status` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'cron_job_status'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_cron_job_status_stmt FROM @schema_111_cron_job_status;
EXECUTE schema_111_cron_job_status_stmt;
DEALLOCATE PREPARE schema_111_cron_job_status_stmt;

-- Child of both `users` and `employees`, through numeric foreign keys only.
SET @schema_111_employee_users = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: employee_users already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `employee_users` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'employee_users'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_employee_users_stmt FROM @schema_111_employee_users;
EXECUTE schema_111_employee_users_stmt;
DEALLOCATE PREPARE schema_111_employee_users_stmt;

SET @schema_111_flexpay_salary_notifications = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: flexpay_salary_notifications already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `flexpay_salary_notifications` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'flexpay_salary_notifications'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_flexpay_salary_notifications_stmt FROM @schema_111_flexpay_salary_notifications;
EXECUTE schema_111_flexpay_salary_notifications_stmt;
DEALLOCATE PREPARE schema_111_flexpay_salary_notifications_stmt;

SET @schema_111_lenders = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: lenders already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `lenders` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'lenders'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_lenders_stmt FROM @schema_111_lenders;
EXECUTE schema_111_lenders_stmt;
DEALLOCATE PREPARE schema_111_lenders_stmt;

SET @schema_111_loans = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: loans already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `loans` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'loans'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_loans_stmt FROM @schema_111_loans;
EXECUTE schema_111_loans_stmt;
DEALLOCATE PREPARE schema_111_loans_stmt;

SET @schema_111_push_subscriptions = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: push_subscriptions already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `push_subscriptions` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'push_subscriptions'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_push_subscriptions_stmt FROM @schema_111_push_subscriptions;
EXECUTE schema_111_push_subscriptions_stmt;
DEALLOCATE PREPARE schema_111_push_subscriptions_stmt;

SET @schema_111_settlements = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: settlements already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `settlements` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'settlements'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_settlements_stmt FROM @schema_111_settlements;
EXECUTE schema_111_settlements_stmt;
DEALLOCATE PREPARE schema_111_settlements_stmt;

SET @schema_111_timesheet_edit_requests = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: timesheet_edit_requests already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `timesheet_edit_requests` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'timesheet_edit_requests'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_timesheet_edit_requests_stmt FROM @schema_111_timesheet_edit_requests;
EXECUTE schema_111_timesheet_edit_requests_stmt;
DEALLOCATE PREPARE schema_111_timesheet_edit_requests_stmt;

SET @schema_111_transactions = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: transactions already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `transactions` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'transactions'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_transactions_stmt FROM @schema_111_transactions;
EXECUTE schema_111_transactions_stmt;
DEALLOCATE PREPARE schema_111_transactions_stmt;

-- Holds `username`, `cccd` and `mobile` — the columns of the join that started this — and three unique indexes over them.
SET @schema_111_users = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: users already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `users` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'users'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_users_stmt FROM @schema_111_users;
EXECUTE schema_111_users_stmt;
DEALLOCATE PREPARE schema_111_users_stmt;

SET @schema_111_wallet_ipn = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: wallet_ipn already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `wallet_ipn` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'wallet_ipn'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_wallet_ipn_stmt FROM @schema_111_wallet_ipn;
EXECUTE schema_111_wallet_ipn_stmt;
DEALLOCATE PREPARE schema_111_wallet_ipn_stmt;

-- `uk_wp_provider_invoice_no` is a unique key over the string columns `provider` + `invoice_no`.
SET @schema_111_wallet_payments = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: wallet_payments already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `wallet_payments` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'wallet_payments'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_wallet_payments_stmt FROM @schema_111_wallet_payments;
EXECUTE schema_111_wallet_payments_stmt;
DEALLOCATE PREPARE schema_111_wallet_payments_stmt;

SET @schema_111_wallet_topups = (
    SELECT IF(
        COUNT(*) = 0,
        'SELECT ''111: wallet_topups already utf8mb4_0900_ai_ci''',
        'ALTER TABLE `wallet_topups` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci'
    )
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'wallet_topups'
      AND table_collation <> 'utf8mb4_0900_ai_ci'
);

PREPARE schema_111_wallet_topups_stmt FROM @schema_111_wallet_topups;
EXECUTE schema_111_wallet_topups_stmt;
DEALLOCATE PREPARE schema_111_wallet_topups_stmt;

-- Step 3 — verify.
-- Expect: tables_on_unicode_ci = 0, every base table on `utf8mb4_0900_ai_ci`,
-- the schema default on `utf8mb4_0900_ai_ci`, no column drifting from its table,
-- and the join that used to raise ERROR 1267 resolving (a count is enough — it
-- only has to execute).
SELECT
    (SELECT COUNT(*) FROM information_schema.tables
      WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
        AND table_collation = 'utf8mb4_unicode_ci') AS tables_on_unicode_ci,
    (SELECT COUNT(*) FROM information_schema.tables
      WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE') AS tables_total,
    (SELECT DEFAULT_COLLATION_NAME FROM information_schema.schemata
      WHERE schema_name = DATABASE()) AS schema_default_collation,
    (SELECT COUNT(*) FROM information_schema.columns AS c
       JOIN information_schema.tables AS t
         ON t.table_schema = c.table_schema AND t.table_name = c.table_name
      WHERE c.table_schema = DATABASE()
        AND c.collation_name IS NOT NULL
        AND c.collation_name <> t.table_collation) AS columns_drifting_from_table,
    (SELECT COUNT(*) FROM users AS u
       JOIN employees AS e ON u.username = e.cccd) AS users_joined_to_employees;
