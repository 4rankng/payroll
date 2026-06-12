# 10 — Loan Management

**Priority**: P1 | **API Prefix**: `/api/v1/loans`, `/api/v1/lenders` | **Risk Level**: High

## Business Rules

### Lender
- Lender represents a source of loan funds (person or institution)
- Can be created, updated, and deleted
- Lender is referenced by loans (foreign key)

### Loan
- Loan has a principal amount, start date, and repayment schedule
- Repayment schedule defines expected payment milestones
- Principal amount must be > 0
- Loan lifecycle: pending → disbursed → partially_repaid → fully_repaid
- Disbursement moves funds from lender to system
- Repayment can be partial (multiple payments tracked)
- Loan can be deleted only while pending (business rule)

## State Machine

```
Loan: [pending] → (disburse) → [disbursed] → (partial repay) → [partially_repaid] → (full repay) → [fully_repaid]

Lender: [created] → active → [deleted]
```

## Test Scenarios

### F19 — Loan

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F19-01 | Create loan with lender | Happy | POST `/loans` with lender_id, principal, start_date, schedule | 201, loan created (pending) |
| F19-02 | Get loan by ID | Happy | GET `/loans/:id` | 200, loan details with schedule |
| F19-03 | List loans | Happy | GET `/loans` | 200, paginated loan list |
| F19-04 | Get repayment schedule | Happy | GET `/loans/:id/schedule` | 200, schedule milestones |
| F19-05 | Disburse loan | Happy | POST `/loans/:id/disburse` | 200, status → disbursed |
| F19-06 | Partial repay | Happy | POST `/loans/:id/repay` with partial amount | 200, status → partially_repaid |
| F19-07 | Full repay | Happy | POST `/loans/:id/repay` with remaining amount | 200, status → fully_repaid |
| F19-08 | Create loan with zero principal | Negative | POST with principal = 0 | 400, amount must be > 0 |
| F19-09 | Disburse already disbursed | Negative | POST disburse on disbursed loan | 400, already disbursed |
| F19-10 | Repay more than remaining | Edge | POST repay with amount > remaining | 400, exceeds remaining balance |
| F19-11 | Loan creates ledger entries on disburse | Integration | Disburse → GET ledger | Cash flow entry created |
| F19-12 | Loan repayment updates ledger | Integration | Repay → GET ledger | Repayment entry created |

### F20 — Lender

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F20-01 | Create lender | Happy | POST `/lenders` with name, contact info | 201, lender created |
| F20-02 | List lenders | Happy | GET `/lenders` | 200, paginated lender list |
| F20-03 | Get lender by ID | Happy | GET `/lenders/:id` | 200, lender details |
| F20-04 | Update lender | Happy | PUT `/lenders/:id` with updated contact info | 200, lender updated |
| F20-05 | Delete lender | Happy | DELETE `/lenders/:id` | 200, soft-deleted |
| F20-06 | Delete lender with active loans | Negative | DELETE lender with pending/disbursed loans | 400, has active loans |
| F20-07 | Create lender without name | Negative | POST without required name field | 400, name required |
| F20-08 | Lender referenced in loan list | Integration | Create lender → create loan → list loans | Lender info appears in loan details |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_loan.go`
