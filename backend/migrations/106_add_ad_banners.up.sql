-- Project-targeted ad campaigns shown in the employee portal ("Quảng cáo").
-- ends_at is NOT NULL by design: an endless campaign must be unrepresentable,
-- not merely discouraged by the UI. Lifetime is additionally capped at 180
-- days in domain validation.
CREATE TABLE IF NOT EXISTS ad_banners (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    body TEXT NULL,
    bullets JSON NULL COMMENT 'JSON string array, max 6 items',
    ctas JSON NULL COMMENT 'JSON array of {label, type: phone|url, value}, max 3 items',
    footer VARCHAR(255) NULL,
    target_project_ids JSON NULL COMMENT 'JSON uint array; NULL or empty = every project',
    priority INT NOT NULL DEFAULT 0,
    starts_at DATETIME(3) NOT NULL,
    ends_at DATETIME(3) NOT NULL COMMENT 'Mandatory campaign end; bounded lifetime',
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    KEY idx_ad_banners_active_window (is_active, starts_at, ends_at),
    KEY idx_ad_banners_deleted_at (deleted_at),
    CONSTRAINT fk_ad_banners_creator FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Append-only CTA click ledger. The unique key makes a tap idempotent per
-- (banner, employee, CTA): repeated taps cannot inflate the admin counters.
CREATE TABLE IF NOT EXISTS ad_banner_cta_clicks (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    banner_id BIGINT UNSIGNED NOT NULL,
    employee_id BIGINT UNSIGNED NOT NULL,
    cta_index TINYINT UNSIGNED NOT NULL,
    clicked_at DATETIME(3) NOT NULL,
    UNIQUE KEY uq_ad_banner_cta_click (banner_id, employee_id, cta_index),
    KEY idx_ad_banner_cta_clicks_banner (banner_id),
    CONSTRAINT fk_ad_banner_click_banner FOREIGN KEY (banner_id) REFERENCES ad_banners(id),
    CONSTRAINT fk_ad_banner_click_employee FOREIGN KEY (employee_id) REFERENCES employees(id)
);
