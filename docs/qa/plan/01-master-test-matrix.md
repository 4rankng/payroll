# 01 — Master Test Matrix

## Flow Registry

| ID | Flow | API Prefix | Priority | State Machine | External Dep | Risk Level |
|----|------|-----------|----------|---------------|-------------|------------|
| F01 | Auth & Login | `/api/v1/auth` | P1 | — | None | Medium |
| F02 | User CRUD | `/api/v1/users` | P1 | active → deleted | None | Low |
| F03 | Employee CRUD | `/api/v1/employees` | P1 | active → deleted | None | Medium |
| F04 | Project CRUD | `/api/v1/projects` | P1 | draft → active → completed | None | Medium |
| F05 | Employee Assignment | `/api/v1/projects/:id/employees` | P1 | assigned → removed | None | High |
| F06 | Payrate CRUD | `/api/v1/projects/:id/payrates` | P1 | — | None | Medium |
| F07 | Timesheet Entry | `/api/v1/timesheets` | P0 | draft → pending → approved → paid | None | Critical |
| F08 | Timesheet Approval | `/api/v1/timesheets/bulk-approve` | P0 | pending → approved | None | Critical |
| F09 | BCC Import | `/api/v1/timesheets/partner-import` | P0 | uploaded → completed/failed | Excel parsing | High |
| F10 | Weekly BCC Import | `/api/v1/timesheets/partner-import` | P0 | uploaded → completed/failed | Excel parsing | High |
| F11 | Bulk Transfer (Auto) | `/api/v1/bulk-transfers` | P0 | initiated → completed | 9Pay/OnePay | Critical |
| F12 | Bulk Transfer (Manual) | `/api/v1/bulk-transfers` | P0 | exported → uploaded → paid | None | Critical |
| F13 | Advance Payment | `/api/v1/advance-payments` | P0 | PENDING → APPROVED → COMPLETED/FAILED | 9Pay | Critical |
| F14 | FlexPay Import | `/api/v1/flex-pay/import` | P1 | uploaded → processed | Excel parsing | High |
| F15 | Wallet | `/api/v1/wallet` | P0 | balance → sync → updated | Provider | Critical |
| F16 | Transaction | `/api/v1/transactions` | P0 | pending → settled → reversed | None | Critical |
| F17 | Settlement | `/api/v1/settlements` | P0 | created → settled | None | Critical |
| F18 | Ledger | `/api/v1/ledger` | P0 | created → reversed | None | Critical |
| F19 | Loan | `/api/v1/loans` | P1 | pending → disbursed → repaid | None | High |
| F20 | Lender | `/api/v1/lenders` | P2 | active → deleted | None | Low |
| F21 | Sao Ke | `/api/v1/saoke` | P1 | sent → settled → updated | Email service | High |
| F22 | Dashboard | `/api/v1/dashboard` | P2 | — (read-only) | None | Low |
| F23 | Notifications | `/api/v1/notifications` | P2 | unread → read | Push (VAPID) | Low |
| F24 | Assets | `/api/v1/assets` | P2 | uploaded → deleted | File storage | Low |
| F25 | Audit Logs | `/api/v1/audit` | P2 | — (append-only) | None | Low |
| F26 | Bank CRUD | `/api/v1/banks` | P2 | active → deleted | None | Low |
| F27 | Settings | `/api/v1/settings` | P2 | — | None | Low |
| F28 | Cron Jobs | `/api/v1/admin/cron` | P2 | enabled ↔ disabled | None | Low |
| F29 | Clock Manipulation | `/api/v1/admin/clock` | P2 | real ↔ frozen | None | Low |
| F30 | Employee Self-Service | `/api/v1/employee/me` | P1 | — | None | Medium |
| F31 | Disbursement | `/api/v1/disbursement` | P0 | — | 9Pay/OnePay | Critical |
| F32 | Metrics | `/api/v1/metrics` | P3 | — (read-only) | None | Low |

## Scenario Count per Flow

