---
title: "Security remediation: money-path, identity, attendance (audit 2026-07-04)"
description: "Remediates the pasted High-finding cluster from the 2026-07-04 STRIDE/OWASP audit: money-path transaction/idempotency gaps (H5/H6/H7), OnePay IPN + fee hardening (M8/M9/M12), identity & RBAC (H8/H10), and attendance override retirement + GPS anti-replay (M1/H3). Every finding was verified against source before planning."
status: pending
priority: P1
branch: "main"
tags: [security, money-path, auth, attendance, audit-2026-07-04]
blockedBy: []
blocks: []
created: "2026-07-04T02:29:31.591Z"
createdBy: "ck:plan"
source: skill
---

# Security remediation: money-path, identity, attendance (audit 2026-07-04)

## Overview

Scope-bounded remediation of the **pasted cluster** from the 2026-07-04 full-codebase STRIDE + OWASP audit (`plans/reports/ck-security-260704-0953-full-codebase-stride-owasp-report.md`). Every finding below was **verified against source** before planning — none are abstract-only concerns.

**Decisions captured (user, 2026-07-04):**
- **M1 override route → retire.** Delete route + handler + `RecordAdminCheckIn`. UI button already removed.
- **Scope → pasted cluster only.** H1/H2/H4/H9/H11–H13 and remaining mediums deferred to a follow-up plan.
- **H3 deployment → verified live** (see Baseline). H3 stays High, not Critical.

## Baseline — pre-plan verification (done)

| Check | Method | Result |
|---|---|---|
| `a3f892c` (accuracy gate) deployed | image `Created` vs commit `%cI` | ✅ prod `2026-07-03T12:43Z` + demo `2026-07-04T01:49Z` both postdate commit `2026-07-02T00:48Z` |
| `bd2b9d1` (checkout gate) deployed | same | ✅ both postdate commit `2026-07-02T11:27Z` |
| H3 severity | — | Stays **High** (not Critical). Residual risk = anti-replay gap, not a live money-loss path. |
| Compose restart policies | `/opt/payroll/docker-compose.yml` | ✅ all services incl. mysql+redis `unless-stopped` |

## Phases

| Phase | Name | Status | Priority |
|-------|------|--------|----------|
| 1 | [Money integrity core](./phase-01-money-integrity-core.md) | Pending | P1 |
| 2 | [OnePay IPN and fee hardening](./phase-02-onepay-ipn-and-fee-hardening.md) | Pending | P1 |
| 3 | [Identity and RBAC](./phase-03-identity-and-rbac.md) | Pending | P1 |
| 4 | [Attendance retire override and GPS anti-replay](./phase-04-attendance-retire-override-and-gps-anti-replay.md) | Pending | P2 |

## Finding → phase map

| Finding | Severity | Phase | One-line |
|---|---|---|---|
| H5 | High | 1 | FlexPay settlement: wrap UPDATE+ledger in tx + file-hash idempotency + propagate ledger errors |
| H6 | High | 1 | RetryDisbursement: in-flight dedup (409 if non-terminal wallet_payment exists) |
| H7 | High | 1 | MarkReconciled: add status precondition (route via FSM or status allowlist) |
| M12 | Medium | 2 | Fee schedule `ResolveFee`: fail-closed when OnePay schedule missing |
| M8 | Medium | 2 | OnePay IPN: cross-check amount vs persisted request |
| M9 | Medium | 2 | OnePay IPN: tighten replay window + nonce store |
| H8 | High | 3 | `LoginWithGoogle`: require `email_verified`; optional hosted-domain |
| H10 | High | 3 | Drop `adv_partner` `PUT /api/v1/users/*`; add ownership middleware |
| M1 | Medium | 4 | Retire override route + `RecordAdminCheckIn` |
| H3 | High | 4 | Server-side `gps_at` anti-replay (±60s) + accuracy-repeat flagging |

## Dependencies

**None blocking.** All four phases touch disjoint files (money-path services, OnePay adapter, auth/casbin, attendance service) and can be implemented in parallel by separate cooks. Phase ordering reflects risk (money first), not technical blocking.

**Cross-plan:** no overlap with `plans/260703-2023-attendance-open-row-unique` (timesheet row uniqueness, not attendance check-in security).

## Out of scope (follow-up plan)

Audit findings outside the pasted cluster — H1 (committed OnePay key), H2 (dev compose exposure), H4 (upload DoS), H9 (login throttle), H11 (supply-chain pinning), H12 (container limits), H13 (weak secrets), and remaining mediums (M2–M7, M10, M11, M13–M25). Track separately.

## Verification posture

- **Backend money/auth fixes:** unit + integration tests (tx rollback, dedup 409, OAuth missing-`email_verified` rejection, casbin `adv_partner` PUT → 403, override route → 404).
- **Deploy:** after merge, `cd backend && make push && make deploy`, then re-run the `verify-fix-deployed-prod` timestamp check against the new image for any commit in this plan.
