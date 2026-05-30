CREATE TABLE `outbox_event_handlers` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `outbox_event_id` VARCHAR(36)
        CHARACTER SET utf8mb4
        COLLATE utf8mb4_0900_ai_ci NOT NULL,
    `handler_type` VARCHAR(100) NOT NULL,
    `status` ENUM('pending', 'processing', 'completed', 'failed')
        NOT NULL DEFAULT 'pending',
    `retry_count` INT NOT NULL DEFAULT 0,
    `max_retries` INT NOT NULL DEFAULT 5,
    `last_error` TEXT,
    `started_at` DATETIME(3),
    `completed_at` DATETIME(3),
    `next_retry_at` DATETIME(3),
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (`id`),

    KEY `idx_outbox_event_id` (`outbox_event_id`),
    KEY `idx_handler_type` (`handler_type`),
    KEY `idx_status` (`status`),
    KEY `idx_next_retry_at` (`next_retry_at`),
    INDEX `idx_outbox_event_handler` (`outbox_event_id`, `handler_type`),

    CONSTRAINT `fk_outbox_event_handlers_outbox_event_id`
        FOREIGN KEY (`outbox_event_id`)
        REFERENCES `outbox_events` (`event_id`)
        ON DELETE CASCADE
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_0900_ai_ci;
