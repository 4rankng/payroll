---
phase: 3
title: "Identity and RBAC"
status: pending
priority: P1
dependencies: []
---

# Phase 3: Identity and RBAC

## Overview
Close the two identity/authorization Highs: Google OAuth accepts an ID token without checking `email_verified` (H8 — account-takeover vector via unverified-email token), and `adv_partner` is granted `PUT /api/v1/users/*` with no ownership check (H10 — can edit any user including admin).

## Requirements
- Functional: an unverified-email Google ID token is rejected; an `adv_partner` cannot PUT a user they don't own and can never modify an admin.
- Non-functional: no regression in legitimate `adv_partner` self-service flows on `/adv-partner/users/*`.

## Architecture
- **H8:** after `idtoken.Validate` and email extraction, require `payload.Claims["email_verified"] == true`; optionally enforce a hosted-domain (`hd`) allowlist if the tenant uses Google Workspace. Reject with the standard invalid-credentials message + audit row.
- **H10:** remove `configs/casbin_policy.csv:119` (`p, adv_partner, /api/v1/users/*, PUT, allow`). The scoped line 122 (`/api/v1/adv-partner/users/*, *, allow`) already covers legitimate adv_partner user management. Add `CanUserModifyUser(actor, target)` in the user-update handler so even the scoped route can't edit an admin or a non-owned user.

## Related Code Files
- Modify: `backend/internal/app/services/auth/auth_service.go` (H8 — `LoginWithGoogle` 471-556, insert after ~line 494)
- Modify: `backend/configs/casbin_policy.csv` (H10 — delete line 119)
- Modify: user-update handler + add ownership check (H10 — locate via `PUT /api/v1/users/*` route registration)
- Reference: `lesson_partner_403_edit_requests` + `lesson_partner_403_employee_detail` (partner scope LIST vs DETAIL nuance — ownership check must not repeat the over-broad-deny mistake)
- Tests: OAuth unverified-email rejection (H8), adv_partner PUT foreign/admin user → 403 (H10)

## Implementation Steps
1. **H8 — `email_verified`.** In `LoginWithGoogle`, after the email is extracted (~line 494), read `payload.Claims["email_verified"]`; if not `true`, audit + return unauthorized. Unit test with a mocked token lacking the claim.
2. **H8 — hosted domain (optional).** If `GOOGLE_HOSTED_DOMAIN` is set, assert `payload.Claims["hd"]` matches.
3. **H10 — drop the grant.** Delete `casbin_policy.csv:119`. Restart policy load (Casbin caches — server restart). Verify adv_partner still hits `/adv-partner/users/*`.
4. **H10 — ownership check.** Add `CanUserModifyUser(actor, target)`: adv_partner may modify only users in their own project scope; admins are never editable by non-admins. Wire into the user PUT handler.
5. **Tests.** Unverified-email token → 401 (H8); adv_partner PUT on admin/foreign user → 403 (H10); self-service `/adv-partner/users/*` still 200.

## Success Criteria
- [ ] H8: a Google ID token with `email_verified: false` is rejected with an audit row.
- [ ] H10: `adv_partner` PUT `/api/v1/users/<admin>` → 403; broad line 119 grant removed from policy.
- [ ] No regression: adv_partner self-service user flows on `/adv-partner/users/*` unaffected.

## Risk Assessment
- **H8 breaks a legit Google account:** Google sets `email_verified: true` for verified accounts, so only genuinely unverified tokens are rejected. *Mitigation:* clear error so a user knows to verify their Google email.
- **H10 removing line 119 breaks an adv_partner flow:** the scoped line 122 covers the legitimate path, but confirm no frontend/client calls the broad `/api/v1/users/*` PUT for adv_partner. *Mitigation:* grep frontend + routes for adv_partner-driven `/api/v1/users/` PUTs before deletion; migrate any to `/adv-partner/users/*` first.
