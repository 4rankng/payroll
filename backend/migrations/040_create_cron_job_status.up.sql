CREATE TABLE IF NOT EXISTS cron_job_status (
    id bigint unsigned NOT NULL AUTO_INCREMENT,
    job_name varchar(100) NOT NULL,
    cron varchar(50) NOT NULL DEFAULT '',
    is_enabled tinyint(1) NOT NULL DEFAULT 1,
    status varchar(20) NOT NULL DEFAULT 'success',
    last_run_at datetime(3) DEFAULT NULL,
    duration_ms bigint DEFAULT NULL,
    last_error text DEFAULT NULL,
    updated_at datetime(3) DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_job_name (job_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
