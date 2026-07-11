# ADR-001: Domain-Driven Design with Clean Architecture

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system handles complex business logic: timesheet calculation, advance payment fees, double-entry ledger accounting, wallet state machines, and payment disbursement. This logic must remain testable, framework-agnostic, and protected from infrastructure churn.

Without strict layer boundaries, business logic tends to leak into handlers (HTTP concerns) or repositories (database concerns), making it hard to test in isolation and fragile to framework upgrades.

## Decision

The backend follows **Domain-Driven Design (DDD) with Clean Architecture**. Dependencies point strictly inward:

```
transport → app → domain ← infra
```

Layer responsibilities:

| Layer | Responsibility | Key Rule |
|-------|---------------|----------|
| `domain/` | Entities, value objects, domain services, ports, specs | **Zero framework imports** (no Gin, no GORM). Only Go stdlib and internal packages. |
| `app/services/` | Application services orchestrating use cases | Depends on domain ports and domain services. Never touches GORM models directly. |
| `infra/persistence/` | GORM repositories implementing domain ports | Depends on GORM. Maps GORM models → domain types via `ToDomain()`. |
| `transport/http/handlers/` | HTTP handlers | Depends on app services and domain types only — never touches GORM models directly. |
| `adapters/` | Anti-corruption layer for external contracts | Isolates third-party API shapes from domain types. |

The domain package defines ports (interfaces) that infrastructure implements. This inverts the dependency — infra depends on domain, not the other way around.

## Consequences

**Positive:**
- Domain logic is fully testable without a database or HTTP server.
- Framework upgrades (Gin, GORM) only affect infra and transport layers.
- Business rules are centralized and discoverable.
- New developers can understand the domain without reading infrastructure code.

**Negative:**
- More files and indirection than a flat structure.
- Mapping between GORM models and domain types adds boilerplate.
- Learning curve for developers unfamiliar with DDD/Clean Architecture.

## Alternatives Considered

1. **MVC (Model-View-Controller)** — Rejected. Business logic would spread across models and controllers. No clear boundary between HTTP and domain.
2. **Hexagonal (Ports and Adapters) without DDD** — Partially adopted. The ports pattern is used (`domain/ports/`), but DDD aggregates (e.g., wallet) provide richer domain modeling than pure hexagonal.
3. **Flat package structure** — Rejected. Would not scale to 26+ service packages and 90+ domain entities.
