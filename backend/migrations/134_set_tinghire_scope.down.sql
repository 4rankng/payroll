-- Revert TingHire key scope to timesheet_read.
UPDATE api_keys
SET scope = 'timesheet_read'
WHERE name = 'TingHire' AND revoked_at IS NULL;
