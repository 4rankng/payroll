<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# configs — RBAC Configuration

## Purpose
Contains Casbin model and policy files for Role-Based Access Control (RBAC). Casbin enforces authorization rules on API endpoints, checking user roles against permitted actions.

## Key Files
| File | Description |
|------|-------------|
| `casbin_model.conf` | Casbin model definition (RBAC with subroles). Defines the policy effect and matching function. |
| `casbin_policy.csv` | Casbin policy rules mapping roles to allowed API endpoints/resources. ~4.8K of role-permission mappings. |

## Subdirectories
_None_

## For AI Agents

### Working In This Directory
- Policy changes require server restart to take effect (loaded at bootstrap)
- The Casbin enforcer is initialized in `internal/app/bootstrap/infrastructure/init.go`
- Middleware `internal/transport/http/middleware/authorization.go` checks each request against these policies
- When adding new admin endpoints, add corresponding policy rows to `casbin_policy.csv`

### Testing Requirements
- Authorization tests in `tests/integration/flow_auth_user.go` verify RBAC enforcement
- Test both allowed and denied access for new endpoints

### Common Patterns
```
# casbin_policy.csv format
p, role, /api/v1/resource, GET
p, role, /api/v1/resource, POST
```

## Dependencies

### Internal
- `internal/app/bootstrap/infrastructure/init.go` — Casbin enforcer initialization
- `internal/transport/http/middleware/authorization.go` — Request authorization check

### External
- `casbin/casbin` — Authorization library
- `casbin/gorm-adapter` — GORM-based policy storage

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
