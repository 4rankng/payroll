-- Enforce "at most one OPEN attendance per employee/day" declaratively. Restores the
-- guard migration 078 dropped (uq_employee_date) WITHOUT breaking re-check-in after a
-- confirmed-no-salary checkout (which legitimately needs a 2nd row for the same
-- employee/date).
--
-- open_key is non-NULL only for OPEN rows (no checkout AND not rejected). MySQL allows
-- multiple NULLs in a unique index, so any number of closed/rejected rows coexist,
-- while at most one OPEN row per employee is enforced. The STORED generated column
-- auto-recomputes on INSERT/UPDATE, so CheckOut (sets check_out_time) and auto-reject
-- (sets salary_reject_reason) free the slot with no application code.
--
-- Pre-flight: any pre-existing duplicate OPEN rows will make ADD UNIQUE fail — run the
-- pre-flight SELECT in plan.md first and reconcile (the auto-reject sweeper closes most).
ALTER TABLE attendances
  ADD COLUMN open_key DATE GENERATED ALWAYS AS
    (CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NULL THEN date ELSE NULL END) STORED,
  ADD UNIQUE KEY uq_attendances_employee_open (employee_id, open_key);
