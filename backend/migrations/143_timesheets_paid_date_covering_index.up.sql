-- Covering index for partner paid aggregates.
--
-- Serves GetPartnerWeeklyPaidStats / GetPartnerMonthlyPaidStats /
-- GetPartnerTopPaidEmployees as an index-only scan:
--   equality: payment_status = 'paid', deleted_at IS NULL
--   range:    date >= ? AND date < ?
--   covering: employee_id, paid_amount
--
-- Replaces idx_timesheets_payment_status_date (001, (payment_status, date)),
-- which this index strictly prefixes. The weekly-stats CTE keeps using
-- idx_timesheets_employee_date_deleted (014) and is unaffected.

CREATE INDEX idx_timesheets_paid_date_cover
  ON timesheets (payment_status, deleted_at, date, employee_id, paid_amount);

DROP INDEX idx_timesheets_payment_status_date ON timesheets;
