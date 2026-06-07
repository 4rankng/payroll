<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# infra — Infrastructure Layer

## Purpose
Implements the domain's port interfaces with concrete technology adapters. Contains database repositories (GORM/MySQL), event bus (Redis streams), payment provider adapters (OnePay, 9Pay), email delivery, file storage, background job processing (asynq), database transaction management, and observability (logging, metrics). Infrastructure code depends on the domain layer but is never imported by it.

## Key Files
_None at this level — all code is in subdirectories._

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `persistence/` | GORM repositories, query builders, database connection (see `persistence/AGENTS.md`) |
| `events/` | Event bus implementation, event handlers, outbox pattern (see `events/AGENTS.md`) |
| `disbursement/` | Payment provider adapters — 9Pay (sandbox) and OnePay (production) (see `disbursement/AGENTS.md`) |
| `asynq/` | Background job client, task handlers, mux server (see `asynq/AGENTS.md`) |
| `email/` | Email sending — Resend provider (production) and sandbox provider (dev) (see `email/AGENTS.md`) |
| `transaction/` | GORM-based transaction manager and unit of work (see `transaction/AGENTS.md`) |
| `storage/` | File storage abstraction for uploads and exports |
| `observability/` | Logging (`slog`), DB metrics, Prometheus metrics |

## For AI Agents

### Working In This Directory
- **Dependency rule:** `infra` depends on `domain` (implements ports), never the reverse
- Repository implementations go in `persistence/` — implement interfaces from `domain/ports/`
- Event handlers go in `events/` — implement `domain.EventHandler` interface
- New external service integrations: define interface in `domain/ports/`, implement in `infra/`
- Observability: use `observability.GetLogger()` for structured logging; never use `fmt.Printf`

### Testing Requirements
- Repository tests use real database (integration test style)
- Event handler tests in `events/` verify correct event processing
- Disbursement provider tests mock HTTP calls to payment providers

### Common Patterns
```go
// Repository implementation
type myRepository struct {
    db *gorm.DB
}

func NewMyRepository(db *gorm.DB) domain.MyRepository {
    return &myRepository{db: db}
}

func (r *myRepository) FindByID(ctx context.Context, id string) (*domain.MyEntity, error) {
    var model MyModel
    if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return model.ToDomain(), nil
}
```

## Dependencies

### Internal
- `internal/domain` — implements port interfaces defined here
- `internal/pkg/clock` — for timestamp generation
- `internal/config` — for infrastructure configuration

### External
- `gorm.io/gorm` — ORM
- `redis/go-redis` — caching and event streaming
- `hibiken/asynq` — background jobs
- `resend/resend-go` — email
- `prometheus/client_golang` — metrics

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
