DELETE FROM settings
WHERE `key` = 'self_check_in_advance_window_open_day'
  AND `value` = '10'
  AND `value_type` = 'number';
