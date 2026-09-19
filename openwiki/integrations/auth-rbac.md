---
type: integration
title: Auth & RBAC
description: JWT issuance and refresh, the Casbin deny-override policy model, role hierarchy (Admin / Partner / Adv-Partner / Employee), project scoping at handler level, and password reset / OTP plumbing.
tags: [auth, jwt, casbin, rbac, partner-scope, middleware]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-410ab36600fefdde765cb24e
    resource: repo://backend/AGENTS.md
  - id: openwiki-source-a4b921e77bb61f1a7a390390
    resource: repo://docs/decisions/ADR-008-casbin-rbac-authorization.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Integration: Auth & RBAC

Authentication is JWT-based. Authorization is Casbin-based with a deny-override policy model and project-scope checks at the handler layer. Together they enforce who can call which endpoint and which rows a partner can see within that endpoint.

ADR-008 is the source of truth for the RBAC design.

## Configuration

| File | Purpose |
|------|---------|
| `backend/configs/casbin_model.conf` | RBAC model definition with role inheritance and deny-override |
| `backend/configs/casbin_policy.csv` | Policy rows: `(role, path, method, allow/deny)` |
| `backend/internal/transport/http/middleware/authorization.go` | Casbin enforcer middleware |
| `backend/internal/transport/http/middleware/auth.go` | JWT middleware |

The model uses `keyMatch2` for URL path matching (so `:id` segments are matched) and `*` for HTTP methods. Deny rules win over allow rules (defense in depth). A new endpoint without a policy row is denied by default — safe by construction, mildly annoying for developers.

## Roles

| Role | Scope |
|------|-------|
| `admin` | Full wildcard access: `/api/*, *, allow` |
| `partner` | Project-scoped: projects, employees, timesheets, payrates. Explicit **deny** on bulk-approve, bulk-reject, reset. |
| `adv_partner` | External partner for the advance-payment pipeline: import, view/export, no financial actions. |
| `employee` | Self-service only: `/api/v1/me/*`, plus employee-mobile flows (check-in/out, advance request, salary slip). |
| Authenticated (any role) | Notifications and push subscription endpoints. |

Adding a new role or changing permissions is a CSV edit — no Go code change required. Integration tests cover the role/path matrix in `backend/tests/integration/flow_auth_user.go`.

## JWT lifecycle

`backend/internal/app/services/auth/` owns:

- Issue on successful login (`/api/v1/auth/login` → `/api/v1/auth/login/verify`).
- Refresh via the standard `Authorization: Bearer` flow.
- Logout (token blacklist) — `/api/v1/auth/logout`.
- Profile endpoints — `/api/v1/auth/me`, `PUT /api/v1/auth/me`.
- Change-password — `/api/v1/auth/change-password`.
- CAPTCHA gate after 3 failed logins — `/api/v1/auth/captcha`.
- Google OAuth — `/api/v1/auth/google` (no OTP required).

Tokens carry the user ID and role. The `auth` middleware extracts them, validates the signature, and sets the actor on the Gin context. Subsequent middleware (authorization, partner-scope, audit context) reads the actor from there.

## Middleware chain (auth-relevant slice)

From `internal/transport/http/middleware/`, in order:

1. **auth** (`auth.go`) — JWT validation; sets actor on context.
2. **authorization** (`authorization.go`) — Casbin enforcer; deny-override.
3. **partner scope** — handler-level checks via `isScopedPartnerRole` + `CanUserAccessProject` / `CanUserAccessEmployee`. Both Casbin and the project-scope checks are required for partner-aware endpoints.
4. **audit context** (`audit_context.go`) — sets per-request audit context.

The rate-limit, security-headers, and timeout middleware run before auth. Recovery is global. See `architecture/transport-http.md` for the full chain.

## Partner scoping — the second layer

Casbin decides which endpoints a partner can call. It cannot decide which rows within that endpoint the partner can see. So partner-aware handlers run a second check after authorization succeeds:

```go
if isScopedPartnerRole(ctx) {
    if !CanUserAccessProject(ctx, projectID) {
        return ErrForbidden
    }
}
```

The predicates (`CanUserAccessProject`, `CanUserAccessEmployee`) live in `internal/app/services/user/` and read the partner's project assignment. They are called inside the handler before any data is returned. The frontend never re-implements this filter — it trusts the server to give back only the rows the partner is allowed to see.

The bank-transfer history endpoint (`/api/v1/payrolls/bank-transfer-histories`) is the canonical example: same response shape as admin, server-filtered to accessible projects.

## Self-service password reset

`/api/v1/auth/password-reset/request` emails a single-use magic link (30-min TTL, SHA-256-hashed token stored in Redis under `pwreset:*`). The endpoint:

- Always returns 200 (anti-enumeration).
- Rate-limits 3 requests/hr per normalized email.
- Is mounted only when `PASSWORD_RESET_ENABLE=true`.

`/api/v1/auth/password-reset/confirm` consumes the token atomically via Redis `GETDEL`, updates the password, and invalidates every existing session for the user (`tokens_invalid_before`) in a single DB transaction.

The Zalo channel (`ZaloResetPassword.tsx`) is the parallel employee-mobile flow. See `integrations/zalo-otp.md`.

## Audit context

The audit middleware sets up a per-request audit context that handlers use to attach `actor_id`, `request_id`, and `ip` to non-blocking audit events. The non-blocking emit pattern is `go func() { _ = auditService.LogXxx(ctx, ...) }()` and always includes `data_type`, `record_count`, `file_name` for imports/exports. See `architecture/application-services.md`.

## Tests

- `authorization_test.go`, `authorization_otp_test.go` (middleware) — happy and deny paths.
- `ip_whitelist_test.go` — webhook IP allow-listing.
- `flow_auth_user.go` (integration) — full role/path matrix.
- `audit_context_test.go` — audit context propagation.
- `frontend/src/contexts/AuthContext.tsx` — login/logout/refresh on the client.

## Relationships

- Transport / HTTP — `architecture/transport-http.md`. The full middleware chain.
- Admin views — `frontend/admin-views.md`.
- Partner views — `frontend/partner-views.md`.
- Employee mobile — `frontend/employee-mobile.md`.
