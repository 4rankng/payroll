# EOD Wallet Settlement — Design Spec

> **Date:** 2026-05-30
> **Status:** Approved
> **Problem:** Wallet payment completions (OnePay/9Pay) mark advance payment requests as COMPLETED but never create ledger entries. The ledger only gets updated when an admin manually uploads a sao ke file — a step that doesn't exist for wallet-disbursed payments.

---

## 1. Overview

A midnight cron job that consolidates all completed wallet payments from the previous day into a single batch of ledger entries, following the same 4-entry pattern used by the bulk transfer result upload flow.

**Key principle:** One aggregated Transaction + 4 ledger entries per day, not per payment.

## 2. Current State (Why This Is Needed)

### Existing Flow — Bulk Transfer Result Upload
```
Admin uploads result file → BulkTransferResultParsedEvent
  → BulkTransferTransactionWorker.createTransactionAndLedger()
    → Transaction (type=revenue, amount=receivable, status=pending)
    → TransactionCreated event → initial double-entry:
        Debit  receivable  76,070,000
        Credit revenue     76,070,000
    → Manual entries from BuildLedgerPlan.Entries():
        Credit cash        75,056,870  (net paid to employees)
        Debit  revenue     75,056,870  (offset)
    → Net effect:
        Receivable debit:  76,070,000
        Revenue credit:     1,013,130  (fee = receivable - cashOut)
        Cash credit:       75,056,870  (net paid out)
```

### Wallet Payment Flow (Broken — No Ledger)
```
IPN arrives → RecordIPN() → OnEnterCompleted hook
  → Update advance request to COMPLETED   ✅
  → Send notifications                     ✅
  → Create ledger entries                  ❌ MISSING
  → Link settlement_transaction_id         ❌ MISSING
```

## 3. Solution Design

### 3.1 Cron Trigger

Registered in `internal/infra/asynq/mux.go`:
- Schedule: `1 0 * * *` (00:01 daily Vietnam time)
- Task type: `TaskWalletSettlement`
- Queue: `low`

### 3.2 Worker: `WalletSettlementWorker`

New file: `internal/app/workers/wallet_settlement_worker.go`

**Dependencies (injected):**
- `db *gorm.DB`
- `walletPaymentRepo wallet.WalletPaymentRepository`
- `advancePaymentReqRepo domain.AdvancePaymentRequestRepository`
- `transactionRepo domain.TransactionRepository`
- `transactionService serviceports.TransactionPort`
- `ledgerService *settlement.LedgerService`
- `settingsConfig *config.SettingsConfigService`
- `logger *slog.Logger`

**Processing flow:**

```
1. Determine target date: yesterday (clock.Now().AddDate(0, 0, -1))
2. Query wallet_payments:
   WHERE status = 'completed'
     AND DATE(updated_at) = <yesterday>
     AND settled_at IS NULL
3. If no results → log "no unsettled wallet payments" → return nil
4. Fetch linked advance_payment_requests via wallet_payment.entity_id → apr.id
5. Aggregate from advance_payment_requests (source of truth for accounting):
   totalReceivable = SUM(apr.request_amount)  // gross amounts (what client owes)
   totalCashOut    = SUM(apr.net_amount)       // net amounts (paid to employees)
   ourFee          = totalReceivable - totalCashOut  // our fee income
   NOTE: wallet_payment.fee is the GATEWAY fee (e.g., 3,300 VND to OnePay), NOT our fee.
   We must NOT use wallet_payment.fee for ledger calculations.
6. Date guard:
   Check if a Transaction exists WHERE description = "Wallet disbursement YYYY-MM-DD"
   If found → log "already processed" → skip
7. Create Transaction:
   type=revenue, amount=totalReceivable, party=partnerCompany,
   description="Wallet disbursement YYYY-MM-DD", status=pending
   → TransactionCreated event fires → LedgerWorker creates initial double-entry
8. Build manual ledger entries via BuildWalletSettlementPlan.Entries():
   - Credit cash, party="Nhân viên", amount=totalCashOut
   - Debit revenue, party=partnerCompany, amount=totalCashOut
9. Persist entries via LedgerService.CreateEntries
10. Batch-update wallet_payments: SET settled_at = NOW() WHERE id IN (processed IDs)
11. Batch-update advance_payment_requests:
    SET settlement_transaction_id = <new transaction ID>
    WHERE id IN (entity_ids from processed wallet_payments)
12. Log summary:
    "Wallet settlement completed: date=YYYY-MM-DD, payments=N, receivable=X, cashOut=Y, fee=Z"
```

