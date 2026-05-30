-- Optimize analytics query performance
-- Migration: 037_optimize_analytics_query_indexes.up.sql
--
-- Targets the following query patterns in timesheet_analytics_repository.go:
--   1. Partner employee list queries (created_by + date range)
--   2. CTE payment lookups (employee_id + payment_status + payment_date)
--   3. paid_at-based weekly aggregations
--   4. project + date profit/financial aggregations

-- (1) Composite index for partner queries filtering by created_by + date.
--     Covers GetPartnerActiveEmployees, GetPartnerDroppedEmployees,
--     GetPartnerPaidEmployees, GetPartnerWeeklyPaidStats, GetPartnerMonthlyPaidStats,
--     GetPartnerEmployeeStats, GetPartnerTopPaidEmployees.
--     Leading column is created_by so the partner-scoped WHERE clause hits the index
--     immediately; date narrows the range scan.
CREATE INDEX idx_timesheets_created_by_date_status
  ON timesheets (created_by, date, payment_status, deleted_at);

-- (2) Index for the CTE inner subquery that finds MAX(payment_date) per employee
--     and the outer join on (employee_id, payment_date).
--     Also covers GetPartnerEmployeeStats paid count sub-query.
--     Column order: employee_id first (equality in JOIN), then payment_status
--     (equality filter), then payment_date (MAX aggregate / range).
CREATE INDEX idx_timesheets_employee_payment_date
  ON timesheets (employee_id, payment_status, payment_date, deleted_at);

-- (3) Index for paid_at-based queries: GetWeeklyPayAggregated, GetPaidSalaryByEmployee.
--     payment_status equality first, then paid_at range scan.
CREATE INDEX idx_timesheets_payment_status_paid_at
  ON timesheets (payment_status, paid_at, deleted_at);

-- (4) Index for project + date aggregations: GetWeeklyProfitByProject,
--     GetMonthlyFinancials, GetProfitSummaryFromTimesheets, GetAllTimeProfitByProject.
--     project_id equality + date range, deleted_at for soft-delete filter.
CREATE INDEX idx_timesheets_project_id_date_deleted
  ON timesheets (project_id, date, deleted_at);
