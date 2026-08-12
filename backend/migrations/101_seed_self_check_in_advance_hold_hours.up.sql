-- +migrate Up
--
-- Seed the Admin-configurable delay between self checkout and advance-quota
-- credit. Keep an existing row unchanged so reruns never overwrite an
-- Admin-configured financial control.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES ('self_check_in_advance_hold_hours', '24', 'number')
ON DUPLICATE KEY UPDATE `key` = `key`;
