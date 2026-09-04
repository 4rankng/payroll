# Test Plan — BCC Zero-Cell Deletion (2026-09-04)

Feature: explicit `0` in a BCC cell deletes the matching **chờ duyệt** (pending_approval)
timesheet row; blank cells change nothing; approved/paid rows immune.

Environment: LOCAL DEV ONLY (`payroll_db` @ payroll-mysql:3306, backend via air :8080).
Real file: `/Users/dev/Downloads/BCC BUMHAN Thang8.xlsx` (copies modified, stored in /tmp).
Unit tests already cover parsers + plan; this matrix covers the live end-to-end path.

## Pre-conditions

- [ ] Backend running on :8080, migrations current.
- [ ] BUMHAN project identified (`projects.name LIKE '%BUMHAN%'`), note its id P.
- [ ] Admin login token obtained from local dev API.

## Matrix

| # | Case | Setup | Action | Expected (DB-verified) |
|---|------|-------|--------|------------------------|
| 0 | Baseline import | original file | upload for_month=2026-08, project P | rows created, status=pending_approval (chờ duyệt); record count N0 + one sample key (emp E, date D, paytype T) |
| 1 | Non-zero overwrite (regression) | copy: change E/D/T hours to different positive value | upload | old row hard-deleted, new row with new hours, still pending |
| 2 | Zero deletes pending row | copy: set E/D/T cell to 0 | upload | row (E,D,T) GONE (hard-deleted), not recreated; other rows intact |
| 3 | Blank leaves row | copy: blank E/D/T cell | upload | row (E,D,T) still present unchanged |
| 4 | All-zero employee row | copy: zero every day-cell for employee E | upload | ALL pending rows of E in 2026-08 gone; no error |
| 5 | All-zero whole file | copy: zero every data cell | upload | completes (no "không có dữ liệu hợp lệ" error); pending rows for project/month reduced to 0 |
| 6 | Zero on approved row | approve one pending row (status=approved via SQL on LOCAL db) then upload 0 for its key | upload | row still exists, status approved; skipped count includes it |
| 7 | Idempotency | re-upload the case-2 copy | upload | same final state, no errors, no duplicates |

## Verification method

Direct SQL before/after each upload:
```sql
SELECT id, employee_id, date, pay_type, hours_worked, status, payment_status
FROM timesheets WHERE project_id = P AND deleted_at IS NULL
ORDER BY employee_id, date, pay_type;
```
Compare row sets step to step. Assert via row counts + sample keys, not API responses alone.

## Out of scope

- Prod (forbidden), demo, frontend UI flows (API-level upload suffices), weekly formats live-test
  (unit-covered; live file at hand is date-row).

## Results (2026-09-04, local dev, backend @ 211c8a3e + this change)

Environment notes: project 56 local payrate id 68 (in force 21–28/08) lacked the NT/OT/T7 label
leaves prod uses (id 80 has them) — patched locally to mirror prod. Substrate rows carried
`transaction_id=237` (restore artifact) — cleared locally to model genuinely-pending rows; the
first upload round incidentally verified transaction-linked protection works.

Fixture = anonymized single-employee copy of the real file (code remapped to assignment
031082006094 / employee 354), bank/mức-lương columns cleared, junk sheets removed. Committed at
`backend/tests/fixtures/bcc-zero/bumhan-single-employee.xlsx` + parser regression test
`bcc_zero_cell_fixture_test.go`. Working copies: /tmp/bcc-zero-test/.

| # | Case | Result | Evidence (assets 487–494 + timesheets table) |
|---|------|--------|----------------------------------------------|
| 0 | Baseline import | PASS | asset 489: routing picked date-row parser; blanks on 21/08 left rows 59675/59676 untouched |
| 1 | Non-zero overwrite (regression) | PASS | 22/08 9h→8h: 61804/61805 hard-deleted, recreated 62841 "ngày nghỉ.t7" 8.00; 24–25/08 replaced with fresh ids |
| 2 | Zero deletes pending row | PASS | 23/08 (CN=0, OT CN=0): 61806 gone, nothing recreated (asset 489). 21/08 (NT=0): 59675+59676 both gone (asset 490) — day-level key, both paytypes |
| 3 | Blank leaves row | PASS | asset 492 (blank-21): 21/08 row 62851 still present 9.00 pending |
| 4 | All-zero employee row | PASS | asset 493: 0 pending rows left in 21–25/08 for employee 354 |
| 5 | All-zero file no error | PASS | assets 493/494 status=completed, errors 0 — the old "không có dữ liệu hợp lệ" failure is gone |
| 6 | Zero on approved row | PASS | 26–28/08 approved+paid rows survived every upload including zeros (skipped as protected) |
| 7 | Idempotency | PASS | asset 494 = re-upload of zero-all: identical final state, 0 errors |
| — | Transaction-linked protection | PASS | pre-cleanup upload (asset 488): rows with transaction_id=237 skipped, never deleted |

Unit: `go test ./...` full backend suite green (exit 0).
Integration: `make api-test` (backend Makefile): 296 total, 273 passed, 0 failed, 23 skipped
(pre-existing env-gated skips).

