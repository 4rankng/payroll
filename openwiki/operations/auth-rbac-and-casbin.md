---
type: operations
title: Auth, RBAC, and Casbin
description: JWT issuance and validation, Casbin policy model, deny-override rules, and the auth middleware chain.
tags: [auth, jwt, casbin, rbac, middleware, authorization]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-0984a00929c961138f6a1d20
    resource: repo://backend/configs/casbin_model.conf
  - id: openwiki-source-6fd4527459c9a156ece12912
    resource: repo://backend/configs/casbin_policy.csv
  - id: openwiki-source-b597dabc4f61ba30a0536915
    resource: repo://backend/internal/transport/http/middleware/auth.go
  - id: openwiki-source-a4b921e77bb61f1a7a390390
    resource: repo://docs/decisions/ADR-008-casbin-rbac-authorization.md
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Auth, RBAC, and Casbin

Authorization in the payroll backend uses **JWT** for stateless authentication and **Casbin** for role-based access control with a deny-override policy. Every protected route runs through the same middleware chain — auth → authorization → audit context — so handlers never need to remember to check permissions. ADR-008 records the rationale and the rejected alternatives (hand-coded middleware, per-endpoint ACLs, OPA, annotation-based).

## Middleware chain

The order matters; each layer assumes the previous one ran successfully:

1. **`Auth` middleware** (`backend/internal/transport/http/middleware/auth.go`) — extracts the JWT from the request, validates the signature and expiry, and sets `user_id` and `role` on the Gin context. Anonymous routes (login, forgot-password, public health endpoints) skip this step.
2. **`Authorization` middleware** (`backend/internal/transport/http/middleware/authorization.go`) — runs after auth. Resolves the request `(role, path, method)` tuple against the Casbin policy and returns 403 on denial. Fail-closed: a new endpoint with no policy row is denied by default.
3. **`Audit context` middleware** (`backend/internal/transport/http/middleware/audit_context.go`) — binds the authenticated `user_id` to the audit logger so subsequent mutations carry actor context.
4. **`Partner scoping` middleware** (where it applies) — for Partner and AdvPartner roles, enforces project-level access control beyond the role check. This is not Casbin — it queries the project-employee relationship at request time.

The chain is registered uniformly in `bootstrap/routes_*.go` so individual handlers do not opt in or out by accident.

## JWT issuance

`backend/internal/app/services/auth/` is the issuing side. Login, OTP login, password reset, and Zalo OA password reset all mint tokens through the same code path so the issuer, audience, expiry, and key id are consistent. Tokens carry `user_id`, `role`, and any project-scoped claims needed for partner authorization. Token refresh is handled by re-login; there is no long-lived refresh-token rotation today.

JWT secret handling must not appear in this page or in source comments — see `docs/standards/security.md` for the rules around secret rotation and storage.

## Role matrix

The four roles map to distinct access shapes:

| Role | Access |
|---|---|
| `admin` | Wildcard access: `/api/*` with any method, plus `/metrics` GET. |
| `partner` | Projects, employees, timesheets, payrates — scoped to assigned projects at query time. Denied on bulk-approve / bulk-reject / bulk-reset / reject-unpaid / approve-all / reset-all / edit-requests. |
| `adv_partner` | External partner for the advance-payment pipeline: import, view, export. No financial actions. |
| `accountant` | Read-only financial surface plus specific denials (e.g., denied on timesheet edit-requests). |
| `employee` | Self-service only: `/api/v1/me/*`. |

All authenticated users can read notifications and manage push subscriptions.

## Casbin policy model

The model definition in `backend/configs/casbin_model.conf`:

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act, eft

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

Three features deserve attention:

- **`keyMatch2`** matches URL paths with `:id` parameters. `/api/v1/users/:id` matches `/api/v1/users/42` but not `/api/v1/users/42/permissions`.
- **`*` wildcard** on HTTP method lets a single row cover all verbs.
- **Deny-override** is the policy effect. A deny rule beats any allow rule for the same `(role, path, method)`. This is the defense-in-depth mechanism that lets a coarse allow list be safely tightened by per-endpoint deny rows.

The policy file at `backend/configs/casbin_policy.csv` is the audit surface for authorization. Adding a new endpoint requires either a new allow row or accepting the safe default (denied). Changing a role's permission is a CSV edit, not a code change.

## Deny-override in practice

Partner timesheet access illustrates the pattern. The bulk rule (`p, partner, /api/v1/timesheets/*, *, allow`) would otherwise let a partner bulk-approve; instead the CSV adds explicit denies:

```
p, partner, /api/v1/timesheets/bulk-approve, *, deny
p, partner, /api/v1/timesheets/bulk-reject, *, deny
p, partner, /api/v1/timesheets/reject-unpaid, *, deny
p, partner, /api/v1/timesheets/bulk-reset, *, deny
p, partner, /api/v1/timesheets/approve-all, *, deny
p, partner, /api/v1/timesheets/reset-all, *, deny
p, partner, /api/v1/timesheets/edit-requests/*, *, deny
```

The deny rows beat the allow row because the policy effect requires "at least one allow AND no deny". Lint the CSV in code review whenever a new timesheet endpoint ships.

## Integration evidence

`backend/tests/integration/flow_auth_user.go` exercises the role/path combinations — login, refresh, change-password, role denial — across admin, partner, accountant, and employee tokens. New endpoints should add a corresponding flow file or extend this one.

The audit context middleware (`audit_context.go`) is what allows the rest of the system to log "who did what" without each handler passing the user id through explicitly.

## What lives outside Casbin

- **Project scoping** for partners — enforced by a partner-scoping middleware that queries the project-employee table. Casbin only knows roles; project membership is dynamic.
- **Record-level authorization** (e.g., "this partner owns this timesheet") — handled inside the application service or the repository query, not in the middleware.
- **Rate limiting** — Redis-backed per-endpoint limits, applied as a separate middleware before auth so anonymous abuse is rejected cheaply.

These belong in their respective subsystems; Casbin is the role gate, not the only authorization mechanism.

## Related pages

- [Architecture Overview](../architecture/overview.md) — where the middleware chain sits in the request lifecycle.
- [Frontend Role and View Architecture](../frontend/role-and-view-architecture.md) — how the SPA consumes the role to decide which routes and chrome to render.
- [Caching and Cache Invalidation](./caching-and-cache-invalidation.md) — interaction between audit context and the post-commit cache invalidation handler.
