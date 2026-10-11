# Brainstorm Summary — MiMo AI Assistant + Approval Workflow for Payroll App

> Date: 2026-10-11 · Status: **agreed, pending implementation plan** · Participants: product owner, CTO advisor

## 1. Problem Statement

Today the payroll app is 100% deterministic (Go + MySQL + Redis, no AI). Two pains drive this work:

1. **Manual sao kê / payroll email dispatch** — an admin prepares and sends statement emails by hand each cycle.
2. **No conversational access** — Admin and Partner want to *ask* for things ("chuẩn bị sao kê", "gửi nhắc nợ") instead of navigating forms. When a Partner asks for something that requires Admin permission, there is no lightweight request/approve loop.

Constraints stated by the business:

- Chatbot for **Admin and Partner** roles (text in/out for now; **no voice**).
- Chat must accept **file attachments**: Excel uploads for import troubleshooting and **error screenshots** (image understanding via `mimo-v2.6` omni-modal vision).
- Chatbot must also absorb recurring support inquiries (why Excel upload failed, wrong timesheet, etc.), escalating to approval cards where admin permission is required.
- Partner requests touching admin-only APIs must become an **approval request** surfaced in **Tổng Quan** + push notification; Admin approves or rejects **with a message**.
- Rejection with guidance (e.g. "gửi mai, hôm nay dữ liệu chưa đủ") must cause the system to **retry the next day**.
- Admin enters the **Xiaomi MiMo API key** in the admin settings page.
- External LLM API with salary/personal data: **accepted** by the business.
- Token budget: MiMo Pro Annual (456B credits, valid to 2027-10-11) — **cost is not a constraint** (~5M flash chat turns/yr). Design for safety and latency, not token frugality.

Model landscape (Xiaomi MiMo API, OpenAI-compatible, `api.xiaomimimo.com/v1`):

| Model | Use here |
|---|---|
| `mimo-v2.6-flash` | **Default** — all interactive chat turns, tool selection, Q&A (latency-bound surface) |
| `mimo-v2.6-pro` | **Async deliverables only** — email drafts, approval summaries, upload-failure diagnosis (quality-bound, latency-free) |
| `mimo-v2.5-pro`, `mimo-v2.5` | **Do not use** — deprecated 2026-10-21, auto-routed to v2.6 from 2026-10-14 |
| `mimo-v2.5-asr/tts/*` | **Out of scope** — speech is zh/en-centric; Vietnamese quality undocumented; TTS free only during promo. Revisit only after a Vietnamese quality spike |

## 2. Evaluated Approaches

| Dimension | A. Approval-first + chat (chosen) | B. Chat-first agent | C. No chatbot (simplest viable) |
|---|---|---|---|
| Core artifact | `approval_requests` domain entity + state machine | Conversation log as business state | Approval queue fed by a button |
| Kills manual sao kê pain | Yes | Yes | Yes |
| "Talk to it" requirement | Yes (as a surface) | Yes | No |
| Audit / crash recovery | Strong — durable rows | Weak — parse chat history | Strong |
| Prompt-injection blast radius | Bounded (model can only file requests) | Large (model mediates everything) | None |
| Frontend surfaces (desktop+mobile × Admin+Partner) | 4 (phased: 2 first) | 4 | 2 |
| Effort | Slice 1 ~1 wk, slice 2 +1–1.5 wk | ~2 wk + rework later | ~1 wk |

**Chosen: A, delivered in two phases.** Rationale: the durable value is the approval state machine and the idempotent scheduled retry, not the chat. Building the engine first means the chatbot is a replaceable convenience layer and can never become load-bearing. Phase 1 (≈C mechanically) delivers the operational win in about a week; Phase 2 bolts the conversation on.

**Why not B:** conversation-as-state is unauditable and unrecoverable in a payroll dispute ("who authorized this email?"). **Why not C alone:** the business explicitly wants users to talk to the system; C remains the fallback if Phase 2 underwhelms.

