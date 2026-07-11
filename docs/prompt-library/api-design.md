# Prompt: API Design

Use when adding a new API endpoint end-to-end.

## Prompt

```
Goal: Add [METHOD] /api/v1/[PATH] — [PURPOSE]

Context to gather:
1. Read docs/api.md for route groups and middleware chain
2. Read docs/decisions/ADR-001-ddd-clean-architecture.md for layer rules
3. Read docs/decisions/ADR-008-casbin-rbac-authorization.md for authz
4. Find the relevant route file: backend/internal/app/bootstrap/routes_[domain].go
5. Find the relevant handler directory: backend/internal/transport/http/handlers/[domain]/
6. Read an existing handler in the same domain as a reference pattern
7. Read internal/app/dto/AGENTS.md for DTO conventions

Constraints:
- Base path: /api/v1
- Handlers depend on app services and domain types only — never GORM models
- DTOs in internal/app/dto/
- Domain errors map to correct HTTP status (404, 400, 403, 409, 500)
- No sensitive data in responses or logs
- All business time uses clock.Clock
- Pagination on list endpoints (page, pageSize query params)
- Vietnamese for any user-facing error messages

Implementation steps:
1. Define request/response DTOs in internal/app/dto/[feature].go
2. Add service method in internal/app/services/[domain]/
3. Add handler in internal/transport/http/handlers/[domain]/
4. Register route in internal/app/bootstrap/routes_[domain].go
5. Add Casbin policy row in configs/casbin_policy.csv
6. Write unit tests for the service (FakeClock)
7. Add integration test flow in backend/tests/integration/

Output format:
- Endpoint: METHOD /api/v1/path
- Request body: { field: type, ... }
- Response body: { field: type, ... }
- Error responses: 400 (validation), 403 (forbidden), 404 (not found), 409 (conflict)
- Required role: admin | partner | employee | adv_partner
- Casbin policy row needed

Verification:
- go build ./... passes
- go test ./... -race passes
- make api-test passes (new flow test)
- Endpoint returns correct status codes
- Casbin denies unauthorized roles
- Pagination works on list endpoints
```

## Example

```
Goal: Add GET /api/v1/expenses — list project expenses with pagination

Request: GET /api/v1/expenses?projectId=123&page=1&pageSize=20
Response 200: { data: [...], total: 100, page: 1, pageSize: 20 }
Error 403: partner accessing another partner's project
Required role: admin, partner
Casbin: /api/v1/expenses, GET, allow (admin, partner)
```
