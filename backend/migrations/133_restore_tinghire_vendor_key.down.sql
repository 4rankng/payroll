-- Inverse of 133: retire the TingHire row again by putting it back under the
-- legacy chatbot scope and revoking it. The original revoked_at instant from
-- migration 130 is not recoverable, so the rollback stamps a fresh revocation
-- time. Rows id=1 and id=2 are untouched (already revoked).
UPDATE api_keys
SET scope = 'chatbot',
    revoked_at = CURRENT_TIMESTAMP(3)
WHERE id = 3
  AND name = 'TingHire'
  AND scope = 'timesheet_read'
  AND revoked_at IS NULL;
