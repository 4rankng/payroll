# ADR-008: Casbin RBAC Authorization

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system has three primary user roles with different access levels, plus a specialized external partner role. Authorization rules are complex: some endpoints are available to all authenticated users, some are role-specific, and some have deny-override rules (e.g., partners can view timesheets but cannot bulk-approve/reject).

A simple middleware check (`if role == "admin"`) would not scale to these nuanced rules and would scatter authorization logic across handlers.

## Decision

Use **Casbin** for RBAC authorization with a **deny-override** policy model.

### Configuration

| File | Purpose |
|------|---------|
| `backend/configs/casbin_model.conf` | RBAC model definition with role inheritance, deny-override |
| `backend/configs/casbin_policy.csv` | Policy definitions (role, path, method, allow/deny) |

### Roles

| Role | Access |
|------|--------|
| `admin` | Full wildcard access: `/api/*, *, allow` |
| `partner` | Scoped access: projects, employees, timesheets, payrates. Explicit **deny** on bulk-approve, bulk-reject, reset |
| `adv_partner` | External partner for advance payment pipeline: import, view/export, no financial actions |
| `employee` | Self-service only: `/api/v1/me/*` |

All authenticated users can access notifications and push subscriptions.

### Model Features

- **`keyMatch2`** for URL path matching (supports path parameters).
- **`*` wildcard** for HTTP methods.
- **Deny-override**: deny rules take precedence over allow rules (defense in depth).

### Middleware

The `Authorization(casbinEnforcer)` middleware in `transport/http/middleware/authorization.go` (6.6K) checks every protected route. It runs after the `Auth` middleware, which sets the user's role in the context.

## Consequences

**Positive:**
- Authorization rules are centralized in CSV files — easy to audit and modify.
- Deny-override provides defense in depth: even if an allow rule is too broad, a deny rule catches it.
- Adding a new role or changing permissions doesn't require code changes — just update the CSV.
- The middleware is applied uniformly — no handler forgets to check permissions.

**Negative:**
- The CSV file must be kept in sync with route changes. A new endpoint without a Casbin policy row is denied by default (safe, but can confuse developers).
- `keyMatch2` syntax has a learning curve.
- Testing all role/path combinations requires integration tests (covered by `flow_auth_user.go`).

## Alternatives Considered

1. **Hand-coded middleware per role** — Rejected. Would scatter authorization logic across handlers and be error-prone.
2. **ACL lists per endpoint** — Rejected. Doesn't scale to 100+ endpoints and 4 roles.
3. **OPA (Open Policy Agent)** — Rejected. Adds a separate service and Rego language dependency. Casbin is embedded in the Go binary.
4. **Spring Security-style annotations** — Rejected. Go doesn't have a dominant equivalent, and Casbin's centralized policy is easier to audit.
