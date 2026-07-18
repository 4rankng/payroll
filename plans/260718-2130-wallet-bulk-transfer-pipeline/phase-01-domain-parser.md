---
phase: 1
title: "Domain & Parser"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Domain & Parser

## Overview

Add the `BulkTransferBatch` domain entity (one record per uploaded "Yêu cầu chuyển tiền" file, linking N `wallet_payments` rows), its repository, the migration, and the Excel parser that turns an `eMB_BulkPayment` sheet into a normalized `[]BulkTransferRow` slice. No business logic, no HTTP, no OnePay calls — just data shapes and parsing.

## Requirements

- **Functional**: Parse the attached `Yeu cau chuyen tien.xlsx` format (sheet `eMB_BulkPayment`, header at row 2, data row 3+). Persist batch metadata. Support content-hash-based duplicate detection.
- **Non-functional**: Parsing must be deterministic, ≤50ms for typical 50-row files. All VND amounts as `int64`. No floats. Parser must fail loudly on unknown headers (no silent column dropping).

## Architecture

### New entity: `BulkTransferBatch`

Lives alongside `BulkTransferFile` (the existing 9Pay timesheet-driven flow) — **do not modify `BulkTransferFile`**. The two coexist:

```
BulkTransferFile     (existing, 9Pay, /payrolls page, timesheet-driven)
BulkTransferBatch    (NEW, OnePay, /admin/wallet page, file-upload-driven)
```

Table: `bulk_transfer_batches`

```
id              bigint unsigned PK
filename        varchar(255)              -- original uploaded filename
content_hash    char(64) UNIQUE           -- SHA-256 of normalized row tuples (dup guard)
cycle           varchar(16) DEFAULT 'flexible'   -- reserved; wallet uploads always 'flexible'
source          varchar(16) DEFAULT 'wallet_upload'
status          varchar(32) DEFAULT 'pending'
                                           -- pending → processing → completed → failed
total_count     int DEFAULT 0
success_count   int DEFAULT 0
failed_count    int DEFAULT 0
transfer_amount bigint DEFAULT 0          -- sum of Amount across rows
total_fee       bigint DEFAULT 0          -- sum of fee across rows (booked at completion)
ledger_txn_id   bigint unsigned NULL      -- FK to transactions.id (set when ledger booked)
asset_id        bigint unsigned NULL      -- FK to assets.id (uploaded file stored via FileStorage)
created_by      bigint unsigned
created_at      datetime
updated_at      datetime
completed_at    datetime NULL
```

Each row maps to one `wallet_payments` row. The link is `wallet_payments.bulk_transfer_batch_id` (NEW nullable FK column — added in the migration). No new "batch row" table; `wallet_payments` is the row table.

### Parser: `YeuCauChuyenTienParser`

Input: `io.Reader` (the raw `.xlsx` bytes). Output: `([]BulkTransferRow, error)`.

`BulkTransferRow` (in-memory DTO, not persisted directly — fields populate `wallet_payments` columns):

```go
type BulkTransferRow struct {
    OrderNo      int       // STT
    AccountNo    string    // "Số tài khoản"
    AccountName  string    // "Tên người thụ hưởng"
    Bank         string    // "Ngân hàng thụ hưởng" (Vietnamese display name e.g. "Quân đội (MB)")
    BankCode     string    // resolved SWIFT/bank code (MB, VCB, etc.) — see step 4
    Amount       int64     // "Số tiền" — VND
    PaymentDetail string   // "Nội dung chuyển khoản" — contains VFIC code
    VFICCode     string    // extracted from PaymentDetail via regex `VFIC[0-9a-f]+`
}
```

### Header map (resilient to column reordering)

The parser reads row 2 as the header and builds a column-index map by matching Vietnamese primary names + English aliases:

