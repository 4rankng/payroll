---
title: "MiMo Assistant — Approval Engine, Sao Kê Automation & AI Chatbot"
description: "Approval-request workflow engine with reject-and-reschedule, automated sao kê email drafting/sending, and a Xiaomi MiMo-powered chatbot (text + file diagnosis) for Admin and Partner roles."
status: pending
priority: P1
effort: "7 phases — P1 (1–4) ≈ 2 weeks; P2 (5–7) gated on measured P1 adoption; estimate rebaselined after phase 4"
tags: [feature, backend, frontend, llm, workflow]
created: 2026-10-11
---

# MiMo Assistant — Approval Engine, Sao Kê Automation & AI Chatbot

## Status 2026-10-11 (all phases implemented; Excel dry-run gated)

Phases 1–7 implemented and live on dev: engine, cards/decisions, sao kê automation (background flash draft → custom-body send → reject-redraft loop), chat with tools + per-role toggles, screenshot vision diagnosis, daily insight digest. Extra rulings shipped mid-build: per-role chatbot switches (settings), no internal IDs anywhere — every surface shows requestor name / summary / approve-effect, and the chat tool feeds the model labels only. api-test 378/0 failures. The ONE remaining gated item: bcc_import dry-run extraction (phase 6) — 20-file payroll-feeding pipeline, equivalence-suite hard gate stands.

## Overview

Adds an AI-assisted operations layer to the payroll app without touching deterministic payroll math. Three deliverables: (1) a durable **approval-request engine** (state machine + reject-with-note → scheduled retry → escalation), (2) **sao kê email automation** (LLM-drafted, human-approved, idempotently sent via the existing email worker), (3) a **chatbot** for Admin+Partner (Q&A, import-failure diagnosis with Excel/screenshot attachments, request filing). LLM writes language and calls tools; all numbers come from deterministic services (boundary of ADR-010 preserved).

Brainstorm record: `plans/2026-10-11-mimo-assistant/brainstorm-summary.md`.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Admin/Partner can file, review, approve/reject (with note) approval requests — cards on Tổng Quan + push notification | P1 |
| 2 | Reject with note → automatic retry next day 08:00 ICT, max 3 retries, then escalation card | P1 |
| 3 | Sao kê emails: prepare → LLM draft → approve → idempotent send (manual send remains as fallback) | P1 |
| 4 | Chatbot: text Q&A + tool-calling (read-only + request filing) for Admin and Partner, desktop + mobile | P2 |
| 5 | Chat diagnosis: explain failed Excel imports from real error data; accept Excel + screenshot attachments | P2 |
| 6 | Admin settings: MiMo API key entry (masked), test connection, audit on change | P1 |
| 7 | Daily ops insight digest on Tổng Quan: deterministic DB signals → LLM triage/narrative, ≤5 items, 7-day dedup, critical-only push | P2 |
| 8 | Chatbot speed & usefulness contract: p95 ≤2s direct / ≤5s tool turns, prompt-cache-stable prompts, quick-action chips, answers + deep links | P2 |

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [LLM Foundation](./phase-01-llm-foundation.md) — AIClient, MiMo key settings, model spike gate | Done (spike pending real key) |
| 2 | [Approval Engine](./phase-02-approval-engine.md) — domain, state machine, API, RBAC, retry jobs | Todo |
| 3 | [Approval Notifications & Cards](./phase-03-approval-notifications.md) — push + Tổng Quan cards, approve/reject UI | Todo |
| 4 | [Sao Kê Automation](./phase-04-sao-ke-automation.md) — draft → approve → idempotent send → reschedule loop | Todo |
| 5 | [Chat MVP](./phase-05-chat-mvp.md) — chat service, tool layer, FAB + panel, both roles | Todo |
| 6 | [Diagnosis & Attachments](./phase-06-diagnosis-attachments.md) — D1 grounded diagnosis, D2 Excel dry-run, screenshots, void/correct proposals | Todo |
| 7 | [Daily Insight Digest](./phase-07-daily-insight-digest.md) — DB-signal collectors → LLM triage → anti-spam digest card on Tổng Quan | Todo |

