<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# domain — Domain Layer

## Purpose
The core of the business logic, following DDD principles. Contains all domain entities, value objects, domain events, repository and service interfaces (ports), domain services with pure business rules, specification objects, transaction abstractions, and the wallet aggregate. This package has zero external framework dependencies — all infrastructure is accessed through interfaces defined in `ports/`.

## Key Files
| File | Description |
|------|-------------|
| `events.go` | Domain event definitions — 50+ event types for the event-driven architecture (14.1K) |
| `errors.go` | Domain error constructors — typed business errors (`NewNotFoundError`, `NewValidationError`, etc.) (4.0K) |
| `error_codes.go` | Error code constants — standardized error identifiers (1.5K) |
| `employee.go` | Employee aggregate — core employee entity with business rules (15.6K) |
| `project.go` | Project aggregate — project entity with configuration (13.2K) |
| `project_employee.go` | Project-Employee assignment — assignment lifecycle and constraints (16.3K) |
| `payrate.go` | Payrate entity — rate configuration with temporal validity (18.5K) |
| `timesheet.go` | Timesheet aggregate — check-in/out records with state (13.7K) |
| `timesheet_types.go` | Timesheet types — day types, overtime rules, shift definitions (16.2K) |
| `transaction.go` | Transaction entity — financial transactions (11.9K) |
| `transaction_code.go` | Transaction codes — categorized transaction identifiers (4.3K) |
| `transaction_manager.go` | Transaction manager interface — unit of work pattern (2.8K) |
| `ledger.go` | Ledger aggregate — double-entry accounting entries (18.6K) |
| `accounting_rules.go` | Accounting rules — debit/credit rules per transaction type (7.6K) |
| `advance_payment.go` | Advance payment entity — payment request lifecycle (7.0K) |
| `advance_payment_request.go` | Advance payment request — request processing and validation (11.7K) |
| `advance_payment_fee_schedule.go` | Fee schedule — tiered fee structure for advance payments (4.9K) |
| `notification.go` | Notification entity — notification types and delivery (5.3K) |
| `settings.go` | Settings entity — system configuration (4.6K) |
| `settlement.go` | Settlement entity — payment settlement tracking (3.4K) |
| `loan.go` | Loan entity — loan management (4.9K) |
| `loan_strategy.go` | Loan strategy — strategy pattern for loan calculation (3.5K) |
| `security.go` | Security — password hashing, token generation (25.1K) |
| `dashboard.go` | Dashboard types — analytics query structures (25.5K) |
| `search.go` | Search types — full-text search structures |
| `audit_diff.go` | Audit diff — change tracking for audit logs (8.6K) |
| `audit_templates.go` | Audit templates — standardized audit message formats (8.4K) |
| `email.go` | Email entity — email template and delivery (6.1K) |
| `user.go` | User entity — user account management (5.6K) |
| `services.go` | Service marker interface (836B) |

### Event Factory Files
| File | Description |
|------|-------------|
| `event_factory_base.go` | Base event factory — shared event creation logic (2.6K) |
| `event_factory_employee.go` | Employee events — hire, update, assignment changes (4.6K) |
| `event_factory_timesheet.go` | Timesheet events — create, update, approve, bulk operations (8.8K) |
| `event_factory_financial.go` | Financial events — transactions, settlements, wallet (9.9K) |
| `event_factory_project.go` | Project events — CRUD, configuration changes (6.8K) |
| `event_factory_system.go` | System events — imports, exports, cron, health (16.4K) |
| `event_factory_user.go` | User events — login, logout, profile changes (4.7K) |
| `event_factory_advance_payment_fee.go` | Advance payment fee events (2.2K) |
| `event_factory_disbursement_fee.go` | Disbursement fee events (2.2K) |
| `payment_cycle_event.go` | Payment cycle events — salary period transitions (1.6K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `ports/` | Repository and service interfaces — contracts that infra implements |
| `services/` | Domain services — pure business logic (no infrastructure dependencies) |
| `specs/` | Business specifications — `WorkingDaysSpec` for date calculations |
| `transactions/` | Transaction abstractions — repository, state machine, unit of work |
| `wallet/` | Wallet aggregate — balance management, payments, IPN handling, state transitions |

## For AI Agents

### Working In This Directory
- **Zero external dependencies** — domain code never imports `infra`, `gorm`, `redis`, or any framework
- Define repository interfaces in `ports/`; implementations live in `infra/persistence/`
- Domain errors: use `domain.NewNotFoundError()`, `domain.NewValidationError()`, `domain.NewConflictError()`, etc.
- Domain events: define in `events.go`, create factories in `event_factory_*.go`
- New entities: define struct with business methods; validate invariants in constructors
- The `wallet/` subdirectory is a self-contained aggregate with its own repository interfaces and state machine

### Testing Requirements
- Domain entity tests co-located in the same package (e.g., `employee_test.go`, `project_test.go`)
- Domain service tests in `services/` subdirectory
- Accounting rules have property-based tests (`accounting_rules_test.go`)
- Fee schedule tests verify tiered calculation logic

### Common Patterns
```go
// Domain entity with business method
type Employee struct {
    ID        string
    Name      string
    // ...
}

// Constructor validates invariants
func NewEmployee(name, phone string) (*Employee, error) {
    if name == "" {
        return nil, NewValidationError("name is required")
    }
    return &Employee{Name: name}, nil
}

// Repository interface in ports/
type EmployeeRepository interface {
    FindByID(ctx context.Context, id string) (*Employee, error)
    Save(ctx context.Context, emp *Employee) error
}
```

## Dependencies

### Internal
- `internal/pkg/clock` — for time-dependent business rules
- No other internal dependencies — this is the innermost layer

### External
- Only standard library (`time`, `fmt`, `errors`, `context`, `math`, `strings`, `crypto/*`)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
