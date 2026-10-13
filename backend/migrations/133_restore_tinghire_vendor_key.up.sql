-- Restore the TingHire vendor key that migration 130 swept up in the chatbot
-- retirement. TingHire is the external payroll vendor integration: its key was
-- created 27/09/2026, before scoped API keys existed, so it carries the legacy
-- 'chatbot' scope label while its actual access is the vendor timesheet read
-- surface. Migration 130 revoked every 'chatbot'/'empty' key when the LLM
-- chatbot was removed, silently cutting the vendor's access mid-use.
--
-- Only the actively-used TingHire key is restored: the never-used historical
-- TingHire/ChatBot rows were revoked manually in September and stay revoked.
-- The scope is normalised to the vendor read scope because the scope guard
-- matches exactly and fails closed on retired values.
UPDATE api_keys
SET revoked_at = NULL,
    scope = 'timesheet_read'
WHERE name = 'TingHire'
  AND scope = 'chatbot'
  AND last_used_at IS NOT NULL;
