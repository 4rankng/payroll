---
phase: 4
title: "KQ Excel & Routes"
status: pending
priority: P1
dependencies: [3]
---

# Phase 3: KQ Excel & Routes

## Overview

Build the "KQ Chuyen Tien" Excel generator and expose HTTP routes. KQ column I reads `wallet_payments.invoice_no` (the OnePay FT number) with `"Đang chờ FT"` fallback when invoice_no still equals the VFIC code (Initiate writes `invoice_no := request_id` at INSERT and `RecordSyncResponse` only overwrites when `ProviderRef != ""`). Column D reads the **persisted Vietnamese display name from `bulk_transfer_batches.data` JSON** (the original `Bank` field from the input Excel).

## Requirements

- **Functional**: Generate `.xlsx` matching the reference layout. Column I reads `invoice_no` with `"Đang chờ FT"` fallback when invoice_no starts with VFIC. Column D reads the Vietnamese display name from `bulk_transfer_batches.data` (matched by `bulk_transfer_order`).
- **Non-functional**: Generation ≤200ms for typical 50-row batches. Filename `KQ_Chuyen_Tien_<batch_id>_<YYYYMMDD_HHMMSS>.xlsx`. Backend enforces upload limits.

## Architecture

### KQ layout (from reference `KQ Chuyen Tien.xlsx`)

Same as v1-v4: sheet `data`, rows 1-4 header, row 5+ data, 9 columns A-I.

### Status mapping + column I source

| `wallet_payments.status` | KQ column H | KQ column I source |
|--------------------------|-------------|--------------------|
| `pending` / `verified` / `authorised` | `Đang xử lý` | empty |
| `completed` | `Thành công` | `invoice_no` — **but if `invoice_no` starts with `VFIC`** (FT not yet received via IPN), display `"Đang chờ FT"` instead |
| `failed` | `Thất bại` | `error_message` |
| `reversed` | `Đã hoàn` | `invoice_no` + " (REVERSED)" |

### Column D source — display name from batch data

The parser stored the original Vietnamese display name (e.g. `"Quân đội (MB)"`) in `BulkTransferRow.Bank`, which was persisted to `bulk_transfer_batches.data` JSON. KQ generation:

1. Load batch.
2. Unmarshal `batch.Data` → `[]BulkTransferRow`.
3. Build map: `orderNo → Bank` (display name).
4. For each `wallet_payment` in the batch (ordered by `bulk_transfer_order`), look up display name by `bulk_transfer_order`.

No `BankRepository.FindByBankCode` reverse-lookup (red-team v2 R2-8: returns arbitrary branch).

### Column G source

`wallet_payments.fee` — stamped by `Initiate` from DB schedule.

### Generator

```go
type KQExcelGenerator struct {
    clock clock.Clock
}

func (g *KQExcelGenerator) Generate(ctx context.Context, batch *BulkTransferBatch, rows []*domaintx.WalletPayment) ([]byte, error)
```

### HTTP routes — real auth pattern

Per red-team v1 S4: `Authorize()` takes NO arguments. Verify existing `routes_wallet.go` wiring (`w := v1.Group("/wallet")` with `w.Use(...)`). Add `/bulk-transfer` sub-group inheriting middleware. Set `router.MaxMultipartMemory = 10 << 20` on the route group.

### Casbin policy

Add explicit deny for partner in `backend/configs/casbin_policy.csv`:

```csv
p, partner, /api/v1/wallet/bulk-transfer/*, *, deny
```

### Upload handler — backend file-size limit + ZIP magic + multipart cap

```go
func (h *WalletBulkTransferHandler) UploadBulkTransfer(c *gin.Context) {
    // http.MaxBytesReader BEFORE c.FormFile (red-team v2 Security H3)
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20+512)

    fileHeader, err := c.FormFile("file")
    if err != nil {
        if strings.Contains(err.Error(), "request body too large") {
            c.JSON(400, gin.H{"error":"file_too_large","max_bytes":10<<20}); return
        }
        c.JSON(400, gin.H{"error":"invalid_file"}); return
    }

    if fileHeader.Size > 10<<20 {
        c.JSON(400, gin.H{"error":"file_too_large","max_bytes":10<<20}); return
    }

    file, err := fileHeader.Open()
    if err != nil { c.JSON(500, gin.H{"error":"internal"}); return }
    defer file.Close()

    head := make([]byte, 512)
    n, _ := io.ReadFull(file, head); head = head[:n]
    if _, err := file.Seek(0, 0); err != nil { c.JSON(500, gin.H{"error":"internal"}); return }
    if !isZipMagic(head) {
        c.JSON(400, gin.H{"error":"invalid_file_type","expected":"xlsx (ZIP)"}); return
    }

    bytes, err := io.ReadAll(io.LimitReader(file, 10<<20+1))
    if err != nil { c.JSON(500, gin.H{"error":"internal"}); return }

    userID := getUserIDFromContext(c)
    result, err := h.svc.Upload(c.Request.Context(), bytes, fileHeader.Filename, userID)
    if err != nil { /* error mapping per table below */ return }
    c.JSON(202, result)
}

func isZipMagic(head []byte) bool {
    return len(head) >= 4 && head[0] == 0x50 && head[1] == 0x4B && head[2] == 0x03 && (head[3] == 0x04 || head[3] == 0x05 || head[3] == 0x07)
}
```

