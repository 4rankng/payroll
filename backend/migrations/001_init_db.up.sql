-- Initial database setup for Payroll Management System
-- This migration contains the complete database schema
-- Generated from production database on 2025-11-13

SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Users table - Core authentication and user management
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `password` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `fullname` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `role` enum('admin','partner','employee') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'employee',
  `deleted_at` datetime(3) DEFAULT NULL,
  `last_login` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  UNIQUE KEY `idx_users_email` (`email`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Banks table - Bank branch information
CREATE TABLE `banks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `branch_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `branch_code` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `transfer_from` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_banks_branch_name` (`branch_name`),
  KEY `idx_banks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Lenders table - Loan lenders information
CREATE TABLE `lenders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `cccd` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `mobile` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_lenders_name` (`name`),
  KEY `idx_lenders_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- AUTHENTICATION TABLES
-- ============================================================================

-- Blacklisted JWT tokens for security
CREATE TABLE `blacklisted_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `token_jti` varchar(255) NOT NULL COMMENT 'JWT ID',
  `user_id` bigint unsigned NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `blacklisted_at` datetime(3) DEFAULT NULL,
  `reason` enum('logout','security','expired','admin_revoke') DEFAULT 'logout',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_blacklisted_tokens_token_jti` (`token_jti`),
  KEY `fk_blacklisted_tokens_user` (`user_id`),
  CONSTRAINT `fk_blacklisted_tokens_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ============================================================================
-- PROJECT MANAGEMENT TABLES
-- ============================================================================

-- Projects table - Client projects
CREATE TABLE `projects` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `client_name` varchar(255) DEFAULT NULL COMMENT 'e.g. Nha may san xuat hoa my pham VERICO',
  `name` varchar(255) NOT NULL COMMENT 'e.g. San xuat xa phong',
  `code` varchar(255) DEFAULT NULL COMMENT 'Initials of client_name and YYMM of created_at date or manually input by users',
  `description` text COMMENT 'Rich text format supported',
  `start_date` date DEFAULT NULL,
  `end_date` date DEFAULT NULL,
  `total_payout_vnd` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT 'Total paid to employees to date',
  `pending_payable_vnd` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT 'Pending payment to employees to date',
  `pending_receivable_vnd` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT 'Pending payment from the partner (staffing company) to date',
  `total_received_vnd` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT 'Total received from the partner to date',
  `project_status` enum('draft','active','paused','completed','cancelled') NOT NULL DEFAULT 'draft',
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `salary_period_from` bigint DEFAULT NULL COMMENT 'Day of previous month payroll period starts (0 or NULL = 1st)',
  `salary_period_to` bigint DEFAULT NULL COMMENT 'Day of current month payroll period ends (0 or NULL = last day)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_projects_code` (`code`),
  KEY `idx_projects_deleted_at` (`deleted_at`),
  KEY `fk_projects_creator` (`created_by`),
  CONSTRAINT `fk_projects_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `chk_salary_period_from_range` CHECK (((`salary_period_from` >= 0) and (`salary_period_from` <= 28))),
  CONSTRAINT `chk_salary_period_to_range` CHECK (((`salary_period_to` >= 0) and (`salary_period_to` <= 28)))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Project-User assignments
CREATE TABLE `project_users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `granted_by` bigint unsigned NOT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_project_user_unique` (`project_id`,`user_id`,`deleted_at`),
  KEY `idx_project_users_user` (`user_id`),
  KEY `idx_project_users_granted_by` (`granted_by`),
  CONSTRAINT `fk_project_users_granted_by` FOREIGN KEY (`granted_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_project_users_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),
  CONSTRAINT `fk_project_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ============================================================================
-- EMPLOYEE MANAGEMENT TABLES
-- ============================================================================

-- Employees table
CREATE TABLE `employees` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `fullname` varchar(255) NOT NULL,
  `email` varchar(255) DEFAULT NULL,
  `cccd` varchar(255) NOT NULL COMMENT 'Citizen ID - Can cong cong dan (12 digits)',
  `address` text,
  `mobile` varchar(15) DEFAULT NULL,
  `bank_id` bigint unsigned DEFAULT NULL,
  `bank_account_number` varchar(30) DEFAULT NULL,
  `bank_account_name` varchar(255) DEFAULT NULL,
  `date_of_birth` date DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `search_normalized` varchar(1000) GENERATED ALWAYS AS (lower(concat(coalesce(`fullname`,_utf8mb4''),_utf8mb4' ',coalesce(`email`,_utf8mb4''),_utf8mb4' ',coalesce(`cccd`,_utf8mb4''),_utf8mb4' ',coalesce(`mobile`,_utf8mb4'')))) VIRTUAL,
  `user_id` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_email_deleted_at` (`email`,`deleted_at`),
  UNIQUE KEY `unique_cccd_deleted_at` (`cccd`,`deleted_at`),
  KEY `idx_employees_deleted_at` (`deleted_at`),
  KEY `fk_employees_creator` (`created_by`),
  KEY `fk_employees_bank` (`bank_id`),
  KEY `idx_employees_user_id` (`user_id`),
  KEY `idx_employees_search_normalized` (`search_normalized`(255)),
  CONSTRAINT `fk_employees_bank` FOREIGN KEY (`bank_id`) REFERENCES `banks` (`id`),
  CONSTRAINT `fk_employees_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_employees_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_employees_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Employee-User assignments
CREATE TABLE `employee_users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `employee_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `granted_by` bigint unsigned NOT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_employee_user` (`employee_id`,`user_id`,`deleted_at`),
  KEY `fk_employee_users_granted_by` (`granted_by`),
  KEY `idx_employee_users_employee_id` (`employee_id`),
  KEY `idx_employee_users_user_id` (`user_id`),
  KEY `idx_employee_users_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_employee_users_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_employee_users_granted_by` FOREIGN KEY (`granted_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_employee_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Project-Employee assignments
CREATE TABLE `project_employees` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `employee_id` bigint unsigned NOT NULL,
  `employee_name` varchar(255) NOT NULL,
  `employee_cccd` varchar(255) NOT NULL COMMENT 'Citizen ID - Can cong cong dan (12 digits)',
  `employee_code` varchar(255) DEFAULT NULL COMMENT 'Factory-assigned code, optional',
  `position` varchar(100) NOT NULL DEFAULT 'phổ thông' COMMENT 'Employee position for payrate calculation',
  `start_date` date NOT NULL,
  `last_date` date DEFAULT NULL COMMENT 'NULL means currently active',
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `payment_schedule` varchar(10) NOT NULL DEFAULT 'weekly' COMMENT 'Current payment schedule: weekly or monthly',
  `pending_payment_schedule` varchar(10) DEFAULT NULL COMMENT 'Pending payment schedule change',
  `schedule_effective_from` date DEFAULT NULL COMMENT 'Date when pending schedule change becomes effective',
  PRIMARY KEY (`id`),
  KEY `idx_project_employees_deleted_at` (`deleted_at`),
  KEY `fk_project_employees_creator` (`created_by`),
  KEY `fk_project_employees_project` (`project_id`),
  KEY `fk_project_employees_employee` (`employee_id`),
  CONSTRAINT `fk_project_employees_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_project_employees_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`),
  CONSTRAINT `fk_project_employees_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ============================================================================
-- PAYROLL MANAGEMENT TABLES
-- ============================================================================

-- Pay rates configuration per project
CREATE TABLE `payrates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `payrate_json` json NOT NULL COMMENT 'custom format per project in format paytype: vnd_per_hour e.g. {"normal": 10000, "overtime": 15000, "weekend": 20000, "holiday": 25000}',
  `from_date` date NOT NULL,
  `to_date` date DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL COMMENT 'User who created the payrate',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_payrates_deleted_at` (`deleted_at`),
  KEY `fk_payrates_project` (`project_id`),
  KEY `fk_payrates_created_user` (`created_by`),
  CONSTRAINT `fk_payrates_created_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_payrates_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Timesheet edit requests
CREATE TABLE `timesheet_edit_requests` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `timesheet_id` bigint unsigned NOT NULL COMMENT 'Foreign key to timesheets table',
  `status` enum('pending','approved','rejected') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' COMMENT 'Request status',
  `requested_by` bigint unsigned NOT NULL COMMENT 'User ID who requested the edit',
  `approved_by` bigint unsigned DEFAULT NULL COMMENT 'Admin user ID who approved the request',
  `rejected_by` bigint unsigned DEFAULT NULL COMMENT 'Admin user ID who rejected the request',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Soft delete timestamp',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_ter_status` (`status`),
  KEY `idx_ter_requested_by` (`requested_by`),
  KEY `idx_ter_created_at` (`created_at` DESC),
  KEY `idx_ter_deleted_at` (`deleted_at`),
  KEY `fk_ter_approved_by` (`approved_by`),
  KEY `fk_ter_rejected_by` (`rejected_by`),
  CONSTRAINT `fk_ter_approved_by` FOREIGN KEY (`approved_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_ter_rejected_by` FOREIGN KEY (`rejected_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_ter_requested_by` FOREIGN KEY (`requested_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Tracks requests to edit approved timesheets';

-- Timesheets
CREATE TABLE `timesheets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint unsigned NOT NULL,
  `employee_id` bigint unsigned NOT NULL,
  `payrate_id` bigint unsigned NOT NULL COMMENT 'Reference to payrate config used',
  `date` date NOT NULL,
  `hours_worked` decimal(4,2) NOT NULL DEFAULT '0.00' COMMENT 'Hours worked (0.00-99.99)',
  `paytype` varchar(100) NOT NULL COMMENT 'Flexible paytype from payrate config',
  `payrate` bigint NOT NULL COMMENT 'VND amount from payrates config',
  `amount` bigint NOT NULL DEFAULT '0',
  `timesheet_status` enum('pending_approval','approved','rejected') NOT NULL DEFAULT 'pending_approval',
  `payment_status` enum('pending','paid','failed','cancelled') NOT NULL DEFAULT 'pending',
  `allowed_edit` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Flag indicating if timesheet is allowed to be edited after approval',
  `request_edit_id` bigint unsigned DEFAULT NULL COMMENT 'Pending edit request id',
  `payment_reference` varchar(255) DEFAULT NULL COMMENT 'Bank transaction reference',
  `payment_date` date DEFAULT NULL,
  `paid_amount` bigint DEFAULT '0' COMMENT 'Total amount paid for this timesheet',
  `paid_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL,
  `approved_by` bigint unsigned DEFAULT NULL COMMENT 'Admin who approved the timesheet',
  `approved_at` datetime(3) DEFAULT NULL,
  `rejection_reason` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `force_payroll` tinyint(1) NOT NULL DEFAULT '0',
  `bank_transfer_ref` varchar(100) DEFAULT NULL COMMENT 'Bank transaction reference or error message from result file (column I5)',
  PRIMARY KEY (`id`),
  KEY `idx_timesheets_deleted_at` (`deleted_at`),
  KEY `fk_timesheets_payrate` (`payrate_id`),
  KEY `idx_timesheets_project_employee_date` (`project_id`,`employee_id`,`date`),
  KEY `idx_timesheets_date_status` (`date`,`timesheet_status`),
  KEY `idx_timesheets_payment_status_date` (`payment_status`,`date`),
  KEY `idx_timesheets_created_by` (`created_by`),
  KEY `idx_timesheets_approved_by` (`approved_by`),
  KEY `idx_timesheets_employee_date` (`employee_id`,`date`),
  KEY `idx_timesheets_force_payroll` (`force_payroll`),
  KEY `idx_timesheets_allowed_edit` (`allowed_edit`),
  KEY `idx_timesheets_request_edit_id` (`request_edit_id`),
  KEY `idx_bank_transfer_ref` (`bank_transfer_ref`),
  CONSTRAINT `fk_timesheets_approved_user` FOREIGN KEY (`approved_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheets_created_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_timesheets_employee` FOREIGN KEY (`employee_id`) REFERENCES `employees` (`id`),
  CONSTRAINT `fk_timesheets_payrate` FOREIGN KEY (`payrate_id`) REFERENCES `payrates` (`id`),
  CONSTRAINT `fk_timesheets_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`),
  CONSTRAINT `fk_timesheets_request_edit_id` FOREIGN KEY (`request_edit_id`) REFERENCES `timesheet_edit_requests` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ============================================================================
-- FINANCIAL MANAGEMENT TABLES
-- ============================================================================

-- Assets table for file storage
CREATE TABLE `assets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `filename` varchar(255) NOT NULL,
  `file_path` varchar(500) NOT NULL,
  `upload_type` varchar(50) NOT NULL DEFAULT 'general',
  `checksum` varchar(64) DEFAULT NULL,
  `uploaded_by` bigint unsigned NOT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_assets_checksum_upload_type` (`checksum`),
  KEY `idx_assets_checksum` (`checksum`),
  KEY `idx_assets_deleted_at` (`deleted_at`),
  KEY `fk_assets_uploader` (`uploaded_by`),
  CONSTRAINT `fk_assets_uploader` FOREIGN KEY (`uploaded_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Loans table
CREATE TABLE `loans` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `lender_id` bigint unsigned NOT NULL,
  `loan_code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `principal_amount` bigint NOT NULL DEFAULT '0',
  `interest_rate_bps` bigint NOT NULL,
  `term_months` bigint NOT NULL,
  `start_date` date NOT NULL,
  `end_date` date NOT NULL,
  `payment_day_of_month` tinyint unsigned NOT NULL DEFAULT '1',
  `status` enum('active','closed') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `disbursed_at` datetime(3) DEFAULT NULL,
  `outstanding_principal` bigint NOT NULL DEFAULT '0',
  `total_interest_paid` bigint NOT NULL DEFAULT '0',
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_loans_loan_code` (`loan_code`),
  KEY `idx_loans_lender_id` (`lender_id`),
  KEY `idx_loans_status` (`status`),
  KEY `idx_loans_dates` (`start_date`,`end_date`),
  KEY `idx_loans_payment_day` (`payment_day_of_month`),
  KEY `idx_loans_created_by` (`created_by`),
  KEY `idx_loans_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_loans_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_loans_lender` FOREIGN KEY (`lender_id`) REFERENCES `lenders` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Transactions table
CREATE TABLE `transactions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'User-provided transaction description',
  `transaction_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Type of transaction from user perspective',
  `amount` bigint NOT NULL DEFAULT '0' COMMENT 'VND',
  `party` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Vendor, client, or other party involved',
  `loan_id` bigint unsigned DEFAULT NULL COMMENT 'Links interest transaction to loan',
  `status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' COMMENT 'Settlement status',
  `url` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Evidence link - external URL',
  `asset_id` bigint unsigned DEFAULT NULL COMMENT 'Reference to asset table for evidence files',
  `user_id` bigint unsigned DEFAULT NULL,
  `reversed_transaction_id` bigint DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL COMMENT 'User who created the transaction',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `settled_amount` bigint NOT NULL DEFAULT '0' COMMENT 'VND',
  `transaction_code` varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Unique transaction identifier',
  PRIMARY KEY (`id`),
  UNIQUE KEY `transaction_code` (`transaction_code`),
  KEY `idx_status` (`status`),
  KEY `idx_transaction_type` (`transaction_type`),
  KEY `idx_created_by` (`created_by`),
  KEY `idx_party` (`party`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`),
  KEY `asset_id` (`asset_id`),
  KEY `idx_settled_amount` (`settled_amount`),
  KEY `fk_transactions_user` (`user_id`),
  KEY `idx_transactions_loan_id` (`loan_id`),
  CONSTRAINT `fk_transactions_loan` FOREIGN KEY (`loan_id`) REFERENCES `loans` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_transactions_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `transactions_ibfk_2` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE RESTRICT,
  CONSTRAINT `transactions_ibfk_3` FOREIGN KEY (`asset_id`) REFERENCES `assets` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Loan repayment schedules
CREATE TABLE `loan_repayment_schedules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `loan_id` bigint unsigned NOT NULL,
  `period` bigint NOT NULL,
  `due_date` date NOT NULL,
  `amount` bigint NOT NULL,
  `status` enum('pending','paid') NOT NULL DEFAULT 'pending',
  `paid_at` datetime(3) DEFAULT NULL,
  `payment_ref` varchar(255) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `transaction_id` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_loan_repayment_schedules_loan_id` (`loan_id`),
  KEY `idx_loan_repayment_schedules_transaction` (`transaction_id`),
  CONSTRAINT `fk_loan_repayment_schedules_transaction` FOREIGN KEY (`transaction_id`) REFERENCES `transactions` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Ledger entries for bookkeeping
CREATE TABLE `ledger_entries` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `date` date NOT NULL,
  `description` text NOT NULL,
  `account` varchar(255) NOT NULL,
  `party` varchar(255) NOT NULL COMMENT 'Who you paid or received from',
  `debit` bigint NOT NULL DEFAULT '0' COMMENT 'Money out (VND)',
  `credit` bigint NOT NULL DEFAULT '0' COMMENT 'Money in (VND)',
  `balance` bigint NOT NULL DEFAULT '0' COMMENT 'Running balance (VND)',
  `reference` text COMMENT 'short description to identify source of fund or url link to transaction',
  `url` varchar(500) DEFAULT NULL COMMENT 'Evidence link - external URL or local file path',
  `asset_id` bigint unsigned DEFAULT NULL COMMENT 'Reference to asset table for evidence files',
  `transaction_id` bigint unsigned DEFAULT NULL COMMENT 'Reference to transaction table for user-facing transactions',
  `deleted_at` datetime(3) DEFAULT NULL,
  `created_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ledger_asset_unique` (`asset_id`,`deleted_at`),
  KEY `idx_ledger_entries_deleted_at` (`deleted_at`),
  KEY `fk_ledger_entries_creator` (`created_by`),
  KEY `idx_transaction_id` (`transaction_id`),
  CONSTRAINT `fk_ledger_entries_asset` FOREIGN KEY (`asset_id`) REFERENCES `assets` (`id`),
  CONSTRAINT `fk_ledger_entries_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `ledger_entries_ibfk_1` FOREIGN KEY (`transaction_id`) REFERENCES `transactions` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Settlements table
CREATE TABLE `settlements` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `transaction_id` bigint unsigned NOT NULL COMMENT 'Parent transaction being settled',
  `amount` bigint NOT NULL COMMENT 'Settlement amount (VND)',
  `settlement_date` date NOT NULL COMMENT 'Date of settlement (business date)',
  `proof_url` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'External URL for settlement proof',
  `proof_asset_id` bigint unsigned DEFAULT NULL COMMENT 'Internal file reference for settlement proof',
  `payment_method` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'cash' COMMENT 'cash, bank_transfer, check, etc.',
  `notes` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Additional notes about settlement',
  `created_by` bigint unsigned NOT NULL COMMENT 'User who created this settlement',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `proof_asset_id` (`proof_asset_id`),
  KEY `idx_transaction_id` (`transaction_id`),
  KEY `idx_settlement_date` (`settlement_date`),
  KEY `idx_deleted_at` (`deleted_at`),
  KEY `idx_created_by` (`created_by`),
  CONSTRAINT `settlements_ibfk_1` FOREIGN KEY (`transaction_id`) REFERENCES `transactions` (`id`) ON DELETE RESTRICT,
  CONSTRAINT `settlements_ibfk_2` FOREIGN KEY (`proof_asset_id`) REFERENCES `assets` (`id`) ON DELETE SET NULL,
  CONSTRAINT `settlements_ibfk_3` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- BULK TRANSFER TABLES
-- ============================================================================

-- Bulk transfer files
CREATE TABLE `bulk_transfer_files` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `filename` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Format: {bankprefix}_{cycle}_{uuid_no_hyphen}',
  `cycle` enum('weekly','monthly') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Payment cycle type',
  `created_by` bigint unsigned NOT NULL COMMENT 'User who initiated the export',
  `from_date` date DEFAULT NULL COMMENT 'Range start for weekly cycle',
  `to_date` date DEFAULT NULL COMMENT 'Range end for weekly cycle',
  `for_month` char(7) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Format: YYYY-MM for monthly cycle',
  `transactions_count` int NOT NULL DEFAULT '0' COMMENT 'Total transaction records',
  `transfer_amount` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Total sum of amounts',
  `data` json NOT NULL COMMENT 'Full data payload to regenerate the file',
  `checksum` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'SHA256 checksum of concatenated transaction codes (sorted)',
  `transfer_result` bigint unsigned DEFAULT NULL COMMENT 'FK to assets.id for uploaded bank result file',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Soft delete timestamp',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_checksum` (`checksum`),
  KEY `idx_created_by` (`created_by`),
  KEY `idx_cycle` (`cycle`),
  KEY `idx_from_date` (`from_date`),
  KEY `idx_to_date` (`to_date`),
  KEY `idx_for_month` (`for_month`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`),
  KEY `idx_transfer_result` (`transfer_result`),
  CONSTRAINT `fk_bulk_transfer_files_created_by` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE RESTRICT,
  CONSTRAINT `fk_bulk_transfer_files_transfer_result` FOREIGN KEY (`transfer_result`) REFERENCES `assets` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bulk transfer histories
CREATE TABLE `bulk_transfer_histories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `asset_id` bigint unsigned NOT NULL COMMENT 'Reference to the uploaded file',
  `transaction_id` bigint unsigned DEFAULT NULL COMMENT 'Reference to transaction created for this bulk transfer',
  `filename` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Original filename of the uploaded file',
  `export_date` date DEFAULT NULL,
  `payment_schedule` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'weekly' COMMENT 'Payment schedule type: weekly or monthly',
  `total_processed` int NOT NULL DEFAULT '0' COMMENT 'Total number of records processed',
  `success_count` int NOT NULL DEFAULT '0' COMMENT 'Number of successful transfers',
  `failed_count` int NOT NULL DEFAULT '0' COMMENT 'Number of failed transfers',
  `processing_result` json NOT NULL COMMENT 'Full processing result including details and errors',
  `uploaded_by` bigint unsigned NOT NULL COMMENT 'User who uploaded the file',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_asset_id` (`asset_id`),
  KEY `idx_transaction_id` (`transaction_id`),
  KEY `idx_uploaded_by` (`uploaded_by`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_export_date` (`export_date`),
  KEY `idx_payment_schedule` (`payment_schedule`),
  CONSTRAINT `bulk_transfer_histories_ibfk_1` FOREIGN KEY (`asset_id`) REFERENCES `assets` (`id`) ON DELETE CASCADE,
  CONSTRAINT `bulk_transfer_histories_ibfk_2` FOREIGN KEY (`transaction_id`) REFERENCES `transactions` (`id`) ON DELETE SET NULL,
  CONSTRAINT `bulk_transfer_histories_ibfk_3` FOREIGN KEY (`uploaded_by`) REFERENCES `users` (`id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- SYSTEM TABLES
-- ============================================================================

-- Notifications table
CREATE TABLE `notifications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sender_id` bigint unsigned NOT NULL COMMENT 'User who initiated/sent the notification',
  `recipient_id` bigint unsigned DEFAULT NULL COMMENT 'User who receives the notification (null for emails to external recipients)',
  `type` varchar(50) NOT NULL,
  `channel` varchar(20) NOT NULL DEFAULT 'push',
  `title` varchar(255) NOT NULL,
  `message` text NOT NULL,
  `content_type` varchar(20) NOT NULL DEFAULT 'plain_text',
  `email_recipients` text,
  `resend_message_id` varchar(255) DEFAULT NULL COMMENT 'Resend API message ID for tracking',
  `read_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_notifications_channel` (`channel`),
  KEY `idx_sender_id` (`sender_id`),
  KEY `idx_recipient_id` (`recipient_id`),
  KEY `idx_notifications_content_type` (`content_type`),
  CONSTRAINT `fk_notifications_recipient` FOREIGN KEY (`recipient_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_notifications_sender` FOREIGN KEY (`sender_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Settings table
CREATE TABLE `settings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(100) NOT NULL,
  `value` text,
  `value_type` enum('string','number','boolean','json') DEFAULT 'string',
  `deleted_at` datetime(3) DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Audit logs for tracking all system actions
CREATE TABLE `audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned DEFAULT NULL,
  `action` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'CREATE, UPDATE, DELETE, APPROVE, REJECT, LOGIN, LOGOUT',
  `entity_type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'user, project, employee, timesheet, payroll',
  `entity_id` bigint unsigned DEFAULT NULL,
  `old_values_json` json DEFAULT NULL,
  `new_values_json` json DEFAULT NULL,
  `ip_address` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_user_created` (`user_id`,`created_at`),
  KEY `idx_audit_logs_entity_created` (`entity_type`,`entity_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API metrics tracking
CREATE TABLE `api_metrics` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `method` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `status_code` int NOT NULL,
  `user_id` bigint unsigned DEFAULT NULL,
  `duration_ms` bigint NOT NULL,
  `called_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_method_path` (`method`,`path`),
  KEY `idx_called_at` (`called_at`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- MIGRATION MANAGEMENT TABLES
-- ============================================================================

-- Migration locks for preventing concurrent migrations
CREATE TABLE `migration_locks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `lock_key` varchar(100) NOT NULL COMMENT 'Lock identifier',
  `is_locked` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Whether lock is currently held',
  `locked_by` varchar(255) DEFAULT NULL COMMENT 'Identifier of process holding lock',
  `locked_at` datetime(3) DEFAULT NULL COMMENT 'When lock was acquired',
  `released_at` datetime(3) DEFAULT NULL COMMENT 'When lock was released',
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_migration_locks_lock_key` (`lock_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Migration history tracking
CREATE TABLE `migration_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `version` varchar(255) NOT NULL COMMENT 'Migration version or filename',
  `type` varchar(50) NOT NULL COMMENT 'Migration type: sql or gorm',
  `applied` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Whether migration was successfully applied',
  `applied_at` datetime(3) DEFAULT NULL COMMENT 'When migration was applied',
  `checksum` varchar(64) DEFAULT NULL COMMENT 'SHA256 checksum of migration content',
  `error` text COMMENT 'Error message if migration failed',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_migration_history_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

SET FOREIGN_KEY_CHECKS = 1;
