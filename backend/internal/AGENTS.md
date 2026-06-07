<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# internal — Core Application Source

## Purpose
Container directory for all internal Go packages. Follows DDD layering where dependencies point inward: `transport` → `app` → `domain` ← `infra`. The `domain` package has zero external dependencies; `infra` implements domain ports; `app` orchestrates use cases; `transport` delivers HTTP APIs. Shared utilities live in `pkg`, adapters in `adapters`, configuration in `config`, and seed data in `seed`.

## Key Files
_None at this level — all code is in subdirectories._

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `adapters/` | Anti-corruption layer — ports for external system contracts (events, cache, notification, services) |
| `app/` | Application layer — bootstrap, services, DTOs, workers (see `app/AGENTS.md`) |
| `config/` | Environment-based configuration loading |
| `domain/` | Domain layer — entities, value objects, events, ports, domain services (see `domain/AGENTS.md`) |
| `infra/` | Infrastructure layer — persistence, events, disbursement, email, storage (see `infra/AGENTS.md`) |
| `pkg/` | Shared utility packages — clock, constants, retry, validation (see `pkg/AGENTS.md`) |
| `seed/` | Database seed data for initial setup and testing |
| `transport/` | Transport layer — HTTP handlers, middleware, routing (see `transport/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- **Dependency direction:** `transport` → `app` → `domain` ← `infra`. Never import `infra` from `domain`.
- Domain types are the source of truth — handlers use domain types, not GORM models
- New features typically touch all layers: domain entity → domain port → infra repository → app service → transport handler → bootstrap wiring → route registration
- The `adapters/` package provides interfaces for external systems to decouple from infrastructure details

### Testing Requirements
- Domain services have unit tests within `domain/services/` and `domain/` itself
- Integration tests in `tests/integration/` exercise the full stack through HTTP

### Common Patterns
```
Adding a new feature:
1. domain/entity.go         — Define entity, errors, events
2. domain/ports/             — Define repository interface
3. infra/persistence/        — Implement repository with GORM
4. app/services/feature/     — Create application service
5. transport/http/handlers/  — Create HTTP handler
6. app/bootstrap/            — Wire everything in container.go and routes.go
```

## Dependencies

### Internal
All subpackages are internal to the `api-server` module.

### External
None at this level — each subdirectory has its own external dependencies.

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
