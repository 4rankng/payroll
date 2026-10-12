-- Roll back the scope column. Keys minted with a timesheet_read scope fall back
-- to 'chatbot' semantics (the pre-127 behaviour: full integration access), so
-- this is a widening, never a lock-out — but revoke any vendor keys before
-- rolling back in an environment where the surface matters.
ALTER TABLE api_keys DROP INDEX idx_api_keys_scope;

ALTER TABLE api_keys DROP COLUMN scope;