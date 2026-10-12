-- Remove the MiMo assistant settings. The sealed API key value is encrypted
-- with a deployment-held key and cannot be restored, so the down migration is
-- a no-op: re-inserting rows here would resurrect a feature that no longer
-- exists and could never decrypt the old key material anyway.
DELETE FROM settings WHERE `key` LIKE 'mimo.%';
