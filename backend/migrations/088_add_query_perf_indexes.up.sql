-- 088: query-performance indexes surfaced by the ck:debug investigation (2026-07-04).
-- All four were verified non-duplicative against the full migration chain:
--
--   audit_logs.entity_id         — dropped in 008, never restored (034 built
--                                  idx_action_entity which can't serve a lone
--                                  entity_id=? predicate). GET /audit/logs?entityId=
--                                  was full-scanning an append-only table.
--   transactions.reversed_transaction_id — FK exists but no index; the
--                                  NOT IN (SELECT reversed_transaction_id ...)
--                                  subquery scanned the whole table twice per
--                                  list/count request.
--   users.last_login             — queried as IS NULL (never-logged-in job)
--                                  and date ranges (dashboard activity); no index.
--   transactions (status, created_at) — common admin list/export combo; only
--                                  single-column indexes existed.
--
-- All InnoDB secondary indexes; added via ALGORITHM=INPLACE (LOCK=NONE) online.

CREATE INDEX idx_audit_logs_entity_id
    ON audit_logs (entity_id, created_at);

CREATE INDEX idx_transactions_reversed_txn_id
    ON transactions (reversed_transaction_id);

CREATE INDEX idx_users_last_login
    ON users (last_login);

CREATE INDEX idx_transactions_status_created
    ON transactions (status, created_at);
