-- +migrate Up
--
-- Seed the Admin-configurable self-check-in advance percentage. Keep an
-- existing row unchanged so reruns never overwrite an Admin-configured value.
-- No quota rewrite is needed here because the default remains 70%.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES ('self_check_in_advance_percentage', '70', 'number')
ON DUPLICATE KEY UPDATE `key` = `key`;
