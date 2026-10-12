-- Retire every key bound to the removed chatbot surface. Legacy rows with an
-- empty scope resolved to the chatbot default, so they are revoked too. Rows
-- stay in place for audit history; revocation is not reversible.
UPDATE api_keys SET revoked_at = CURRENT_TIMESTAMP(3)
WHERE revoked_at IS NULL AND (scope = 'chatbot' OR scope = '');
