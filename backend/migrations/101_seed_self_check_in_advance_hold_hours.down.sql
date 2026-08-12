-- +migrate Down
DELETE FROM settings
WHERE `key` = 'self_check_in_advance_hold_hours'
  AND `value` = '24'
  AND `value_type` = 'number';
