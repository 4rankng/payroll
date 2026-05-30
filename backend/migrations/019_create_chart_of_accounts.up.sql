-- Create accounts table for chart of accounts
-- This provides a proper account classification system for accounting
CREATE TABLE IF NOT EXISTS `accounts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(20) NOT NULL COMMENT 'Account code (e.g., 1000, 1100)',
  `name` varchar(100) NOT NULL COMMENT 'Account name',
  `type` enum('asset','liability','equity','revenue','expense') NOT NULL COMMENT 'Account type for classification',
  `parent_id` bigint unsigned DEFAULT NULL COMMENT 'Parent account for hierarchy',
  `deleted_at` timestamp(3) NULL DEFAULT NULL,
  `created_at` timestamp(3) NULL DEFAULT NULL,
  `updated_at` timestamp(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_accounts_code` (`code`),
  KEY `idx_accounts_deleted_at` (`deleted_at`),
  KEY `idx_accounts_type` (`type`),
  CONSTRAINT `fk_accounts_children` FOREIGN KEY (`parent_id`) REFERENCES `accounts` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Chart of accounts for proper account classification';

-- Insert standard account types
INSERT INTO `accounts` (`code`, `name`, `type`, `created_at`, `updated_at`) VALUES
('1000', 'Cash', 'asset', NOW(3), NOW(3)),
('1100', 'Accounts Receivable', 'asset', NOW(3), NOW(3)),
('2000', 'Accounts Payable', 'liability', NOW(3), NOW(3)),
('2100', 'Loans Payable', 'liability', NOW(3), NOW(3)),
('5000', 'Owner\'s Equity', 'equity', NOW(3), NOW(3)),
('4000', 'Service Revenue', 'revenue', NOW(3), NOW(3)),
('3000', 'Operating Expenses', 'expense', NOW(3), NOW(3));