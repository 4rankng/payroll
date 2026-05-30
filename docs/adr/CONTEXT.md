# Advance Payment Partner — Context

## Glossary

### adv_partner
A user role (`UserRole = "adv_partner"`) for external partners who manage the advance payment pipeline end-to-end. Replaces the defunct `tester` role in the enum. Not scoped to specific projects — sees all projects that have advance payment data for a given period.

### Advance Payment Partner Scope
The set of operations an `adv_partner` can perform: upload FlexPay data, view request lists and status, export sao kê and reconciliation files. Excludes financial actions (cancel, settle, upload-result, send-email) and general payroll operations.

## Role Enum
After migration: `enum('admin','partner','employee','adv_partner')`. The `tester` role is removed.

## Access Matrix

| Endpoint | adv_partner |
|----------|:-----------:|
| `GET /advance-payments` | YES |
| `GET /advance-payments/summary` | YES |
| `GET /advance-payments/available-months` | YES |
| `GET /advance-payments/employees` | YES |
| `GET /advance-payments/employees/export` | YES |
| `GET /advance-payments/export` | YES |
| `GET /advance-payments/reconciliation/export` | YES |
| `POST /advance-payments/import` | YES |
| `GET /advance-payments/import/:id` | YES |
| `POST /advance-payments/import-employee-list` | YES |
| `POST /advance-payments/:id/cancel` | NO |
| `POST /advance-payments/upload-result` | NO |
| `POST /advance-payments/reconciliation/send-email` | NO |
| `POST /advance-payments/reconciliation/settle` | NO |
| `GET /advance-payments/files` | NO |
| `GET /advance-payments/files/:id/download` | NO |
| `GET /advance-payments/transfer-histories/:id/download` | NO |
| `/admin/advance-payment-fees/*` | NO |
| `/admin/disbursement-fees/*` | NO |

## Decisions

- **No project_users scoping**: adv_partner sees all projects with advance payment data. No per-project assignment needed.
- **Shared frontend, role-gated**: Same admin panel, sidebar hides non-advance-payment menus. Backend enforces via Casbin.
- **Authentication**: Username/email/CCCD + password, or Google OAuth. Admin creates accounts manually.
- **Dashboard**: Uses `/advance-payments/summary` as their dashboard. No general dashboard access.
