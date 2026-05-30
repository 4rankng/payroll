-- +migrate Up
--
-- Seed the JSON-array disbursement-provider fee schedule into the existing
-- settings table and retire the legacy single-value key it supersedes.
--
-- Same pattern as 044_seed_advance_payment_fee_schedules: the new design stores
-- all schedule history in a single settings row whose value is a JSON array of
-- {id, effective_date, fee_vnd, ...} entries. The disbursement service picks
-- the entry with the latest effective_date that is <= the transaction date.
-- See:
--   internal/domain/disbursement_fee_schedule.go
--   internal/app/services/disbursement/fee_schedule_service.go
--
-- This migration is idempotent — INSERT … ON DUPLICATE KEY UPDATE leaves the
-- value alone if the row already exists, so re-running it never clobbers
-- admin-created schedules.

-- Bootstrap entry preserves current behaviour: flat 200 VND per transfer
-- (the 9pay rate as of 2026-05), effective from 2020-01-01 so every existing
-- transaction resolves to it.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES (
    'disbursement_fee_schedules',
    '[{"id":"00000000-0000-4000-8000-000000000050","effective_date":"2020-01-01","fee_vnd":200,"notes":"Bootstrap from disbursement_fee_vnd setting","created_at":"2020-01-01T00:00:00Z"}]',
    'json'
)
ON DUPLICATE KEY UPDATE `key` = `key`;

-- Retire the legacy single-value key. The new schedules row replaces it; code
-- that previously read `disbursement_fee_vnd` now resolves the active schedule
-- and reads its fee_vnd. Soft-deleted via the GORM DeletedAt column so it
-- remains queryable in audit views.
UPDATE settings
SET deleted_at = NOW()
WHERE `key` = 'disbursement_fee_vnd' AND deleted_at IS NULL;
