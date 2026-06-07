<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# handlers — HTTP Request Handlers

## Purpose
Contains all HTTP request handlers organized by domain area. Each handler struct receives application services via DI, parses incoming HTTP requests, delegates to the appropriate service method, and formats the response. Handlers are thin adapters — they contain no business logic, only HTTP-level concerns like request parsing, response formatting, and file upload handling.

## Key Files (Root-Level Handlers)
| File | Description |
|------|-------------|
| `transaction.go` | Transaction handler — financial transaction CRUD, wallet operations (27.1K) |
| `wallet_handler.go` | Wallet handler — balance queries, topup, payment history (15.7K) |
| `payroll.go` | Payroll handler — payroll summary, salary calculation (17.9K) |
| `payrate.go` | Payrate handler — payrate CRUD and validation (13.1K) |
| `payrate_validate.go` | Payrate validation — cross-field validation rules (14.9K) |
| `dashboard.go` | Dashboard handler — admin and employee analytics (20.5K) |
| `metric_handler.go` | Metric handler — API performance metrics (18.7K) |
| `ledger.go` | Ledger handler — double-entry ledger queries (17.8K) |
| `notification.go` | Notification handler — notification list and push management (11.9K) |
| `employee_profile.go` | Employee profile handler — self-service profile endpoints (13.1K) |
| `user.go` | User handler — user management CRUD (9.3K) |
| `health.go` | Health check handler — service health and dependency status (2.5K) |
| `auth.go` | Auth handler — login, logout, token refresh (6.6K) |
| `audit.go` | Audit handler — audit log queries (6.6K) |
| `email_handler.go` | Email handler — email template management (5.8K) |
| `cron_handler.go` | Cron handler — cron job management (3.1K) |
| `db_export_handler.go` | DB export handler — full database export (3.8K) |
| `cache.go` | Cache handler — cache management endpoints (1.4K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `admin/` | Admin endpoints — clock manipulation (non-prod), attendance, provider transaction queries |
| `adv_partner/` | Advance partner handler — partner user management |
| `advance_payment/` | Advance payment handlers — FlexPay lifecycle, file import/export, fee schedules, reconciliation |
| `attendance/` | Attendance handler — check-in/out operations |
| `bank/` | Bank handlers — bank CRUD and listing |
| `disbursement/` | Disbursement handlers — manual disbursement, fee schedules, reconciliation, webhook handling |
| `employee/` | Employee handlers — CRUD, import, export, payroll, summary, timesheets, user accounts |
| `lender/` | Lender handler — lender management |
| `loan/` | Loan handler — loan lifecycle management (19.7K) |
| `project/` | Project handlers — CRUD, payrates, summary, timesheets, users, entry tables (28.5K) |
| `project_employee/` | Project-employee handlers — assignment management, batch operations |
| `push/` | Push notification handler — Web Push subscription management |
| `settings/` | Settings handler — system configuration management (11.1K) |
| `settlement/` | Settlement handlers — settlement upload and processing |
| `timesheet/` | Timesheet handlers — CRUD, bulk create/update, approval, BCC import, export, templates |

## For AI Agents

### Working In This Directory
- Handlers are constructed in `bootstrap/container.go` with service dependencies injected
- Routes are registered in `bootstrap/routes.go`
- Use `response.BadRequest`, `response.NotFound`, `response.HandleError` for error responses
- Parse request bodies with `c.ShouldBindJSON(&req)` or `helpers.ParseRequest(c, &req)`
- For file uploads: use `c.FormFile("file")` and stream processing
- Always use `c.Request.Context()` for context propagation
- For paginated results, use `helpers.ParsePagination(c)` and return total count header

### Testing Requirements
- Tested through integration tests in `tests/integration/` via HTTP
- Each flow file covers a handler group
- File upload handlers tested with fixture files from `tests/fixtures/`

### Common Patterns
```go
// Handler struct with DI
type MyHandler struct {
    service    *services.MyService
    auditSvc   *infrastructure.AuditService
}

// Constructor
func NewMyHandler(svc *services.MyService, audit *infrastructure.AuditService) *MyHandler {
    return &MyHandler{service: svc, auditSvc: audit}
}

// Handler method
func (h *MyHandler) Create(c *gin.Context) {
    ctx := c.Request.Context()
    var req dto.CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    result, err := h.service.Create(ctx, req)
    if err != nil {
        response.HandleError(c, err)
        return
    }
    response.Created(c, result)
}
```

## Dependencies

### Internal
- `internal/app/services/*` — service dependencies
- `internal/app/dto` — request/response types
- `internal/transport/http/response` — response formatting
- `internal/transport/http/helpers` — request parsing utilities
- `internal/pkg/clock` — for time-dependent operations

### External
- `gin-gonic/gin` — HTTP context and binding

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
