<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# transport — Transport Layer

## Purpose
Container directory for all transport protocol implementations. Currently contains only the HTTP transport layer (Gin-based), but structured to allow future additions (e.g., gRPC, WebSocket). The transport layer is the outermost ring in the architecture — it receives external requests, applies middleware, delegates to application services via handlers, and returns HTTP responses.

## Key Files
_None at this level — all code is in subdirectories._

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `http/` | HTTP API layer — handlers, middleware, response utilities, validation (see `http/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- Transport depends on `app/services` and `domain` — never the reverse
- Handlers translate HTTP concerns to domain/application calls
- Keep transport logic thin: parse request → call service → format response
- Error translation happens in `http/response/error_translator.go`

### Testing Requirements
- Handler behavior tested through integration tests in `tests/integration/`
- Middleware has dedicated unit tests

### Common Patterns
```go
// Handler pattern
func (h *MyHandler) Create(c *gin.Context) {
    var req dto.CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    result, err := h.service.Create(c.Request.Context(), req)
    if err != nil {
        response.HandleError(c, err)
        return
    }
    response.Created(c, result)
}
```

## Dependencies

### Internal
- `internal/app/services` — application services
- `internal/app/dto` — request/response types
- `internal/domain` — domain errors for error translation

### External
- `gin-gonic/gin` — HTTP framework

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
