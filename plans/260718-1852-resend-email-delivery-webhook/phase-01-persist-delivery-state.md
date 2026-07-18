---
phase: 1
title: "Persist Delivery State"
status: pending
priority: P1
dependencies: []
effort: "2-3 days"
---

# Phase 1: Persist Delivery State

## Overview

Replace best-effort post-send history creation with a pre-send tracked attempt. Add per-recipient delivery rows and a privacy-minimal receipt inbox, then derive aggregate history state transactionally.

## Context Links

- [Resend contract](./research/resend-webhook-contract.md)
- [Email architecture scout](./reports/email-architecture-scout.md)
- `backend/internal/app/services/notification/email_service.go:612`
- `backend/internal/app/services/notification/email_publisher.go:68`
- `backend/internal/domain/notification.go:48`
- `backend/internal/infra/persistence/notification_repository.go:20`
- `backend/migrations/001_init_db.up.sql:536`

## Requirements

- Pre-create an email attempt and unique tracking token before calling the provider. A tracking write failure prevents the send.
- Record To, CC, and BCC as unique normalized expected recipients with role metadata and initial `pending` state; move them to `accepted` only after provider API acceptance.
- After provider response, atomically attach `email_provider` and provider message ID; on synchronous API error mark submission `failed` with a normalized internal code.
- Persist one minimal receipt per `svix-id`: tracking token, provider email ID, event type/time, recipient-row ID when resolved, result state, and timestamps.
- Keep provider event time separate from local observation time. Initial/submission state has no provider ordering timestamp.
- Serialize distinct callbacks for the same tracked email and recompute aggregate counts/status in the same transaction.
- Do not store raw payloads, raw diagnostic messages, subjects, bodies, or recipient addresses in the receipt inbox.

## Architecture

Refactor `EmailEventPublisher` into a focused delivery tracker used by `EmailService.dispatchWithMeta`:

1. `Begin` creates the notification, expected recipient rows, and an unguessable token.
2. The service adds tracking tags to the outbound `EmailMessage` and calls the provider.
3. `MarkSubmissionFailed` records a safe failure code when the API call fails.
4. `MarkAccepted` stores provider/name/message ID and claims any signed receipt that raced the provider response.

Add `EmailRecipientDelivery` and `ResendWebhookReceipt` domain contracts. Keep persistence in the existing notification repository to avoid a parallel email model. Receipt processing locks the target notification (`SELECT ... FOR UPDATE`) or uses a compare-and-swap predicate/retry, updates one recipient by normalized impacted address, then recomputes:

- all expected delivered -> `delivered`
- any delivered plus any terminal failure -> `partially_delivered`
- any terminal failure with no delivered recipient -> `failed` (terminal only when every recipient is terminal)
- otherwise delayed > sent > accepted/pending

Per-recipient provider states remain `accepted`, `sent`, `delivery_delayed`, `delivered`, `bounced`, `failed`, or `suppressed`. Older provider timestamps lose; equal timestamps use documented deterministic precedence; non-terminal events cannot overwrite a terminal recipient result.

## Related Code Files

- Create: `backend/migrations/092_resend_email_delivery_tracking.up.sql` — additive notification columns, recipient table, receipt inbox, indexes.
- Create: `backend/migrations/092_resend_email_delivery_tracking.down.sql` — recovery-oriented rollback; never drop populated data without export/confirmation.
- Create: `backend/internal/domain/email_webhook.go` — status, recipient, receipt, tracker/repository contracts and aggregate reducer.
- Create: `backend/internal/domain/email_webhook_test.go` — per-recipient and aggregate transition matrix.
- Modify: `backend/internal/domain/email.go` — tracking token/tags and delivery tracker contract.
- Modify: `backend/internal/domain/notification.go` — provider, tracking token, aggregate status/counts, observation/provider timestamps.
- Modify: `backend/internal/app/services/notification/email_publisher.go` — implement Begin/accepted/failed tracking operations.
- Modify: `backend/internal/app/services/notification/email_service.go` — pre-create, tag, send, finalize sequence.
- Modify: `backend/internal/app/services/notification/email_service_test.go` — both constructor callers and failure ordering.
- Modify: `backend/internal/infra/persistence/notification_repository.go` — tracking lifecycle and locked receipt application.
- Create: `backend/internal/infra/persistence/notification_repository_email_webhook_test.go` — real-DB concurrency/race tests.

