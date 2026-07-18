---
phase: 2
title: "Receive Verified Resend Events"
status: pending
priority: P1
dependencies: [1]
effort: "2 days"
---

# Phase 2: Receive Verified Resend Events

## Overview

Add a static public POST route that verifies raw bytes with the pinned Resend SDK, accepts only bounded email events, and durably applies or classifies each callback before responding.

## Context Links

- [Resend contract](./research/resend-webhook-contract.md)
- `backend/internal/infra/email/resend_provider.go:23`
- `backend/internal/transport/http/handlers/disbursement/webhook.go:118`
- `backend/internal/app/bootstrap/routes.go:26`
- `backend/internal/transport/http/middleware/rate_limit.go:154`
- Pinned verifier: `github.com/resend/resend-go/v2@v2.28.0`

## Requirements

- Route: `POST /api/v1/webhooks/email/resend`, outside JWT/Casbin and outside the generic `/api/v1` rate-limit group.
- Verify untouched body plus `svix-id`, `svix-timestamp`, and `svix-signature` before parsing.
- Support `email.sent`, `email.delivery_delayed`, `email.delivered`, `email.bounced`, `email.failed`, and `email.suppressed`.
- Parse `data.email_id`, exactly one impacted `data.to` recipient, event time, and exact tag contract.
- Return `200` for applied/stale/duplicate/pending/dead-letter/ignored supported results; return 200 for authenticated permanent invalid/unsupported events after a minimal classified receipt; reserve 5xx for transient durable-storage failure.
- Bound request bytes, identifiers, tags, collection sizes, and failure-code fields. Reject secrets without the `whsec_` prefix, valid base64, and a non-empty bounded decoded key at startup.
- Route is absent when disabled; `RESEND_WEBHOOK_ENABLED=true` requires one syntactically valid `whsec_...` secret.

## Architecture

Add an SDK-backed verifier adapter behind a domain port. `ResendWebhookService` parses a verified narrow payload, enforces:

- `app_source=tingting_payroll`
- `email_kind` in `generic|payroll_report|advance_payment_report`
- one well-formed `tracking_token`
- exactly one impacted recipient for outcome events

OTP lacks an allowed tracking contract and is acknowledged/ignored. Legacy untagged events may fall back to one unambiguous provider/message row; ambiguous or unknown events become classified receipts without mutating history.

The handler owns only bounded body/header extraction and HTTP translation. Use a short request timeout and a dedicated concurrency/burst policy sized for Resend replay traffic; do not let the shared 10,000/minute API bucket reject provider backlog before signature handling.

## Related Code Files

- Modify: `backend/internal/domain/email.go` — verifier headers/port and exact outbound tags.
- Modify: `backend/internal/infra/email/resend_provider.go` — map application tags into `resend.SendEmailRequest.Tags`.
- Create: `backend/internal/infra/email/resend_provider_test.go` — tag/request regression tests.
- Create: `backend/internal/infra/email/resend_webhook_verifier.go` — SDK `Webhooks.Verify` adapter.
- Create: `backend/internal/infra/email/resend_webhook_verifier_test.go` — valid/tampered/missing/expired cases.
- Create: `backend/internal/app/services/notification/resend_webhook_service.go` — payload normalization and receipt orchestration.
- Create: `backend/internal/app/services/notification/resend_webhook_service_test.go` — result/HTTP matrix and tag/event mapping.
- Create: `backend/internal/transport/http/handlers/resend_webhook_handler.go` — public bounded receiver.
- Create: `backend/internal/transport/http/handlers/resend_webhook_handler_test.go` — security and status contract.
- Modify: `backend/internal/config/config.go` — enable flag, signing secret, syntactic validation.
- Modify: `backend/internal/config/config_test.go` — empty, whitespace, malformed-prefix/base64, enabled/disabled cases.
- Modify: `backend/.env.example` — names/setup only; no secret values.
- Modify: `backend/internal/app/bootstrap/services/init.go` — construct/expose verifier and webhook service.
- Modify: `backend/internal/app/bootstrap/container.go` — construct/expose handler and dedicated middleware settings.
- Create: `backend/internal/app/bootstrap/routes_email_webhook.go` — static route on the engine, not protected v1 group.
- Modify: `backend/internal/app/bootstrap/routes.go` — mount the provider route before generic API grouping.

## Implementation Steps

1. Add `RESEND_WEBHOOK_ENABLED=false` and `RESEND_WEBHOOK_SECRET`; trim-reject whitespace and validate the `whsec_` prefix, base64, and non-empty bounded decoded key at boot. Keep one-secret scope.
2. Add exact outbound tag names/values and validate the email-kind allowlist. Do not repurpose arbitrary headers as tags.
3. Wrap `resend.Client.Webhooks.Verify`; verify raw bytes first and retain the SDK five-minute replay window.
4. Define narrow JSON structs and a complete result matrix for supported, unsupported, malformed-authenticated, foreign-source, OTP, legacy, duplicate, stale, and storage-failure cases.
5. Map bounce/failed/suppressed payloads to bounded internal codes/types. Discard raw messages, control characters, addresses, IPs, and SMTP identifiers from diagnostic text.
6. Implement handler body cap, request timeout, concurrency cap, structured low-PII logs, cheap indexed duplicate path, and dedicated replay-friendly rate policy.
7. Wire/mount only when enabled. The path must remain public but is not considered authenticated until signature verification succeeds.
8. Unit-test each event, exact/over-limit input, duplicate tags, multiple/zero impacted recipients, bad token, tampered body, expired/missing headers, malformed secret, replay burst, and storage outage.

## Todo List

- [ ] Verification always precedes JSON parsing and persistence.
- [ ] Exact tag/event/result contracts are exhaustive and tested.
- [ ] Multi-recipient events update exactly one expected recipient.
- [ ] Generic API limiting cannot cause provider retry storms.
- [ ] Permanent authenticated invalid events cannot trigger endless retries.

## Success Criteria

- [ ] Valid callbacks return 200 only after durable classification.
- [ ] Invalid signatures and oversized bodies write zero rows.
- [ ] `data.message_id` is never used for correlation.
- [ ] Captured replay cannot bypass `svix-id` dedupe or exhaust unbounded handler/database concurrency.
- [ ] `cd backend && go test ./internal/infra/email ./internal/app/services/notification ./internal/transport/http/handlers ./internal/config -race -v` passes.

## Risk Assessment

- Separate engine route can accidentally inherit/omit middleware. Mitigation: route-list and handler tests assert exact method/path and only intended security/timeout/concurrency middleware.
- Provider payload evolves. Mitigation: narrow parser, authenticated permanent-invalid receipt, metrics, and safe 200 response.
- One-secret rotation has a short controlled interruption. Mitigation: pause/replace endpoint, restart, then replay missed dashboard events; do not claim zero downtime.

## Security Considerations

- Never log the signing secret, raw body, recipient, subject, or provider message text.
- Signature is authenticity; JWT, CORS, source IP, and API key are not substitutes.
- Bound all data before JSON allocation/persistence where possible.

## Next Steps

Phase 3 exposes aggregate counts/status, adds bounded refresh, cleanup, and executable rollout checks.