## 3. Final Solution Shape

### Phase 1 — Approval engine + sao kê automation
- New domain entity `approval_request`: type (`send_sao_ke_email`, `send_payroll_email`, `send_reminder`, …), JSON payload, requester, approver, status (`draft → pending → approved → executed | rejected → rescheduled | escalated | expired`), decision note, `scheduled_at`, `retry_count`, idempotency key.
- "Chuẩn bị sao kê" action (button in Phase 1, chatbot in Phase 2) generates draft email(s) via `mimo-v2.6-pro` → files `pending` approval → push notification + card in **Tổng Quan**.
- Admin approves → deterministic send path executes (existing `email_service`), guarded by an idempotency key per (cycle, recipient, report type) — precedent: `bcc_import_dedup`.
- Admin rejects with note → asynq job (ADR-005) retries at **08:00 ICT next day** (`clock.Now`, ADR-006), max **3** auto-retries, then **escalation card** to admin. Each retry re-checks **deterministic preconditions** (e.g. all statements of the cycle generated) before re-drafting; the LLM is never the judge of data completeness.

### Phase 2 — Chatbot (Admin + Partner)
Capability classes, different risk tiers:

| Class | Examples | Behavior |
|---|---|---|
| 1. Diagnose & explain (read-only) | "why did my Excel upload fail?", error-code questions, request status | Answer grounded in real system state |
| 2. Mutate, caller lacks permission | void/correct timesheet, resend sao kê | **Always** → approval card + PN via the Phase 1 engine |
| 3. Mutate, caller has permission | partner's own edit requests | **Out of Phase 2 scope** — chat reads and files requests only; execution stays in existing UI |

- Split delivery: **2a** = Q&A + D1 diagnosis + request filing (read-only); **2b** = attachments (Excel dry-run + screenshot vision).
- **Diagnosis (D1 + D2), grounded, never free-form:**
  - D1: tool fetches the caller's latest failed import + real per-row validation errors + the BCC template doc → LLM explains in Vietnamese.
  - D2: attachments in chat — Excel (`.xlsx/.xls`) **parsed server-side** (headers, sample/failing rows, template-validation **dry-run**) and sent to the model as structured text; raw file bytes never reach the model. Dry-run must be provably side-effect-free.
  - Screenshots (`png`/`jpg`) go to `mimo-v2.6-flash`/`pro` as **image input** (omni-modal). Best answers reconcile vision output with the real job state via tools; screenshot text is untrusted data, never instructions.
  - PII minimization: send only the error-relevant slice (headers + failing rows), not full spreadsheets.
- **Wrong timesheet → void/correct, not delete:** chatbot proposes void/correct or the existing timesheet edit-request flow (`HasRequestEdit`). Approval card shows a **deterministic impact summary** (rows, cycle, payroll already computed?) generated by Go. Hard delete is a separate rare approval type, never the default verb.
- **Graceful dead end:** requests with no matching tool get designed Vietnamese fallback copy ("tính năng này chưa hỗ trợ qua chat — thao tác tại X"). The tool manifest is the contract; the chatbot never hallucinates capability.
- Chat surface: desktop + mobile for both roles (project rule: one delivery scope). Attachments reuse the existing `asset` service hardening (MIME sniffing, size cap, private storage, RBAC-gated download, retention cap).
- Model routing per the Cross-cutting model-selection decision (flash default; pro for deliverables incl. image-heavy diagnosis).
- Conversation history + attachments stored in MySQL with a retention decision (default proposal: 12 months, admin purge).
- Unattended runs (scheduled retry) may only create drafts + approval requests, **never send**.

