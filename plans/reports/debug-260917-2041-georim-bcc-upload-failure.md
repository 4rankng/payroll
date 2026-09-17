# Prod investigation: GEORIM weekly BCC upload failure (project Georim/77)

**Date:** 2026-09-17 20:36 · **Data:** synced prod → local MySQL (`payroll-mysql`, payroll_db) · **Status:** ROOT CAUSE CONFIRMED, FIX IMPLEMENTED (local main, not yet deployed)

## Symptom

Partner uploaded `GEORIM-luongtuan2TH9.xlsx` to project **Georim (77)**, month 2026-09, 4 attempts 19:29–19:35 (users 1194, 976). All failed: 0 created, 0 skipped, 1 error:

> "Không thể xử lý dòng dữ liệu này (ngày 08/09/2026)" — generic fallback; no actionable info.

## Root cause

The workbook's second sheet is named **`BCC-OT150 `** (trailing space — stray keystroke when the partner created the sheet):

1. `ExtractShiftType` (`excel/format_detector.go:89`) returns `"OT150 "` untrimmed.
2. `shiftInConfig` (`bcc_import_weekly_rates.go:50`) compares `ToLower` only → `"ot150 "` ≠ config leaf `"ot150"` → false.
3. `weeklyBCCConfigError` (`bcc_import_weekly_bcc.go:187`) fails the **whole import** (all-or-nothing): `ca làm "OT150 " không có trong cấu hình lương cho ngày 2026-09-08. Các ca làm khả dụng: OT30, OT270, OT390, HC, OT300, OT200, OT150` — note OT150 IS configured; only the space broke the match.
4. `import-errors.ts` has no pattern for "không có trong cấu hình lương" → frontend masks it with the generic fallback. The date 08/09 in the UI is just the first worked date iterated (config check fails identically for all dates).

Contrast: `BCC-HC` / `BCC-OT200` sheet names were clean → they parsed fine; the single bad sheet name blocked all 3 sheets' ~155 entries.

## Evidence

- `SELECT metadata FROM assets WHERE id IN (542..545)` — 4 identical failures, raw `error_detail` carries `"OT150 "` (with space) and an available list that includes `OT150`.
- `openpyxl` on the uploaded file: sheet titles `'BCC-HC'`, `'BCC-OT150 '`, `'BCC-OT200'`.
- Prior week's file `GEORIM-luongtuan1TH9.xlsx` (asset 510) succeeded — clean sheet names.

## Fix (implemented)

`ExtractShiftType` now `strings.TrimSpace`s the extracted shift (`excel/format_detector.go`), so `"BCC-OT150 "` → `"OT150"`; the same clean value then flows into `shiftInConfig`, `buildShiftRatesForShift`, and the persisted `HourType`. TDD: 4 new cases in `TestExtractShiftType` (`weekly_bcc_parser_test.go`): trailing space ×2, space after dash, whitespace-only suffix.

## Verification

- `go test ./internal/app/services/excel/` green (red→green on new cases).
- `go build ./...`, `go vet` on touched packages, targeted `go test ./internal/app/services/ -run 'WeeklyBCC|BCCImport|...'` green.
- **Real prod file replay:** `go run ./cmd/bccinspect /Users/dev/Downloads/GEORIM-luongtuan2TH9.xlsx` → `format=weekly-bcc`, `OK: sheet BCC-HC / BCC-OT150 / BCC-OT200 — 5 employees each`. Same file now parses clean.

## Deployment

Fix is uncommitted on local `main`. `make deploy` required; afterwards the partner re-uploads the **same file** — no file correction needed.

## Unresolved / follow-ups

1. Deploy + ask partner to re-upload.
2. Optional: frontend `import-errors.ts` has no mapping for "không có trong cấu hình lương" — a genuinely unconfigured shift (e.g. "OT400") would again show the generic fallback. Suggest mapping → "Ca làm chưa được cấu hình trong bảng lương của dự án".
3. Optional: defensively trim inside `shiftInConfig`/`buildShiftRatesForShift` too (currently safe: `ExtractShiftType` is the sole `ShiftType` producer).