| Field | Vietnamese header | English alias |
|-------|-------------------|---------------|
| OrderNo | `STT` | `Ord. No.` |
| AccountNo | `Số tài khoản` | `Account No.` |
| AccountName | `Tên người thụ hưởng` | `Beneficiary` |
| Bank | `Ngân hàng thụ hưởng/Chi nhánh` | `Beneficiary Bank` |
| Amount | `Số tiền` | `Amount` |
| PaymentDetail | `Nội dung chuyển khoản` | `Payment Detail` |

Unknown headers → return `ErrUnknownExcelHeader` with the offending cell value. No silent skip.

### Bank name → bank code resolution

The Excel uses Vietnamese display names like `"Ngoại thương Việt Nam (VCB)"`. The bank code is the parenthesized short code. The parser extracts it via regex `\(([A-Z]+)\)`. If no code is found, return `ErrUnresolvedBank` with the row index + bank string.

The second sheet `Tên NH Gợi ý` in the reference file is a dropdown source list — **ignored** by the parser (informational only).

## Related Code Files

- **Create**: `backend/internal/domain/bulk_transfer_batch.go` — entity + repository interface.
- **Create**: `backend/internal/infra/persistence/bulk_transfer_batch_repository.go` — GORM impl.
- **Create**: `backend/internal/app/services/wallet_bulk/parser.go` — `YeuCauChuyenTienParser`.
- **Create**: `backend/internal/app/services/wallet_bulk/parser_test.go` — golden-file test against the attached `Yeu cau chuyen tien.xlsx`.
- **Create**: `backend/internal/app/services/wallet_bulk/types.go` — `BulkTransferRow` DTO + error sentinels.
- **Create**: `backend/migrations/092_bulk_transfer_batches.up.sql` + `.down.sql`.
- **Create**: `backend/migrations/093_seed_onepay_fee_3850.up.sql` + `.down.sql` (corrects placeholder 3,300 → contract 3,850 for `provider='1pay'`).
- **Modify**: `backend/internal/app/bootstrap/repositories/init.go` — register `BulkTransferBatch` repo.
- **Modify**: `backend/internal/domain/transactions/wallet_payment.go` — add `BulkTransferBatchID *uint64` + `BulkTransferOrder *uint` fields + column tags.

## Implementation Steps

1. **Write the migration** `092_bulk_transfer_batches.up.sql`:
   - Create `bulk_transfer_batches` table per schema above.
   - `ALTER TABLE wallet_payments ADD COLUMN bulk_transfer_batch_id BIGINT UNSIGNED NULL, ADD COLUMN bulk_transfer_order INT UNSIGNED NULL, ADD INDEX idx_wallet_payments_bulk_batch (bulk_transfer_batch_id);`
     (`bulk_transfer_order` preserves the original STT from the uploaded Excel so KQ output keeps the same row order — required by Phase 3.)
   - `.down.sql`: drop index, drop both columns, drop table.

1b. **Write the seed-update migration** `093_seed_onepay_fee_3850.up.sql`:
   - Update the active `DisbursementFeeScheduleEntry` for `provider='1pay'` to `fee_vnd=3850`, `effective_date=<today>`, `notes='OnePay contract rate per Dieu 6.3.b'`.
   - Resolves the gap: migration 058 seeded 3,300 (placeholder) but every AC and Decision #4 in `plan.md` requires 3,850.
   - `.down.sql`: revert to 3,300 (or leave forward-only — team preference).

2. **Add `BulkTransferBatchID *uint64` and `BulkTransferOrder *uint`** to the `WalletPayment` struct in `backend/internal/domain/transactions/wallet_payment.go` (after `BatchID *string`). Add `gorm:"column:bulk_transfer_batch_id"` and `gorm:"column:bulk_transfer_order"` tags.

3. **Write `bulk_transfer_batch.go`** domain entity: struct + `TableName()` + `BulkTransferBatchRepository` interface with methods:
   ```go
   type BulkTransferBatchRepository interface {
       Create(ctx context.Context, b *BulkTransferBatch) error
       GetByID(ctx context.Context, id uint64) (*BulkTransferBatch, error)
       GetByContentHash(ctx context.Context, hash string) (*BulkTransferBatch, error)
       Update(ctx context.Context, b *BulkTransferBatch) error
       UpdateWithLock(ctx context.Context, id uint64, fn func(*BulkTransferBatch) error) error
       List(ctx context.Context, filter BatchFilter) ([]*BulkTransferBatch, int64, error)
   }
   ```