| ID | Flow | Happy Path | Negative | Edge Case | Integration | Total |
|----|------|-----------|----------|-----------|-------------|-------|
| F01 | Auth & Login | 2 | 3 | 1 | 1 | 7 |
| F02 | User CRUD | 4 | 2 | 1 | 1 | 8 |
| F03 | Employee CRUD | 6 | 3 | 2 | 2 | 13 |
| F04 | Project CRUD | 5 | 3 | 2 | 2 | 12 |
| F05 | Employee Assignment | 4 | 3 | 2 | 3 | 12 |
| F06 | Payrate CRUD | 4 | 2 | 1 | 2 | 9 |
| F07 | Timesheet Entry | 5 | 4 | 3 | 3 | 15 |
| F08 | Timesheet Approval | 3 | 2 | 2 | 2 | 9 |
| F09 | BCC Import | 4 | 3 | 2 | 3 | 12 |
| F10 | Weekly BCC Import | 4 | 3 | 2 | 3 | 12 |
| F11 | Bulk Transfer (Auto) | 4 | 3 | 2 | 4 | 13 |
| F12 | Bulk Transfer (Manual) | 3 | 2 | 1 | 2 | 8 |
| F13 | Advance Payment | 8 | 5 | 4 | 4 | 21 |
| F14 | FlexPay Import | 3 | 2 | 1 | 2 | 8 |
| F15 | Wallet | 4 | 2 | 2 | 3 | 11 |
| F16 | Transaction | 5 | 3 | 2 | 3 | 13 |
| F17 | Settlement | 4 | 2 | 2 | 2 | 10 |
| F18 | Ledger | 4 | 3 | 2 | 3 | 12 |
| F19 | Loan | 5 | 2 | 2 | 2 | 11 |
| F20 | Lender | 4 | 2 | 1 | 1 | 8 |
| F21 | Sao Ke | 5 | 2 | 2 | 3 | 12 |
| F22 | Dashboard | 8 | 1 | 1 | 1 | 11 |
| F23 | Notifications | 4 | 2 | 1 | 2 | 9 |
| F24 | Assets | 3 | 1 | 1 | 2 | 7 |
| F25 | Audit Logs | 2 | 1 | 1 | 1 | 5 |
| F26 | Bank CRUD | 5 | 2 | 1 | 1 | 9 |
| F27 | Settings | 4 | 2 | 1 | 1 | 8 |
| F28 | Cron Jobs | 3 | 1 | 1 | 1 | 6 |
| F29 | Clock Manipulation | 4 | 2 | 2 | 2 | 10 |
| F30 | Self-Service | 5 | 1 | 1 | 2 | 9 |
| F31 | Disbursement | 3 | 2 | 1 | 2 | 8 |
| F32 | Metrics | 4 | 0 | 1 | 1 | 6 |
| | **TOTAL** | **133** | **71** | **49** | **67** | **320** |

## Cross-Module Dependencies

```
                    ┌─────────────┐
                    │   Project   │
                    └──────┬──────┘
                           │ assigns
                    ┌──────▼──────┐
              ┌─────│  Employee   │─────┐
              │     └──────┬──────┘     │
              │            │            │
       ┌──────▼───┐  ┌────▼─────┐  ┌──▼──────────┐
       │ Payrate  │  │Timesheet │  │Advance Pay  │
       └──────┬───┘  └────┬─────┘  └──┬───────────┘
              │           │           │
              │     ┌─────▼─────┐     │
              │     │ Approval  │     │
              │     └─────┬─────┘     │
              │           │           │
              │    ┌──────▼───────────▼──────┐
              │    │    Bulk Transfer        │
              │    └──────────┬──────────────┘
              │               │
         ┌────▼───────────────▼──────────┐
         │     Wallet / Ledger           │
         └──────────┬────────────────────┘
                    │
         ┌──────────▼──────────┐
         │  Transaction /      │
         │  Settlement / Loan  │
         └─────────────────────┘
```
