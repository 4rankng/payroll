ALTER TABLE timesheet_import_jobs
    ADD COLUMN include_flexible_employees BOOLEAN NOT NULL DEFAULT FALSE
    AFTER uploader_role;
