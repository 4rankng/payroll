# Task Context

## Intent

- **Goal:** Prevent Payroll Zalo credentials from becoming unusable when the access token expires.
- **Observable success:** Invalid refresh credentials fail at save time; valid credentials rotate and continue automatically under concurrency.
- **In scope:** Immediate validation, distributed refresh ownership, lost-update protection, tests, runbook.
- **Out of scope:** UI redesign, shared ChatBot credentials, commit, deployment.

## Authority

- **Explicit user decision:** Approved `ck:cook --auto` with no human review pauses.
- **User-owned work to preserve:** Existing unrelated loan reminder changes in the dirty worktree.

## Active Path

- **Entry point:** Admin credential save and ZNS send.
- **Owner:** `zaloconnect.Service` plus `zalo.Provider`.
- **Persisted contract:** `settings.key = zalo.credentials` JSON.
- **Closest precedent:** ChatBot Redis `SET NX EX` refresh ownership plus authoritative DB re-read.

## Decisions

| Decision | Evidence | Consequence |
|---|---|---|
| Validate a supplied refresh token immediately | Production accepted an unusable refresh token and discovered it only after access expiry | Configuration fails early instead of after roughly 24 hours |
| Coordinate refresh through Redis and local mutex | Refresh tokens rotate and are single-use; local mutex covers only one process | One app-level token authority across instances |
| Protect JSON mutations with CAS | Read-modify-write status updates can restore stale token JSON | Stale writers retry against the latest pair |
| Treat `-14014` separately from OAuth transport failures | A consumed refresh token requires replacement, but an upstream outage may recover | `-14014` prompts a fresh Payroll-specific pair; transport failures remain retryable |

## Progress

| Work item | Status | Evidence / next action |
|---|---|---|
| Locate active path | PASS | Provider, service, repository, bootstrap, and ChatBot precedent traced |
| Confirm contracts | PASS | Existing APIs and settings key identified |
| Implement | PASS | Save validation, Redis lease, post-lock re-read, and CAS persistence implemented |
| Focused verification | PASS | Touched packages passed repeated race tests |
| Required broader verification | PASS | Full unit suite and vet pass; live API suite unavailable because localhost:8080 was not running; unrelated full-race failure classified |
| Review completion claim | PASS | Tester, debugger, and production reviewer completed; blocking findings fixed and rechecked |

## Handoff

- **Changed files:** Zalo provider/coordinator, connection service, settings CAS/logger, bootstrap wiring, focused tests, runbook, journal.
- **Remaining work:** Authorized deploy, then save one fresh Payroll-specific pair and run a controlled OTP.
- **Blockers:** Current production refresh token is already rejected and cannot be recovered in code.
- **Latest verified evidence:** Repeated touched-package race tests, full unit suite, full vet, diff check, graphify update.
| Production credential handoff | PENDING | After any authorized deploy, an administrator must save one fresh Payroll-specific pair and run a controlled OTP test; no live validation claimed here |
