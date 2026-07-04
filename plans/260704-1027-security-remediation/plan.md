---
title: "Security remediation: money-path, identity, attendance (audit 2026-07-04)"
description: "Remediates the pasted High-finding cluster from the 2026-07-04 STRIDE/OWASP audit: money-path transaction/idempotency gaps (H5/H6/H7), OnePay IPN + fee hardening (M8/M9/M12), identity & RBAC (H8/H10), and attendance override retirement + GPS anti-replay (M1/H3). Every finding verified against source; plan passed a 3-reviewer red-team and was reshaped accordingly."
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

Scope-bounded remediation of the **pasted cluster** from the 2026-07-04 full-codebase STRIDE + OWASP audit (`plans/reports/ck-security-260704-0953-full-codebase-stride-owasp-report.md`). Every finding was verified against source, and the plan then passed a 3-reviewer red-team (Security Adversary, Failure Mode Analyst, Assumption Destroyer) — **14 findings applied**, including 3 Critical design-level reshapes (H3 anti-replay was theater; H5 design/path was wrong; H6 severity was overstated). See `## Red Team Review`.

**Decisions captured (user, 2026-07-04):**
- **M1 override route → retire** (delete route + handler + `RecordAdminCheckIn` + 3 frontend call sites).
- **Scope → pasted cluster only.** H1/H2/H4/H9/H11–H13 and remaining mediums deferred to a follow-up plan.
- **H3 deployment → verified live** (see Baseline). H3 stays High; the ±60s fix was reshaped (it was theater).

## Baseline — pre-plan verification (done)

| Check | Method | Result |
|---|---|---|
| `a3f892c` (accuracy gate) deployed | image `Created` vs commit `%cI` | ✅ prod `2026-07-03T12:43Z` + demo `2026-07-04T01:49Z` both postdate commit `2026-07-02T00:48Z` |
| `bd2b9d1` (checkout gate) deployed | same | ✅ both postdate commit `2026-07-02T11:27Z` |
| H3 severity | — | Stays **High** (not Critical). Deployed; residual = anti-replay gap (now honestly scoped — see Phase 4). |
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
| H5 | High | 1 | FlexPay settlement: per-iteration tx + `settlement_uploads` file-hash idempotency + propagate errors + add `*gorm.DB` wiring |
| H6 | ~~High~~ **Medium** | 1 | RetryDisbursement: asynq `UniqueTTL` + `entity_id` dedup (worker already prevents double-pay; honest success criterion) |
| H7 | High | 1 | MarkReconciled: status precondition, allowlist `{Failed, Completed, Authorised, Verified}` from `ReconcilePayment` switch |
| M12 | Medium | 2 | `disbursement/fee_schedule_service.go` `ResolveFee` → `(fee, waived, err)` + cascade through `GetDisbursementFeeVND`/`Initiate`/worker |
| M8 | Medium | 2 | OnePay IPN: amount cross-check in the **async asynq processor** (not the webhook receiver); audit row after check |
| M9 | Medium | 2 | OnePay IPN: nonce in `Receive` (keyed on real fields, fail-open) + verify `clock.Now()` before tightening ±60s |
| H8 | High | 3 | `LoginWithGoogle`: require `email_verified` BEFORE email extraction (generic error; audit absent vs false) |
| H10 | High | 3 | Drop `adv_partner` `PUT /api/v1/users/*` (content-match) + ownership gate in `GetEmployeeForUpdate(actor)` covering BOTH PUT paths + password/username |
| M1 | Medium | 4 | Retire override route + `RecordAdminCheckIn` + 3 frontend callers (`api.config.ts`, `dashboard.service.ts`, `useDashboard.ts`) |
| H3 | High | 4 | Server-side replay guard (real) + handler-layer input-sanity (honestly labeled); device attestation = follow-up |

## Dependencies

**Phase 2 (`blockedBy: [phase-01-money-integrity-core]`):** H5 adds `*gorm.DB` to `FlexPaySettlementService` and M12 changes `ResolveFee`'s signature — both land in `bootstrap/services/init.go` money-flow wiring. Sequence H5 before M12, OR run a combined settlement-upload → disbursement integration test post-merge if parallel.

**Otherwise disjoint:** Phase 3 (auth/casbin) and Phase 4 (attendance) touch separate files and can proceed independently of Phases 1–2.

**Cross-plan:** no overlap with `plans/260703-2023-attendance-open-row-unique` (timesheet row uniqueness).

## Out of scope (follow-up plan)

Audit findings outside the pasted cluster — H1 (committed OnePay key), H2 (dev compose exposure), H4 (upload DoS), H9 (login throttle), H11 (supply-chain pinning), H12 (container limits), H13 (weak secrets), and remaining mediums (M2–M7, M10, M11, M13–M25). **Also tracked here:** `advance_payment/fee_schedule_service.go:237` fail-open (twin of M12); real H3 spoofing-defeat (device attestation + payload signing).

## Verification posture

- **Tests:** per-iteration tx rollback (H5); replay/payload rejection + stale-fix rejection (H3); unverified-email → generic 401 + audit (H8); adv_partner PUT foreign/admin → 403 on BOTH routes + CSV-load (H10); forged-amount IPN rejection in the processor (M8); override route → 404 + frontend type-check (M1).
- **Deploy:** see `## Deploy & Rollback`. After merge, re-run the `verify-fix-deployed-prod` timestamp check against the new image for each commit in this plan.

## Deploy & Rollback

