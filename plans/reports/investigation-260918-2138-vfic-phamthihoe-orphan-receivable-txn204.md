# Investigation: VFIC/PhamThiHoe orphan receivable — txn `bc039188` (id 204)

Date: 18/09/2026 · Scope: local synced DB (`payroll-mysql`, `payroll_db`) + local sao kê files

## Question
UI shows revenue transaction `bc039188-867d-4ca4-a3eb-ac859cb04503` (VFIC Manpower,
1,835,982 ₫, "Trả lương cho VFIC Manpower, file: PhamThiHoe.xlsx", created 10/08 22:59 local =
10/08 21:59 UTC) stuck "Chờ TT" with "Đã TT 0 ₫". Which sao kê covers Phạm Thị Hoè, has the
customer paid, and what is the action item?

## Answer
**The money is received and fully reconciled — under txn 205, not 204.** Txn 204 is a duplicate
orphaned receivable with zero linked timesheets; no upload can ever settle it.

## Evidence

### 1. Current DB state
| txn | code | amount | settled | status | asset (source file) | timesheets |
|---|---|---|---|---|---|---|
| 204 | `bc039188-…4503` | 1,835,982 | 0 | **pending** | 400 `PhamThiHoe.xlsx` | **0** |
| 205 | `92b3ed71-…0bf` | 173,094,390 | 173,094,390 | settled | 403 `Kết quả chuyển tiền theo bảng kê - 26081022381994008.xlsx` | 711 |

Settlements on 205 (none on 204):
- `id 220` — 1,835,979 ₫ — 2026-09-10 — proof asset 452 (`sao_ke_tt_2026-08-26.xlsx`), notes `Settlement from upload: sao_ke_email`
- `id 226` — 171,258,408 ₫ — 2026-09-18 — proof asset 555 (`sao_ke_tt_2026-09-01.xlsx`)

### 2. The three timesheets behind txn 204 are already paid
`timesheets 54856/54857/54858` — employee 1284 (Phạm Thị Hoè), project 56 (BUMHAN), dates
2026-08-05/06/07, `amount` 599,994, `revenue_receivable` 611,993 each:
- `revenue_paid = 1`
- `transaction_id = 205`

Sum = 3 × 611,993 = **1,835,979 ₫** = settlement 220 exactly.

### 3. Root cause — last-write-wins re-link across three bulk files
All parsed rows live in `bulk_transfer_files.data`.

| file | filename | uploaded | her row | → txn |
|---|---|---|---|---|
| 129 | `Kết quả chuyển tiền theo bảng kê - 26081021052552580.xlsx` | 10/08 21:09 | `transfer_status: **failed**`, acct `0392073184`, no bank ref | 203 (not linked — only `completed` rows link) |
| 130 | `PhamThiHoe.xlsx` | 10/08 21:59 | `completed`, acct `0392073184`, `bank_txn_ref: MANUAL` | **204** |
| 131 | `Kết quả chuyển tiền theo bảng kê - 26081022381994008.xlsx` | 10/08 23:16 | `completed`, acct `108885601671`, `bank_txn_ref: FT26222736780247` | **205** |

All three carry the same `timesheet_ids: [54858, 54857, 54856]` and `amount: 1799982`.

After the batch row failed (file 129), a manual file was created for her alone (file 130) →
receivable 204. The bank's later batch result (file 131) reported the same transfer with a real
bank reference to her **registered** account (`employees.bank_account_number = 108885601671`,
status `valid`) → receivable 205, and `BulkUpdateTransactionID` re-pointed 54856-58 to 205.

`BulkTransferTransactionWorker.linkTimesheetsToTransaction`
(`backend/internal/app/workers/bulk_transfer_transaction_worker.go:343`) calls
`timesheetRepo.BulkUpdateTransactionID` unconditionally for every `completed` row of every
completed file → last write wins; the earlier receivable is left with no timesheets.

### 4. Uploads cannot fix 204 (by design)
`SettlementUploadService.ProcessSettlementFileWithDedup`
(`backend/internal/app/services/settlement/upload_service.go:267`) settles via the file's
INTERNAL sheet → `timesheet.transaction_id` → transaction. `timesheets WHERE transaction_id=204`
is empty, so no INTERNAL sheet can reference 204.

### 5. Both sao kê files are already 100% reconciled
| file | INTERNAL ids | paid (`revenue_paid=1`) | unpaid |
|---|---|---|---|
| `sao_ke_tt_2026-08-26.xlsx` | 1,673 | 1,673 | 0 |
| `sao_ke_tt_2026-09-01.xlsx` | 4,965 | 4,965 | 0 |

Zero ID overlap between the two files. Her ids are in the 08-26 file only; the 09-01 file has no
`BUMHAN` sheet at all.

### 6. Which sao kê holds Phạm Thị Hoè
`sao_ke_tt_2026-08-26.xlsx` → sheet **`BUMHAN`**:
- row 23 (STT 15): `Phạm Thị Hoè | 089168018363 | BUMHAN | 1799982 | 10/08/2026` ← the 05-07/08 batch
- row 67 (STT 59): same person/amount, `24/08/2026` ← separate later batch

The 24/08 payment covers timesheets 59923/59924/59925 → linked to txn 228, `revenue_paid=1`.
Note her 2026-08-21 timesheet (59926) is still `revenue_paid=0` — outside both files.

## Why 204 is the orphan (not 205)
- Her timesheets are attached to 205 and paid; 205 is fully settled with real bank ref and her
  registered account.
- VFIC's sao kê lists each payment once per date; the 10/08 line maps to one receivable.
- 204 is the only `bulk_transfer_result`-backed transaction in the DB with zero timesheets —
  all 83 settled and the other 10 pending/partial ones have links.
- `transactions.asset_id = 400` whose file is a hand-made `PhamThiHoe.xlsx` targeting the stale
  account `0392073184`.

## Recommended action
Cancel txn 204 — the UI's **Hủy giao dịch** button. `DELETE /transactions/{id}`
(`handlers/transaction.go:378`) soft-deletes a pending transaction plus its ledger entries;
settled transactions must instead be reversed. 204 is pending, so the delete path applies.
205 remains the settled source of truth. Re-uploading the sao kê files changes nothing.

## Follow-ups (not actioned)
1. **Re-link bug:** `linkTimesheetsToTransaction` should skip timesheets already attached to a
   live (non-settled) receivable, or merge/re-point the earlier receivable instead of orphaning it.
2. **Wrong account on manual file:** `PhamThiHoe.xlsx` paid `0392073184`, but the employee's
   registered account is `108885601671`. Worth confirming whether the manual transfer actually
   executed, and why the manual file used a stale number.
3. Rounding tolerance: `constants.RoundingTolerance = 1000` ₫ absorbs the 3 ₫ (txn 205) and 51 ₫
   (txn 203) residuals between `settled_amount` and the sum of settlement rows. Cosmetic.
