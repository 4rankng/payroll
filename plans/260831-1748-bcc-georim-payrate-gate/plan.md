# BCC Georim payrate gate + review hardening

Date: 2026-08-31 · Status: verification-complete · Mode: /ak-fix --auto

## Symptom

GEORIM-luongtuan4TH8.xlsx (project 77 Georim, first upload, weekly BCC-HC/
OT30/OT200/OT150 sheets) failed 5× on prod 16:00-16:32 with a single sanitized
error "Không thể xử lý dòng dữ liệu này". Raw prod error_detail:
"không tìm thấy bảng lương cho dự án: no active payrate found for this project
and date".

## Root cause (evidence: prod assets 457-461 + code)

`prepareImportContext` (bcc_import_lifecycle.go) gates the whole import on a
payrate active at **month start**. Georim's payrate exists but starts
**2026-08-24** (project created 08-30, config verified in admin UI) — from_date
> Aug 1 → NotFound → whole import rejected before any row is read, even though
every worked day in the file (Aug 26-28) is covered. The weekly flow already
resolves payrates per entry date downstream; only this shared gate probed
month start.

Secondary: the raw reason matched no frontend sanitizer rule → displayed as
the vague fallback instead of an actionable config hint.

## Changes

1. `bcc_import_lifecycle.go` — prepareImportContext now takes
   `earliestEntryDay`; on NotFound at monthStart it retries at
   `time.Date(year, month, earliestEntryDay)`. User-confirmed rule: a mid-month
   payrate is accepted ONLY when already active on/before the file's earliest
   worked day — never retroactively for days before it takes effect.
2. Four callers compute earliestDay from their parsed entries:
   bcc_import_process.go (legacy DayNum), bcc_import_weekly.go ×2 (weekly
   Date.Day(), payment Day), bcc_import_multi_position.go (DayNum).
3. `frontend/src/utils/import-errors.ts` — new rule: 'bảng lương'/'payrate' →
   "Chưa cấu hình bảng lương cho dự án" (+ test).

## Code-review hardening folded into the same series (Samsung reviewer, REQUEST_CHANGES)

1. STK blank spacer rows no longer terminate parsing (removed-guard regression)
   + TestParseSTKSheet_BlankSpacerRow.
2. Partially-labelled STK headers keep legacy dataStart via new
   detectSTKHeaderRowLoose + test.
3. shiftLabelHourType token-matching (TCN/TCNN/NN no longer mis-bucketed as
   regular hours) + test cases.
4. flatRatesHaveBucket membership guard on the rateless label fallback + test.
5. rateRow == day-row guard (date serials can never parse as VND rates).

## Verification

- Unit: full excel package PASS; services package only the 7 pre-existing
  date-drift failures; gofmt/vet clean; frontend vitest 5/5 + tsc 0 errors.
- `make api-test`: 296 total, 273 passed, 0 failed, 23 skipped (env-gated).
- E2E local dev, ORIGINAL GEORIM file (asset 463, project 77, 2026-08):
  completed, created 10, errors 0. Rows verified: Công HC 8h × Aug 26-27;
  Tùng/Vịnh HC 2h + OT30 6h × Aug 27-28; paytypes
  "up công lgd.ngày thường.hc/.ot30"; matched the 3 existing 12-digit-CCCD
  employees (no duplicates created).

## Follow-ups (documented, deferred)

- Holiday blindness of determineDayType on rateless templates (finding 6).
- chức danh column scan-order priority + Department field has no consumers
  (finding 7).
- GEORIM file hygiene (partner): CCCD column lost the 12th digit on all rows
  (matched leniently this time, but should be fixed / formatted as text), and
  the 4th listed person (03120500319x, zero hours) is not assigned to the
  project.
