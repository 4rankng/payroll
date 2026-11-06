-- Seed the Admin-configurable day of month the self-check-in advance request
-- window opens (bounds 10-28; days 1-8 belong to the regular flow's previous
-- period tail and day 9 is its locked gap). Keep an existing row unchanged so
-- reruns never overwrite an Admin-configured control.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES ('self_check_in_advance_window_open_day', '10', 'number')
ON DUPLICATE KEY UPDATE `key` = `key`;
