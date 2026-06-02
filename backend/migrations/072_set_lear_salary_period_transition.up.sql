-- Restore Lear Cát Hải's previous salary period configuration so the transition
-- safety net can recover its orphaned April timesheets.
-- Lear changed from (from=15, to=14) → (from=0, to=0) on 2026-05-25. The May 26
-- sao ke was generated after the change, so the April 22–27 timesheets were never
-- included in any export window. Setting last_salary_from/last_salary_to marks the
-- project as "in transition", which causes the safety net to use an unconstrained
-- date query and pick up all unsettled paid timesheets regardless of date.
UPDATE projects
SET last_salary_from = 15,
    last_salary_to   = 14
WHERE id = 19;
