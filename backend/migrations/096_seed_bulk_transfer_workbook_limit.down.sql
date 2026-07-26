-- +migrate Down
DELETE FROM settings
WHERE `key` = 'bulk_transfer_workbook_limit_vnd'
  AND `value` = '400000000'
  AND `value_type` = 'number';
