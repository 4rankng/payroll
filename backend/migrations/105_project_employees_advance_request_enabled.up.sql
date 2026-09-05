-- Per-employee advance payment kill switch ("tạm ngừng ứng lương").
-- 0 = the employee cannot create NEW advance payment requests (both the
-- regular flow and the self-check-in flow). Existing pending/approved
-- requests are untouched. Default 1 = allowed, preserving current behavior.

ALTER TABLE project_employees
  ADD COLUMN advance_request_enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Advance request kill switch: 0 = new advance requests blocked for this employee';
