-- +migrate Up
--
-- Seed the JSON-array weekly-payment fee schedule into the settings table.
--
-- The weekly-payment fee (phí trả lương tuần) is the percentage service fee
-- charged on weekly salary disbursements: partner receivable = transfer ×
-- (1 + fee%). Historically it shared the FlexPay advance-payment fee schedule
-- (first-tier percentage), which coupled two unrelated fee domains. This
-- migration seeds the decoupled weekly schedule so weekly disbursements keep
-- resolving to 2% — identical to dev/prod today — while FlexPay advance
-- fees stay independent going forward.
--
-- Idempotent: ON DUPLICATE KEY UPDATE leaves any existing row untouched, so
-- re-running never clobbers admin-created schedules.

INSERT INTO settings (`key`, `value`, `value_type`)
VALUES (
    'weekly_payment_fee_schedules',
    '[{"id":"00000000-0000-4000-8000-000000000001","effective_date":"2020-01-01","percentage":2.0,"notes":"Bootstrap from the shared fee schedule"}]',
    'json'
)
ON DUPLICATE KEY UPDATE `key` = `key`;
