<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# tests — Integration Test Suite

## Purpose
End-to-end integration tests that exercise the full HTTP stack against a running server. Tests use a real database (MySQL) and Redis instance, making actual HTTP requests through a test client. The suite covers all major flows: authentication, employee CRUD, timesheet management, advance payments, wallet operations, payroll processing, dashboard analytics, and more. Test results are written to an HTML report.

## Key Files
| File | Description |
|------|-------------|
| `integration/main.go` | Test runner entry point — discovers test data, runs all flows, generates HTML report |
| `integration/client.go` | HTTP test client with auth, request helpers, and response parsing |
| `integration/models.go` | Test data models and response structures (34K — comprehensive) |
| `integration/config.go` | Test configuration (base URL, credentials, timeouts) |
| `integration/reporter.go` | HTML test report generator with pass/fail/skip statistics |
| `integration/assertions.go` | Custom assertion helpers for test validation |
| `integration/clock_helper.go` | Server clock manipulation helpers (fake clock for non-prod) |
| `integration/results/` | Generated HTML test reports (gitignored) |
| `http/` | HTTP-specific test utilities (currently minimal) |
| `fixtures/` | Test fixture files (Excel templates for import tests) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `integration/` | Full integration test suite with flow-based test files |
| `http/` | HTTP-specific test utilities |
| `fixtures/` | Test data files (Excel templates, sample data) |

## For AI Agents

### Working In This Directory
- Run with `make api-test` from the backend root
- Server must be running (on hot-reload via `air`) before tests execute
- Backend port: 8080, DB: `root:rootpassword@tcp(localhost:3306)/payroll_db`
- **Always use `pageSize=200`** when querying timesheets to avoid pagination false failures
- Use `findAvailableDateWithMin(client, projectID, employeeID, start, minDate)` for date selection
- Test data pollution from prior runs is a known issue — use DB cleanup when dates are exhausted
- Monthly timesheet tests skip gracefully when all dates are occupied by paid records

### Testing Requirements
- When adding new features, add corresponding `flow_*.go` test file or extend existing ones
- Each flow file follows the pattern: setup → action → assertion → cleanup
- Use `TestData` struct from `models.go` to share discovered resources across flows
- Clock manipulation available via `SetServerTime`, `AdvanceServerTime`, `ResetServerTime` (non-prod only)

### Common Patterns
```go
// Test flow pattern
func RunMyFeatureFlow(t *testing.T, client *integration.TestClient, data *integration.TestData) {
    result := &integration.TestResult{Name: "My Feature Flow"}
    
    // Setup
    resp := client.POST("/api/v1/resource", payload)
    
    // Assert
    if resp.StatusCode != 200 {
        result.Fail("expected 200, got %d", resp.StatusCode)
    }
    
    // Cleanup
    client.DELETE("/api/v1/resource/" + id)
    result.Pass()
}

// Date selection for timesheets
date := findAvailableDateWithMin(client, projectID, employeeID, startDate, minDate)
```

### Test Flow Files
| Flow File | Coverage |
|-----------|----------|
| `flow_advance_payment.go` | FlexPay lifecycle, fee schedules, file import/export |
| `flow_auth_user.go` | Login, JWT, RBAC, unauthorized access |
| `flow_bcc_import.go` | BCC (bank statement) import processing |
| `flow_bulk_transfer.go` | Bulk payment transfer workflows |
| `flow_clock_manipulation.go` | Fake clock set/advance/reset |
| `flow_dashboard.go` | Dashboard analytics endpoints |
| `flow_employee_crud.go` | Employee create/read/update/delete |
| `flow_employee_self_service.go` | Employee self-service features |
| `flow_flexpay_import.go` | FlexPay file import |
| `flow_ledger.go` | Double-entry ledger operations |
| `flow_loan.go` | Loan management lifecycle |
| `flow_notification.go` | Notification delivery |
| `flow_payrate_crud.go` | Payrate management |
| `flow_project_crud.go` | Project CRUD with assignments |
| `flow_saoke.go` | Sao Ke (bank statement) import/export |
| `flow_settings.go` | System settings management |
| `flow_timesheet_extended.go` | Extended timesheet operations |
| `flow_transaction.go` | Financial transaction flows |
| `flow_wallet.go` | Wallet balance and payment operations |

## Dependencies

### Internal
- All backend packages (full stack integration)
- `internal/pkg/clock` — clock manipulation for time-dependent tests

### External
- Server must be running on `localhost:8080`
- MySQL accessible at `localhost:3306`
- Redis accessible at `localhost:6379`

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
