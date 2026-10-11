---
title: "Phase 7: Daily Insight Digest"
status: todo
---

# Phase 7: Daily Insight Digest

## Overview

A daily "what matters" card on Tổng Quan: deterministic DB-signal collectors compute candidate findings, hard anti-spam filters decide what survives, and the LLM only triages + writes the Vietnamese narrative. Facts are computed in Go; the model never derives numbers. Depends only on Phase 1 (AIClient) + Phase 3 card patterns — can ship before Phases 5–6.

## Requirements

- [ ] Collectors (deterministic, Go): disbursement failures, timesheet-import failure spikes, cron job failures (`cron_job_status`), attendance check-in anomalies, wallet/ledger alerts, audit-log spikes (e.g. failed logins). Each emits a `daily_finding` with: type, severity (info|warning|critical), computed facts (numbers, deltas vs yesterday), entity deep-links
- [ ] `daily_findings` table with **fingerprints** (type + entity key) for dedup; findings carry facts as structured JSON — the LLM receives facts as quoted read-only strings
- [ ] Anti-spam hard filters (structural, not prompt-based): suppress already-reported fingerprints within 7 days; **max 5 items per digest**; suppress "ongoing/unchanged" conditions; info-severity hidden unless nothing better
- [ ] LLM triage step (`mimo-v2.6-pro`, async so latency-free): given surviving candidates + facts, select only new/worsening/actionable items and write ≤3-sentence Vietnamese narrative each with a "vì sao đáng chú ý" line; may drop items but **may not invent or alter numbers**
- [ ] Daily run at **07:30 ICT** via `scheduler_jobs.go` cron pattern (idempotent per date — `RunJobOnce`-style guard); read-only by construction: the digest never mutates business data
- [ ] **Push only for severity=critical** findings (e.g. disbursement failure spike) via existing `PushService`; everything else lives quietly on the card
- [ ] Admin-only card on Tổng Quan desktop + mobile: today's digest + 7-day history, per-item dismiss ("không hữu ích") → suppression rule stored against the fingerprint family
- [ ] No raw logs anywhere in this phase — sources are DB state only (approved scope)
- [ ] **Notifier consolidation**: `scheduler_jobs.go` already pushes overlapping signals (`monitor_high_error_rates`, `advance_payment_reminder`, sao kê reminder crons days 1/26/9) — either retire the overlapping pushes or route them through the digest's suppression/dedup so the same condition doesn't notify 3×; the ≤5/critical-only budget means nothing if legacy jobs spam alongside
- [ ] Digest respects the **≤2 dashboard cards** binding (UU migration plan) — one card or an existing-zone slot, never a third block above the fold

## Implementation Steps

1. `internal/app/services/insights/` — collector registry (one file per collector), finding domain type + fingerprinting, `daily_findings` persistence.
2. Filter pipeline: dedup window, per-day cap, severity gate, suppression rules table (fed by dismiss actions).
3. LLM narrative step in `llm/` (pro model; prompt = fact sheet as quoted strings; contract: "never compute or modify numbers"); store rendered digest alongside findings.
4. Scheduler job `daily_insight_digest` at `0 30 7 * * *`-equivalent (see `scheduler_jobs.go` cron entries; Asia/Ho_Chi_Minh via `clock.Now()`); date-keyed idempotency.
5. API: `GET /api/v1/insights/digest?date=`, `POST /api/v1/insights/:id/dismiss` (admin only; Casbin rows).
6. Frontend: `InsightDigestCard` in `pages/admin/DashboardPage/index.tsx` + `pages/mobile/admin/DashboardPage/index.tsx` disclosure section; dismiss buttons; deep links to source pages (system-health, cron-health, timesheet imports, transactions).
7. Tests: collector unit tests (each signal type), filter tests (dedup within 7 days, cap of 5, critical-only push — non-critical must NOT push), narrative test with fixed facts (numbers unchanged), `flow_insights.go` integration.

## Related Code Files

- Create: `internal/app/services/insights/`, `internal/domain/daily_finding.go`, `backend/tests/integration/flow_insights.go`, `frontend/src/components/admin-dashboard/InsightDigestCard.tsx`, `frontend/src/hooks/api/useInsights.ts`
- Modify: `internal/app/bootstrap/scheduler_jobs.go`, `configs/casbin_policy.csv`, both DashboardPage variants, `config/api.config.ts`, `services/api/` (new `insights.service.ts`)

## Success Criteria

- [ ] Digest appears daily at 07:30 with ≤5 items; no duplicate fingerprints within 7 days (test evidence)
- [ ] Numbers in narratives match collector facts verbatim (frozen-facts test)
- [ ] Non-critical finding triggers zero push notifications (test); critical triggers exactly one
- [ ] Dismissed finding family stays suppressed (test)
- [ ] Desktop + mobile admin card verified (1280/390/320)

## Risks

- Alert fatigue kills the feature — the cap/dedup/dismiss loop is the product, not a detail; treat filter regressions as bugs.
- Digest cannibalizing system-health/cron-health pages — mitigated by deep-linking every item into those pages rather than duplicating their detail.
- "Important" drifts over time — the dismiss feedback is the tuning signal; review suppression rules quarterly.

## Follow-ups (explicitly out of scope here)

- Redacted error-log fingerprints as an additional collector (needs redaction filter + business sign-off).
- Partner business-subset digest.
