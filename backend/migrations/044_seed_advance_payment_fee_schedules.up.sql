-- +migrate Up
--
-- Seed the JSON-array advance-payment fee schedule into the existing settings
-- table and retire the legacy flat-percentage key it supersedes.
--
-- The new design stores all schedule history in a single settings row whose
-- value is a JSON array of {id, effective_date, tiers, min_fee_vnd, ...}
-- entries. The calculator picks the entry with the latest effective_date
-- that is <= the transaction date. See:
--   internal/domain/advance_payment_fee_schedule.go
--   internal/app/services/advance_payment/fee_schedule_service.go
--
-- This migration is idempotent — INSERT … ON DUPLICATE KEY UPDATE leaves the
-- value alone if the row already exists, so re-running it never clobbers
-- admin-created schedules.

-- Bootstrap entry preserves current behaviour: flat 2% with a 10,000 VND floor,
-- effective from 2020-01-01 so every existing transaction resolves to it.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES (
    'advance_payment_fee_schedules',
    '[{"id":"00000000-0000-4000-8000-000000000001","effective_date":"2020-01-01","tiers":[{"min_amount":0,"percentage":2.0}],"min_fee_vnd":10000,"notes":"Bootstrap from legacy hard-coded config","created_at":"2020-01-01T00:00:00Z"}]',
    'json'
)
ON DUPLICATE KEY UPDATE `key` = `key`;

-- Retire the legacy flat-percentage key. The new schedules row replaces it;
-- code that previously read `advance_cash_fee_percentage` now resolves the
-- active schedule and reads its first-tier percentage. Soft-deleted via the
-- GORM DeletedAt column so it remains queryable in audit views.
UPDATE settings
SET deleted_at = NOW()
WHERE `key` = 'advance_cash_fee_percentage' AND deleted_at IS NULL;
