# Test Plan — MBank "Kết quả giao dịch" new statement template (260910)

Context: MBank switched the bulk-transfer result export from the flat
"Kết quả chuyển tiền theo bảng kê" layout to a banner+header statement
("Kết quả giao dịch - <ref>.xlsx"). Sample files (real, weekly cycle, codes
present in local dev DB):

- F1 = `~/Downloads/Ket qua giao dich - 202609102419883866.xlsx` (166 rows, 399,206,690 ₫)
- F2 = `~/Downloads/Ket qua giao dich - 202609102419903986.xlsx` (22 rows, 45,495,355 ₫)

Change under test: new `MBankStatementStrategy` (payroll island,
`/payrolls/bulk-transfer-result`), shared header locator
(`excelkit.LocateMBankStatementHeader`), and the same-template branch in
advance_payment `ProcessBankResult` (`/advance-payments/upload-result`).

## Unit (already run)

| # | Case | Expected | Result |
|---|------|----------|--------|
| U1 | Locator: new-template header row + column map | idx 11; C=account F=name G=bank H=amount J=code Q=status X=ref | PASS |
| U2 | Locator: NFD-form headers | still matched | PASS |
| U3 | Locator: legacy flat layout / banner-only | NOT matched | PASS |
| U4 | Strategy: parse new template (3 rows incl. 1 failed) | codes/status/ref/error split correct | PASS |
| U5 | Factory: new → MBankStatementStrategy; old flat → TransactionCodeStrategy (unchanged) | routing correct | PASS |
| U6 | Advance extractor: statement rows (Thất bại→THẤT BẠI), legacy rows byte-identical semantics | PASS | PASS |
| U7 | Real-file env-gated (MBANK_RESULT_REAL_FILE=Downloads) | F1 166/166 completed; F2 22/22 | PASS |
| U8 | Full `go test ./...` | 304+ ok, 0 fail | PASS (81 pkgs ok, 0 FAIL; vet clean) |

## Live dev E2E (localhost:3000 / :8080, admin frankng)

| # | Case | Steps | Expected | Result |
|---|------|-------|----------|--------|
| E1 | New template upload (API) | POST /api/v1/payrolls/bulk-transfer-result with F1 | 200; 166/166/0; items with employee/bank/account/amount/FT ref; bulk_transfer_files (weekly) + transaction/ledger | PASS — 200 in 0.27s, 166 items; btf #144 (166/166/0, 399,206,690 ₫); txn #250; +4 ledger entries |
| E2 | Duplicate re-upload (API) | same file again | 200 with same stats (self-healing, no dupes) | PASS — 166/166/0; no new btf/txn/ledger rows |
| E3 | Old template negative (API) | synthetic old-format xlsx, unknown codes | error naming unknown VFIC code (legacy parse intact) | PASS — "không tìm thấy mã giao dịch trong hệ thống: VFICtest0001" (first attempt with wrong fixture confirmed len<9 guard still skips gapless rows) |
| E4 | UI upload + PDF | browser → /admin/timesheet → "Nhập KQ chuyển lô" → upload F2 → "Tải PDF" | 22/22 shown; PDF downloads | PASS — toast "22 records processed, 22 succeeded, 0 failed"; btf #145 (22/22/0, 45,495,355 ₫); txn #251; URL.createObjectURL hook captured application/pdf blob, 29,529 bytes |

Notes:
- E1/E2/E4 write to local dev DB only (marks weekly payments completed,
  creates transaction/ledger/bulk_transfer_files rows). Accepted — same as
  prior live-dev E2E sessions.
- No prod data touched. Real files never committed.
