# Testplan — BCC date-row (BUMHAN M1) import

Scope (user-directed): update BUMHAN payrate config + parsing logic ONLY. New template parses the real partner file; **old-template behavior unchanged**. New-template BCC sheet carries employee info → employee upsert via the existing STK machinery.

## Prereqs
- Backend :8080 (air), frontend :3000 (vite), local DB (project 56 BUMHAN).
- Real file: ~/Downloads/BCC BUMHAN T09.2026 thợ phụ chốt ứng lương - Copy.xlsx (27 employees, period 21/08→23/09).

## Matrix — code (run first)

| # | Check | Method | Expected | Result |
|---|---|---|---|---|
| 1 | Format detection routes M1-style sheet | unit `TestDetectFormat_DateRowBCC` | FormatDateRow, sheets=[M1] | PASS |
| 2 | Date-row parse: dates/labels/hours/FullDate | unit `TestParseDateRowBCCFile` | (8/21 NT 9), (8/21 OT 1), (8/23 CN 8), FullDate set | PASS |
| 3 | Real-file parse | unit `TestParseDateRowBCCFile_RealFile` (new) | 27 employees, 301 entries, first=031082006094 with Aug 22 T7=9, OT T7=1, Aug 23 CN=8 | PASS |
| 4 | labelRateTarget label→bucket | unit `TestLabelRateTarget` | prefers ngày thường; zero-rate/unknown reject | PASS |
| 5 | Excel package regression | `go test ./internal/app/services/excel/` | PASS | PASS |
| 6 | BCC services regression | `go test ./internal/app/services/ -run 'BCC\|LabelRate\|ShiftLabel'` | PASS | PASS |
| 7 | Whole backend builds | `go build ./...` | PASS | PASS |
| 8 | Known pre-existing failures excluded | `go test ./internal/app/services/` | only wallet_demand_forecast day-20/21 drift fails (feedback_period_threshold_day20; files untouched) | PASS |

## Matrix — live (after code green)

| # | Check | Method | Expected |
|---|---|---|---|
| L1 | Backend runs new code | air rebuilt; import 510 error message came from the new parser | PASS — import 510's error came from the new parser (route live) |
| L2 | Payrate config carries all file codes | POST /api/v1/projects/56/payrate effective 2026-09-01, rates: phổ thông/ngày thường += NT 66666, OT 100000, T7 100000, OT T7 100000, CN 100000, OT CN 100000 (rest unchanged) | PASS with caveat — row 93 (8-22→open) created via API carries all 6 codes; row 68 (8-1→8-31, no codes) NOT editable/deletable (closed row + used-by-timesheets rules) → 68/93 overlap on 8-22..31 is a local-only artifact; month-start lookup resolves 68 for Aug |
| L3 | Live parse | upload real file: POST /api/v1/timesheets/partner-import, project_id=56, for_month=2026-09, fresh Idempotency-Key | PASS — import 513: completed, total 26, created 288, skipped 0, errors 0 |
| L4 | Correct dates imported | DB: timesheets for Sep 1–30 only | ADAPTED — file holds only Aug 22–28 data (Sep columns empty) so the correct month was 2026-08; DB shows paytypes phổ thông.ngày thường.{nt,ot,t7,ot t7,cn} for 8-22..8-28; no out-of-month rows; NT rate 66,666 = file's 600,000/9h |
| L5 | Hours correct | DB sample vs source file (Khổng Văn Tiến: Sep 1 NT=9, OT=1... per M1 data) | PASS — Aug 22: 22×T7 9h + 22×OT T7 1h; Aug 23: 20×CN 8h (matches file) |
| L6 | Employee upsert works | DB: employees/project_employees for file CCCDs | PASS — 26 file employees (27th was the filtered Cộng totals row) all assigned; new ones (e.g. Sìn Văn Đài 002086006112, Nguyễn Thế Anh 031092000816) auto-created with bank account + bank resolved by name |
| L7 | Old-template behavior unchanged | code path check: legacy/multi-position/weekly parsers + STK flow untouched; excel pkg suite green | PASS |

## Local-test artifacts (cleaned)
- Payrate row 68 patched with codes for the import test, then restored to its original JSON.
- frankng (local admin) password temporarily reset, original hash restored.
- Row 93 (codes config, 8-22→open) kept — it is the go-forward config the admin would create anyway.
- Import assets 510–513 and created timesheets remain (local dev DB, same residue class as api-test runs).

## Negative cases (unit-covered)
- Label missing from config → row error "không tìm thấy mức lương cho ca X" (TestLabelRateTarget rejects unknown/zero-rate).
- Mid-month: Aug days of the T09 file are dropped by the in-month filter (FullDate path unit-verified).

## Environment notes
- Login: POST /api/v1/auth/login {username, password} (local frankng, temp password during test, original hash restored after).
- Import processing is async (asynq); poll GET /timesheets/partner-import/:id.
