# Google OAuth No-OTP Residual Risk

**Date:** 2026-07-04
**Source:** Commits `e5dd9f7`, `63c72fb`
**Tags:** [security, auth, oauth, risk-acceptance]

## Context

The system requires OTP (one-time password) for standard email/password login. When Google OAuth login was added, the question was whether to require OTP after Google authentication as well.

Google OAuth already provides a strong authentication signal — Google has verified the user's identity. Requiring an additional OTP would add friction without meaningfully improving security for users who have already authenticated via Google.

## Decision / Outcome

Google OAuth login **skips OTP** (commit `e5dd9f7`), with replay/issuer hardening:
- Replay protection — prevents token reuse.
- Issuer validation — verifies the Google JWT issuer.

The residual risk was documented (commit `63c72fb`): if a user's Google account is compromised, the attacker gains access without the OTP second factor. This is accepted because:
1. Google accounts have their own 2FA options.
2. The friction of OTP after OAuth would reduce adoption of Google login.
3. The system's most sensitive operations (financial) have additional authorization checks via Casbin.

## Lesson

**Security decisions are trade-offs, not absolutes.** When deciding whether to add a security control:

1. **Identify what signal already exists.** Google OAuth already verifies identity — OTP adds verification of a different factor, but the incremental security gain is small compared to password-based login.
2. **Document accepted residual risk.** Don't just accept the risk silently — write it down so future developers understand why and can revisit it.
3. **Harden the implementation even when accepting risk.** Skipping OTP doesn't mean skipping all security — replay protection and issuer validation were still added.
4. **Layer defense at the operation level.** Even if login security is relaxed, financial operations have their own authorization checks (Casbin RBAC, ownership validation, audit logging).

**Generalized rule:** Every security convenience decision should be paired with (a) hardening of the implementation, (b) documentation of the residual risk, and (c) compensating controls at the operation level.

## References

- Commit `e5dd9f7`: feat(auth): Google OAuth no OTP + replay/issuer hardening
- Commit `63c72fb`: Documented Google-OAuth-no-OTP residual-risk decision
- [Security Standards](../standards/security.md)
- [ADR-008: Casbin RBAC](../decisions/ADR-008-casbin-rbac-authorization.md)
