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

## What was actually applied (migration 116)

The obvious repair was wrong, and the dry run is what caught it. The two
pipelines **overwrite** each other rather than stacking: `BatchUpsert` sets
`max_adv_amount` to the workbook amount, and `AccumulateSalary` replaces it with
`floor((salary + earning) * percent / 100)`. A period that received both
therefore already holds a **purely check-in-derived cap**.

Verified in production: all 58 such rows had `max_adv_amount = salary` (credited
while the advance percentage was 100) and none equalled
`floor(salary * 70 / 100)`, while every check-in-only row created since the
percentage became configurable follows 70%.

So there was no "workbook part" left in the numbers to remove — the check-in
side had already won on all of them. Migration 116 clears only the stale
`last_applied_asset_id`, which was making the request backstop refuse clean
check-in periods. **No amount was rewritten.** Verification after applying:
`remaining_mixed_rows = 0`, `unexpected_caps = 0`, cap and request totals
unchanged.

## Open decision: the 100% rows

Those 58 rows were credited when the advanceable percentage was 100; it is 70
now. Recomputing them would cut the cap of 23 employees by 30% — and 25 of the
rows already carry `completed` requests, several of which drew MORE than 70%
would allow. That is a policy decision about money already earned and paid,
not a data defect, so it is not automated. Two ways to act:

- Leave it. The stored cap reflects the percentage that was in force when the
  money was earned.
- Reapply today's percentage to the live check-in rows with
  `AdvancePaymentRepository.RecomputeActiveCheckInMaxAdvance` (already invoked
  when the admin saves the percentage setting). Review `completed` vs
  `max_adv_amount` first: any employee already paid above the new cap is now
  visibly overdrawn, which is accurate but not reversible by script.

