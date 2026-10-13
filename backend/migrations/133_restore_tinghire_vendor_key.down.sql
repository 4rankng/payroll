-- Inverse of 133: put the TingHire key back under the retired chatbot scope and
-- revoke it again. The original revoked_at instant from migration 130 is not
-- recoverable, so the rollback stamps a fresh revocation time.
UPDATE api_keys
SET scope = 'chatbot',
    revoked_at = CURRENT_TIMESTAMP(3)
WHERE name = 'TingHire'
  AND scope = 'timesheet_read'
  AND revoked_at IS NULL
  AND last_used_at IS NOT NULL;
