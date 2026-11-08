-- Backfill weekly_pay.from_date on legacy transaction codes.
--
-- All current writers persist FromDate (export_service.go, ninepay_service.go,
-- onepay_exporter.go); only rows created before that convention lack it, which
-- forces the bank-transfer-histories cycle resolution to fall back to a bulk
-- timesheet-date fetch (GetTimesheetDatesByIDs) on every request. After this
-- backfill the persisted value serves all resolvable codes and the fallback
-- path is dead for them.
--
-- from_date := MIN(timesheet date) over the code's own weekly_pay timesheet
-- IDs — exactly the value the runtime fallback computes
-- (resolveWeeklyHistoryCycle). Stored as RFC3339 UTC-midnight so it parses
-- into CyclePayData.FromDate *time.Time.
--
-- to_date is intentionally NOT backfilled: no consumer reads it in this path.
-- Codes whose timesheets are all deleted stay NULL — the runtime fallback
-- still serves them.
--
-- Idempotent: the WHERE clause skips rows that already have from_date.

UPDATE transaction_codes tc
JOIN (
    SELECT tc2.id, MIN(CAST(ts.date AS DATE)) AS min_date
    FROM transaction_codes tc2
    JOIN JSON_TABLE(
        tc2.data, '$.weekly_pay.timesheet_ids[*]'
        COLUMNS (ts_id BIGINT PATH '$')
    ) jt
    JOIN timesheets ts ON ts.id = jt.ts_id
    WHERE JSON_EXTRACT(tc2.data, '$.weekly_pay') IS NOT NULL
      AND JSON_EXTRACT(tc2.data, '$.weekly_pay.from_date') IS NULL
    GROUP BY tc2.id
) m ON m.id = tc.id
SET tc.data = JSON_SET(
    tc.data, '$.weekly_pay.from_date',
    CONCAT(DATE_FORMAT(m.min_date, '%Y-%m-%d'), 'T00:00:00Z')
);
