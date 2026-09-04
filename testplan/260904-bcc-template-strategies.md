# Testplan — BCC template strategies (T09 BUMHAN "Mẫu mức lương")

Source of truth: real file `/Users/dev/Downloads/BCC BUMHAN T09.2026 Mẫu mức lương.xlsx`
(sheets: `BCC` = date-row layout named like the legacy sheet, `Truy lĩnh`, `Công TrT T5` = 2018-era artifacts).

## Root cause (proven by probe)

`DetectFormat` gives a literal `BCC` sheet name the top tiebreak → `FormatLegacy`
→ legacy parser yields 14 employees / **0 entries / 0 rates** → import fails with
"không có dữ liệu hợp lệ để tạo bảng chấm công". The sheet is a date-row layout.

## Strategy routing (ParseBCCData cascade)

| # | Scenario | Expect |
|---|----------|--------|
| 1 | Sheet named `BCC`, date-row layout (T09) | Legacy parse unusable → date-row strategy parses; 17 employees, in-month entries, format reported = date-row |
| 2 | True legacy file named `BCC` | Legacy parse usable (rates>0) → accepted; date-row never runs; behavior byte-identical to today |
| 3 | Legacy parse errors | Router falls through to date-row before failing |
| 4 | No strategy matches | Vietnamese router error (không nhận diện được...) |
| 5 | Non-"BCC" sheets (Truy lĩnh / Công TrT T5 artifacts) | Never matched/routed |

## Vị trí (Position) column

| # | Scenario | Expect |
|---|----------|--------|
| 6 | Header `Vị trí` at col CX (31-day month) | Parsed per employee into `BCCEmployeeData.Position` |
| 7 | Header `Vị trí` at earlier col (30/28-day month → CW/BV…) | Same — located by header name, never by letter |
| 8 | Header written without diacritics (`Vi tri`) | Still matched |
| 9 | No `Vị trí` column | `Position` = "" ; assignment falls back to deduce/phổ thông |
| 10 | Date-row employee with Position, no existing assignment | Auto-assignment Position = file value (not phổ thông) |
| 11 | Employee already assigned | Existing assignment untouched (no position rewrite) |

## Regression gates

- `go build ./...`; full `excel` + `services` package tests green (existing
  date-row, label-rate, format-detector tests unchanged in behavior).
- Real-file guarded tests (os.Stat skip): T08-style file + T09 file both parse
  via their strategies; T09 shows `LE`/`OT LE` labels parsed (holiday codes —
  rate resolution depends on project payrate config having ngày lễ leaves,
  which is data, not code).
- Multi-position / weekly-payment / weekly-BCC flows untouched.

## Live import gate (local dev, project 78)

| # | Scenario | Expect |
|---|----------|--------|
| 12 | POST `/api/v1/timesheets/partner-import` (admin, project 78, for_month=2026-09) with the real T09 file against the rebuilt backend | status `completed`; 17 employees found; timesheets created for Sep days; no "nhân viên không tìm thấy" errors |
| 13 | Same upload re-run | idempotent: re-upload = safe upsert, no duplicate employees/assignments |