4. **Write the GORM repository** in `infra/persistence/`. Mirror the locking pattern from `bulk_transfer_file_repository.go` (`UpdateWithLock` uses `SELECT ... FOR UPDATE`).

5. **Write `types.go`** in `wallet_bulk/`: `BulkTransferRow` struct, sentinel errors `ErrUnknownExcelHeader`, `ErrUnresolvedBank`, `ErrMissingVFICCode`, `ErrInvalidAmount`, `ErrEmptyFile`, `ErrInvalidSheet`.

6. **Write `parser.go`**:
   - Open with `excelize.OpenReader(r)`.
   - Verify sheet `eMB_BulkPayment` exists (else `ErrInvalidSheet`).
   - Read header row 2, build column map.
   - Iterate data rows from row 3 until 2 consecutive empty rows (matches reference file's trailing blank rows at R5-R6).
   - For each row: extract the 6 fields, run regex `VFIC[0-9a-f]+` on `PaymentDetail`, extract bank code via `\(([A-Z]+)\)`, validate `Amount ≥ 100000` (OnePay minimum per `onepay/provider.go:279`).
   - Return `[]BulkTransferRow` on success.

7. **Wire the repo** in `backend/internal/app/bootstrap/repositories/init.go`: instantiate `NewBulkTransferBatchRepository(db.DB)` and add to the `Repositories` struct.

8. **Write parser tests** (`parser_test.go`):
   - Copy `Yeu cau chuyen tien.xlsx` to `backend/testdata/yeu_cau_chuyen_tien.xlsx`.
   - Test happy path: parse → assert 2 rows (Lâm Văn Bách / Lò Thị Hiêm) with correct Amount, VFIC codes, bank codes.
   - Test missing VFIC code in PaymentDetail → `ErrMissingVFICCode`.
   - Test unknown bank name without `(CODE)` → `ErrUnresolvedBank`.
   - Test reordered columns → parser still maps correctly.
   - Test empty file (no data rows) → `ErrEmptyFile`.

9. **Run `make api-test`** to confirm the migration applies cleanly and nothing regresses.

## Success Criteria

- [ ] Migration `092` creates the table and adds `wallet_payments.bulk_transfer_batch_id` + `bulk_transfer_order` columns cleanly; `.down.sql` reverses it.
- [ ] Migration `093` updates the active `1pay` fee schedule entry to `fee_vnd=3850`; existing rows using the old schedule are unaffected (effective-dated).
- [ ] `BulkTransferBatch` entity + repository interface defined and GORM impl registered in bootstrap.
- [ ] `WalletPayment` struct has the new nullable `BulkTransferBatchID` field.
- [ ] `YeuCauChuyenTienParser` parses the reference file into 2 correct rows.
- [ ] Parser rejects: missing VFIC, unresolved bank, unknown header, empty file.
- [ ] Parser handles column reordering via header-name mapping.
- [ ] `cd backend && go test ./internal/app/services/wallet_bulk/... -v -race -cover` passes with ≥85% coverage on `parser.go`.
- [ ] `make api-test` green (no regression).

## Risk Assessment

- **Risk**: Excel cell types (number vs string for Amount). **Mitigation**: `excelize.GetCellValue` returns string; parse via `strconv.ParseInt` after trimming. Reject non-numeric with `ErrInvalidAmount`.
- **Risk**: VFIC code format drift (uppercase hex vs other). **Mitigation**: regex is case-insensitive (`(?iVFIC[0-9a-f]+)`); documented in parser comment.
- **Risk**: Bank code already exists as `bank.go` domain table — don't duplicate. **Mitigation**: Phase 1 only extracts the short code string; bank-code → SWIFT mapping happens in Phase 2 via existing `BankRepository`. Document this boundary in the parser.