Dependencies: 1 → 2 → 3 → 4; 1,2 → 5 → 6; 1,3 → 7. Phases 4 and 5 can proceed in parallel after 3; phase 7 can ship any time after 1+3, independent of 5–6.

## Cross-cutting rules (apply to every phase)

- LLM output is **never** a source of truth; every tool call, draft, and approval transition is audit-logged (existing `audit` event pipeline → `audit_logs`).
- Money/numbers computed only by deterministic services; approval impact summaries generated in Go.
- Unattended runs (scheduled retries) may create drafts + approval requests only — **never send**.
- All business time via `clock.Now()` (ADR-006); background jobs via asynq (ADR-005).
- RBAC re-checked server-side at execution time for every action; partner-triggered admin actions can only produce `pending` requests (Casbin `configs/casbin_policy.csv`).
- Frontend parity is non-negotiable: desktop + mobile × Admin + Partner, verified at 1280 / 390 / 320 px (repo testing rules).
- Vietnamese copy for all user-facing text; `mimo-v2.5*` models are deprecated — never use them.

## Success Criteria

- [ ] `make api-test` green including new `flow_approval.go`, `flow_sao_ke_automation.go`, `flow_chat.go`
- [ ] Injection test: partner message "send it now" cannot bypass approval (only a request is filed)
- [ ] Zero double-sent emails across 3 payroll cycles (idempotency key evidence in logs)
- [ ] Reject→retry loop: fires at 08:00 ICT next day, escalates after 3 retries (verified with `clock_helper` time advance)
- [ ] UI matrix verified with screenshots: Admin + Partner × desktop + mobile (1280/390/320)
- [ ] LLM outage degrades to clear Vietnamese error; no partial sends; deterministic flows unaffected
- [ ] Chat latency budget met (p95): ≤2s direct answers, ≤5s tool-calling turns — if missed in the phase-5 spike, SSE streaming becomes a required fast-follow
- [ ] Digest noise budget respected: ≤5 items/day, zero duplicate fingerprints within 7 days, non-critical findings trigger zero pushes

## Red-Team Constraints (ak:predict 2026-10-11 — verdict CAUTION, full report in `reports/2026-10-11-predict-red-team.md`)

Binding constraints folded into phases (non-negotiable before code):
1. `mimo.api_key` **must** be registered in `pkg/secret/keys.go` `ProtectedKeys` (sealed at rest + masked) — otherwise any partner/employee token can read it via `GET /settings/key/:key`
2. Explicit Casbin **deny rows** for `/approvals/:id/{approve,reject,cancel}` for partner/adv_partner/employee/accountant; `send_*` payloads server-derived; creation rate-limited
3. Approval card distinguishes **"Thử lại ngày mai"** (reject+reschedule) from **"Huỷ / không gửi lại"** (terminal)
4. Ownership checks at handler level for chat/messages/attachments/approvals (Casbin can't express ownership)
5. bcc_import dry-run extraction carries a **before/after equivalence suite** as a hard gate (payroll-feeding path)
6. Instrumentation before claims: per-turn latency metrics from day one; measured sao kê baseline + top-3 real partner import questions before phases 5–6 start
7. Error-catalog quick win (`frontend/src/utils/import-errors.ts` extension + fix-hints) ships regardless of chat timing

**Sequencing (user decision 2026-10-11):** ship **all 7 phases straight through in dependency order (1→2→3→4; 5→6 parallel with 4; 7 after 3)** — no adoption gate between P1 and P2 (explicit user decision after red-team; the red-team recommendation to gate P2 on measured P1 was considered and declined). Instrumentation and business baselines (sao kê cycle cost, top-3 partner questions) are still collected — they inform tuning and the success metrics, not sequencing.

## Open Questions (defaults proposed, reversible)

- Chat + attachment retention: default 12 months with admin purge.
- Escalation card UX after max retries: dedicated banner on Tổng Quan (see phase-02/03).
- Which email types join the approval flow after sao kê (loan reminders next?).
- Chat streaming: MVP is non-streaming (no SSE infra exists); revisit if latency complaints.

<!-- slug: mimo-assistant -->
