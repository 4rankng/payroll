-- BCC import "add on top" mode: when the uploader ticks the supplement
-- checkbox, approved/paid rows matching an uploaded cell are not skipped —
-- the import creates a NEW pending supplement row (same paytype, file hours)
-- so the existing finalized row keeps its paid amount untouched.
ALTER TABLE timesheet_import_jobs
    ADD COLUMN add_on_top BOOLEAN NOT NULL DEFAULT false AFTER include_flexible_employees;
