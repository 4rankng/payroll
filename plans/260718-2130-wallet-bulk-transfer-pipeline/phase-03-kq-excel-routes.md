---
phase: 3
title: "KQ Excel & Routes"
status: pending
priority: P1
dependencies: [2]
---

# Phase 3: KQ Excel & Routes

## Overview

Build the "KQ Chuyen Tien" Excel generator that mirrors the attached `KQ Chuyen Tien.xlsx` layout, and expose all `WalletBulkTransferService` operations over HTTP under `/api/v1/wallet/bulk-transfer/*`. The generator reads batch + wallet_payments state and produces a downloadable `.xlsx` even before the batch fully completes (so admins can grab partial results), though the typical flow is download after completion.

## Requirements

- **Functional**: Generate `.xlsx` matching the reference layout exactly (sheet `data`, rows 1-4 header, row 5+ data, 9 columns). Support download at any batch status (rows show their current status — `Thành công`/`Thất bại`/`Đang xử lý`).
- **Non-functional**: Generation ≤200ms for typical 50-row batches. Streaming not needed (files are small). Filename follows the reference convention `KQ_Chuyen_Tien_<batch_id>_<YYYYMMDD_HHMMSS>.xlsx`.

## Architecture

### KQ Excel layout (from reference file)

```
Row 1: A1 = "DANH SÁCH GIAO DỊCH/ LIST OF BULK TRANSACTION"  (merged A1:I1)
Row 2: A2 = "Số tham chiếu (Ref No): <batch_ref>"            (batch_ref = "WB<batch_id>")
Row 3: A3 = "Ngày giao dịch (Transaction date): DD/MM/YYYY"   (batch.created_at in Asia/Ho_Chi_Minh)
Row 4 (headers, 9 columns A-I):
   A: "STT\n(No.)"
   B: "SỐ TÀI KHOẢN\n(Account No)"
   C: "TÊN ĐƠN VỊ THỤ HƯỞNG\n(Beneficiary Name)"
   D: "NGÂN HÀNG THỤ HƯỞNG\n(Beneficiary Bank)"
   E: "SỐ TIỀN GIAO DỊCH\n(Amount)"
   F: "CHI TIẾT THANH TOÁN\n(Payment Detail)"
   G: "PHÍ (CHƯA BAO GỒM VAT)\n(Charge not VAT)"
   H: "TRẠNG THÁI GIAO DỊCH\n(Transaction status)"
   I: "BÚT TOÁN/LÝ DO\n(FT number/ERROR)"
Row 5+: one data row per wallet_payment in the batch.
```

### Status mapping

| `wallet_payments.status` | KQ column H | KQ column I source |
|--------------------------|-------------|--------------------|
| `pending` / `verified` / `authorised` | `Đang xử lý` | empty |
| `completed` | `Thành công` | `bank_ref` (FT number, e.g. `FT26199097048888`) |
| `failed` | `Thất bại` | `error_message` (Vietnamese-translated via `translateMessage`) |
| `reversed` | `Đã hoàn` | `bank_ref` + " (REVERSED)" |

### Column G (PHÍ) source

Per locked decision #4: source is the DB fee schedule value stamped on the `wallet_payments.fee` column at upload time. **Not** read from any external KQ. This is the audited amount.

### Generator: `KQExcelGenerator`

```go
type KQExcelGenerator struct {
    clock clock.Clock
}

func (g *KQExcelGenerator) Generate(ctx context.Context, batch *BulkTransferBatch, rows []*domaintx.WalletPayment) ([]byte, error)
```

Uses `excelize.NewFile()` → set sheet name to `data` → write rows 1-4 → iterate `rows` sorted by `wallet_payments.bulk_transfer_order` (populated by Phase 2) → write data rows → return bytes.

### HTTP routes

All under existing `/api/v1/wallet` group with `Authenticate() + Authorize("admin")`:

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| POST | `/wallet/bulk-transfer/upload` | `UploadBulkTransfer` | Multipart upload, returns 202 + batch_id |
| GET | `/wallet/bulk-transfer/batches` | `ListBulkTransferBatches` | Paginated batch history |
| GET | `/wallet/bulk-transfer/batches/:id` | `GetBulkTransferBatch` | Single batch detail (all rows) |
| GET | `/wallet/bulk-transfer/batches/:id/kq` | `DownloadBulkTransferKQ` | Stream KQ `.xlsx` |
| GET | `/wallet/bulk-transfer/template` | `DownloadTemplate` | (Optional) Re-serve the Yêu cầu template |

