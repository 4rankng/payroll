# Red-Team Report — ak:predict, 5-persona pre-implementation review

> Date: 2026-10-11 · Plan: `plans/261011-0512-mimo-assistant/` · Method: 5 independent persona analyses (Architect, Security, Performance, UX, Devil's Advocate), each reading plan + repo source, then consensus synthesis.

## Verdict: CAUTION — proceed after spec-level fixes (applied to plan files on this date)

No STOP trigger survived: the architecture is sound, but 3 findings are certain to manifest if implementation starts against the plan as written.

## Critical findings (fixed in plan)

1. **MiMo API key exposure** — settings key GET (`routes_admin.go`) has `Authenticate()` only; Casbin grants partner/employee `GET /api/v1/settings/*`; masking is allowlist-driven via `secret.ProtectedKeys` (`pkg/secret/keys.go`, currently only `zalo.credentials`). Plan must mandate: register `mimo.api_key` in `ProtectedKeys` (sealed at rest — the zaloconnect precedent is *sealed*, not plaintext), masked responses, no values in audit/logs.
2. **Reject ≠ cancel** — `rejected → rescheduled` means "Từ chối" silently re-files tomorrow; no terminal "Huỷ" on the card. Fix: distinct actions "Thử lại ngày mai" (reject+reschedule) vs "Huỷ / không gửi lại" (cancel, terminal).
3. **Confused deputy on `send_*` approvals** — untyped payload JSON + filer-influenced recipients + "≤2 clicks" approval = phishing relay via company mailbox. Fix: server-derived payload builders, partner-fileable type allowlist, creation rate limit, explicit Casbin deny rows for `/approvals/:id/{approve,reject,cancel}` (keyMatch2 widening is documented in `casbin_policy.csv` itself).

## High findings (constraints folded into phases)

4. Chat/approval/attachment **ownership checks** must be handler-level (Casbin is role×path only; `authorization.go` scopes only `/projects/:id`, `/employees/:id`).
5. **bcc_import dry-run extraction** is the highest-risk step in the program — mandatory before/after equivalence suite; STOP if treated as routine plumbing.
6. **Unmeasured claims** — p95 budget (3-sample spike can't establish p95), sao kê −80% baseline, chat adoption. Fix: instrumentation before spike; business baseline + top-3 real partner import questions before phase 5/6 go.
7. **Error-catalog quick win** — `frontend/src/utils/import-errors.ts` already provides pattern→Vietnamese explanations rendered in `BCCUploadModal`/`UploadHistorySheet`. Error codes + catalog extension (days, not a phase) is the durable fix; chat diagnosis layers on top. Now a plan constraint.

## Medium findings (noted in phases)

- Approval `executed` transition + idempotent send = dual write without reconciliation (stuck-`approved` → manual fallback double-send path).
- LLM client belongs in `internal/infra/llm/` behind a port (`internal/infra/zalo/` pattern), not `app/services/llm/client.go`.
- No budget/quotas/circuit breaker on the shared key across 4 consumers.
- Digest must consolidate/dedupe existing notifiers (`monitor_high_error_rates`, `advance_payment_reminder`, sao kê reminder crons) or admins get the same signal 3×.
- "≤2 dashboard cards" binding (UU migration plan) conflicts with ApprovalRequestsCard + InsightDigestCard + escalation banner + existing banner.
- `adv_partner` reuses `AdminLayout` — chat FAB would leak to an external role unless explicitly scoped.
- PII (CCCD in `ImportError.Reason` verbatim) enters prompts + 12-month chat retention; `.xlsx` = ZIP needs decompression limits; screenshots: HEIC from iOS camera rolls vs png/jpg pipeline.
- Retry TaskID must be attempt-scoped (repo documents retained-TaskID collision swallowing retries — `infra/asynq/client.go:93-105`).
- All 08:00 retries fire in one burst (N × Excel gen + pro LLM call); stagger.
- Chat panel must lazy-load (`lib/lazy.tsx`); mobile entry point must be decided (bell FAB returns `null` on mobile — `NotificationFAB.tsx:34`).

## Scope challenge (RESOLVED 2026-10-11)

Devil's Advocate: phases 5–7 (38 of 79 tasks) are P2 speculative value; option C (~1 week, phases 1–4 mechanics) + error-catalog work was the brainstorm's own "fallback if chat underwhelms". **Resolution: user decision — ship all 7 phases straight through, no adoption gate.** The recommendation to gate P2 on measured P1 adoption was explicitly considered and declined; instrumentation + business baselines remain as tuning inputs. This is a user decision — do not re-litigate it in later reviews absent new evidence.

## Open business inputs (unblockers for phase 5/6 go)

- Measured current cost per sao kê cycle (minutes, this month).
- Top-3 real partner import questions with examples (verify against `import-errors.ts` coverage).
