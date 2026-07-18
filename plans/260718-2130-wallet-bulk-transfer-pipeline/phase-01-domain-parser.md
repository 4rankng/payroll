---
phase: 1
title: "Domain & Parser"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Domain & Parser

## Overview

Add the `BulkTransferBatch` domain entity (one per upload, with a `data` JSON column storing parsed rows for outbox re-enqueue + KQ display-name lookup), the migration (5 new columns + 2 indexes; **no new unique index** — relies on existing `idx_wp_request_id`), and the Excel parser. **The parser reads the SWIFT code directly from a column in the input file** — no `BankRepository.FindByBankCode` resolution needed (the input file's admin/producer pre-resolves the bank).

## Requirements

- **Functional**: Parse the "Yêu cầu chuyển tiền" `.xlsx` format (sheet `eMB_BulkPayment`, header row 2, data row 3+) WITH a SWIFT code column. Persist batch metadata + parsed rows in `data` JSON. Support content-hash-based duplicate detection. TOCTOU-safe per-row duplicate rejection via existing `idx_wp_request_id` UNIQUE.
- **Non-functional**: Parsing deterministic, ≤50ms for typical 50-row files. All VND amounts as `int64`. Fail loudly on unknown headers. Hard cap at 5,000 rows (zip-bomb defense). `excelize.OpenReader` with `UnzipSizeLimit: 50<<20, UnzipXMLSizeLimit: 10<<20` (default is 16GB — red-team v2 Security H1).

## Architecture

### New entity: `BulkTransferBatch`

Lives alongside `BulkTransferFile` (the existing 9Pay timesheet-driven flow) — do NOT modify `BulkTransferFile`.

Table: `bulk_transfer_batches`

```
id              bigint unsigned PK
filename        varchar(255)              -- sanitized: basename only, control chars stripped, ≤128 chars
content_hash    char(64) UNIQUE           -- SHA-256 of normalized row tuples
source          varchar(16) DEFAULT 'wallet_upload'
status          varchar(32) DEFAULT 'pending'
                                          -- pending → processing → completing → completed → failed
enqueue_state   varchar(16) DEFAULT 'pending'  -- pending → enqueued (batch-level outbox)
total_count     int DEFAULT 0
success_count   int DEFAULT 0
failed_count    int DEFAULT 0
transfer_amount bigint DEFAULT 0
total_fee       bigint DEFAULT 0
ledger_txn_id   bigint unsigned NULL       -- FK to transactions.id (set when ledger booked)
fee_booked_at   datetime NULL              -- idempotency timestamp
data            json NOT NULL             -- []BulkTransferRow JSON (source of truth for re-enqueue + KQ)
asset_id        bigint unsigned NULL
created_by      bigint unsigned
created_at, updated_at, completed_at datetime
```

### New columns on `wallet_payments` (migration 093)

```sql
ALTER TABLE wallet_payments
  ADD COLUMN bulk_transfer_batch_id BIGINT UNSIGNED NULL,
  ADD COLUMN bulk_transfer_order    INT UNSIGNED NULL,
  ADD COLUMN vfic_code              VARCHAR(32) NULL,
  ADD COLUMN enqueue_state          VARCHAR(16) NULL DEFAULT 'enqueued',
  ADD COLUMN sweeper_retry_count    INT UNSIGNED NOT NULL DEFAULT 0,
  ADD INDEX idx_wp_bulk_batch (bulk_transfer_batch_id),
  ADD INDEX idx_wp_vfic_code (vfic_code);
```

**No new UNIQUE index** — the existing `idx_wp_request_id` UNIQUE at `migrations/054_wallet_tables.up.sql:25` already catches every VFIC duplicate (both old advance-payment rows `request_id='VFIC3ba3ec31'` and new bulk rows use VFIC as request_id). Red-team v2 Security C2 proved the v3 generated-column approach was redundant.

### Note on `tx_wallet_payment_repository.go` vs `wallet_payment_repository.go`

Two repos exist:
- `tx_wallet_payment_repository.go` — GORM-based, used by `WalletPaymentService` (our path). Auto-handles new columns via struct tags.
- `wallet_payment_repository.go` — legacy hand-written SQL with hardcoded `paymentColumns`/`Create`/`Update` strings. Used by `walletService` for the CSV reconciliation flow. NOT in our path.

New repo methods go in `tx_wallet_payment_repository.go` + the `domaintx.WalletPaymentRepository` interface. Do NOT touch the legacy SQL repo unless a test queries through it.

### Parser: `YeuCauChuyenTienParser`

Input: `io.Reader`. Output: `[]BulkTransferRow`.

```go
type BulkTransferRow struct {
    OrderNo       int    `json:"order_no"`
    AccountNo     string `json:"account_no"`
    AccountName   string `json:"account_name"`
    Bank          string `json:"bank"`           // Vietnamese display name (e.g. "Quân đội (MB)") for KQ column D
    SwiftCode     string `json:"swift_code"`     // READ DIRECTLY from input — no BankRepository lookup
    Amount        int64  `json:"amount"`
    PaymentDetail string `json:"payment_detail"`
    VFICCode      string `json:"vfic_code"`      // extracted from PaymentDetail via regex
}
```

### Input file format (with SWIFT column)

The admin/producer includes a SWIFT code column. The reference `Yeu cau chuyen tien.xlsx` in `~/Downloads/` does NOT have it — that file is from a Vietnamese bank export and only has the Vietnamese display name. The actual input file uploaded to `/admin/wallet` MUST include the SWIFT column.

Expected header map (resilient to column reordering — match by name, not offset):

| Field | Vietnamese header | English alias |
|-------|-------------------|---------------|
| OrderNo | `STT` | `Ord. No.` |
| AccountNo | `Số tài khoản` | `Account No.` |
| AccountName | `Tên người thụ hưởng` | `Beneficiary` |
| Bank (display name) | `Ngân hàng thụ hưởng/Chi nhánh` | `Beneficiary Bank` |
| **SwiftCode** | `Mã SWIFT` / `Mã SWIFT/BIC` | `SWIFT Code` / `SWIFT/BIC` / `BIC` |
| Amount | `Số tiền` | `Amount` |
| PaymentDetail | `Nội dung chuyển khoản` | `Payment Detail` |

Unknown headers → `ErrUnknownExcelHeader`. Missing SWIFT column → `ErrMissingSwiftColumn`. SWIFT value doesn't match `^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$` (8 or 11 chars, ISO 9362 format) → `ErrInvalidSwift`.

### VFIC code extraction

Regex `(?i)VFIC[0-9a-f]+` against `PaymentDetail`. Verified against production (`app.log:46260`: `VFIC3ba3ec31`). If missing → `ErrMissingVFICCode`.

## Related Code Files

- **Create**: `backend/internal/domain/bulk_transfer_batch.go` — entity + repository interface.
- **Create**: `backend/internal/infra/persistence/bulk_transfer_batch_repository.go` — GORM impl. `UpdateWithLock` returns `(batch *BulkTransferBatch, shouldBook bool, err error)` — NOT sentinel-error pattern (GORM rolls back on non-nil error; red-team v2 R2-3).
- **Create**: `backend/internal/app/services/wallet_bulk/parser.go` — `YeuCauChuyenTienParser`.
- **Create**: `backend/internal/app/services/wallet_bulk/parser_test.go` — golden-file test.
- **Create**: `backend/internal/app/services/wallet_bulk/types.go` — `BulkTransferRow` DTO + error sentinels.
- **Create**: `backend/migrations/092_bulk_transfer_batches.up.sql` + `.down.sql`.
- **Create**: `backend/migrations/093_wallet_payments_bulk_columns.up.sql` + `.down.sql`.
- **Create**: `backend/testdata/yeu_cau_chuyen_tien_with_swift.xlsx` — anonymized fixture WITH SWIFT column.
- **Create**: `backend/testdata/kq_chuyen_tien_reference.xlsx`.
- **Modify**: `backend/internal/app/bootstrap/repositories/init.go` — register `BulkTransferBatch` repo.
- **Modify**: `backend/internal/domain/transactions/wallet_payment.go` — add `BulkTransferBatchID *uint64`, `BulkTransferOrder *uint`, `VFICCode *string`, `EnqueueState *string`, `SweeperRetryCount int` fields + GORM tags.
- **Modify**: `backend/internal/domain/transactions/repository.go` — add new read methods to interface.

### Fee schedule migration — DROPPED

Production already at 3,850 (`app.log:46260`). No fee-schedule migration needed. Tests seed 3,850 explicitly for determinism.

## Implementation Steps

1. **Anonymize + commit test fixtures**: build `yeu_cau_chuyen_tien_with_swift.xlsx` from the user's attached file by adding a `Mã SWIFT` column (use real SWIFT codes from `migrations/059_add_swift_code_to_banks.up.sql` — e.g. MB → `MBBEVNVX`, VCB → `BFTVVNVX`). Replace names with `Nguyen Test A/B`, accounts with `99990001/99990002`. Keep VFIC codes + amounts intact.

2. **Write migration `092_bulk_transfer_batches.up.sql`** — create table per schema above. `.down.sql`: drop table.

3. **Write migration `093_wallet_payments_bulk_columns.up.sql`** — 5 new columns + 2 indexes. **No new UNIQUE index** (existing `idx_wp_request_id` covers it). `.down.sql`: drop indexes, drop columns.

4. **Add fields** to `WalletPayment` struct with GORM column tags.

5. **Write `bulk_transfer_batch.go`** entity + repository interface:
   ```go
   type BulkTransferBatchRepository interface {
       Create(ctx, b *BulkTransferBatch) error
       GetByID(ctx, id uint64) (*BulkTransferBatch, error)
       GetByContentHash(ctx, hash string) (*BulkTransferBatch, error)
       Update(ctx, b *BulkTransferBatch) error
       UpdateWithLock(ctx, id uint64, fn func(*BulkTransferBatch) (shouldBook bool, err error)) (*BulkTransferBatch, bool, error)
       List(ctx, filter BatchFilter) ([]*BulkTransferBatch, int64, error)
       UpdateEnqueueState(ctx, id uint64, state string) error
       ListByEnqueueState(ctx, state string, olderThan time.Time, limit int) ([]*BulkTransferBatch, error)
       ListByStatusAndOlderThan(ctx, status string, olderThan time.Time, limit int) ([]*BulkTransferBatch, error)
   }
   ```

6. **Write GORM repository** with `UpdateWithLock` returning `(batch, shouldBook, err)`:
   ```go
   func (r *bulkTransferBatchRepository) UpdateWithLock(ctx, id, fn) (*BulkTransferBatch, bool, error) {
       var shouldBook bool
       var result *BulkTransferBatch
       err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
           var b BulkTransferBatch
           if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&b, id).Error; err != nil { return err }
           var fnErr error
           shouldBook, fnErr = fn(&b)
           if fnErr != nil { return fnErr }  // rolls back
           if err := tx.Save(&b).Error; err != nil { return err }
           result = &b
           return nil  // commits
       })
       return result, shouldBook, err
   }
   ```

7. **Write `types.go`**: `BulkTransferRow`, sentinel errors (`ErrUnknownExcelHeader`, `ErrMissingSwiftColumn`, `ErrInvalidSwift`, `ErrMissingVFICCode`, `ErrInvalidAmount`, `ErrEmptyFile`, `ErrInvalidSheet`, `ErrTooManyRows`).

8. **Write `parser.go`**:
   - `NewYeuCauChuyenTienParser(logger *slog.Logger)` — **no `BankRepository` dependency**.
   - `Parse(ctx, r io.Reader) ([]BulkTransferRow, error)`:
     - Open with `excelize.OpenReader(r, excelize.Options{UnzipSizeLimit: 50<<20, UnzipXMLSizeLimit: 10<<20})`.
     - Verify sheet `eMB_BulkPayment` exists.
     - Read header row 2, build column map.
     - Iterate data rows from row 3 until 2 consecutive empty rows.
     - **Hard cap row count at 5,000** — reject larger with `ErrTooManyRows`.
     - For each row: extract 7 fields (incl. SWIFT), extract VFIC via `(?i)VFIC[0-9a-f]+`, validate `Amount ≥ 100000`, validate SWIFT format `^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`.
     - Return `[]BulkTransferRow`.

9. **Wire the repo** in `bootstrap/repositories/init.go`.

10. **Write parser tests**:
    - Happy path: parse the fixture (2 rows) → assert correct Amount, VFIC codes, SWIFT codes (`MBBEVNVX`, `BFTVVNVX`).
    - Missing SWIFT column → `ErrMissingSwiftColumn`.
    - Invalid SWIFT format (e.g. `MB`) → `ErrInvalidSwift`.
    - Missing VFIC in PaymentDetail → `ErrMissingVFICCode`.
    - Unknown header → `ErrUnknownExcelHeader`.
    - Reordered columns → parser still maps correctly.
    - Empty file → `ErrEmptyFile`.
    - 5,001-row file → `ErrTooManyRows`.

11. **Run `make api-test`**.

## Success Criteria

- [ ] Migration `092` creates `bulk_transfer_batches`.
- [ ] Migration `093` adds 5 columns + 2 indexes; **no new UNIQUE index** (existing `idx_wp_request_id` covers VFIC dups).
- [ ] No fee-schedule migration.
- [ ] `BulkTransferBatch` entity + repo interface; `UpdateWithLock` returns `(batch, shouldBook, error)`.
- [ ] `WalletPayment` struct has 5 new fields.
- [ ] Test fixture `yeu_cau_chuyen_tien_with_swift.xlsx` committed (anonymized, WITH SWIFT column).
- [ ] Parser reads SWIFT directly from input — **no `BankRepository.FindByBankCode` calls in parser code or tests**.
- [ ] Parser validates SWIFT format (ISO 9362).
- [ ] Parser parses fixture into 2 correct rows.
- [ ] Parser rejects: missing SWIFT column, invalid SWIFT, missing VFIC, unknown header, empty file, >5000 rows.
- [ ] Parser handles column reordering.
- [ ] `excelize.OpenReader` called with `UnzipSizeLimit` options.
- [ ] `cd backend && go test ./internal/app/services/wallet_bulk/... -v -race -cover` ≥85% on parser.

## Risk Assessment

- **Risk**: Excel cell types (number vs string for Amount). **Mitigation**: parse via `strconv.ParseInt` after trim.
- **Risk**: SWIFT code format drift (lowercase, whitespace). **Mitigation**: uppercase + trim before regex validation; document expected format.
- **Risk**: PII in committed fixture. **Mitigation**: anonymize names/accounts; deterministic VFIC codes generated in tests.
- **Risk**: Admin uploads a Yêu cầu file WITHOUT the SWIFT column (legacy format). **Mitigation**: parser returns clear `ErrMissingSwiftColumn` with instructions to add the column; UI surfaces a helpful error message.
