<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# persistence — Database Repositories (GORM)

## Purpose
Implements all repository interfaces defined in `domain/ports/` using GORM with MySQL. Contains individual repository files for each aggregate, shared query builders for complex queries, batch processing helpers, relationship loaders for eager loading, and the database connection setup. This is the only layer that directly interacts with the database.

## Key Files
| File | Description |
|------|-------------|
| `database.go` | Database connection setup — GORM configuration, connection pooling, hooks (10K) |
| `base_repository.go` | Base repository — shared CRUD operations and pagination (6.5K) |
| `redis.go` | Redis client setup for caching |
| `health.go` | Database health check for monitoring endpoints |
| `project_repository.go` | Project repository — complex queries with assignments, payrates, users (28.6K) |
| `project_employee_repository.go` | Assignment repository — employee-project assignments with caching (24.1K) |
| `timesheet_repository.go` | Timesheet repository — bulk operations, date queries, aggregations (34K) |
| `employee_repository_crud.go` | Employee CRUD operations (5.2K) |
| `employee_repository_queries.go` | Employee query operations — search, filter, statistics (12.1K) |
| `employee_repository_stats.go` | Employee statistics — aggregated employee data (8.5K) |
| `advance_payment_repository.go` | Advance payment repository (13.4K) |
| `advance_payment_request_repository.go` | Advance payment request repository — complex lifecycle queries (25.7K) |
| `bulk_transfer_file_repository.go` | Bulk transfer file repository — batch payment processing (18.5K) |
| `transaction_repository.go` | Transaction repository — financial transaction queries (14.6K) |
| `tx_wallet_payment_repository.go` | Wallet payment repository — within-transaction operations (13.6K) |
| `wallet_payment_repository.go` | Wallet payment repository — payment tracking (10.8K) |
| `wallet_topup_repository.go` | Wallet topup repository — topup tracking (4.8K) |
| `ledger_repository_crud.go` | Ledger CRUD — double-entry entry management (7.3K) |
| `ledger_repository_balance.go` | Ledger balance — account balance queries (5.5K) |
| `ledger_repository_analytics.go` | Ledger analytics — financial reporting queries (10.6K) |
| `user_repository.go` | User repository — authentication and user management (14.6K) |
| `payrate_repository.go` | Payrate repository — rate configuration with temporal queries (11.9K) |
| `notification_repository.go` | Notification repository — notification delivery tracking (4.5K) |
| `settlement_repository.go` | Settlement repository — payment settlement tracking (5.3K) |
| `security_repository.go` | Security repository — login attempts, security events (7.7K) |
| `settings_repository.go` | Settings repository — dynamic configuration (3.1K) |
| `loan_repository.go` | Loan repository — loan management (4.1K) |
| `lender_repository.go` | Lender repository — lender management (3.2K) |
| `bank_repository.go` | Bank repository — bank list management (3.6K) |
| `asset_repository.go` | Asset repository — asset tracking, emits `AssetCreatedEvent` (5.4K) |
| `attendance_repository.go` | Attendance repository — check-in/out records (3.8K) |
| `api_metric_repository.go` | API metric repository — endpoint performance tracking (21K) |
| `cron_job_status_repository.go` | Cron job status — job execution tracking (2.4K) |
| `push_subscription_repository.go` | Push subscription repository — Web Push subscriptions (1.3K) |
| `employee_repository.go` | Employee repository constructor (1K) |
| `employee_user_repository.go` | Employee-user link repository (2.9K) |
| `project_user_repository.go` | Project-user link repository (2.8K) |
| `wallet_ipn_repository.go` | Wallet IPN repository — instant payment notification tracking (1.7K) |
| `timesheet_edit_request_repository.go` | Timesheet edit request repository (5.3K) |
| `transaction_code_repository.go` | Transaction code repository — categorized codes (3.0K) |
| `loan_repayment_schedule_repository.go` | Loan repayment schedule repository (3.2K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `common/` | Shared helpers — query builders, batch processor, error handler, filter builder, transaction helper |
| `query_builders/` | Complex query builders — employee, project, timesheet, ledger, employee statistics |
| `relationship_loaders/` | Eager loading — timesheet relationship loader |
| `repositories/` | Split repository pattern — timesheet analytics/command/query repositories |

## For AI Agents

### Working In This Directory
- Repositories implement interfaces from `domain/ports/` — check port interfaces before adding methods
- Use `common/filter_builder.go` for building dynamic WHERE clauses
- Use `common/batch_processor.go` for batch insert/update operations
- Complex queries should use dedicated query builders in `query_builders/`
- The timesheet repository is split into three files in `repositories/` (analytics, command, query) due to size
- Always use `db.WithContext(ctx)` for GORM operations to respect context cancellation
- Cache invalidation: `ProjectEmployeeService.UpdateAssignment` invalidates Redis key `assignment:{projectID}:{employeeID}` after commit

### Testing Requirements
- Repository tests use real database connection
- Some repository tests in `advance_payment_request_repository_test.go` and `provider_transaction_repository_test.go`
- Filter builder has tests in `common/`

### Common Patterns
```go
// Repository with GORM
type myRepository struct {
    db *gorm.DB
}

// Use context, handle errors, map to domain
func (r *myRepository) FindByID(ctx context.Context, id string) (*domain.MyEntity, error) {
    var model MyModel
    if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, domain.NewNotFoundError("entity not found")
        }
        return nil, err
    }
    return model.ToDomain(), nil
}
```

## Dependencies

### Internal
- `internal/domain` — implements port interfaces
- `internal/domain/ports` — repository interface contracts
- `internal/pkg/clock` — for timestamp generation
- `internal/config` — database configuration

### External
- `gorm.io/gorm` — ORM framework
- `gorm.io/driver/mysql` — MySQL driver
- `redis/go-redis` — caching layer

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