**All DB writes in steps 6-10 happen inside a single `db.Transaction()` for atomicity.**

### 3.3 Builder: `BuildWalletSettlementPlan`

New file: `internal/app/services/disbursement/wallet_settlement_builder.go`

```go
type WalletSettlementPlan struct {
    Date            time.Time
    TotalReceivable int64  // SUM(apr.request_amount) — gross amounts from client
    TotalCashOut    int64  // SUM(apr.net_amount) — net amounts paid to employees
    OurFee          int64  // TotalReceivable - TotalCashOut — our fee income
    Description     string
    PaymentCount    int
}

// BuildWalletSettlementPlan aggregates advance payment request amounts for ledger creation.
// IMPORTANT: Uses advance_payment_request amounts (source of truth for accounting), NOT wallet_payment amounts.
// wallet_payment.fee is the gateway processing fee (e.g., 3,300 VND to OnePay) — irrelevant for the ledger.
func BuildWalletSettlementPlan(
    date time.Time,
    requests []domain.AdvancePaymentRequest,
    partnerCompany string,
) WalletSettlementPlan {
    var totalReceivable, totalCashOut int64
    for _, r := range requests {
        totalReceivable += int64(r.RequestAmount)
        totalCashOut += int64(r.NetAmount)
    }
    return WalletSettlementPlan{
        Date:            date,
        TotalReceivable: totalReceivable,
        TotalCashOut:    totalCashOut,
        OurFee:          totalReceivable - totalCashOut,
        Description:     fmt.Sprintf("Wallet disbursement %s", date.Format("2006-01-02")),
        PaymentCount:    len(requests),
    }
}

func (p WalletSettlementPlan) Entries(
    now time.Time,
    processedBy uint,
    partnerCompany string,
) []*domain.LedgerEntry {
    cashOutEntry := &domain.LedgerEntry{
        Date:      now,
        Account:   domain.AccountCash,
        Party:     "Nhân viên",
        Debit:     0,
        Credit:    p.TotalCashOut,
        CreatedBy: processedBy,
    }
    revenueOffsetEntry := &domain.LedgerEntry{
        Date:      now,
        Account:   domain.AccountRevenue,
        Party:     partnerCompany,
        Debit:     p.TotalCashOut,
        Credit:    0,
        CreatedBy: processedBy,
    }
    return []*domain.LedgerEntry{cashOutEntry, revenueOffsetEntry}
}
```

### 3.4 Repository Additions

**`internal/domain/wallet/repository.go`** — add to `WalletPaymentRepository` interface:

```go
// GetCompletedUnsettled returns wallet_payments completed on the given date
// that have not yet been settled (settled_at IS NULL).
GetCompletedUnsettled(ctx context.Context, date time.Time) ([]*WalletPayment, error)

// MarkSettled sets settled_at = NOW() for the given wallet payment IDs.
MarkSettled(ctx context.Context, ids []uint64) error
```

**`internal/infra/persistence/wallet_payment_repository.go`** — implement:

```go
func (r *walletPaymentRepository) GetCompletedUnsettled(ctx context.Context, date time.Time) ([]*wallet.WalletPayment, error) {
    var payments []*wallet.WalletPayment
    start := date.Truncate(24 * time.Hour)
    end := start.AddDate(0, 0, 1)
    err := r.db.WithContext(ctx).
        Where("status = ?", "completed").
        Where("settled_at IS NULL").
        Where("updated_at >= ? AND updated_at < ?", start, end).
        Find(&payments).Error
    return payments, err
}

func (r *walletPaymentRepository) MarkSettled(ctx context.Context, ids []uint64) error {
    if len(ids) == 0 {
        return nil
    }
    return r.db.WithContext(ctx).
        Model(&wallet.WalletPayment{}).
        Where("id IN ?", ids).
        Update("settled_at", clock.Now()).Error
}
```