### Handler: `WalletBulkTransferHandler`

New file `backend/internal/transport/http/handlers/wallet_bulk_transfer_handler.go`. Mirrors the structure of `handlers/payroll.go` (lines 188-300) for the existing bulk transfer handlers.

```go
type WalletBulkTransferHandler struct {
    svc    *wallet_bulk.Service
    logger *slog.Logger
}
```

### Error → HTTP mapping

| Service error | HTTP status | Body |
|---------------|-------------|------|
| `ErrDuplicateBatch` (content hash) | 409 | `{"error":"duplicate_batch","existing_batch_id":N}` |
| `ErrDuplicateVFIC` (per-row) | 409 | `{"error":"duplicate_vfic","conflicts":["VFICxxx","VFICyyy"]}` |
| `ErrUnresolvedBank` | 400 | `{"error":"unresolved_bank","row":N,"bank":"..."}` |
| `ErrInvalidAmount` | 400 | `{"error":"invalid_amount","row":N}` |
| `ErrMissingVFICCode` | 400 | `{"error":"missing_vfic","row":N}` |
| `ErrUnknownExcelHeader` | 400 | `{"error":"invalid_excel_format","detail":"..."}` |
| `ErrInvalidSheet` | 400 | `{"error":"invalid_sheet","expected":"eMB_BulkPayment"}` |
| `ErrEmptyFile` | 400 | `{"error":"empty_file"}` |

## Related Code Files

- **Create**: `backend/internal/app/services/wallet_bulk/kq_generator.go` — `KQExcelGenerator`.
- **Create**: `backend/internal/app/services/wallet_bulk/kq_generator_test.go` — golden-file test.
- **Create**: `backend/internal/transport/http/handlers/wallet_bulk_transfer_handler.go`.
- **Modify**: `backend/internal/app/services/wallet_bulk/service.go` — implement `DownloadKQ` (was stub in Phase 2).
- **Modify**: `backend/internal/app/bootstrap/routes_wallet.go` — register the 4-5 new routes.
- **Modify**: `backend/internal/app/bootstrap/container.go` — instantiate the handler, inject into the routes registrar.
- **Reuse**: existing `wallet_handler.go` patterns for multipart parsing (`c.FormFile("file")`).
- **Test data**: copy `KQ Chuyen Tien.xlsx` to `backend/testdata/kq_chuyen_tien_reference.xlsx`.

## Implementation Steps

1. **STT preservation**: Phase 1 migration 092 already added `wallet_payments.bulk_transfer_order` and Phase 2 step 8 populates it from `row.OrderNo`. This step just iterates `rows` sorted by `bulk_transfer_order` ASC. No schema change needed here.

2. **Write `KQExcelGenerator.Generate`**:
   - `f := excelize.NewFile()`
   - `f.SetSheetName(f.GetSheetName(0), "data")`
   - Set row 1 merged A1:I1 with the title (use `f.MergeCell("data", "A1", "I1")`).
   - Set row 2 with ref no `WB<batch_id>`.
   - Set row 3 with `batch.CreatedAt` formatted `clock.Now()`-style in Asia/Ho_Chi_Minh (`02/01/2006`).
   - Set row 4 headers (use the exact strings from the reference file, including `\n` line breaks — `excelize` handles `\n` in cells).
   - Iterate `rows` sorted by `bulk_transfer_order`:
     - Column A: order number (1-based).
     - Column B: `RecipientAccountNo`.
     - Column C: `RecipientName`.
     - Column D: bank display name (reverse-lookup from BankCode via `bankRepo` — or store display name on the wallet_payment at creation; simpler to store).
     - Column E: `RequestedAmount` (number format `#,##0`).
     - Column F: `Description` (the VFIC PaymentDetail).
     - Column G: `Fee` (number format `#,##0`).
     - Column H: status mapping per table above.
     - Column I: FT number or error.
   - Apply column widths matching reference (~12, 18, 25, 28, 16, 22, 18, 22, 25).
   - `f.WriteToBuffer()` → return bytes.

3. **Bank display name already on `wallet_payment`**: Phase 2 step 8 stores `RecipientBank = row.Bank` (the Vietnamese display name like `"Quân đội (MB)"`). KQ column D reads it directly. No extra lookup needed.

4. **Implement `Service.DownloadKQ`** in `service.go`:
   - Load batch by ID → if not found, 404.
   - Load all `wallet_payments` with `bulk_transfer_batch_id = id` ordered by `bulk_transfer_order`.
   - Call `KQExcelGenerator.Generate(ctx, batch, rows)`.
   - Return `(bytes, filename)` where filename = `fmt.Sprintf("KQ_Chuyen_Tien_%d_%s.xlsx", batch.ID, batch.CreatedAt.Format("20060102_150405"))`.

