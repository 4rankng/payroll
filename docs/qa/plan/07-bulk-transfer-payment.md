# 07 — Bulk Transfer & Payment

**Priority**: P0 | **API Prefix**: `/api/v1/bulk-transfers`, `/api/v1/disbursement` | **Risk Level**: Critical

## Business Rules

### Auto Bulk Transfer (9Pay/OnePay)
- Requires `ENABLE_NINEPAY` (or provider) configuration
- IPN (Instant Payment Notification) arrives ~3s after batch completion (async)
- Tests must poll for `payment_status='paid'` rather than checking immediately
- Cannot use both `forMonth` and date range simultaneously
- `toDate` required when `fromDate` provided
- 9Pay mock sandbox sends IPNs to `http://host.docker.internal:8080/api/v1/webhooks/disbursement/9pay`

### Manual Bulk Transfer
- Export Excel with approved timesheets
- Admin processes with bank externally
- Upload result file → timesheets marked as paid synchronously
- Manual upload marks timesheets as paid directly (no IPN polling)

### Disbursement Provider
- Production uses OnePay (NOT 9Pay); system is provider-agnostic
- BalanceService does pre-flight auth check, daily reconciliation (sandbox/dev only)
- Check account validates bank code + account number

## State Machine

```
Auto Transfer: [initiated] → processing → completed (via IPN)
                                       ↘ failed (IPN timeout)

Manual Transfer: [exported] → uploaded → paid (synchronous)
```

## Test Scenarios

### F11 — Bulk Transfer (Auto via Provider)

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F11-01 | Check auto-bulk-transfer config | Happy | GET `/bulk-transfers/config` | 200, returns enabled/disabled status |
| F11-02 | Estimate fee for weekly period | Happy | GET `/bulk-transfers/estimate-fee?fromDate=X&toDate=Y&schedule=weekly` | 200, fee estimate returned |
| F11-03 | Estimate fee for monthly period | Happy | GET `/bulk-transfers/estimate-fee?forMonth=2026-06` | 200, fee estimate returned |
| F11-04 | Initiate weekly bulk transfer | Happy | POST `/bulk-transfers/initiate` with weekly date range | 200, transfer initiated, provider processes |
| F11-05 | Initiate monthly bulk transfer | Happy | POST `/bulk-transfers/initiate` with forMonth | 200, transfer initiated |
| F11-06 | Poll for payment completion | Happy | Initiate → poll timesheets for payment_status='paid' | Timesheets eventually marked 'paid' |
| F11-07 | Both forMonth and date range | Negative | POST with both forMonth and fromDate+toDate | 400, "dong thoi" error |
| F11-08 | FromDate without ToDate | Negative | POST with fromDate but no toDate | 400, toDate required |
| F11-09 | Future date range (no timesheets) | Edge | Estimate fee for future period | 200, zero fee (no approved timesheets) |
| F11-10 | Transfer marks timesheets as paid | Integration | Initiate → wait for IPN → check timesheets | Timesheets payment_status='paid' |
| F11-11 | Transfer updates wallet balance | Integration | Complete transfer → GET wallet balance | Balance reflects payment |
| F11-12 | Transfer creates ledger entries | Integration | Complete transfer → GET ledger | Double-entry entries created |
| F11-13 | Transfer updates project financials | Integration | Complete transfer → GET project summary | total_payout_vnd increases |

### F12 — Bulk Transfer (Manual)

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F12-01 | Export bulk transfer Excel | Happy | GET `/bulk-transfers/export?forMonth=2026-06` | 200, binary Excel file |
| F12-02 | Upload result file | Happy | POST `/bulk-transfers/upload-result` with Excel | 200, timesheets marked paid synchronously |
| F12-03 | Export for non-existent month | Negative | GET `/bulk-transfers/export?forMonth=2020-01` | 200, empty Excel (no data) |
| F12-04 | Upload marks timesheets paid | Integration | Upload → check timesheets | payment_status='paid' immediately |
| F12-05 | Upload updates wallet | Integration | Upload → GET wallet | Payment records created |
| F12-06 | Upload creates audit trail | Integration | Upload → check audit logs | Upload event logged |
| F12-07 | Re-upload same result | Edge | Upload result file twice | Idempotent handling |
| F12-08 | Export respects project filter | Integration | Export with project_id filter | Only matching timesheets included |

### F31 — Disbursement

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F31-01 | Get provider stats | Happy | GET `/disbursement/stats` | 200, provider statistics |
| F31-02 | Get manual banks list | Happy | GET `/disbursement/banks` | 200, list of supported banks |
| F31-03 | Get manual balance | Happy | GET `/disbursement/balance` | 200, current balance |
| F31-04 | Check account (valid) | Happy | POST `/disbursement/check-account` with valid bank code + account | 200, account info returned |
| F31-05 | Check account (invalid bank) | Negative | POST with invalid bank code | Error response |
| F31-06 | List disbursements | Happy | GET `/disbursement` | 200, paginated disbursement history |
| F31-07 | Balance reflects transfers | Integration | Complete transfer → check balance | Balance decreased by transfer amount |
| F31-08 | Provider stats reflect activity | Integration | Complete transfer → check stats | Stats updated with new transfer data |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_bulk_transfer.go`
- Integration test: `backend/tests/integration/flow_manual_bulk_transfer.go`
- Integration test: `backend/tests/integration/flow_disbursement.go`
- Important: Use `pageSize=200` when polling timesheets to avoid pagination issues
