---
phase: 3
title: "Expose Status and Verify"
status: pending
priority: P1
dependencies: [2]
effort: "2 days"
---

# Phase 3: Expose Status and Verify

## Overview

Expose truthful aggregate status/counts in the existing history API, keep open history views fresh while delivery is non-terminal, bound receipt retention, and document an executable migration-first production rollout.

## Context Links

- `backend/internal/app/services/notification/email_service.go:328`
- `backend/internal/app/dto/email.go:75`
- `backend/tests/integration/flow_saoke.go:80`
- `backend/tests/integration/config.go:10`
- `frontend/src/hooks/api/useEmails.ts:37`
- `frontend/src/types/api/email.types.ts:36`
- `frontend/src/components/timesheet/EmailHistorySheet.tsx:190`
- `frontend/src/components/transaction/SaoKeHistoryDialog.tsx:51`
- `docs/deployment-guide.md:93`

## Requirements

- `GET /api/v1/email/history` adds `status`, `statusUpdatedAt`, `recipientCount`, `deliveredCount`, and `failedCount`; no raw failure message.
- Both history surfaces show shared Vietnamese labels for pending, accepted, sent, delayed, delivered, partial, and failed.
- Settlement state stays independent from delivery state.
- While an open history view contains non-terminal rows, refetch on a bounded interval; stop when terminal/closed and retain manual refresh.
- Applied/stale/ignored receipts expire after 30 days; pending becomes dead-letter after 24 hours; dead-letter expires after 90 days with daily counts/oldest-age alerting.
- Integration coverage is opt-in/capability-probed with a deterministic test secret. Core race/order tests remain mandatory real-DB/service tests.

## Architecture

Extend the current DTO/endpoint; do not add another read API. Put the status union, count-aware terminal predicate, label, icon intent, and visual variant in one `.ts` utility used by both `.tsx` surfaces. A `failed` aggregate with outstanding recipients is still non-terminal; a legacy row with `recipientCount=0` is historical and non-pollable. The hook enables a 10-second refetch only while a history surface is open and any loaded row is non-terminal.

Add receipt cleanup/reconciliation methods to `ResendWebhookService` and register one daily scheduler job. Pending rows older than 24 hours are marked dead-letter and produce a structured alert/admin notification with counts only; the cleanup is batched and idempotent.

## Related Code Files

- Modify: `backend/internal/app/dto/email.go` — aggregate delivery fields.
- Modify: `backend/internal/app/services/notification/email_service.go` — map null legacy state and new counts.
- Modify: `backend/internal/app/services/notification/email_service_test.go` — API mapping and legacy fallback.
- Modify: `backend/internal/app/services/notification/resend_webhook_service.go` — batched cleanup/dead-letter metrics.
- Modify: `backend/internal/app/bootstrap/scheduler_jobs.go` — daily cleanup/dead-letter alert job.
- Modify: `backend/internal/app/bootstrap/container.go` — pass webhook service through both `initScheduler` caller/definition and `registerSchedulerJobs`.
- Modify: `backend/tests/integration/config.go` — optional webhook test secret/capability.
- Modify: `backend/tests/integration/flow_saoke.go` — poll async history, signed HTTP update, replay/order assertions.
- Modify: `backend/tests/integration/models.go` only if shared history types live there.
- Modify: `frontend/src/types/api/email.types.ts` — full lowercase aggregate union/counts.
- Create: `frontend/src/utils/email-delivery-status.ts` — shared presentation and terminal predicate.
- Create: `frontend/src/utils/email-delivery-status.test.ts` — mandatory table-driven Vitest.
- Modify: `frontend/src/hooks/api/useEmails.ts` — visibility/non-terminal bounded refetch.
- Modify: `frontend/src/components/timesheet/EmailHistorySheet.tsx` — status/count badge and manual refresh.
- Modify: `frontend/src/components/transaction/SaoKeHistoryDialog.tsx` — status/count badge separate from settlement.
- Modify: `docs/deployment-guide.md` — preflight, manual DDL, endpoint/events, secret rotation/replay, rollback.

## Implementation Steps

