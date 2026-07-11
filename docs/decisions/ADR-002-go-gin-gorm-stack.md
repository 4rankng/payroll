# ADR-002: Go + Gin + GORM + MySQL 8 Stack

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll backend needs to handle financial transactions, background job processing, and serve a React SPA. The system processes bulk payments, double-entry accounting, and integrates with Vietnamese payment providers (OnePay, 9Pay). Reliability, type safety, and performance are critical — this is a financial system.

## Decision

The backend is written in **Go 1.26** using:
- **Gin** (`github.com/gin-gonic/gin v1.10.1`) as the HTTP framework.
- **GORM** (`gorm.io/gorm v1.30.1` with `gorm.io/driver/mysql v1.6.0`) as the ORM.
- **MySQL 8** as the primary database.

The Go module is `api-server` (import path: `api-server/internal/...`).

Production server is **x86_64/amd64 only** — never build arm64 Docker images.

## Consequences

**Positive:**
- Go's static typing and compile-time safety reduce runtime errors in financial logic.
- Gin provides fast HTTP routing with a mature middleware ecosystem.
- GORM offers a familiar ORM pattern with migrations, hooks, and preloading.
- MySQL 8 brings window functions, CTEs, and improved JSON support.
- Go's goroutine model enables non-blocking event publishing and audit logging.
- Single binary deployment simplifies Docker images.

**Negative:**
- GORM's auto-migration is not used — raw SQL `.up.sql` files are the migration system (by design, for control).
- Go's error handling verbosity adds boilerplate.
- GORM can hide N+1 queries if `Preload` is not used correctly (mitigated by code review).

## Alternatives Considered

1. **Node.js / TypeScript (Express/NestJS)** — Rejected. Go's performance, goroutine model, and compile-time safety are better suited for a financial system with background jobs.
2. **Python (FastAPI)** — Rejected. While excellent for rapid development, Go's type safety and single-binary deployment are preferable for a production financial system.
3. **Java (Spring Boot)** — Rejected. Heavier runtime, higher memory footprint, slower startup. The team has strong Go expertise.
4. **sqlx instead of GORM** — Rejected for initial development. GORM's hooks, preloading, and migration tooling accelerated development. Raw SQL is still used in migrations and complex queries where GORM is insufficient.