5. **Write `WalletBulkTransferHandler`**:
   - `UploadBulkTransfer`: `c.FormFile("file")` → open → read bytes → `svc.Upload(ctx, bytes, file.Filename, userID)` → map errors per table → return 202 with `{batch_id, total_count, transfer_amount, estimated_fee_total}`.
   - `ListBulkTransferBatches`: parse `page`, `page_size` query params (defaults 1/20) → `svc.ListBatches` → return paginated list (summary fields only, no rows).
   - `GetBulkTransferBatch`: parse `:id` → `svc.GetBatch` → return full detail including rows.
   - `DownloadBulkTransferKQ`: parse `:id` → `svc.DownloadKQ` → set `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` + `Content-Disposition: attachment; filename="..."` → `c.Data(200, ct, bytes)`.

6. **Wire routes** in `routes_wallet.go`:
   ```go
   bulk := walletGroup.Group("/bulk-transfer")
   {
       bulk.POST("/upload", container.Handlers.WalletBulkTransfer.UploadBulkTransfer)
       bulk.GET("/batches", container.Handlers.WalletBulkTransfer.ListBulkTransferBatches)
       bulk.GET("/batches/:id", container.Handlers.WalletBulkTransfer.GetBulkTransferBatch)
       bulk.GET("/batches/:id/kq", container.Handlers.WalletBulkTransfer.DownloadBulkTransferKQ)
   }
   ```
   Reuse the existing `walletGroup := v1.Group("/wallet", Authenticate(), Authorize("admin"))` block.

7. **Wire container**: add `WalletBulkTransfer *handlers.WalletBulkTransferHandler` to the handlers struct. Instantiate after the service (Phase 2 step 7).

8. **Write KQ generator tests** (`kq_generator_test.go`):
   - Build a mock batch + 3 wallet_payments (1 completed, 1 failed, 1 pending).
   - Generate bytes → parse with `excelize.OpenReader(bytes.NewReader(b))`.
   - Assert sheet name, header rows 1-4 match expected strings.
   - Assert data row 5 = completed row with FT number in column I.
   - Assert data row 6 = failed row with error in column I.
   - Assert data row 7 = pending row with empty column I and "Đang xử lý" in column H.
   - Assert column G = fee value (3850) for all rows.

9. **Run `make api-test`** including new integration tests (Phase 5 will add the full end-to-end).

## Success Criteria

- [ ] `KQExcelGenerator.Generate` produces bytes parseable by `excelize` and matching the reference layout for headers, status mapping, and fee column.
- [ ] `POST /api/v1/wallet/bulk-transfer/upload` accepts multipart `.xlsx`, returns 202 with batch_id.
- [ ] `GET /api/v1/wallet/bulk-transfer/batches/:id` returns the batch detail with all rows.
- [ ] `GET /api/v1/wallet/bulk-transfer/batches/:id/kq` returns a valid `.xlsx` with correct `Content-Type` and `Content-Disposition`.
- [ ] All error cases return the documented HTTP status codes and JSON error bodies.
- [ ] KQ download works at any batch status (partial results shown for `processing` batches).
- [ ] Filename format: `KQ_Chuyen_Tien_<batch_id>_<YYYYMMDD_HHMMSS>.xlsx`.
- [ ] `cd backend && go test ./internal/app/services/wallet_bulk/... ./internal/transport/http/handlers/... -v -race` passes.

## Risk Assessment

- **Risk**: Vietnamese characters in headers break Excel encoding. **Mitigation**: `excelize` handles UTF-8 natively; golden-file test compares bytes against the reference file structure (not byte-equality, since timestamps differ).
- **Risk**: Large batches (500 rows) make the KQ file large. **Mitigation**: 500 rows × 9 columns ≈ 50KB — trivial. No streaming needed.
- **Risk**: Admin downloads KQ mid-processing, then rows change status → confusing. **Mitigation**: KQ always reflects current DB state; column H shows "Đang xử lý" for unfinished rows. Filename timestamp makes it obvious when it was generated. UI in Phase 4 will show a "regenerate" hint if batch is still processing.
- **Risk**: Routes collide with existing `/wallet/*` routes. **Mitigation**: All new routes are under `/wallet/bulk-transfer/*` sub-group — no collision with existing `/wallet/balance`, `/wallet/payments`, etc.
