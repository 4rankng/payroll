---
title: "Resend Email Delivery Webhook"
description: "Receive verified Resend events and show reliable aggregate and per-recipient delivery outcomes in admin email history."
status: pending
priority: P2
branch: "main"
tags: [feature, backend, frontend, api, database, security]
blockedBy: []
blocks: []
created: "2026-07-18T10:55:20.501Z"
createdBy: "ck:plan"
source: skill
---

# Resend Email Delivery Webhook

## Overview

Add `POST /api/v1/webhooks/email/resend` for signed Resend callbacks. Pre-create a tracked email attempt before provider submission, persist per-recipient outcomes, and derive one truthful aggregate status for the existing admin history.

`email.sent` means accepted for delivery; `email.delivered` confirms recipient-server acceptance. Resend emits distinct outcome events per impacted recipient, so the system must not use last-event-wins at message level.

## Scope

- In: emails sent through `EmailService`; submission failures; signed receipt inbox; To/CC/BCC recipient outcomes; aggregate status/counts; both admin history surfaces; cleanup/alerts; staged production setup.
- Out: OTP history (OTP bypasses `EmailService`), open/click/complaint analytics, automatic email resend, raw webhook payload retention, and provider diagnostic text retention.
- Legacy: pre-feature history remains readable as `accepted`; untagged callbacks update only when one existing provider/message correlation is unambiguous.

## Key Decisions

- Pre-create tracking before sending; if tracking fails, do not call Resend.
- Generate an unguessable tracking token and send exact tags: `app_source=tingting_payroll`, `email_kind=<allowlisted kind>`, `tracking_token=<token>`.
- Correlate signed callbacks by tracking token, then cross-check `data.email_id`; never use `data.message_id`.
- Store expected recipients and provider state per normalized address. Aggregate as `pending`, `accepted`, `sent`, `delivery_delayed`, `delivered`, `partially_delivered`, or `failed`.
- Store normalized failure codes/types only. Do not persist raw SMTP/provider messages.
- Deduplicate receipts by `svix-id`; serialize same-email updates with a row lock or compare-and-swap retry; order only provider events against provider timestamps.
- Use one active signing secret with a documented coordinated rotation/replay procedure.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Persist Delivery State](./phase-01-persist-delivery-state.md) | Pending |
| 2 | [Receive Verified Resend Events](./phase-02-receive-verified-resend-events.md) | Pending |
| 3 | [Expose Status and Verify](./phase-03-expose-status-and-verify.md) | Pending |

## Dependencies

- Existing Resend ID, notification history, scheduler, and integration reporter.
- Public HTTPS endpoint and Resend dashboard signing secret.

## Acceptance Criteria

- Missing/invalid/expired signatures, tampered bodies, and oversized callbacks write nothing; authenticated schema-invalid callbacks write only a minimal classified receipt.
- Duplicate, concurrent, raced, and out-of-order events produce deterministic per-recipient and aggregate state.
- Multi-recipient email cannot show `delivered` unless every expected recipient is delivered; mixed outcomes show `partially_delivered` with counts.
- Provider submission failure is recorded even when no webhook will arrive.
- Receipts have bounded retention; stale pending items become visible dead letters.
- Migration preflight/apply/postcheck precedes code deployment; rollback preserves captured data.
- Backend race tests, opt-in HTTP integration, frontend lint/Vitest, `make api-test`, and `graphify update .` are explicit gates.

## References

- https://resend.com/docs/webhooks/event-types
- https://resend.com/docs/webhooks/verify-webhooks-requests
- https://resend.com/docs/webhooks/retries-and-replays
- https://resend.com/changelog/webhook-event-visibility

## Red Team Review

Three adversarial reviews produced 23 raw findings, consolidated into 12 material corrections. All 12 were accepted or accepted with a narrower implementation; none remain unresolved.

| Correction | Plan response |
|------------|---------------|
| Resend outcomes are per impacted recipient | Added expected-recipient rows and aggregate counts/status. |
| Post-send history can be lost | Moved durable tracking before provider submission. |
| Local acceptance time can wrongly stale provider events | Separated local observation time from provider ordering time. |
| Distinct concurrent events can race | Required row locking or compare-and-swap retry plus transactional reduction. |
| Code can deploy before schema | Added manual migration preflight/apply/postcheck as a hard deploy gate. |
| Receipt storage can grow or orphan | Added terminal retention, pending dead-lettering, cleanup, and alerts. |
| Replay traffic can exhaust ingress | Added body/data bounds and dedicated timeout/concurrency/replay policy. |
| One secret cannot provide zero-downtime rotation | Documented a controlled interruption and dashboard replay procedure. |
| UI state would remain stale | Added open-and-nonterminal bounded polling plus manual refresh. |
| Live integration assumptions were not executable | Added opt-in secret/capability probing and kept race/order checks in real-DB tests. |
| Provider/message correlation can be ambiguous | Added exact application tags, a random tracking token, and `data.email_id` cross-checking. |
| Raw provider diagnostics create privacy risk | Limited persistence and UI to normalized internal codes and counts. |

## Validation Log

- Mode: Standard verification after hard-mode red team.
- Evidence: project instructions, current backend/frontend source, migration/deploy docs, pinned `resend-go/v2@v2.28.0`, and official Resend webhook documentation.
- Claims checked: 30 (10 per phase); verified: 30; failed: 0; unresolved: 0.
- Contract checks: phase dependencies are acyclic; every referenced existing file was found; planned new files are marked `Create`; repository commands match `Makefile`/`package.json`; phase success criteria cover security, ordering, concurrency, rollout, API, and UI behavior.
- Whole-plan sweep: re-read `plan.md` and all three phase files after red-team edits. Removed last-event-wins, message-level delivery, unbounded retention, raw diagnostic storage, generic-rate-limit dependency, and zero-downtime rotation assumptions. No stale phase references or contradictory status semantics remain.
