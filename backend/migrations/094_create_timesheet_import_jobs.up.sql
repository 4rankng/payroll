CREATE TABLE IF NOT EXISTS timesheet_import_jobs (
    asset_id BIGINT UNSIGNED NOT NULL,
    project_id BIGINT UNSIGNED NOT NULL,
    for_month CHAR(7) NOT NULL,
    uploaded_by BIGINT UNSIGNED NOT NULL,
    uploader_role VARCHAR(32) NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'failed') NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(128) NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    active_scope_key VARCHAR(64) NULL,
    attempt INT UNSIGNED NOT NULL DEFAULT 0,
    lease_expires_at DATETIME(3) NULL,
    started_at DATETIME(3) NULL,
    processed_at DATETIME(3) NULL,
    audit_logged_at DATETIME(3) NULL,
    last_error TEXT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (asset_id),
    UNIQUE KEY uq_timesheet_import_idempotency (uploaded_by, idempotency_key),
    UNIQUE KEY uq_timesheet_import_active_scope (active_scope_key),
    KEY idx_timesheet_import_recovery (status, lease_expires_at),
    CONSTRAINT fk_timesheet_import_asset
        FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE,
    CONSTRAINT fk_timesheet_import_project
        FOREIGN KEY (project_id) REFERENCES projects(id),
    CONSTRAINT fk_timesheet_import_uploader
        FOREIGN KEY (uploaded_by) REFERENCES users(id)
);
