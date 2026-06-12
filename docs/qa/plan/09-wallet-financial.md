# 09 — Wallet & Financial (Wallet, Transaction, Ledger, Settlement)

**Priority**: P0 | **API Prefix**: `/api/v1/wallet`, `/api/v1/transactions`, `/api/v1/ledger`, `/api/v1/settlements` | **Risk Level**: Critical

## Business Rules

### Wallet
- Balance tracking: available, pending, total
- Sync balance with provider (9Pay/OnePay)
- Available balance = total - pending (deducts pending disbursements)
- Payment records linked to bulk transfers and advance payments
- Topup records track incoming funds
- SyncBalance: available deducts pending, guard skips transient states

### Transaction
- Double-entry accounting: every transaction creates paired ledger entries
- Types: expense, revenue
- Status: pending → partially_settled → settled; settled → reversed
- Amount must be > 0
- Partial settlement supported (multiple settlements per transaction)
- Export available (binary)

### Ledger
- Double-entry bookkeeping: each entry has account, party, debit, credit, date
- Cannot create entry with zero debit AND credit
- Reversal creates a mirror entry with reason
- Cash flow summary and ledger summary endpoints
- Account types defined in accounting rules

### Settlement
- Settles (partially or fully) a transaction
- Proof URL and proof asset support (receipt images/files)
- Settlement date cannot be in the future
- Multiple settlements can be applied to one transaction (partial settle)
- Settlement method tracked (cash, bank transfer, etc.)

## State Machines

```
Transaction:
[pending] → (partial settle) → [partially_settled] → (full settle) → [settled]
[settled] → (reverse with reason) → [reversed]

Ledger Entry:
[created] → (reverse with reason) → [reversed] (+ reversal entry created)

Settlement:
[created] → part of transaction lifecycle
```

## Test Scenarios

### F15 — Wallet

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F15-01 | Get wallet balance | Happy | GET `/wallet/balance` | 200, available, pending, total balance |
| F15-02 | Sync balance with provider | Happy | POST `/wallet/sync` | 200, balance updated from provider |
| F15-03 | List wallet payments | Happy | GET `/wallet/payments` | 200, paginated payment records |
| F15-04 | List wallet topups | Happy | GET `/wallet/topups` | 200, paginated topup records |
| F15-05 | Resolve wallet payment | Happy | POST `/wallet/payments/:id/resolve` | 200, payment resolved |
| F15-06 | Resolve non-existent payment | Negative | POST `/wallet/payments/99999/resolve` | 404 or error |
| F15-07 | Sync after bulk transfer | Edge | Complete bulk transfer → sync balance | Balance reflects transfer deduction |
| F15-08 | Available balance deducts pending | Edge | Initiate transfer → check balance | Available = total - pending |
| F15-09 | Wallet balance reflects advance payment | Integration | Complete advance → check wallet | Payment record appears, balance updated |
| F15-10 | Wallet balance reflects manual upload | Integration | Upload manual result → check wallet | Balance updated synchronously |
| F15-11 | Sync updates dashboard financials | Integration | Sync → GET dashboard/financial | Dashboard shows updated balance |

### F16 — Transaction

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F16-01 | Get transaction metadata | Happy | GET `/transactions/metadata` | 200, types, statuses |
| F16-02 | Create expense transaction | Happy | POST `/transactions` with type=expense, amount, party | 201, transaction + ledger entries created |
| F16-03 | Create revenue transaction | Happy | POST `/transactions` with type=revenue, amount, party | 201, transaction + ledger entries created |
| F16-04 | List transactions | Happy | GET `/transactions` | 200, paginated list |
| F16-05 | Get transaction by ID | Happy | GET `/transactions/:id` | 200, transaction details with settlements |
| F16-06 | Partial settle transaction | Happy | POST `/transactions/:id/settle` with partial amount | 200, status → partially_settled |
| F16-07 | Full settle transaction | Happy | POST `/transactions/:id/settle` with remaining amount | 200, status → settled |
| F16-08 | Reverse settled transaction | Happy | POST `/transactions/:id/reverse` with reason | 200, status → reversed, reversal entries created |
| F16-09 | Export transactions | Happy | GET `/transactions/export` | 200, binary export file |
| F16-10 | Create transaction with zero amount | Negative | POST with amount = 0 | 400, amount must be > 0 |
| F16-11 | Settle with amount > remaining | Negative | Settle with amount exceeding balance | 400, cannot exceed transaction amount |
| F16-12 | Reverse non-settled transaction | Negative | Reverse a pending transaction | 400, only settled can be reversed |
| F16-13 | Transaction creates double-entry | Integration | Create transaction → GET ledger | Paired debit/credit entries exist |

### F17 — Settlement

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F17-01 | Create settlement with proof | Happy | POST `/settlements` with transaction_id, amount, date, proof_url | 201, settlement created |
| F17-02 | Create settlement with proof asset | Happy | POST with proof_asset_id (uploaded file) | 201, linked to asset |
| F17-03 | List settlements for transaction | Happy | GET `/settlements?transaction_id=X` | 200, all settlements for transaction |
| F17-04 | Get total settled amount | Happy | GET aggregated settlement total | Returns sum of all settlements |
| F17-05 | Settlement date in future | Negative | POST with future settlement_date | 400, "ngày thanh toán không được ở tương lai" |
| F17-06 | Settlement with zero amount | Negative | POST with amount = 0 | 400, amount must be > 0 |
| F17-07 | Multiple partial settlements | Edge | Create 3 partial settlements → check total | Total matches sum of partial amounts |
| F17-08 | Settlement updates transaction status | Integration | Partial settle → check transaction | Transaction status = partially_settled |
| F17-09 | Full settlement via multiple partials | Integration | Settle in 2 parts → check transaction | Transaction status → settled |
| F17-10 | Settlement recorded in audit log | Integration | Create settlement → check audit | Settlement event logged |

### F18 — Ledger

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F18-01 | Get ledger accounts | Happy | GET `/ledger/accounts` | 200, account types and metadata |
| F18-02 | Create single entry | Happy | POST `/ledger/entries` with account, party, debit, credit, date | 201, entry created |
| F18-03 | Create bulk entries | Happy | POST `/ledger/entries/bulk` with multiple entries | 201, all entries created |
| F18-04 | List entries with filters | Happy | GET `/ledger/entries?fromDate=X&toDate=Y&account=Z` | 200, filtered entries |
| F18-05 | Get cash flow summary | Happy | GET `/ledger/cash-flow?fromDate=X&toDate=Y` | 200, cash flow data |
| F18-06 | Get ledger summary | Happy | GET `/ledger/summary` | 200, aggregated summary |
| F18-07 | Reverse entry | Happy | POST `/ledger/entries/:id/reverse` with reason | 200, reversal entry created |
| F18-08 | Create entry with zero debit AND credit | Negative | POST with debit=0 and credit=0 | 400, cannot be both zero |
| F18-09 | Reverse non-existent entry | Negative | POST reverse with invalid ID | 404 or error |
| F18-10 | Export ledger | Happy | GET `/ledger/export` | 200, binary export file |
| F18-11 | Reversal creates mirror entry | Edge | Reverse entry → check both entries | Reversal has opposite debit/credit |
| F18-12 | Ledger entries linked to transaction | Integration | Create transaction → GET ledger | Entries reference transaction |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_wallet.go`
- Integration test: `backend/tests/integration/flow_transaction.go`
- Integration test: `backend/tests/integration/flow_ledger.go`
- Integration test: `backend/tests/integration/flow_saoke.go` (settlement via sao ke)
- Domain: `backend/internal/domain/transaction_manager.go` — double-entry coordination
