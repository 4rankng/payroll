# 08 — Advance Payment (FlexPay)

**Priority**: P0 | **API Prefix**: `/api/v1/advance-payments` | **Risk Level**: Critical

## Business Rules

### Request Window (3-Phase Cutoff)
```
Day  1-10:  OPEN for previous calendar month's period
Day 11-20:  LOCKED (inter-period gap, requests rejected)
Day 21-31:  OPEN for current month ONLY IF admin uploaded "bang luong" for current month
```

### Amount Validation
- Minimum amount: 5,000 VND
- Maximum amount: determined by FlexPay import (bang luong)
- Cannot exceed remaining available amount

### Fee Calculation
- Fee schedule defined per lender/provider
- Fee preview available before request submission

### Disbursement Lifecycle
1. Employee creates request → `PENDING`
2. Poller claims request → `APPROVED`
3. Provider (9Pay) executes → `COMPLETED` (success) or `FAILED`
4. Employee can cancel → `CANCELLED` (only while PENDING)

### Cancellation Rules
- Only PENDING requests can be cancelled
- Cannot cancel APPROVED, COMPLETED, FAILED, or already CANCELLED

### Employee Visibility
- After removal from project: employee STILL visible for original forMonth
- Employee NOT visible for current month after removal
- Re-adding employee restores visibility

## State Machine

```
Advance Payment:
[PENDING] → (poller claims) → [APPROVED] → (9Pay executes) → [COMPLETED]
[PENDING] → (employee cancels) → [CANCELLED]
[APPROVED] → (9Pay fails) → [FAILED]

Request Window:
Day 1-10:  OPEN (previous month)
Day 11-20: LOCKED
Day 21-31: OPEN (current month, IF bang luong uploaded)
```

## Test Scenarios

### F13 — Advance Payment

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| **Happy Path — Employee** ||||
| F13-01 | Get advance payment info | Happy | GET `/advance-payments/my-info` | 200, eligibility, max amount, current period |
| F13-02 | Calculate fee preview | Happy | POST `/advance-payments/calculate-fee` with amount | 200, fee breakdown returned |
| F13-03 | Create advance request | Happy | POST `/advance-payments` with valid amount | 201, request created (PENDING) |
| F13-04 | View my advance history | Happy | GET `/advance-payments/my-history` | 200, list of past requests with status |
| F13-05 | Cancel PENDING request | Happy | POST `/advance-payments/:id/cancel` (while PENDING) | 200, status → CANCELLED |
| **Happy Path — Admin** ||||
| F13-06 | List pending requests | Happy | GET `/advance-payments?status=pending` (admin) | 200, filtered list |
| F13-07 | Get advance summary | Happy | GET `/advance-payments/summary` | 200, aggregated advance statistics |
| F13-08 | Get available months | Happy | GET `/advance-payments/available-months` | 200, months with bang luong data |
| **Disbursement Lifecycle** ||||
| F13-09 | Poller claims and completes | Happy | Create request → wait for poller → 9Pay executes | Status: PENDING → APPROVED → COMPLETED |
| F13-10 | Verify wallet payment after completion | Integration | Complete advance → GET wallet payments | Payment record created with correct amount |
| F13-11 | Verify employee completedAmount | Integration | Complete advance → GET employee info | completedAmount increased |
| **Amount Validation** ||||
| F13-12 | Below minimum amount | Negative | POST with amount < 5000 | 400, minimum amount validation |
| F13-13 | Zero amount | Negative | POST with amount = 0 | 400, validation error |
| F13-14 | Exceeds max amount | Negative | POST with amount > max | 400, exceeds available amount |
| F13-15 | Calculate fee with zero | Negative | POST calculate-fee with amount = 0 | 400, validation error |
| **Request Window** ||||
| F13-16 | Request during Day 1-10 (OPEN) | Edge | Set clock to day 5 → create request | 201, forMonth = previous month |
| F13-17 | Request during Day 11-20 (LOCKED) | Negative | Set clock to day 15 → create request | 400, request window locked |
| F13-18 | Request during Day 21-31 (OPEN) | Edge | Set clock to day 25 → create request (bang luong uploaded) | 201, forMonth = current month |
| F13-19 | Request Day 21-31 without bang luong | Negative | Set clock to day 25 (no bang luong) → create request | 400, locked until bang luong uploaded |
| **Cancellation Edge Cases** ||||
| F13-20 | Cancel already cancelled | Negative | Cancel → cancel again | 400, already cancelled |
| F13-21 | Cancel APPROVED request | Negative | Cancel after poller claimed | 400, cannot cancel non-PENDING |
| **Cross-Module Integration** ||||
| F13-22 | FlexPay import enables Phase 3 | Integration | Upload bang luong → request on day 25 | Request accepted (phase 3 enabled) |
| F13-23 | Advance payment appears in dashboard | Integration | Create request → GET dashboard | Advance counts/amounts reflected |
| F13-24 | Advance payment creates ledger entries | Integration | Complete advance → GET ledger | Financial entries created |
| F13-25 | Reconciliation cancels outstanding | Integration | Send reconciliation email → check advances | Outstanding requests cancelled |

### F14 — FlexPay Import

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F14-01 | Upload bang luong Excel | Happy | POST `/flex-pay/import` with forMonth + Excel file | 200, import job created |
| F14-02 | Import updates advance limits | Integration | Upload → employee checks advance info | Max advance amount updated |
| F14-03 | Upload for future month | Negative | POST with forMonth in future | Rejected or processed with warning |
| F14-04 | Re-upload same month | Edge | Upload bang luong for same month twice | Data updated (latest wins) |
| F14-05 | Import triggers quota calculation | Integration | Upload → GET available months | Month appears in available list |
| F14-06 | Import appears in audit log | Integration | Upload → check audit logs | Import event logged |
| F14-07 | Invalid Excel format | Negative | Upload non-Excel file | 400, parsing error |
| F14-08 | Upload without forMonth | Negative | POST without forMonth | 400, forMonth required |

### F-AV — Advance Removal Visibility

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| AV-01 | Removed employee visible for original forMonth | Happy | Record forMonth → remove employee → list advances | Employee still visible for original month |
| AV-02 | Removed employee not visible for current month | Happy | Remove employee → list advances for current month | Employee not in list |
| AV-03 | Re-added employee restored | Edge | Remove → re-add → list advances | Employee visible for current month |
| AV-04 | Removal preserves historical data | Integration | Remove → check advance history | Past advance records intact |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_advance_payment.go` (684 lines, largest flow)
- Integration test: `backend/tests/integration/flow_flexpay_import.go`
- Integration test: `backend/tests/integration/flow_advance_removal_visibility.go`
- Integration test: `backend/tests/integration/flow_clock_manipulation.go` (for cutoff testing)
- Clock helpers: `SetServerTime`, `AdvanceServerTime`, `ResetServerTime`
