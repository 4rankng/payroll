# Security: IDOR, Spray, and Token Revocation

**Date:** 2026-07-04
**Source:** Commit `619f9cd` (red-team fix)
**Tags:** [security, idor, rate-limiting, token-revocation, red-team]

## Context

A red-team security review identified three critical gaps:

1. **IDOR (Insecure Direct Object Reference)** — Users could access other users' resources by changing an ID in the URL. The system trusted client-provided IDs without server-side ownership validation.

2. **Login spray** — Attackers could try many passwords across many accounts without rate limiting, bypassing per-account lockout by rotating accounts.

3. **Token revocation** — Logged-out users could continue using old JWT tokens because the blacklist wasn't consistently enforced, and there was no `invalid_before` timestamp for password resets.

## Decision / Outcome

All three were fixed in commit `619f9cd`:

### IDOR Fix
- Every endpoint that takes a resource ID now validates ownership server-side.
- Repository methods accept `userID` and `role` parameters to scope queries.
- Casbin policies enforce role-based access, but ownership is checked at the service layer.

### Spray Fix
- Per-account login rate limiting (commit `473a49e`): Redis-backed, returns Vietnamese message with actual `retry_after` seconds.
- Self-hosted CAPTCHA after 3 failed attempts (commit `6327168`).

### Token Revocation Fix
- `blacklisted_tokens` table checked on every authenticated request.
- Migration 086: added `invalid_before` to user tokens — password reset sets this timestamp, invalidating all tokens issued before it.
- Both logout and password reset now revoke the token.

## Lesson

**Three patterns every API must enforce from day one:**

1. **Never trust client-provided IDs for authorization.** Every resource access must validate that the requesting user owns or is authorized to access that resource. Casbin handles role-based access, but ownership is a separate concern that must be checked at the service/repository layer.

   ```go
   // WRONG
   repo.FindByID(id)
   // CORRECT
   repo.FindByIDForUser(id, userID, role)
   ```

2. **Rate limit by account AND by IP.** Per-IP rate limiting alone allows rotating accounts. Per-account rate limiting alone allows rotating IPs. Both are needed to prevent spray attacks.

3. **Token revocation needs two mechanisms:**
   - **Blacklist** for explicit logout (token is valid but revoked).
   - **`invalid_before` timestamp** for password reset (all tokens issued before this time are invalid).
   
   A blacklist alone is insufficient because you can't enumerate all outstanding tokens for a user. `invalid_before` provides a bulk revocation mechanism.

**Generalized rule:** Security is not a feature you add later. IDOR, spray, and token revocation are baseline requirements for any authenticated API. A red-team review will find them if you don't implement them proactively.

## References

- Commit `619f9cd`: fix(security): close IDOR, spray, and token-revocation gaps (red-team)
- Commit `473a49e`: feat(rate-limit): per-account login rate limiting
- Commit `6327168`: feat(auth): self-hosted image CAPTCHA after 3 failed logins
- Migration 086: Add `invalid_before` to user tokens
- [Security Standards](../standards/security.md)
- [ADR-008: Casbin RBAC](../decisions/ADR-008-casbin-rbac-authorization.md)
