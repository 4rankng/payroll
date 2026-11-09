DELETE FROM settings
WHERE `key` = 'advance_request_stop_day'
  AND `value` = '9'
  AND `value_type` = 'number';