- **Migration ordering (H5):** `settlement_uploads` is additive — deploy migration **before** the backend image referencing it. Paired `.up.sql`/`.down.sql`, idempotent. demo/prod `schema_migrations` is dump-seeded/empty → apply via the manual DDL runbook in `lesson_apply_sql_migration_demo_prod` (sed pw from server `DB_DSN` → `MYSQL_PWD` → `docker exec payroll-mysql`); `make demo-db` alone is not sufficient.
- **Staged rollout (H3, M9):** ship wide, tighten. H3 input-sanity start at 300s; M9 — log `elapsed` distribution 24h at current ±900s and verify `clock.Now()` before narrowing. Rollback trigger: rejection rate >1% for 30min → widen + redeploy.
- **Flag-gated (M12):** fail-closed behind a flag for first deploy; explicit zero-fee allowlist so legit routes aren't blocked.
- **Restart coordination (H10):** Casbin caches in-process — server restart required after `casbin_policy.csv` edit; `make deploy`'s `--force-recreate` applies it.
- **Post-deploy verify:** `cd backend && make push && make deploy`, then re-run the `verify-fix-deployed-prod` timestamp check; confirm `docker inspect payroll-backend --format {{.State.StartedAt}}` is fresh.
- **Rollback:** `golang-migrate down` (paired `.down.sql`) for H5; redeploy prior image for the rest. All compose services are `restart: unless-stopped` (verified 2026-07-04), so a rollback reboot won't strand mysql/redis. Cite `lesson_demo_deploy_gotchas`, `lesson_apply_sql_migration_demo_prod`, `lesson_drop_index_blocked_by_fk`.

## Red Team Review

### Session — 2026-07-04
**Reviewers:** Security Adversary, Failure Mode Analyst, Assumption Destroyer (3, scaled to 4 phases). 27 raw findings → deduplicated to 16 → **14 accepted, 2 rejected/folded**.
**Severity breakdown (accepted):** 3 Critical, 7 High, 3 Medium, 1 Low. All accepted findings carry `file:line` evidence (auto-reject filter passed).
**Reports:** `reports/from-code-reviewer-to-planner-red-team-{security-adversary,failure-mode-analyst,assumption-destroyer}-plan-review-report.md`.

| # | Finding | Severity | Disposition | Applied To |
|---|---|---|---|---|
| 1 | H5 wrong design + no `*gorm.DB` + multi-row semantics | Critical | Accept | Phase 1 |
| 2 | H5 migration flawed (table unset, MySQL, no `.down.sql`) | Critical | Accept | Phase 1 |
| 3 | H3 ±60s theater (`gps_at` client-controlled) | Critical | Accept | Phase 4 |
| 4 | H6 severity overstated + column `advance_request_id` nonexistent (real: `entity_id`) | High | Accept | Phase 1 |
| 5 | M1 frontend not removed (3 live callers) | High | Accept | Phase 4 |
| 6 | H10 scoped handler no ownership check; scoped path stays open (incl. password) | High | Accept | Phase 3 |
| 7 | H7 allowlist under-specified → breaks `failed→completed` | High | Accept | Phase 1 |
| 8 | M12 wrong `ResolveFee` (two impls) + signature cascade | High | Accept | Phase 2 |
| 9 | M8 wrong layer (async asynq processor, not `Receive`) | High | Accept | Phase 2 |
| 10 | M9 nonce unwired + `jti` nonexistent + wrong clock + fail-open unset | High | Accept | Phase 2 |
| 11 | H8 placement = enumeration oracle + absent-claim | Medium | Accept | Phase 3 |
| 12 | H10 casbin deletion line-fragile (file shifted to 158 lines) | Medium | Accept | Phase 3 |
| 13 | No deploy/rollback runbook | Medium | Accept | plan.md (Deploy & Rollback) |
| 14 | Cross-phase H5+M12 not disjoint (both `bootstrap/init.go`) | Low | Accept | plan.md (Dependencies) |
| — | H8 second Google entry point | Medium | **Reject** | Folded into #11; reviewer found no second entry point |
| — | Deploy-timestamp re-verification | — | **Reject** | Verified this session (Baseline) |

### Whole-Plan Consistency Sweep
- **Decision deltas applied:** H6 severity High→Medium; H5 design = per-iteration tx + `settlement_uploads` audit table + `*gorm.DB` wiring; H3 = server-side replay guard + honest input-sanity (not ±60s theater); M8 = async processor layer; M9 = `Receive`-wired nonce + `clock.Now()` verification; M12 = `disbursement/` path pinned + cascade documented; H7 = allowlist baked from `ReconcilePayment` switch; H10 = both PUT paths + `GetEmployeeForUpdate(actor)`; H8 = pre-extraction placement; M1 = + frontend cleanup.
- **Stale-term sweep:** searched all 5 plan files for rejected terms — "±60s anti-replay" (Phase 4, corrected to input-sanity), "advance_request_id" column (Phase 1, corrected to `entity_id`), "mirror `ApplySettlement`" (Phase 1, replaced), "UNIQUE ... WHERE" (Phase 1, dropped — MySQL), "coordinate with owner" for H7 (Phase 1, replaced with baked allowlist), "button already removed" baseline (Phase 4, corrected to "frontend callers still live"). All instances resolved.
- **Cross-file reconciliation:** finding map (this file) matches each phase's Overview/Success Criteria; Phase 2 frontmatter `dependencies: [phase-01-money-integrity-core]` matches the Dependencies section; H6 severity consistent across finding map + Phase 1.
- **Unresolved contradictions:** none. Plan is ready for implementation.