## Implementation Steps

1. Specify exact domain constants, terminal sets, aggregation table, maximum identifier/tag lengths, recipient normalization/dedup rules, and normalized failure codes.
2. Run production preflight queries: notification row count/size, non-empty provider-ID count, duplicate IDs, long IDs, and index presence. Duplicate/oversize results are a hard rollout stop until quarantined/remediated.
3. Design migration 092 as additive/nullable. Avoid a table-wide status backfill; map legacy null to `accepted` in reads. Use MySQL 8 online/instant DDL where supported, explicit lock timeout, backup verification, and post-schema assertions.
4. Add a normal index for legacy provider-ID lookup and unique indexes for new tracking tokens, `(notification_id, normalized_recipient)`, and `svix_id`. Do not let sandbox counter reuse weaken production correlation.
5. Implement pure recipient/aggregate reducers. Test older/newer/equal timestamps, terminal protection, concurrent delivered/bounced recipients, all-delivered, all-failed, failed-with-pending, partial, and submission failure.
6. Implement `Begin` in one transaction. Create the email notification only for messages passing `EmailMessage.Validate`; include To/CC/BCC recipient rows and preserve settlement metadata.
7. Change `dispatchWithMeta`: begin tracking before send; add tags; call provider; mark safe submission failure or accepted result. If finalization fails after provider acceptance, retain the pre-created row/token for webhook recovery and return the provider ID without inviting a duplicate send.
8. Implement receipt insert/apply under notification lock/CAS. Cross-check token, provider, message ID, and impacted recipient. Quarantine mismatches instead of updating history.
9. Add focused fault-injection tests for Begin failure (provider not called), provider error, finalization failure, webhook-before-finalize, duplicate `svix-id`, and concurrent distinct events.

## Todo List

- [ ] Pre-send tracking makes immediate API failures visible and prevents untracked sends.
- [ ] Multi-recipient aggregate semantics are explicit and table-tested.
- [ ] Callback updates serialize per notification and are idempotent.
- [ ] Migration has production preflight, online-DDL, postcheck, and recovery steps.
- [ ] No raw provider diagnostics or duplicate recipient PII enter the receipt inbox.

## Success Criteria

- [ ] `delivered` is impossible until all expected recipients are delivered.
- [ ] A mixed delivered/failure email becomes `partially_delivered` regardless of callback order.
- [ ] A callback received before provider finalization is later claimed by token and cannot be made stale by a local timestamp.
- [ ] Begin failure performs zero provider calls; provider failure leaves a visible failed history row.
- [ ] `cd backend && go test ./internal/domain ./internal/app/services/notification ./internal/infra/persistence -race -v` passes.

## Risk Assessment

- Refactoring send tracking changes failure semantics. Mitigation: preserve provider success response on finalization failure, use token-based recovery, and enumerate both existing `NewEmailService` callers.
- Online index creation can still contend on production. Mitigation: row/size preflight, backup, maintenance window, lock timeout, and abort criteria.
- Duplicate recipient addresses across To/CC/BCC can distort totals. Mitigation: normalize and dedupe once, retaining the highest-visibility role only for display.

## Security Considerations

- Tracking tokens are random, unique, non-secret correlation capabilities; never use sequential notification IDs in tags.
- Normalize and bound every provider-controlled identifier before DB entry.
- Persist reason codes/types only; redact/drop provider message text.

## Next Steps

Phase 2 verifies and normalizes Resend callbacks into this transactional contract.
