-- 058_seed_disbursement_fee_schedules_v2.up.sql
-- Production has no real data here; bootstrap entries from migration
-- 050 are placeholders. Reseed with the provider-aware shape.
-- NOTE: created_at must be RFC3339 (Go time.Time unmarshal format),
--       NOT MySQL UTC_TIMESTAMP() which produces "YYYY-MM-DD HH:MM:SS".
UPDATE settings
SET value = JSON_ARRAY(
    JSON_OBJECT(
        'id',              UUID(),
        'provider',        '9pay',
        'effective_date',  '2026-01-01',
        'fee_vnd',         3300,
        'notes',           'Seed entry for 9Pay (migrated)',
        'created_at',      '2026-01-01T00:00:00Z'
    ),
    JSON_OBJECT(
        'id',              UUID(),
        'provider',        '1pay',
        'effective_date',  '2026-01-01',
        'fee_vnd',         3300,
        'notes',           'Placeholder for OnePay — update with contract rate',
        'created_at',      '2026-01-01T00:00:00Z'
    )
)
WHERE `key` = 'disbursement_fee_schedules';
