# Assignment start_date defaults blocked weekly BCC imports (2026-09-17)

## What happened

Partner uploaded `LUONGTUAN_CBS_KY2-09_v1.xlsx` (weekly BCC, project CBS/65,
entries Sept 10–14). Failed 8× with a generic error: *"lỗi tạo bảng chấm công:
không thể thay thế an toàn: 9 dòng không hợp lệ"* — no employee, no date, no
reason. The partner retried blindly for ~2 hours and eventually self-recovered
by backdating the assignment start in the UI.

## Root causes (two independent defects)

1. **Assignment start defaulted to "today"** — backend
   (`project_employee_batch.go`) defaulted a missing `start_date` to today, and
   the frontend pre-filled today's date and sent it explicitly. BCC files
   always cover past dates, so `ValidateEmployeeAssignment` rejected every row
   dated before the assignment start ("thời gian phân công bắt đầu sau ngày
   chấm công"). The employee had been created 4 minutes before the first
   upload with `start_date = 2026-09-17`.
2. **Weekly import paths hid per-row failures** — when the safe-replacement
   transaction rolled back, the weekly formats returned one generic line while
   the legacy format already surfaced per-row errors (employee + date + safe
   reason). The partner could not self-diagnose.

## Fixes (branch investigate-prod-upload-fail)

1. Weekly BCC + weekly payment paths surface per-row errors like legacy.
2. `SuggestAssignmentStart` rule: omitted `start_date` = day after the
   employee's last recorded timesheet (any project), else 1st of current
   month — applied at every assignment-creation path (UI, employee import,
   all 5 BCC auto-create sites, FlexPay).
3. Self-healing imports: `backdateAssignmentsForImport` aligns an existing
   assignment's start to the file's earliest entry before validation
   (compare-and-set, after-commit cache invalidation per ADR-007).
4. Frontend no longer pre-fills/sends today.

## Lessons

- **Default values are business decisions.** "Today" is almost never the
  right default for a payroll period field; defaults must match the data the
  feature actually processes (BCC files cover the current month's past days).
- **An error message that cannot be acted on is a defect.** The generic
  "N dòng không hợp lệ" cost a partner 2 hours; the same event already had
  per-row detail available on another code path. When one path surfaces
  detail, all sibling paths must.
- **Cache invalidation ordering matters after silent data edits** — a
  backdated assignment row without `assignment:{p}:{e}` cache deletion keeps
  failing validation until TTL expiry (ADR-007).
- Prod diagnosis pattern that worked: audit table (`partner_bcc_import_failed`)
  → asset metadata (`error_detail`) → `docker logs | grep "validation failed"`
  gives employee/date/reason triple directly.

## Latent findings (not fixed here)

- 1,258 historical timesheet rows predate their assignment start across
  projects 7/10/19/15/17/11/67; recent case: project 67, Phạm Trung Hiếu,
  22 rows Aug 18–22 vs start 08-24. Re-importing those periods will now
  self-heal instead of failing.
