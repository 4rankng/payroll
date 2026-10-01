# Runbook: periods funded by both the workbook and self check-in

A salary period's advanceable amount may come from **one** pipeline only: the
admin workbook upload (bảng công) or self check-in/out earnings. Both write into
the same `advance_payments` row, so a row carrying both signatures would let an
employee withdraw against the sum of two independent sources.

| Signature | Field | Written by |
|---|---|---|
| Workbook | `last_applied_asset_id IS NOT NULL` | `BatchUpsert` (import) |
| Self check-in | `salary > 0` | `AccumulateSalary` (check-out credit) |

Code prevents new conflicts at three points: the workbook import skips check-in
periods, enabling self check-in refuses a start month the workbook already
funded, and both request flows refuse a period carrying both signatures. This
runbook only concerns rows that predate that enforcement.

## Finding affected periods

Read-only. Run against `payroll_db`:

```sql
SELECT
    ap.project_id,
    p.code              AS project,
    ap.employee_id,
    e.fullname,
    ap.for_month,
    ap.salary,                              -- 100% earned by check-in/out
    ap.max_adv_amount,                      -- what the employee may actually request
    ap.last_applied_asset_id,               -- the workbook upload that also wrote this row
    (SELECT COUNT(*) FROM advance_payment_requests r
      WHERE r.adv_pay_id = ap.id)           AS requests_made,
    (SELECT IFNULL(SUM(r.request_amount), 0) FROM advance_payment_requests r
      WHERE r.adv_pay_id = ap.id
        AND r.status = 'completed')         AS paid_out
FROM advance_payments ap
JOIN employees e ON e.id = ap.employee_id
LEFT JOIN projects p ON p.id = ap.project_id
WHERE ap.salary > 0
  AND ap.last_applied_asset_id IS NOT NULL
ORDER BY ap.for_month DESC, ap.employee_id;
```

Exposure summary:

```sql
SELECT
    COUNT(*)                     AS mixed_rows,
    COUNT(DISTINCT ap.employee_id) AS employees,
    MIN(ap.for_month)            AS from_month,
    MAX(ap.for_month)            AS to_month,
    SUM(ap.max_adv_amount)       AS inflated_max_adv_total
FROM advance_payments ap
WHERE ap.salary > 0
  AND ap.last_applied_asset_id IS NOT NULL;
```

`paid_out > 0` means money already moved against the inflated cap — resolve
those by hand, never by script.

## Resolving a row

There is no automated repair: both amounts are money the employee may have been
paid against, so each case is the admin's call. The two defensible outcomes:

- **The check-in side wins** (matches what the import enforces going forward):
  drop the workbook contribution and keep the check-in-earned cap
  `floor(salary * advance_percent / 100)`.
- **The workbook wins**: drop the check-in-earned portion and keep
  `max_adv_amount` as the workbook set it.

Before touching a row, check its requests: anything already `completed` has
been paid and must not be clawed back by the correction.
