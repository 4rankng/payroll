-- Normalize the TingHire integration credential scope to 'tinghire'.
-- Migration 133 previously restored row id 3 under 'timesheet_read'
-- to avoid resurrecting the retired 'chatbot' scope.
-- TingHire is the recruitment/employee integration platform, not the
-- vendor reading timesheets, so it must carry its dedicated 'tinghire' scope.
UPDATE api_keys
SET scope = 'tinghire'
WHERE name = 'TingHire' AND revoked_at IS NULL;
