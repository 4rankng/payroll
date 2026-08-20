-- Deferred self-checkin activation: enabling check-in for an employee takes
-- effect on day 1 of the NEXT month (enable 29 Aug -> active 1 Sep).
-- Mirrors the pending_payment_schedule / schedule_effective_from pair.

ALTER TABLE project_employees
  ADD COLUMN pending_check_in_enabled TINYINT(1) NULL DEFAULT NULL COMMENT 'Pending check-in enable awaiting activation (NULL = no pending change)',
  ADD COLUMN check_in_effective_from  DATE       NULL DEFAULT NULL COMMENT 'Date when the pending check-in enable becomes effective (day 1 of next month)';
