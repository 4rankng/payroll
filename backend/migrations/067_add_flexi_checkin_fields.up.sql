ALTER TABLE projects ADD COLUMN is_flexible BOOLEAN NOT NULL DEFAULT FALSE;

-- Set LG Display project to flexible as per spec
UPDATE projects SET is_flexible = true WHERE id=58;

ALTER TABLE project_employees ADD COLUMN check_in_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE attendances (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  employee_id      BIGINT UNSIGNED NOT NULL,
  project_id       BIGINT UNSIGNED NOT NULL,
  date             DATE NOT NULL,            -- ngày check-in (ca đêm = ngày bắt đầu ca)
  check_in_time    DATETIME(3) NOT NULL,
  check_out_time   DATETIME(3) NULL,
  check_in_lat     DECIMAL(10,7) NOT NULL,
  check_in_lng     DECIMAL(10,7) NOT NULL,
  check_out_lat    DECIMAL(10,7) NULL,
  check_out_lng    DECIMAL(10,7) NULL,
  check_in_gate    VARCHAR(50) NOT NULL,
  check_out_gate   VARCHAR(50) NULL,
  earning_amount   BIGINT NULL DEFAULT 0,    -- VND
  created_at       DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  updated_at       DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_employee_date (employee_id, date),
  INDEX idx_project_date (project_id, date),
  CONSTRAINT fk_attendances_employee FOREIGN KEY (employee_id) REFERENCES employees(id),
  CONSTRAINT fk_attendances_project FOREIGN KEY (project_id) REFERENCES projects(id)
);