### Error → HTTP mapping

| Service error | HTTP | Body |
|---------------|------|------|
| `ErrDuplicateBatch` | 409 | `{"error":"duplicate_batch","existing_batch_id":N}` |
| `ErrDuplicateVFIC` | 409 | `{"error":"duplicate_vfic","conflicts":[...]}` |
| `ErrMissingSwiftColumn` | 400 | `{"error":"missing_swift_column"}` |
| `ErrInvalidSwift` | 400 | `{"error":"invalid_swift","row":N}` |
| `ErrUnresolvedBank` / `ErrInvalidAmount` / `ErrMissingVFICCode` | 400 | `{"error":"invalid_row","detail":"..."}` |
| `ErrUnknownExcelHeader` / `ErrInvalidSheet` | 400 | `{"error":"invalid_excel_format","detail":"..."}` |
| `ErrEmptyFile` | 400 | `{"error":"empty_file"}` |
| `ErrTooManyRows` | 400 | `{"error":"too_many_rows","max":5000}` |
| File > 10MB | 400 | `{"error":"file_too_large","max_bytes":10485760}` |
| Non-ZIP content | 400 | `{"error":"invalid_file_type","expected":"xlsx (ZIP)"}` |

## Related Code Files

- **Create**: `backend/internal/app/services/wallet_bulk/kq_generator.go`.
- **Create**: `backend/internal/app/services/wallet_bulk/kq_generator_test.go`.
- **Create**: `backend/internal/transport/http/handlers/wallet_bulk_transfer_handler.go`.
- **Modify**: `backend/internal/app/services/wallet_bulk/service.go` — implement `DownloadKQ`.
- **Modify**: `backend/internal/app/bootstrap/routes_wallet.go` — register `/bulk-transfer` sub-group + set `MaxMultipartMemory`.
- **Modify**: `backend/internal/app/bootstrap/container.go` — instantiate handler.
- **Modify**: `backend/configs/casbin_policy.csv` — partner deny.

## Implementation Steps

1. **Write `KQExcelGenerator.Generate`**:
   - Sheet `data`, rows 1-4 header (exact Vietnamese strings + `\n`).
   - Unmarshal `batch.Data` → `[]BulkTransferRow`; build `orderNo → Bank` map.
   - Iterate `rows` sorted by `bulk_transfer_order`:
     - A: order, B: account_no, C: name, D: display name from map, E: amount, F: PaymentDetail (from `description`), G: fee, H: status mapping, I: invoice_no / "Đang chờ FT" / error_message.
     - For column I on completed rows: `if strings.HasPrefix(invoice_no, "VFIC") { cell = "Đang chờ FT" } else { cell = invoice_no }`.
   - Column widths match reference.
   - `f.WriteToBuffer()` → bytes.

2. **Implement `Service.DownloadKQ`**: load batch + wallet_payments ordered by `bulk_transfer_order` → generate → return `(bytes, filename)`.

3. **Write `WalletBulkTransferHandler`**: upload (with size + magic + multipart cap), list, get, download KQ.

4. **Wire routes** in `routes_wallet.go`: verify real auth pattern; add `/bulk-transfer` sub-group; set `MaxMultipartMemory`.

5. **Add Casbin deny** for partner.

6. **Wire container**: add handler to handlers struct.

7. **Write KQ generator tests**:
   - Mock batch + 4 wallet_payments: completed-with-FT (`invoice_no="FT26199097048888"`), completed-without-FT (`invoice_no="VFIC3ba3ec31"`), failed, pending.
   - Assert row 5 H=`Thành công`, I=`FT26199097048888`.
   - Assert row 6 H=`Thành công`, I=`Đang chờ FT`.
   - Assert row 7 H=`Thất bại`, I=error.
   - Assert row 8 H=`Đang xử lý`, I=empty.
   - Assert column D = Vietnamese display name (e.g. `"Quân đội (MB)"`), NOT SWIFT.
   - Assert column G = 3850.

8. Run `make api-test`.

## Success Criteria

- [ ] KQ column I reads `invoice_no`; falls back to `"Đang chờ FT"` when invoice_no starts with VFIC.
- [ ] KQ column D reads Vietnamese display name from `batch.data` JSON, NOT SWIFT code.
- [ ] `POST /upload` enforces 10MB + ZIP magic + `http.MaxBytesReader` + `MaxMultipartMemory`.
- [ ] All 4 routes work; partner denied (Casbin).
- [ ] KQ download works at any batch status.
- [ ] Filename: `KQ_Chuyen_Tien_<batch_id>_<YYYYMMDD_HHMMSS>.xlsx`.
- [ ] `cd backend && go test ./internal/app/services/wallet_bulk/... ./internal/transport/http/handlers/... -v -race` passes.

## Risk Assessment

- **Risk**: Vietnamese chars. **Mitigation**: excelize handles UTF-8.
- **Risk**: Mid-processing download. **Mitigation**: KQ reflects current state.
- **Risk**: XXE in excelize parsing the GENERATED file. **Mitigation**: Generator uses excelize programmatically; no XML injection surface.
- **Risk**: ZIP bomb on upload. **Mitigation**: 10MB hard cap + `io.LimitReader` + `excelize.UnzipSizeLimit` in parser.
- **Risk**: Route collision. **Mitigation**: Sub-group under `/wallet/bulk-transfer/*`.
