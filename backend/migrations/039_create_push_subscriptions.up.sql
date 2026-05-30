CREATE TABLE IF NOT EXISTS push_subscriptions (
    id bigint unsigned NOT NULL AUTO_INCREMENT,
    user_id bigint unsigned NOT NULL,
    endpoint varchar(500) NOT NULL,
    p256dh varchar(200) NOT NULL,
    auth varchar(100) NOT NULL,
    device_type varchar(20) NOT NULL DEFAULT 'web',
    created_at datetime(3) DEFAULT NULL,
    updated_at datetime(3) DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX idx_push_sub_user_endpoint (user_id, endpoint(191)),
    INDEX idx_push_sub_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
