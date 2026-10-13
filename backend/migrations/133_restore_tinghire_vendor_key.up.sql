-- Reactivate the EXISTING TingHire credential — no new key is issued, so the
-- vendor keeps its current secret and nothing needs re-distribution.
--
-- Target: api_keys row id=3 (name 'TingHire', key prefix ttk_70...), created
-- 27/09/2026 and live (last_used 2026-10-12 14:41) until migration 130
-- revoked every legacy 'chatbot'-scope key when the LLM chatbot was retired.
-- That row is the key TingHire uses for password-reset OTP, self check-in,
-- employee lookup, and the Zalo OA token pull.
--
-- 1) clear revoked_at (idempotent — a re-run matches nothing once the scope is
--    live), 2) move the scope from the retired 'chatbot' value to the live
--    vendor scope the middleware checks
--    (RequireAPIKeyScope(timesheet_read) on /api/v1/integration/* including
--    /api/v1/integration/zalo/token). Rows id=1 and id=2 (the September
--    ChatBot / superseded TingHire keys, manually revoked) are left revoked.
--
-- The name + legacy-scope predicates ride along with the id so the statement
-- can never revive a different row that happens to hold id=3 in another
-- environment.
UPDATE api_keys
SET revoked_at = NULL,
    scope = 'timesheet_read'
WHERE id = 3
  AND name = 'TingHire'
  AND scope IN ('chatbot', '');
