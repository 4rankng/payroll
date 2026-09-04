# BCC BUMHAN template — parse failure diagnosis & docx guide spec

Status: diagnosis only, no code changed. Date: 2026-09-04.
Input: `~/Downloads/BCC BUMHAN T09.2026 thợ phụ chốt ứng lương - Copy.xlsx`
Contract: `backend/internal/app/services/excel/bcc_parser.go`, `format_detector.go`, `backend/internal/app/services/bcc_import_process.go`

## 1. Target model (user-confirmed)

- Hour-marking column codes (`NT, OT, T7, OT T7, CN, OT CN`) must exist in the payrate config as shift columns.
- One position (`vị trí`) for all employees. Every day is ngày thường. Rate is determined purely by the label.
- Target config shape: one position → `ngày thường` → leaves `NT, OT, T7, OT T7, CN, OT CN`, one non-zero rate each.
- This shape is NOT achievable via config alone today. It needs one code fix (§6, Option B).

## 2. What the partner sent (verified by cell dump)

Three sheets, three layouts:

- `M1`: the attendance sheet. Dates 21/08 → 23/09 (mid-month cycle). Sheet says "Tháng 8 năm 2026"; filename says T09.2026.
- `Truy lĩnh`: salary payment sheet (Mẫu số 02-LĐTL). Not attendance.
- `Công TrT T5`: second attendance variant, days 1–31, weekday row, totals columns.

M1 layout: R3 title, R4 "Tháng 8", R5 headers `STT | Mã NV | Họ và tên | TK Ngân hàng | Ngân hàng | Mức lương /9h | Ngày thử việc | Ngày nghỉ việc` (no position column), R6 full date serials 21/08→23/09 every 2nd column, R7 weekday numbers (not day-of-month), R8 shift codes `NT|OT|T7|OT T7|CN|OT CN`, R9+ employee rows. Two sub-columns per day. No rate row. Decimal hours.

## 3. What the parser accepts (the contract)

Four fingerprints in `DetectFormat` (`format_detector.go:46-126`), priority: sheet named `BCC` → legacy; `BCC-` prefix → weekly BCC; row 8 headers incl. `Lương 8h` + row 10 codes HC/TCN/NN/TCNN → weekly payment; row 4 `STT+Mã nhân viên+Họ và tên` → multi-position. None of the 3 sheets matches → error "không nhận diện được định dạng file BCC" (format_detector.go:125).

Legacy layout contract (`ParseBCCFile`, bcc_parser.go):

- Sheet name `BCC` (else first sheet) — bcc_parser.go:143-154
- Employee headers scanned only rows 7–8, cols 1–10: STT, CCCD, Mã nhân viên, Họ và tên, Bộ phận — bcc_parser.go:47-108
- Day numbers: row 7 or 8, day-of-month 1–31, propagated across sub-columns — bcc_parser.go:227-240
- Shift codes: first row in rows 9–11 with ≥2 letter cells that are not `T2…CN` — bcc_parser.go:289-323
- Rate row: directly above shift row — bcc_parser.go:343-370
- Data rows: from max(shiftRow, rateRow)+1; stop at empty CCCD+name or "tổng cộng" — bcc_parser.go:374-435
- Fallback risk: headers not found → hardcoded cols 1/2/2/4; on M1 that maps nameCol→col D = "TK Ngân hàng" (bank numbers as employee names)

## 4. Failure chain (each gap verified in source)

1. Format detection fails outright: no sheet named `BCC`; headers at R5 not R4; not weekly-payment fingerprint.
2. Headers at R5 vs scan window rows 7–8 → fallback mis-maps nameCol to col D (TK Ngân hàng).
3. R7 holds weekday numbers, not day-of-month → all dates wrong. Real dates in R6 are never read.
4. Shift codes at R8, below the rows 9–11 scan band → no labels → zero entries → "không có dữ liệu hợp lệ để tạo bảng chấm công" (bcc_import_process.go:470).
5. No rate row, and labels NT/T7 are outside the rateless vocabulary (OT→tăng ca; CB/CN/HC→ca ngày; bcc_import_helpers.go:109-124) → per-entry error "không tìm thấy mức lương cho ca NT (0 VND)" (bcc_import_process.go:423).
6. Rateless path forces day type from the calendar: Sat/Sun → "ngày nghỉ" (bcc_import_weekly_rates.go:219-224). Cannot price T7 vs CN separately; contradicts "every day = ngày thường".
7. Mid-month span impossible: date = (forMonth year, month, dayNum); entries landing outside the month are silently dropped (bcc_import_process.go:399-401). One upload = one calendar month.
8. Employees must pre-exist and be assigned (CCCD match; "nhân viên không tìm thấy trong hệ thống"). No STK sheet in this file → no auto-create.

