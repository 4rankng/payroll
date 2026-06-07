<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# http — HTTP Layer

## Purpose
The complete HTTP API layer built on Gin. Contains request handlers grouped by domain, middleware for cross-cutting concerns (authentication, RBAC, rate limiting, security headers), request/response helpers, error translation, and request validation. All API endpoints are registered in `bootstrap/routes.go` and served on port 8080.

## Key Files
_None at this level — all code is in subdirectories._

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `handlers/` | HTTP request handlers — 15+ handler groups by domain (see `handlers/AGENTS.md`) |
| `middleware/` | HTTP middleware — auth, RBAC, rate limiting, security (see `middleware/AGENTS.md`) |
| `response/` | Response utilities — error translation, standardized response formatting |
| `helpers/` | Request helpers — context parsing, pagination, formatting, request binding |
| `validation/` | Request validation — reusable validation rules |

## For AI Agents

### Working In This Directory
- Use `response.BadRequest(c, msg)`, `response.NotFound(c, msg)`, etc. for consistent error responses
- Use `helpers.ParsePagination(c)` for list endpoints
- Use `validation.Validate(req)` for request validation
- Error translation in `response/error_translator.go` maps domain errors to HTTP status codes
- Always use `c.Request.Context()` for context propagation

### Testing Requirements
- Integration tests in `tests/integration/` exercise handlers through HTTP
- Middleware has unit tests in `middleware/`
- Response error translator has comprehensive tests

### Common Patterns
```go
// Standard handler flow
func (h *MyHandler) List(c *gin.Context) {
    ctx := c.Request.Context()
    pagination := helpers.ParsePagination(c)
    filter := helpers.ParseFilter(c)

    result, err := h.service.List(ctx, filter, pagination)
    if err != nil {
        response.HandleError(c, err)
        return
    }
    response.OK(c, result)
}
```

## Dependencies

### Internal
- `internal/app/services` — business logic delegation
- `internal/app/dto` — request/response types
- `internal/domain` — domain errors
- `internal/pkg/clock` — for time-dependent handlers

### External
- `gin-gonic/gin` — HTTP framework

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
