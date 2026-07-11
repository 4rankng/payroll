# Prompt: Security Review

Use when reviewing a change or component for security vulnerabilities. Follows STRIDE + OWASP methodology.

## Prompt

```
Goal: Security review of [COMPONENT/CHANGE] — [SCOPE_DESCRIPTION]

Context to gather:
1. Read docs/standards/security.md for known threat models and mitigations
2. Read docs/decisions/ADR-008-casbin-rbac-authorization.md for authz model
3. Read docs/decisions/ADR-003-payment-provider-abstraction.md for webhook security
4. Check docs/lessons/ for past security incidents
5. Read the relevant handler, service, and middleware code

Review checklist (STRIDE):

S — Spoofing:
- [ ] Authentication validates JWT tokens correctly
- [ ] Password hashing uses Argon2id
- [ ] No hardcoded credentials

T — Tampering:
- [ ] Input validation on all endpoints (Zod frontend, domain validation backend)
- [ ] Webhook signatures verified (VerifyAndParseWebhook)
- [ ] IDOR prevention — resource ownership validated server-side
- [ ] Casbin policy covers all new endpoints

R — Repudiation:
- [ ] Audit logs for all financial and state-changing operations
- [ ] Audit events emitted via non-blocking goroutine

I — Information Disclosure:
- [ ] No sensitive data in logs (tokens, passwords, bank accounts)
- [ ] Error messages don't leak internal state
- [ ] No stack traces in API responses

D — Denial of Service:
- [ ] Rate limiting on auth and expensive endpoints
- [ ] Pagination on list endpoints (no unbounded queries)
- [ ] Request timeout middleware active

E — Elevation of Privilege:
- [ ] Role checks via Casbin, not manual if-statements
- [ ] No privilege escalation paths (e.g., employee → admin)
- [ ] Token revocation on logout and password reset

OWASP Top 10:
- [ ] A01: Broken Access Control — IDOR checks, Casbin policies
- [ ] A02: Cryptographic Failures — Argon2id, TLS, no weak crypto
- [ ] A03: Injection — Parameterized queries (GORM), no raw SQL with user input
- [ ] A04: Insecure Design — Threat modeling for new features
- [ ] A05: Security Misconfiguration — No debug endpoints in prod, security headers
- [ ] A07: Auth Failures — CAPTCHA, rate limiting, token revocation
- [ ] A08: Software/Data Integrity — Webhook signature verification
- [ ] A09: Logging/Monitoring — Audit logs, slog, Prometheus metrics
- [ ] A10: SSRF — No server-side requests with user-controlled URLs

Output format:
1. List each finding with severity (Critical/High/Medium/Low)
2. For each finding: description, affected file, exploit scenario, recommended fix
3. Summary: overall risk assessment and go/no-go recommendation
```

## Reference

- [Security Standards](../standards/security.md)
- [Security Lessons](../lessons/2026-07-04-security-idor-spray-token-revocation.md)
- CI runs `gosec` and `npm audit --audit-level=high`