## 5. Payrate config side

Config flattens as `position.dayType.hourType → VND/giờ`. Two resolution paths today:

- Rated path ("rate is king"): file has a rate row; every label's amount must exactly equal a non-zero config rate (bcc_import_process.go:404-406). Labels free; config column names irrelevant.
- Rateless path: label vocabulary only (OT→tăng ca; CB/CN/HC→ca ngày), day type calendar-forced, buckets must exist by exact normalized name: dayType ∈ {ngày thường, ngày nghỉ} × hourType ∈ {ca ngày, tăng ca} (bcc_import_helpers.go:130-150, zero rate = absent).

Consequence: the Zalo-thread plan to rename config shift columns to NT/OT/T7 breaks the rateless path today (code looks for ca ngày/tăng ca). Renaming alone fixes nothing.

## 6. Options for the fixing agent

**Option A — no code change (stopgap).** Legacy layout + a numeric rate row above the shift row; amounts must exactly match config rates. Works today; rate-keyed, not label-keyed; config rate changes break future uploads.

**Option B — label-keyed rateless resolution (RECOMMENDED; implements §1).** In the rateless branch (bcc_import_process.go:406-418), before the vocabulary/calendar fallback: normalize the label (NFC + diacritic-strip + trim); find a non-zero config bucket whose hour-type leaf equals the label (position-agnostic — the BUMHAN model assumes a single position and the app does NOT check employee position for rate lookup; prefer dayType ngày thường); if matched, use that bucket's dayType + hourType=label as the target (calendar bypassed). Keep the existing fallback order: label-keyed → vocabulary+calendar → row error, so Samsung-SDS-style files keep working. Match semantics: exact leaf equality after normalization; `NT` matches `nt`, not `NT.` (punctuation must match). Tests: NT/T7/OT T7 label-keyed; CB/OT still fall back to calendar; NT with no config leaf → row error naming the label.

## 7. Docx guide spec (for the agent producing the docx)

Audience: partner staff. **The entire docx must be written in Vietnamese** (project rule: all user-facing text is Vietnamese — no English sections). Checklist style; include an annotated example grid and the error table.

Compliant layout (preserves the partner's 2-sub-column style, inside parser row bands):

- Row 7: `STT | Mã nhân viên (CCCD) | Họ và tên | Bộ phận (tùy chọn)` + day-of-month numbers 1–31 repeated across each day's sub-columns.
- Row 8: weekday abbreviations `T2…CN` (optional).
- Row 9: shift codes under each sub-column, must equal payrate shift names.
- Row 10+: employee rows; CCCD as text; hours numeric, decimals OK, blank/0 = no work.
- Sheet name exactly `BCC`; other sheets allowed but ignored.
- One file = one calendar month, containing that month's days 1 → end-of-month (all of them, no days from other months). A 21→20 cycle is stitched from two consecutive monthly files, NOT by splitting one month's days across files (a for_month can only hold one upload scope, so a month's days must stay in that month's single file).
- Codes must exist in payrate config (single position, all under ngày thường, non-zero rates). Until Option B ships, use Option A's rate row instead.
- ≤10MB, .xlsx. Re-upload replaces pending rows; after timesheet approval, re-upload is rejected.

Error table for the guide appendix:

| App message | Cause | Partner action |
|---|---|---|
| "không nhận diện được định dạng file BCC" | Sheet not named BCC | Rename sheet to `BCC` |
| "không có dữ liệu hợp lệ để tạo bảng chấm công" | Shift codes not in rows 9–11 | Use the guide layout (codes at row 9, data from row 10) |
| "không tìm thấy mức lương cho ca X (N VND)" | Code missing from payrate config | Add the code to payrate config |
| "nhân viên không tìm thấy trong hệ thống" | CCCD not assigned to project | Assign employee (or add STK sheet) |
| Days 21–31 missing after import | Mid-month span dropped | Split into two calendar-month files |

## 8. Open questions

1. Mid-month cycle (21→20) vs calendar-month model: split-files acceptable? Intersects unfinished day-20 cutoff work.
2. Confirm exact position name in BUMHAN config ("phổ thông" vs wage-tier naming like "Mức 600"); confirm whether label lookup should be position-scoped or position-agnostic.
3. Holidays: is "every day = ngày thường" permanent for BUMHAN? Code has no holiday detection.
4. Recommend the fixing agent also ship a downloadable compliant template xlsx (none exists today; `export-entries-template` is for manual entry, not partner BCC import).
5. Note: worktree has an unrelated 3-line uncommitted edit in `frontend/src/pages/admin/PayrateEditPage/index.tsx` — the fixing agent must not trample it.
