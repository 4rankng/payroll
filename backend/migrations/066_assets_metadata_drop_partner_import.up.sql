-- Add metadata JSON column for storing import results (BCC import, etc.)
ALTER TABLE assets ADD COLUMN metadata JSON NULL;

-- Drop the now-redundant partner_import_files table
DROP TABLE IF EXISTS partner_import_files;
