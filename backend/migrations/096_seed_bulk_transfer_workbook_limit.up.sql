-- +migrate Up
--
-- Seed the strict total threshold used when partitioning manual MBank
-- Chuyển lô workbooks. Keep an existing row unchanged so rerunning the
-- migration never overwrites an Admin-configured value.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES ('bulk_transfer_workbook_limit_vnd', '400000000', 'number')
ON DUPLICATE KEY UPDATE `key` = `key`;
