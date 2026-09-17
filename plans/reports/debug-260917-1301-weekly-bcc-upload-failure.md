# Prod investigation: BCC lương tuần upload failure (project CBS)

**Date:** 2026-09-17 · **Server:** tingting.vip · **Status:** RESOLVED (partner self-fixed); root cause confirmed; code fix + feature plan follow-ups

## Resolution (12:42–12:43)

Partner backdated assignment 1723 `start_date` 09-17 → 09-09 at 12:42:52 (UI). Re-upload at 12:43:20 succeeded: **status=completed, 110 rows created, 0 errors** (asset 531). No ops intervention needed.

## Other employees with similar issues (checked 2026-09-17 ~13:45)

- **Today's CBS failures: only Nguyễn Văn Linh (1493).** All 9 failed rows in every attempt belong to him. Now resolved.
- **Latent, project 65:** Nguyễn Văn Sơn (1018, start 09-14, created 11:37) and Võ Thị Thu Hà (1382, start 09-08) — if upcoming files contain their rows before those dates, same failure. No rows failed yet.
- **Historical, 1,258 timesheet rows predate assignment start** across projects 7 (676), 10 (392), 19 (89), 15 (39), 17 (37), 67 (22), 11 (3). Only project 67 is recent: **Phạm Trung Hiếu (1354), 22 rows Aug 18–22 vs start 08-24** — re-importing that week will fail the same way (already had a test-case-worthy precedent: `TestImportErrorsFromBulkFailuresMapsAssignmentError` uses exactly 2026-08-24).

## Follow-ups on branch investigate-prod-upload-fail

1. **Code fix (done, uncommitted):** weekly BCC + weekly payment paths now surface per-row errors (`bcc_import_weekly_bcc.go`, `bcc_import_weekly_payment.go` — mirrors legacy pattern). Build + unit tests green.
2. **Feature plan (written):** `plans/260917-1351-assignment-smart-start-date/` — smart assignment start defaults + self-healing BCC import backdate.


## Symptom

Partner `hoalt@vfic.com.vn` (user 1193) failed to upload `LUONGTUAN_CBS_KY2-09_v1.xlsx` to project **CBS (65)** — 8 attempts 10:06–11:55, plus `LUONGTUAN_CBS_KY1-09_v2.xlsx` at 11:33. All attempts end `partner_bcc_import_failed`. User-facing error (asset 520–530 metadata):

> "lỗi tạo bảng chấm công: không thể thay thế an toàn: 9 dòng không hợp lệ"

No employee name, no date, no reason → user retried blindly.

## Root cause chain (verified from prod logs + DB)

1. **10:02:15** — partner manually created employee **Nguyễn Văn Linh (id 1493, CCCD 031099015066)** and assignment (id 1723, project 65) with `start_date = 2026-09-17` (= upload day; likely UI default).
2. File contains Linh's shifts on **2026-09-10 → 09-14** (HC, OT150, OT200) — all before assignment start.
3. `ValidateEmployeeAssignment` (`timesheet_validation_assignment.go:23`) rejects each row: *"thời gian phân công nhân viên bắt đầu sau ngày chấm công"*. Exactly **9 rows, all employee_id=1493** (verified in docker logs, e.g. index 58–62, 184–186, 188, 250).
4. `applyTimesheetReplacementPrepared` (`bcc_import_replacement.go:161`) rolls back the **entire** import when any row fails → 101 valid rows also discarded (intentional "safe replacement" semantics).
5. Weekly-BCC path (`bcc_import_weekly_bcc.go:130-133`) **discards `result.FailedEntries`** on the error path → generic one-line error, no row detail.

Evidence: `docker logs payroll-backend --since 72h | grep "validation failed"`; `SELECT metadata FROM assets WHERE id IN (520,525,530)`; `SELECT * FROM project_employees WHERE employee_id=1493` (start_date 2026-09-17).

## Contributing product bug

Legacy path (`bcc_import_process.go:284-293`) and multi-position path (`bcc_import_multi_position.go:123-133`) DO surface per-row failures via `failWithImportErrors` + `importErrorsFromBulkFailures`. The two weekly paths don't:

- `bcc_import_weekly_bcc.go:130-133` (WBCC — this incident)
- `bcc_import_weekly_payment.go:136-138` (weekly payment)

Had detail been surfaced, `safeBulkFailureReason` would have shown *"Nhân viên chưa được phân công vào dự án cho ngày chấm công"* for Linh's rows and the partner could likely have self-corrected.

## Remedies (status)

1. **Data unblock — DONE by partner** (backdate 1723 → 09-09 at 12:42; upload succeeded 12:43). Latent cases above remain.
2. **Code fix — DONE on this branch** (see Follow-ups).
3. **Systemic prevention — PLANNED** (`plans/260917-1351-assignment-smart-start-date/`): smart start defaults + self-healing import.

## Unresolved

- Optional: consider whether all-or-nothing rollback should instead import valid rows and report failures (behavior change — needs product decision, not done here).
- Latent assignments (1018 start 09-14, 1382 start 09-08 in project 65; 1354 in project 67) — fix on demand or via the planned feature.

