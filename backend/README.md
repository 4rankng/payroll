# Payroll Backend

A modular monolith backend for payroll management built with Go, following Domain-Driven Design (DDD) principles.

## Quick Start

```bash
# Install dependencies
go mod download

# Copy environment variables
cp .env.example .env

# Start MySQL and Redis
docker-compose up -d

# Run migrations
make migrate-up

# Start the server
make run
```

## Architecture

This project uses a **modular monolith** architecture with clear separation of concerns:

- ✅ **Transport Layer**: HTTP handlers (Gin framework)
- ✅ **Service Layer**: Business logic and orchestration
- ✅ **Repository Layer**: Data access and persistence
- ✅ **Domain Layer**: Business entities and interfaces
- ✅ **Infrastructure**: Cross-cutting concerns (logging, caching, etc.)

### Key Achievements

- ✅ Container reduced from 643 → 208 lines (-67.7%)
- ✅ Bootstrap modules for clean initialization
- ✅ Common components (FilterBuilder, BatchProcessor, ErrorHandler)
- ✅ Zero linting errors
- ✅ All tests passing

## Documentation

### Core Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Comprehensive architecture overview
  - Architecture layers and directory structure
  - Design patterns (Repository, DI, CQRS, Event-Driven)
  - Key concepts and best practices
  - Performance considerations

- **[docs/architecture/service-creation.md](docs/architecture/service-creation.md)** - Service creation guide
  - How to add a new service
  - Simple vs Config patterns
  - Event-driven services
  - Testing strategies
  - Troubleshooting guide

- **[docs/service-analysis.md](docs/service-analysis.md)** - Service analysis
  - Analysis of large services
  - Decomposition opportunities
  - Config pattern recommendations

### Code Examples

- **[examples/README.md](examples/README.md)** - Code examples
  - Simple Service Pattern (this example)
  - Config Service Pattern (for 7+ dependencies)
  - Repository with FilterBuilder
  - CQRS Pattern
  - Event-Driven Services
  - Batch Processing

## Project Structure

```
payroll-backend/
├── cmd/                          # Application entry points
├── internal/
│   ├── app/
│   │   ├── bootstrap/            # Application initialization
│   │   │   ├── container.go      # Main container (208 lines)
│   │   │   ├── infrastructure/   # DB, Redis, EventBus
│   │   │   ├── repositories/    # Repository initialization
│   │   │   └── services/        # Service initialization
│   │   └── services/            # Business logic
│   ├── config/                  # Configuration
│   ├── domain/                  # Domain entities & interfaces
│   ├── infra/                   # Infrastructure
│   │   ├── persistence/
│   │   │   ├── common/          # Shared components
│   │   │   ├── repositories/    # Repository implementations
│   │   │   └── query_builders/  # Query builders
│   │   └── observability/       # Logging & metrics
│   └── transport/               # HTTP layer
├── docs/                        # Documentation
├── examples/                    # Code examples
└── tests/                       # Tests
```

## Key Components

### Bootstrap Modules

- **Infrastructure** (`bootstrap/infrastructure/`) - DB, Redis, EventBus initialization
- **Repositories** (`bootstrap/repositories/`) - Repository initialization
- **Services** (`bootstrap/services/`) - Service initialization

### Common Components

- **FilterBuilder** - Composable filter functions for queries
- **BaseQueryBuilder** - Template for query builders
- **BatchProcessor** - Centralized batch processing
- **RepoErrorHandler** - Standardized error handling

### Design Patterns

- Repository Pattern - Data access abstraction
- Dependency Injection - Loose coupling
- CQRS - Separate read/write operations
- Strategy Pattern - Interchangeable algorithms
- Observer Pattern - Event-driven communication
- Facade Pattern - Simplified interfaces

## Getting Started

### Adding a New Service

1. **Choose your pattern**:
   - Simple Pattern (<7 dependencies)
   - Config Pattern (7+ dependencies)

2. **Create the service**:
   ```bash
   # Follow the guide
   docs/architecture/service-creation.md
   ```

3. **Add to bootstrap**:
   - Add repository to `bootstrap/repositories/`
   - Add service to `bootstrap/services/`
   - Add handler to `bootstrap/container.go`

4. **Test your service**:
   ```bash
   go test ./internal/app/services/your-service/...
   ```

### Code Examples

See the `examples/` directory for working examples:

```bash
# Simple service pattern
cd examples/simple_service
go test -v

# Config service pattern (7+ dependencies)
cd examples/config_service
go test -v
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/app/services/employee/...

# Run bootstrap tests
go test ./internal/app/bootstrap/...
```

### Linting

```bash
# Run linting
make lint

# Format code
gofmt -s -w .
```

### Database Operations

```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create a new migration
make migrate-create NAME=create_users_table
```

## Configuration

Configuration is managed through environment variables and config files:

- `.env` - Local development environment
- `internal/config/` - Configuration structs

### Key Configuration

```env
# Database
DATABASE_DSN=root:password@tcp(localhost:3306)/payroll_db

# Redis
REDIS_ADDRESS=localhost:6379

# Server
SERVER_PORT=8080
SERVER_MODE=debug
```

## Performance

### Optimization Techniques

- Batch processing for large datasets
- Connection pooling (DB, Redis)
- Eager loading to prevent N+1 queries
- Pagination for large result sets
- Caching frequently accessed data
- Index optimization

### Monitoring

- Structured logging with `log/slog`
- Performance metrics
- Error tracking
- Database query logging

## Testing Strategy

### Test Types

- Unit tests - Test individual components
- Integration tests - Test component interactions
- E2E tests - Test complete workflows

### Coverage Goals

- Common components: 80%+
- Repositories: 60%+
- Services: 60%+
- Handlers: 40%+

## Contributing

1. Follow the code style guide
2. Write tests for new features
3. Update documentation
4. Run linting before committing
5. Create pull requests for review

### Code Review Checklist

- [ ] Tests pass
- [ ] Linting passes
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
- [ ] Performance considered

## Troubleshooting

### Common Issues

**Circular Dependencies**
- Extract common logic to separate service
- Use event-driven architecture
- Create interface for dependency

**Constructor Explosion**
- Use Config pattern for 7+ dependencies
- Group related dependencies

**Testing Complex Services**
- Create interfaces for all dependencies
- Use mock implementations
- Focus on one behavior per test

## Resources

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Architecture overview
- **[docs/architecture/service-creation.md](docs/architecture/service-creation.md)** - Service creation guide
- **[docs/service-analysis.md](docs/service-analysis.md)** - Service analysis
- **[examples/](examples/)** - Code examples

## License

Internal project - All rights reserved
