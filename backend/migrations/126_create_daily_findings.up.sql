CREATE TABLE IF NOT EXISTS `daily_findings` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `date` VARCHAR(10) NOT NULL,
  `fingerprint` VARCHAR(191) NOT NULL,
  `type` VARCHAR(64) NOT NULL,
  `severity` VARCHAR(16) NOT NULL DEFAULT 'info',
  `facts` JSON NOT NULL,
  `narrative` TEXT NULL,
  `dismissed_at` DATETIME(3) NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_daily_findings_date_fingerprint` (`date`, `fingerprint`),
  KEY `idx_daily_findings_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
