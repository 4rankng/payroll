# 11 — Sao Ke & Statements

**Priority**: P1 | **API Prefix**: `/api/v1/saoke` | **Risk Level**: High

## Business Rules

### Payroll Report Email (Bảng Công Sao Kê)
- Sends payroll report with attached Excel asset
- Email includes `saoKeAssetId` (attached Excel file)
- Email history tracked with metadata

### Reconciliation Email (Ứng Lương Sao Kê)
- Sends advance payment reconciliation email
- **Cancels outstanding advance requests** as part of reconciliation
- Settlement is idempotent — can re-settle without error

### Settlement Flow
- Settle payroll email → marks as settled
- Settle advance/reconciliation email → marks as settled
- Re-settle is idempotent (no duplicate processing)

### Admin Re-upload
- Admin can re-upload to already-settled notifications
- Dedup guard prevents duplicate processing
- Updated data replaces previous

### Export
- Reconciliation preview available for export (binary Excel)

## State Machine

```
Email: [sent] → (settle) → [settled] → (admin re-upload) → [updated]
                                                         ↘ (re-settle) → [settled] (idempotent)
```

## Test Scenarios

### F21 — Sao Ke

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F21-01 | Send payroll report email | Happy | POST `/saoke/payroll-email` with project_id, forMonth, saoKeAssetId | 200, email sent with attachment |
| F21-02 | Send reconciliation email | Happy | POST `/saoke/reconciliation-email` with filters | 200, email sent, outstanding advances cancelled |
| F21-03 | Get email history | Happy | GET `/saoke/emails` | 200, paginated email history with metadata |
| F21-04 | Verify email metadata | Happy | GET `/saoke/emails/:id` | 200, metadata includes asset_id, recipient, dates |
| F21-05 | Settle payroll email | Happy | POST `/saoke/emails/:id/settle` | 200, email marked as settled |
| F21-06 | Settle reconciliation email | Happy | POST `/saoke/emails/:id/settle` (reconciliation) | 200, settled + advances cancelled |
| F21-07 | Re-settle (idempotent) | Edge | Settle already-settled email | 200, idempotent — no duplicate processing |
| F21-08 | Admin re-upload to settled | Edge | Upload to already-settled notification | 200, dedup guard prevents duplicates, data updated |
| F21-09 | Send email without required fields | Negative | POST payroll-email without saoKeAssetId | 400, asset required |
| F21-10 | Send email for non-existent project | Negative | POST with invalid project_id | 400+ error |
| F21-11 | Reconciliation cancels outstanding advances | Integration | Send reconciliation → check advance requests | Outstanding requests cancelled |
| F21-12 | Email asset downloadable | Integration | Send email → download saoKeAssetId | 200, binary file matches uploaded asset |
| F21-13 | Settlement reflected in dashboard | Integration | Settle → GET dashboard | Financial figures updated |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_saoke.go`
