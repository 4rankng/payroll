-- +migrate Down
DROP INDEX idx_attendances_quota_credit_eligible ON attendances;
ALTER TABLE attendances
    DROP COLUMN quota_credit_eligible_at;
