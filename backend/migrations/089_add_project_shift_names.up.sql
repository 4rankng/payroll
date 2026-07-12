-- 089: admin-named shifts for check-in-enabled projects
-- Maps a payrate shift time-range to a display name (e.g. "09:00-18:00" -> "Ca làm").
-- Payrate remains the source of truth for times/pricing; this column only carries labels.
-- Nullable: existing projects have no shift names and behave as before.
ALTER TABLE projects ADD COLUMN shift_names JSON NULL;