1. Add aggregate DTO fields. Legacy rows with null status map to `accepted` and `recipientCount=0` as an explicit historical/unknown sentinel; never pretend all recipients delivered or poll legacy rows forever.
2. Implement and unit-test one frontend mapping: `Đang chuẩn bị`, `Đã tiếp nhận`, `Đã gửi`, `Giao chậm`, `Đã giao`, `Giao một phần`, `Gửi thất bại`.
3. Render accessible text/icon badges and delivered/failed totals in both surfaces. Keep settlement labels/actions visually and logically separate.
4. Add 10-second bounded refetch only when open + non-terminal, plus manual refresh. Derive terminality from status and recipient counts; test failed-with-pending, fully terminal, legacy/unknown-count, and unknown-status handling.
5. Add daily batched cleanup: terminal receipts 30 days, pending -> dead-letter at 24 hours, dead-letter 90 days. Log/notify total pending, oldest pending age, dead-letter count, and cleanup failures.
6. Make live-server webhook tests opt-in. Add `RESEND_WEBHOOK_TEST_SECRET`; probe the route and skip with an explicit reason when disabled. Use synthetic callbacks against a known history message/token; keep webhook-before-finalize and concurrency in real-DB/service tests where ordering is controllable.
7. Run focused backend/frontend tests, full backend race suite, existing API suite, and graph update.
8. Production pre-deploy gate: take/verify backup; run row/duplicate/length/index preflight; copy migration 092 to server; apply it manually before container recreation; verify new tables/columns/indexes; abort deploy on any DDL/postcheck failure.
9. Deploy code with webhook disabled and verify history. Register `https://tingting.vip/api/v1/webhooks/email/resend` with `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.bounced`, `email.failed`, `email.suppressed`; install the `whsec_...` secret; enable/restart; replay test events and verify aggregate counts.
10. Rotate with controlled interruption: pause/delete old endpoint, create replacement, install its one new secret, restart, then replay events that occurred during the window. Do not support or claim overlapping secrets.

## Todo List

- [ ] API/frontend aggregate contract handles multi-recipient partial outcomes.
- [ ] Open views refresh only while non-terminal and can refresh manually.
- [ ] Receipt retention/dead-letter policy is batched, observable, and tested.
- [ ] Integration setup is reproducible or explicitly skipped by capability probe.
- [ ] Migration is manually applied and verified before code deploy.

## Success Criteria

- [ ] A two-recipient delivered+bounced email shows `partially_delivered` with 1 delivered/1 failed in both UIs.
- [ ] Open history changes from non-terminal to terminal without closing/reopening; polling stops afterward.
- [ ] Duplicate/reordered signed callbacks do not change counts twice or regress state.
- [ ] `cd backend && go test ./... -v -race -cover` passes.
- [ ] `make api-test` passes; webhook subflow either passes with capability enabled or reports an intentional skip.
- [ ] `cd frontend && pnpm lint && pnpm test:run -- src/utils/email-delivery-status.test.ts` passes.
- [ ] `graphify update .` completes after implementation.

## Risk Assessment

- Manual production DDL can partially apply. Mitigation: additive statements, backup, pre/postchecks, lock timeout, idempotent repair notes, and deploy abort.
- Polling can multiply load. Mitigation: only open/non-terminal queries, 10-second floor, existing cache/query key, and immediate stop on terminal.
- Receipt cleanup can delete evidence too early. Mitigation: documented 30/90-day policy, batched state filters, and dry-run counts before deletion.

## Security Considerations

- UI renders internal status/counters only; no raw provider-controlled diagnostic text.
- Production docs name secrets but never include values.
- Admin alerts contain counts/IDs only, not recipient addresses.

## Rollback

1. Disable the webhook route and pause the Resend endpoint; provider dashboard retains replay capability.
2. Roll back frontend/API use of additive fields; old clients ignore them.
3. Keep additive schema/receipt data during application rollback. Export and run the down migration only after explicit approval; never make destructive DDL the first rollback step.
4. If migration 092 partially applied, run documented schema postchecks and idempotent forward-repair statements before any down action.

## Next Steps

After validation, implement sequentially with `/ck:cook /Users/dev/Documents/projects/payroll/plans/260718-1852-resend-email-delivery-webhook/plan.md`.
