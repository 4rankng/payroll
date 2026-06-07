<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# dto — Data Transfer Objects

## Purpose
Defines request and response shapes for the HTTP API layer. DTOs decouple the internal domain model from the external API contract, controlling what data is exposed and how it's serialized. Each file corresponds to a domain area and contains both request (input) and response (output) structs with JSON tags.

## Key Files
| File | Description |
|------|-------------|
| `employee.go` | Employee DTOs — create/update requests, list responses, profile data (10.5K) |
| `employee_import.go` | Employee import DTOs — bulk import request shapes |
| `employee_profile.go` | Employee profile DTOs — self-service profile views |
| `project.go` | Project DTOs — CRUD requests, list/detail responses (9.5K) |
| `project_employee.go` | Assignment DTOs — employee-project assignment management (7.7K) |
| `timesheet.go` | Timesheet DTOs — create/update/approval requests, list responses (10.9K) |
| `timesheet_entry_table.go` | Timesheet entry table DTOs — tabular data for UI rendering |
| `timesheet_edit_request.go` | Timesheet edit request DTOs — edit request workflow |
| `timesheet_export.go` | Timesheet export DTOs — export configuration |
| `payroll.go` | Payroll DTOs — payroll summary and calculation responses (13K) |
| `dashboard.go` | Dashboard DTOs — admin and employee dashboard data (25.5K) |
| `advance_payment.go` | Advance payment DTOs — requests, fee schedules, payment info (14.1K) |
| `transaction.go` | Transaction DTOs — financial transaction requests/responses (5.4K) |
| `ledger.go` | Ledger DTOs — ledger entry queries and balance responses (4.6K) |
| `payrate.go` | Payrate DTOs — payrate configuration for projects (2.9K) |
| `notification.go` | Notification DTOs — notification list and push subscription (3.4K) |
| `user.go` | User DTOs — login, profile, user management (6.4K) |
| `settings.go` | Settings DTOs — system configuration management |
| `loan.go` | Loan DTOs — loan management and repayment schedules (7.6K) |
| `email.go` | Email DTOs — email template and sending requests (4.1K) |
| `export.go` | Export DTOs — generic export request/response shapes (7.5K) |
| `auth.go` | Auth DTOs — login request, JWT response |
| `bank.go` | Bank DTOs — bank CRUD |
| `lender.go` | Lender DTOs — lender management |
| `asset.go` | Asset DTOs — asset tracking |
| `attendance.go` | Attendance DTOs — check-in/out requests (2.1K) |
| `adv_partner.go` | Advance partner DTOs — partner-specific advance payment data |
| `notification.go` | Notification DTOs — notification delivery and subscription |
| `flexpay_reconciliation_email.go` | FlexPay reconciliation email DTOs |
| `flexpay_reconciliation_export.go` | FlexPay reconciliation export DTOs |
| `disbursement_fee_schedule.go` | Disbursement fee schedule DTOs (2.3K) |
| `advance_payment_fee_schedule.go` | Advance payment fee schedule DTOs (2.5K) |

## Subdirectories
_None_

## For AI Agents

### Working In This Directory
- DTOs use `json:"field_name"` tags for serialization
- Request DTOs may include `validate:"required"` tags for input validation
- Keep DTOs flat — avoid deep nesting; use embedded structs for shared fields
- Response DTOs should not expose internal domain fields (IDs, internal timestamps)
- The `dashboard.go` file is the largest (25.5K) due to many aggregated response types

### Testing Requirements
- DTOs are tested indirectly through handler integration tests
- Ensure JSON tags match API contract expectations

### Common Patterns
```go
// Request DTO
type CreateEmployeeRequest struct {
    Name     string `json:"name" validate:"required"`
    Phone    string `json:"phone"`
    BankCode string `json:"bank_code"`
}

// Response DTO
type EmployeeResponse struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    CreatedAt string `json:"created_at"`
}
```

## Dependencies

### Internal
- `internal/domain` — domain entity types (DTOs map to/from these)

### External
_None_ — pure data structures with no external dependencies

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
