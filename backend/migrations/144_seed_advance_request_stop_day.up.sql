-- Seed the Admin-configurable advance request stop day: the day of month from
-- which new advance requests are refused (the previous-period request tail is
-- days 1..stopDay-1). Default 9 reproduces the historical hardcode — requests
-- allowed through day 8, day 9 reserved for the admin's sao kê. Keep an
-- existing row unchanged so reruns never overwrite an Admin-configured control.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES ('advance_request_stop_day', '9', 'number')
ON DUPLICATE KEY UPDATE `key` = `key`;
