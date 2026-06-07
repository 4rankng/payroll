<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# middleware — HTTP Middleware

## Purpose
Contains all HTTP middleware that executes before request handlers. Provides cross-cutting concerns: JWT authentication, Casbin-based RBAC authorization, rate limiting, IP whitelisting for webhooks, security headers, request timeout, error recovery, audit context propagation, and tenant-aware concurrency control.

## Key Files
| File | Description |
|------|-------------|
| `auth.go` | JWT authentication middleware — extracts and validates JWT tokens from Authorization header, sets user context (2.5K) |
| `authorization.go` | RBAC authorization middleware — Casbin policy enforcement, checks user role against endpoint permissions (6.6K) |
| `authorization_test.go` | Authorization tests — verifies role-based access control |
| `rate_limit.go` | Rate limiting middleware — per-IP and per-user request throttling (3.1K) |
| `ip_whitelist.go` | IP whitelist middleware — restricts webhook endpoints to trusted provider IPs (2.3K) |
| `ip_whitelist_test.go` | IP whitelist tests — verifies allowed/denied IPs |
| `security_headers.go` | Security headers middleware — adds X-Content-Type-Options, X-Frame-Options, CSP headers (1.6K) |
| `security_headers_test.go` | Security headers tests |
| `error_middleware.go` | Error recovery middleware — catches panics, logs stack traces, returns 500 (2.8K) |
| `error_middleware_test.go` | Error middleware tests |
| `request_timeout.go` | Request timeout middleware — enforces maximum request duration (441B) |
| `request_timeout_test.go` | Request timeout tests |
| `audit_context.go` | Audit context middleware — propagates audit metadata through request context (878B) |
| `audit_context_test.go` | Audit context tests |
| `tenant_semaphore.go` | Tenant semaphore middleware — limits concurrent requests per tenant (1.5K) |
| `api_metrics.go` | API metrics middleware — records request latency, status codes, endpoint metrics (6.1K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `logs/` | Middleware processing logs (gitignored) |

## For AI Agents

### Working In This Directory
- Middleware chain is configured in `bootstrap/middleware.go`
- **Execution order matters:** recovery → security headers → logging → auth → RBAC → rate limit → handler
- Auth middleware sets user info in context: `c.Set("user_id", ...)` and `c.Set("role", ...)`
- Authorization uses Casbin policies from `configs/casbin_policy.csv`
- IP whitelist is applied only to webhook routes (payment provider IPN endpoints)
- Rate limiting uses Redis for distributed rate tracking
- Audit context middleware sets `actor_id` and `action` for audit trail

### Testing Requirements
- Each middleware has dedicated unit tests (`*_test.go`)
- Integration tests verify the full middleware chain end-to-end
- Authorization tests in `tests/integration/flow_auth_user.go`

### Common Patterns
```go
// Middleware registration in bootstrap/middleware.go
router.Use(
    middleware.ErrorRecovery(),
    middleware.SecurityHeaders(),
    middleware.RequestTimeout(30*time.Second),
    middleware.APIMetrics(metricsCollector),
)

// Protected routes use auth + RBAC
api := router.Group("/api/v1")
api.Use(middleware.Auth(jwtSecret))
api.Use(middleware.Authorization(casbinEnforcer))

// Webhook routes use IP whitelist
webhook := router.Group("/webhooks")
webhook.Use(middleware.IPWhitelist(allowedIPs))
```

## Dependencies

### Internal
- `internal/config` — JWT secret, rate limit config, allowed IPs
- `internal/domain` — user context types
- `internal/infra/observability` — logging
- `internal/infra/persistence` — Redis for rate limiting, metrics storage
- `configs/` — Casbin model and policy files

### External
- `gin-gonic/gin` — middleware interface (`gin.HandlerFunc`)
- `casbin/casbin` — RBAC enforcement
- `golang-jwt/jwt` — JWT token validation

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
