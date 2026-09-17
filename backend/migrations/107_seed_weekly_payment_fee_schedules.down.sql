-- +migrate Down
--
-- Remove the weekly-payment fee schedule settings row seeded by the up
-- migration. Soft-delete keeps the row queryable in audit views.

UPDATE settings
SET deleted_at = NOW()
WHERE `key` = 'weekly_payment_fee_schedules' AND deleted_at IS NULL;
