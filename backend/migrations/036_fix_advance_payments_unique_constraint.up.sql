-- +migrate Up

-- Replace regular index with proper unique constraint
ALTER TABLE advance_payments DROP INDEX idx_adv_pay_unique,
ADD UNIQUE INDEX idx_adv_pay_unique (employee_id, project_id, for_month, upload_date);
