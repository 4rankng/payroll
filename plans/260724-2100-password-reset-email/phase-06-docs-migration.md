---
phase: 6
title: "Docs & Migration"
status: pending
priority: P3
dependencies: [3, 5]
---

# Phase 6: Docs & Migration

## Overview

Update the evergreen docs to reflect the new endpoints and feature, and confirm no database migration is required (the feature is Redis-only). Add env vars to the deployment reference.

## Requirements

- **Functional:** `docs/api.md` lists the two new routes; `docs/system-architecture.md` mentions the reset flow at a high level; the deployment doc records the new env vars.
- **Non-functional:** No DB migration (confirmed: tokens live in Redis, and `users.tokens_invalid_before` already exists). Docs stay under 800 LOC per the `docs/AGENTS.md` convention.

## Architecture

### Database

**No migration needed.** The feature reuses:
- `users.email` (existing column, `varchar(255)`, nullable — only users with a non-null email can self-reset).
- `users.password` (existing column — updated in place by the confirm flow).
- `users.tokens_invalid_before` (existing column — set by `UpdateTokensInvalidBefore` to kill active sessions).

All ephemeral state (reset tokens) lives in Redis under the `pwreset:*` key namespace, with TTL enforced by Redis. No `.up.sql` / `.down.sql` files.

### Environment variables

| Var | Default | Purpose |
|-----|---------|---------|
| `PASSWORD_RESET_ENABLE` | `true` | Feature flag. Set `false` to disable the flow (endpoints return a VN "disabled" message). |
| `PASSWORD_RESET_TOKEN_TTL` | `30m` | Token lifetime. |
| `PASSWORD_RESET_RATE_LIMIT` | `3` | Max requests per email per hour. |
| `PASSWORD_RESET_URL` | `https://tingting.vip/reset-password` | Base URL for the magic link (the frontend reset page). Must match the deployed frontend origin. |

`RESEND_API_KEY` is already required for OTP and is reused for reset emails.

## Related Code Files

- **Modify:** `docs/api.md` — add the two routes to the auth section
- **Modify:** `docs/system-architecture.md` — add a one-paragraph note on the reset flow
- **Modify:** `docs/deployment-guide.md` — add the four env vars to the env table
- **Modify:** `backend/CLAUDE.md` or `backend/AGENTS.md` — (only if they enumerate env vars; check first)

## Implementation Steps

1. **Read `docs/api.md`** to find the auth-routes section (it already documents `/auth/login`, `/auth/login/verify`, etc.). Add:
   ```
   ### POST /auth/password-reset/request
   Body: { "email": "user@example.com" }
   Response: 200 { "message": "Nếu email tồn tại..." }   (always 200 — anti-enumeration)
   Rate limit: 3/hour per email.
   Auth: none (public).

   ### POST /auth/password-reset/confirm
   Body: { "token": "<from magic link>", "new_password": "..." }
   Response: 200 { "message": "Đặt lại mật khẩu thành công..." } | 401 (invalid/expired token)
   Side effects: invalidates all active sessions for the user (sets tokens_invalid_before).
   Auth: none (public, token-gated).
   ```

2. **Read `docs/system-architecture.md`**, find the auth/login section, and add a short subsection: "Self-service password reset: users with an email request a single-use magic link (30-min TTL, Redis-backed), click through to set a new password, which invalidates all active sessions."

3. **Read `docs/deployment-guide.md`**, find the env-var table, and append the four new vars with their defaults and purposes.

4. **Confirm no migration:** `ls backend/migrations/ | tail` shows `094_*` as the latest. **Do not add `095_*`** — verify once more that no MySQL schema change is needed (it isn't; the plan is Redis-only). Document this decision in the deployment guide: "Password reset: no migration (Redis-only)."

5. **Cross-check the feature flag default in code vs. docs:** `config.go` defaults `PASSWORD_RESET_ENABLE` to `true`. Make sure the deployment doc matches. If ops wants it off by default, flip the default in `config.go` and document accordingly. **Recommendation:** ship enabled by default — it's low-risk self-service and the rate limiter + anti-enumeration contain abuse.

6. **Final whole-repo validation gate** (run before declaring done):
   - `cd backend && go test ./... -race -cover` — all green.
   - `make api-test` — including the new `auth_password_reset_test.go`.
   - `cd frontend && pnpm lint && pnpm type-check && pnpm test` — all green.
   - Manual smoke on `make dev`: full flow from `/login` → forgot → email → click link → reset → login with new password.

## Success Criteria

- [ ] `docs/api.md` documents both new routes with bodies, responses, rate limits, and auth requirements.
- [ ] `docs/system-architecture.md` mentions the reset flow.
- [ ] `docs/deployment-guide.md` lists the four env vars.
- [ ] No new migration file was added (confirmed intentional, documented).
- [ ] All backend tests pass with `-race`.
- [ ] `make api-test` is green.
- [ ] All frontend checks (`lint`, `type-check`, `test`) are green.
- [ ] Manual end-to-end smoke succeeds on the dev environment.

## Risk Assessment

- **Docs drift:** The codebase docs are well-maintained; adding to them is low-risk as long as the additions match the existing format. Read each file before editing.
- **Feature-flag default surprise:** If ops deploys without setting `PASSWORD_RESET_ENABLE`, the feature goes live. This is intended (recommendation: enabled by default), but the deployment guide must state it explicitly so it's not a surprise.
