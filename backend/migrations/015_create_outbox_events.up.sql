-- Create outbox_events table for transactional outbox pattern
-- This ensures events are published reliably even if the service crashes
CREATE TABLE IF NOT EXISTS `outbox_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `event_id` varchar(36) NOT NULL COMMENT 'UUID of the event for idempotency',
  `event_type` varchar(100) NOT NULL COMMENT 'Type of event (e.g., TransactionCreated, SettlementCreated)',
  `aggregate_type` varchar(50) NOT NULL COMMENT 'Type of aggregate (transaction, settlement, ledger_entry)',
  `aggregate_id` bigint unsigned NOT NULL COMMENT 'ID of the aggregate',
  `payload` json NOT NULL COMMENT 'Event payload as JSON',
  `status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'pending, published, failed',
  `retry_count` int NOT NULL DEFAULT 0 COMMENT 'Number of retry attempts',
  `max_retries` int NOT NULL DEFAULT 5 COMMENT 'Maximum number of retries',
  `last_error` text DEFAULT NULL COMMENT 'Last error message if failed',
  `published_at` timestamp NULL DEFAULT NULL COMMENT 'When the event was successfully published',
  `next_retry_at` timestamp NULL DEFAULT NULL COMMENT 'When to retry next (exponential backoff)',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_event_id` (`event_id`),
  KEY `idx_status` (`status`),
  KEY `idx_aggregate` (`aggregate_type`, `aggregate_id`),
  KEY `idx_next_retry` (`status`, `next_retry_at`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Transactional outbox for reliable event publishing';
