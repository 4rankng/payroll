-- +migrate Up

-- Table: advance_payments
-- Stores monthly advance payment limits from Flexible Payroll Template uploads
CREATE TABLE advance_payments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    project_id BIGINT UNSIGNED NOT NULL,
    employee_id BIGINT UNSIGNED NOT NULL,
    for_month VARCHAR(7) NOT NULL COMMENT 'Month in YYYY-MM format',
    upload_date VARCHAR(7) NOT NULL COMMENT 'Upload month in YYYY-MM format',
    salary BIGINT UNSIGNED NOT NULL COMMENT 'Full salary amount for the month',
    max_adv_amount BIGINT UNSIGNED NOT NULL COMMENT 'Maximum advance = salary * percentage',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_adv_pay_employee (employee_id),
    INDEX idx_adv_pay_month (for_month),
    INDEX idx_adv_pay_project_month (project_id, for_month),
    INDEX idx_adv_pay_unique (employee_id, project_id, for_month, upload_date),

    CONSTRAINT fk_adv_pay_project FOREIGN KEY (project_id)
        REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_adv_pay_employee FOREIGN KEY (employee_id)
        REFERENCES employees(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table: advance_payment_requests
-- Stores individual advance payment requests from employees
CREATE TABLE advance_payment_requests (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    adv_pay_id BIGINT UNSIGNED NOT NULL,
    project_id BIGINT UNSIGNED NOT NULL COMMENT 'Denormalized for query performance',
    employee_id BIGINT UNSIGNED NOT NULL COMMENT 'Denormalized for query performance',
    request_amount BIGINT UNSIGNED NOT NULL,
    fee BIGINT UNSIGNED NOT NULL COMMENT 'Fee = MAX(10000, 2% * request_amount)',
    net_amount BIGINT UNSIGNED NOT NULL COMMENT 'Net = request_amount - fee',
    status ENUM('PENDING', 'CANCELLED', 'COMPLETED', 'APPROVED', 'FAILED') NOT NULL DEFAULT 'PENDING',
    payment_reference VARCHAR(255) NULL COMMENT 'Bank transaction reference',
    paid_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_apr_adv_pay (adv_pay_id),
    INDEX idx_apr_employee_status (employee_id, status),
    INDEX idx_apr_status_created (status, created_at),

    CONSTRAINT fk_apr_adv_pay FOREIGN KEY (adv_pay_id)
        REFERENCES advance_payments(id) ON DELETE CASCADE,
    CONSTRAINT fk_apr_project FOREIGN KEY (project_id)
        REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_apr_employee FOREIGN KEY (employee_id)
        REFERENCES employees(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table: transaction_codes
-- Tracks transaction codes linking bank transfers to advance payment requests
CREATE TABLE transaction_codes (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(255) NOT NULL COMMENT 'Unique code VFIC-{uuid}',
    data JSON NOT NULL COMMENT '{"adv_pay_requests": [1, 2, 3]}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Modify bulk_transfer_files cycle enum to include 'flexible'
ALTER TABLE bulk_transfer_files
MODIFY COLUMN cycle ENUM('weekly', 'monthly', 'flexible') NOT NULL;

ALTER TABLE assets
ADD COLUMN metadata JSON NULL COMMENT 'JSON metadata for asset-specific tracking (e.g., import progress)';

ALTER TABLE advance_payment_requests
ADD COLUMN receivable_settled_at TIMESTAMP NULL;

