-- Restore the original 001 index and drop the covering replacement.

CREATE INDEX idx_timesheets_payment_status_date
  ON timesheets (payment_status, date);

DROP INDEX idx_timesheets_paid_date_cover ON timesheets;