### Cross-cutting
- Thin `AIClient` interface (OpenAI-compatible `chat.completions` + function calling + streaming) — provider swappable in an afternoon.
- **Model selection (decided 2026-10-11):** one `AIClient` with a per-call `model` param. Default `mimo-v2.6-flash`; `mimo-v2.6-pro` only for async deliverables (drafts, summaries, diagnosis). Rationale: cost is non-binding (~1.8M pro turns/yr vs ~5.3M flash turns/yr on the 456B-credit plan); the real axes are latency (interactive chat — pro thinking is felt in the spinner; `mimo-v2.6-pro-ultraspeed` is NOT in the 8-model subscription) and output quality (trust-critical Vietnamese copy, async so latency-free). **Spike gate before Phase 2:** 3 real Vietnamese samples on both models — measure turn latency, human-judge output — collapse to flash-only if indistinguishable (YAGNI). Two model IDs = two prompt/eval baselines; the spike is what keeps that cost from being paid needlessly. `mimo-v2.5*` never (deprecated 2026-10-21).
- MiMo API key: reuse existing `apikey` encrypted-storage pattern; server-side only; test-connection button; audit row on change; graceful human-readable error when missing/invalid (Vietnamese UI copy per convention).
- Every tool call, generated draft, and approval transition written to the existing `audit` service.

## 4. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Prompt injection / confused deputy ("just send it now") | Structural: tool whitelist, per-caller RBAC re-check at *execution* time, no execute-as-admin tool |
| Double-send on retry | Idempotency key (cycle, recipient, report type) enforced in send path |
| Silent retry loops | Max 3 retries + escalation card |
| LLM decides data completeness wrongly | Deterministic precondition checks gate each retry |
| API key leakage | Encrypted at rest, never in frontend bundle, audited rotation |
| Vietnamese output quality | Human approves every outbound message in Phase 1; spot-check Vietnamese copy in Phase 2 QA |
| Scope creep in frontend | 4 surfaces is the real cost; Phase 1 ships only 2 (Tổng Quan card + prepare button) |

**Boundary with ADR-010 (statistical-forecast-over-ML):** preserved. LLM writes language and calls tools; all numbers, balances, and payroll math stay in deterministic services. The model is never a source of truth.

## 5. Success Metrics

- Time admin spends per sao kê cycle: from manual send to **≤ 2 approve clicks** (target: −80% hands-on time).
- Zero double-sent emails over 3 payroll cycles (idempotency evidence in logs).
- Approval round-trip (request → decision) < 1 business day median, measured from audit rows.
- Zero permission escalations executed without an approval record.
- Chat slice: ≥ 50% of partner requests fully handled by chat without admin intervention in UI (target, first month).

## 6. Validation Criteria (definition of done)

- `make api-test` green incl. new integration scenarios: approval lifecycle, reject-with-retry, max-retry escalation, idempotent send, RBAC check on every tool.
- Desktop + mobile verified for **both** Admin and Partner (1280px + 390px, 320px overflow check) per project testing rules.
- Injection test: partner prompt "send it now" cannot bypass approval.
- LLM outage test: feature degrades to clear Vietnamese error; no partial sends; deterministic flows unaffected.

## 7. Next Steps & Dependencies

1. **Detailed implementation plan** (`/ck:plan`) — pending user go-ahead.
2. Dependency: MiMo API key from the business (Pro Annual plan active until 2027-10-11).
3. Open decisions to confirm during planning (defaults proposed, reversible):
   - Retention of chat history (default 12 months).
   - Exact escalation UX for the >3-retries card.
   - Which email types join the approval flow after sao kê (loan reminders next?).
4. Explicitly deferred: voice (ASR/TTS), employee-facing chat, MiMo self-hosting (only if compliance posture changes).

## Appendix — rejected ideas (kept for the record)

- Voice features (ASR/TTS/voice clone): zh/en-first models, undocumented Vietnamese quality, TTS pricing cliff after promo. Revisit only after a Vietnamese quality spike.
- Letting the chatbot write SQL directly: replaced by RBAC-checked tools over the service layer.
- Self-hosted vLLM (open weights): unnecessary while external API is business-accepted; noted as the fallback if compliance changes.
