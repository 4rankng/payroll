# Test Plan — OnePay Fee Import Template Variants

**Date:** 2026-09-04
**Bug:** `BBĐS VFICPO Tháng 08.2026.xlsx` fails with `GD · Không tìm thấy sheet chi tiết giao dịch GD` in "Nhập phí OnePay" dialog.
**Scope:** `backend/internal/app/services/settlement/onepay_fee_import_service.go` — parse level only. No schema, no frontend contract change (frontend renders server issues as-is).

## Known Variants (evidence)

| | V1 (existing, tested) | V2 (failing file) |
|---|---|---|
| Detail sheet name | `GD` | `CHI TIET THANG` |
| Layout | headers in row 1, data from row 2 | banner rows 1–7, VN header row 8, EN header row 9, data from row 10, trailing `Tổng cuối (Total)` row |
| Date column | `Create Date` (`06-06-2026 12:26 PM`) | `Thời gian giao dịch` / `Transaction Date` (`31/08/2026 19:29:37`) |
| Amount column | `Amount` | `Giá trị GD` / `Transaction Amount` |
| State column | `State` (required today) | absent |
| Currency column | `Currency` (optional today) | absent |
| Beneficiary cols | `Beneficiary Account/Name/Bank` | `Số Tài khoản` / `Tên chủ tài khoản` / `Ngân hàng` |
| Summary sheet | `PHI THANG` (same in both) | identical |

## Matrix

### Unit (Go) — `settlement` package

| # | Case | Expected |
|---|------|----------|
| U1 | V2 fixture: sheet `CHI TIET THANG`, banner rows, VN+EN header rows 8/9, data from row 10, leading empty column A, trailing total row, no State/Currency cols, dates `31/08/2026 19:29:37` | parse OK, 0 issues; summary (period 2026-08-01→31, count, fee/txn, total fee, import ref) correct; 2 details mapped correctly |
| U2 | Real file via env `ONEPAY_FEE_REAL_FILE` (skips when unset; never committed) | parse OK, 177 details, 0 issues, total fee 681,450 |
| U3 | Detail sheet renamed arbitrarily (`Detail 08`) but headers intact | still detected via content fallback |
| U4 | Summary sheet renamed but `SLGD` row intact | still detected via content fallback |
| U5 | Old V1 flat `GD` fixture (existing tests) | unchanged, pass |
| U6 | Official layout without optional metadata (existing test) | unchanged, pass |
| U7 | State column present with value `Pending` | `state_mismatch` still fires (strictness preserved) |
| U8 | State column absent | NO state issue |
| U9 | Second bilingual header row (EN row after VN row) | skipped, not treated as a data row |
| U10 | `parseOnePayDate("31/08/2026 19:29:37")` | parses |
| U11 | No detail sheet at all | `missing_sheet` issue, message mentions both known names |

### E2E — local dev (per user: test in local dev)

| # | Case | Expected |
|---|------|----------|
| E1 | `POST /api/v1/ledger/onepay-fee-reports` with real file, admin token | NO template errors (`missing_sheet`/`missing_column`/`date_out_of_period`). Either success (if dev DB holds the 177 wallet payments) or only legitimate data-mismatch issues. |
| E2 | Full `go test ./internal/app/services/settlement/...` | pass |

## Non-goals
- No new reconciliation validations (e.g., per-row fee sum) — would risk false-positive errors on data that is fine.
- No frontend changes.

## Rollback
Revert the single commit; parse-only change, no migrations.
