---
phase: 3
title: "Identity and RBAC"
status: pending
priority: P1
dependencies: []
---

# Phase 3: Identity and RBAC

## Overview
Close two identity/authorization Highs. **H8:** `LoginWithGoogle` never checks `email_verified` (insert before email extraction to avoid a user-enumeration oracle). **H10:** `adv_partner` can `PUT /api/v1/users/*` (casbin) AND the scoped `/adv-partner/users/:id` handler has no ownership check — deleting the broad casbin grant alone leaves the IDOR open on the scoped path. Red-team corrected the H8 placement, the H10 scope (two handlers), and the line-fragile deletion; see `## Red Team Corrections`.

## Requirements
- Functional: an unverified-email Google ID token is rejected with a generic error (no enumeration); an `adv_partner` cannot modify any user/employee it doesn't own, and can never modify an admin or another partner's employee (incl. password/username).
- Non-functional: legitimate `adv_partner` self-service on `/adv-partner/users/*` still works; casbin reload coordinated (in-process cache).

## Architecture
- **H8 (corrected placement):** insert the `email_verified` check **immediately after `idtoken.Validate` succeeds (after line 482), BEFORE email extraction (line 484)** — placing it after extraction leaks user-existence via the distinct `MsgGoogleAccountNotLinkedVN` error (lines 501/506). Use the generic `MsgInvalidCredentialsVN`. Hard-reject `email_verified != true`; audit-log **absent** (token-shape anomaly) vs **false** distinctly. Optional hosted-domain (`hd`) only after grepping the users table for non-domain emails. Add a test asserting `idtoken.Validate` is called with the configured `googleClientID` (not `""`), so a future second client can't silently weaken it.
- **H10 (corrected scope):** there are **two** user-PUT handlers:
  - `UserHandler.UpdateUser` (`user.go:92-110`) — the broad one behind deleted casbin line 119; no actor/ownership today.
  - `UpdateAdvPartnerUser` (`adv_partner/user_handler.go:34-130`) — the scoped one (still allowed by casbin line 122 `/adv-partner/users/*, *, allow`); `GetEmployeeForUpdate(ctx, employeeID)` has **no actor scope**, and it exposes `Username` + `Password` mutation.
  Deleting casbin line 119 alone does NOT close the IDOR — the scoped handler still lets any adv_partner edit any employee. Enforcement point: `EmployeeService.GetEmployeeForUpdate` must take `(actorID, actorRole)` and return `NotFound` for employees outside the actor's project scope; gate `Username`/`Password` mutation too.
- **H10 (line-safe deletion):** delete by **content match** (`grep 'adv_partner, /api/v1/users/\*, PUT, allow'`), NOT line number — the file has shifted to 158 lines. Add a CSV-load test asserting the line is gone AND the scoped line 122 grant still resolves.

## Related Code Files
- Modify: `backend/internal/app/services/auth/auth_service.go` (H8 — `LoginWithGoogle` 471-556; insert after :482)
- Modify: `backend/configs/casbin_policy.csv` (H10 — content-match delete of the `adv_partner /api/v1/users/* PUT allow` grant)
- Modify: `backend/internal/transport/http/handlers/adv_partner/user_handler.go` (H10 — `UpdateAdvPartnerUser` 34-130; ownership gate on `GetEmployeeForUpdate`; protect Username/Password)
- Modify: `backend/internal/transport/http/handlers/user.go` (H10 — `UpdateUser` 92-110; ownership check; actor/scope source from JWT via `middleware/auth.go`)
- Reference: `lesson_partner_403_edit_requests` (Casbin caches in-process — server restart required); `lesson_partner_403_employee_detail` (LIST vs DETAIL scope nuance — ownership must not over-deny)
- Tests: unverified-email token → generic 401 (H8); `absent` vs `false` audit distinction; adv_partner PUT foreign/admin employee → 403 on BOTH paths (H10); CSV-load test

## Implementation Steps
1. **H8 — placement.** Insert the check after `idtoken.Validate` (line 482), before email extraction. Generic error. Audit absent vs false.
2. **H8 — single-client invariant test.** Assert `idtoken.Validate` receives the configured `googleClientID`.
3. **H8 — hosted domain (optional).** Only if `GOOGLE_HOSTED_DOMAIN` set AND users table has no `@gmail.com` rows.
4. **H10 — ownership gate.** `GetEmployeeForUpdate(ctx, actorID, actorRole, employeeID)` returns `NotFound` for out-of-scope employees; cover `Username`/`Password` mutation. Source `actorID`/`actorRole`/project scope from the JWT (`middleware/auth.go`) — same join backing `/adv-partner/users/*`.
5. **H10 — casbin deletion.** Content-match delete; restart server (policy cache). Grep frontend (`src/`) for adv_partner-driven `PUT /api/v1/users/` calls before deletion — if any, migrate to `/adv-partner/users/*` first.
6. **Tests.** Unverified/absent → 401 (H8); adv_partner PUT on admin/foreign employee → 403 on both routes (H10); self-service `/adv-partner/users/*` still 200; CSV-load assertion.

## Success Criteria
- [ ] H8: `email_verified: false` OR absent → generic 401 with audit; no enumeration (same error as invalid token).
- [ ] H10: `adv_partner` PUT `/api/v1/users/<admin>` AND `/api/v1/adv-partner/users/<not-owned>` → 403; password/username mutation gated.
- [ ] H10: scoped self-service unaffected; casbin grant deleted by content (verified by CSV-load test).

## Risk Assessment
- **H8 breaks a legit Google account:** only genuinely unverified tokens reject. *Mitigation:* clear error so the user verifies their Google email.
- **H10 ownership gate over-denies:** could break legitimate adv_partner edits if the project-scope join is wrong. *Mitigation:* reuse the exact join backing the existing scoped route; test both allow (own employee) and deny (foreign/admin).
- **H10 casbin line shift:** deleting by number could remove the scoped grant instead. *Mitigation:* content-match deletion + CSV-load test.

## Red Team Corrections
- **H8 placement = enumeration oracle (Medium):** moved before email extraction; generic error; absent vs false audited.
- **H10 scoped handler ignored (High):** v1 only addressed the broad casbin line — the scoped `/adv-partner/users/:id` handler has no ownership check and exposes password mutation. Now covered: enforcement in `GetEmployeeForUpdate(actor)`, both paths in the success criterion.
- **H10 line-fragile deletion (Medium):** file shifted to 158 lines; content-match + CSV-load test mandated.
