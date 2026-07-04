-- Reverse of 088. Drops the four query-performance indexes added by the
-- ck:debug 2026-07-04 audit.
DROP INDEX idx_transactions_status_created   ON transactions;
DROP INDEX idx_users_last_login              ON users;
DROP INDEX idx_transactions_reversed_txn_id  ON transactions;
DROP INDEX idx_audit_logs_entity_id          ON audit_logs;