### 3.5 Idempotency

Two-layer protection ensures safe re-runs:

1. **Date guard (Transaction-level):** Before creating the Transaction, query for an existing transaction with `description = "Wallet disbursement YYYY-MM-DD"`. If found, the batch for that date was already processed — skip entirely.

2. **settled_at guard (Payment-level):** Only query wallet_payments where `settled_at IS NULL`. Once processed in step 9, they're excluded from future runs.

If the worker crashes between step 6 (Transaction created) and step 9 (settled_at set), the date guard prevents a duplicate Transaction on the next run. The wallet_payments remain with `settled_at IS NULL`, so they'd be picked up again — but the date guard catches it before any new Transaction is created.

**Recovery scenario:** If manual entries (step 8) fail but Transaction (step 6) was committed, the Transaction exists without its manual entries. On next run, the date guard sees the Transaction and skips. An admin would need to manually create the offset entries — this is an acceptable edge case. A future improvement could check for missing entries and fill them in.

## 4. Files Changed

| File | Action | Description |
|---|---|---|
| `internal/app/workers/wallet_settlement_worker.go` | **New** | Cron worker |
| `internal/app/services/disbursement/wallet_settlement_builder.go` | **New** | Settlement plan builder |
| `internal/domain/wallet/repository.go` | **Update** | Add `GetCompletedUnsettled`, `MarkSettled` |
| `internal/infra/persistence/wallet_payment_repository.go` | **Update** | Implement new repo methods |
| `internal/infra/asynq/mux.go` | **Update** | Register `TaskWalletSettlement` periodic task |
| `internal/infra/asynq/handlers.go` | **Update** | Add handler for `TaskWalletSettlement` |
| `internal/app/bootstrap/services/init.go` | **Update** | Create `WalletSettlementWorker` |
| `internal/app/bootstrap/container.go` | **Update** | Add worker to container |

## 5. What This Does NOT Do

- Does not change the `OnEnterCompleted` hook (no per-payment ledger creation)
- Does not modify the existing bulk transfer flow
- Does not handle reconciliation (covered by `ReconcilePayment`)
- Does not retroactively fix request #36 (that will be handled by the first cron run or a manual trigger)

## 6. Testing Strategy

- **Unit tests** for `BuildWalletSettlementPlan` with various payment counts and amounts
- **Unit tests** for idempotency: verify date guard skips when Transaction already exists
- **Integration test**: create wallet_payments → trigger cron → verify Transaction + 4 ledger entries exist → verify `settled_at` is set → trigger cron again → verify no duplicates

## 7. Ledger Entry Example

For a day with 3 completed wallet payments (using advance_payment_request amounts):
- Request A: request_amount=5,000,000, fee=65,000, net_amount=4,935,000
- Request B: request_amount=2,000,000, fee=40,000, net_amount=1,960,000
- Request C: request_amount=3,000,000, fee=50,000, net_amount=2,950,000

Aggregated:
- totalReceivable = 10,000,000 (sum of request_amounts = what client owes)
- totalCashOut = 9,845,000 (sum of net_amounts = paid to employees)
- ourFee = 155,000 (totalReceivable - totalCashOut = our fee income)

Ledger entries (4 total):

| # | Account | Party | Debit | Credit |
|---|---------|-------|-------|--------|
| 1 | receivable | VFIC Manpower | 10,000,000 | 0 |
| 2 | revenue | VFIC Manpower | 0 | 10,000,000 |
| 3 | cash | Nhân viên | 0 | 9,845,000 |
| 4 | revenue | VFIC Manpower | 9,845,000 | 0 |

Net effect:
- Receivable: **debit 10,000,000** (owed from client)
- Revenue: **credit 155,000** (our fee income)
- Cash: **credit 9,845,000** (paid out to employees)
- Balance: 10,000,000 = 155,000 + 9,845,000 ✅
