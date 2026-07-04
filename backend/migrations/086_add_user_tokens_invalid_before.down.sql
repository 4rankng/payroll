-- Reverse of 086. Drops the per-user JWT invalidation column.
ALTER TABLE users DROP COLUMN tokens_invalid_before;
