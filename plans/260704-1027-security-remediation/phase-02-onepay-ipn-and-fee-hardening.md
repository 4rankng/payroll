---
phase: 2
title: "OnePay IPN and fee hardening"
status: pending
priority: P1
dependencies: [phase-01-money-integrity-core]
---

# Phase 2: OnePay IPN and fee hardening

## Overview
Harden the OnePay money-in path. **M8:** IPN amount isn't cross-checked (must run in the async asynq processor, not the webhook receiver). **M9:** replay window is attacker-controllable and the nonce has no wiring point (and the clock source is unverified). **M12:** the disbursement `ResolveFee` fails open to `fee=0` (and the signature change cascades). Red-team corrected layers, paths, and the clock source; see `## Red Team Corrections`.

## Requirements
- Functional: a forged-amount IPN is rejected before the FSM applies it; a replayed IPN is rejected; a missing fee schedule surfaces as an error, not `fee=0`.
- Non-functional: nonce store must not block disbursement reconcile when Redis blips (fail-open + alert); tighten the replay window only after proving the clock source.

## Architecture
- **M8 (corrected layer):** the webhook happy path is **async** — `WebhookHandler.Receive` calls `asynqClient.EnqueueIPNProcess` (`disbursement/webhook.go:152`); it only `applyIPNSync` on asynq failure. So the amount cross-check must run in the **asynq IPN processor** (service layer, `WalletPaymentService`), NOT in `Receive` or in `onepay/webhook.go` (which has no repo handle). Order in the processor: load persisted request via `GetByRequestID` (`wallet_payment_service.go:220`) → assert `event.Amount == persisted.Amount` (and currency) → reject + non-2xx on mismatch. Move the `WalletIPN` audit-row write to **after** the cross-check passes (today it's written before, `:126-149`).
- **M9 (corrected wiring + clock):** `onepay/webhook.go` has **no Redis dep** today; adding one to `parseAndVerifyWebhook` would cascade through the `Provider.VerifyAndParseWebhook` interface + 9Pay adapter. Instead, wire the nonce check in **`WebhookHandler.Receive` AFTER `VerifyAndParseWebhook` succeeds**, keyed on `event.RequestID + event.ProviderRef` (the payload has **no `jti`** — uses `FundsTransferID`/`TransactionID`). **Fail-open** (log + alert, accept IPN) so a Redis outage doesn't block all disbursement reconcile. **Clock:** before tightening ±900s → ±60s, verify `p.client.now == clock.Now()` (current default is `time.Now` at `webhook.go:17` — prod scratch containers run UTC while the DSN forces `time.Local` to Asia/Ho_Chi_Minh, per `lesson_timezone_loc_local_prod_utc`); log the `elapsed` distribution for 24h at the current window first. Asymmetric bound: reject future-dated entirely, or `elapsed < -5 || elapsed > 60`.
- **M12 (corrected path + cascade):** pin to **`internal/app/services/disbursement/fee_schedule_service.go:279`** (NOT `advance_payment/fee_schedule_service.go:237` — different signature, different path). Change `ResolveFee(ctx, provider, at)` → `(fee int64, waived bool, err error)`; missing OnePay schedule → `err`. **Cascade:** `ResolveFee` → `GetDisbursementFeeVND` (`:66-68`) → `Initiate` (`wallet_payment_service.go:135`) must propagate the error (today `Initiate` swallows the fee path); worker (`disbursement_execute_worker.go:108-114`) must surface it as a real failure, not fall into `ErrDuplicatePaymentInProgress`. Preserve `feature_fee_zero_on_preflight` (zero-on-preflight, honest at transfer). The `advance_payment` `ResolveFee` (fails open to `0.02*amount`) is **explicitly out of scope** here — tracked as a follow-up.

## Related Code Files
- Modify: asynq IPN processor + `backend/internal/app/services/disbursement/wallet_payment_service.go` (M8 — amount cross-check in `ApplyIPN`/processor; `GetByRequestID` :220; move audit-row write)
- Modify: `backend/internal/transport/http/handlers/disbursement/webhook.go` (M9 — nonce in `Receive` after verify; audit-row ordering)
- Modify: `backend/internal/infra/disbursement/onepay/webhook.go` (M9 — verify/fix `now` clock source at :17)
- Modify: `backend/internal/app/services/disbursement/fee_schedule_service.go` (M12 — `ResolveFee` :279, `GetDisbursementFeeVND` :66-68) + `wallet_payment_service.go:135` (`Initiate` propagation) + `disbursement_execute_worker.go:108-114`
- Reference: `feature_fee_zero_on_preflight` (impl 2026-06-25, NOT deployed — confirm no signature conflict if it ships first); Redis client in `internal/infra/cache`
- Tests: forged-amount IPN rejection (M8, via the processor), replay-within-TTL rejection + Redis-down fail-open (M9), missing-schedule error propagation through `Initiate` + worker (M12)

## Implementation Steps
1. **M8 — site.** Move/insert the amount+currency cross-check in the **asynq IPN processor** (after `GetByRequestID`, before status update). Reorder the audit-row write to after the check.
2. **M8 — surface parse errors.** Replace the discarded `Int64()` error with a logged rejection.
3. **M9 — clock verification.** Grep-confirm `p.client.now == clock.Now()`; if not, wire `clock.Now()`. Log `elapsed` distribution for 24h before tightening.
4. **M9 — window.** Asymmetric bound (`elapsed < -5 || elapsed > 60`); env-configurable; monitor rejection rate.
5. **M9 — nonce.** In `Receive`, after `VerifyAndParseWebhook` succeeds: `SETNX ipn:nonce:{RequestID}:{ProviderRef} 1 EX 600`; on conflict, reject. Fail-open on Redis error (log + alert, accept).
6. **M12 — signature + cascade.** Change `disbursement.ResolveFee` → `(fee, waived, err)`; propagate through `GetDisbursementFeeVND` → `Initiate` → worker. Missing OnePay schedule for a chargeable route → `err`. Stage behind a flag for first deploy.
7. **Tests.** Forged amount (M8); replay/TTL + Redis-down fail-open (M9); missing-schedule error reaches the worker (M12).

## Success Criteria
- [ ] M8: a forged-amount IPN is rejected in the processor and never reaches the FSM; audit row written only after the check.
- [ ] M9: replayed IPN within TTL rejected; Redis-down fails open (IPN accepted, alert fired); clock source verified as `clock.Now()`.
- [ ] M12: missing OnePay schedule for a chargeable route surfaces as an error at the worker (no silent `fee=0`); `feature_fee_zero_on_preflight` preserved.
- [ ] `advance_payment/fee_schedule_service.go` fail-open documented as a separate follow-up (not silently included).

## Risk Assessment
- **M8 placed in `Receive`:** the whole point — forged IPNs would still be enqueued + applied async. *Mitigation:* site is the processor; review must confirm it's not in `Receive`.
- **M9 window too tight / wrong clock:** mass-rejects legit IPNs. *Mitigation:* verify clock first; 24h distribution log; env-configurable; auto-widen monitor if rejection > threshold.
- **M9 Redis fail-closed:** would block all disbursement reconcile. *Mitigation:* fail-open + alert.
- **M12 fail-closed blocks disbursements:** if a schedule legitimately doesn't exist for a new route. *Mitigation:* flag-gated first deploy; explicit zero-fee allowlist.

## Red Team Corrections
- **M8 wrong layer (High):** webhook.go has no repo handle; happy path is async. Moved to the asynq IPN processor; audit-row write reordered.
- **M9 unwired + nonexistent `jti` + wrong clock + no fail policy (High):** nonce wired into `Receive` (not the adapter interface), keyed on real fields, fail-open, clock verified before tightening.
- **M12 wrong path + missing cascade (High):** pinned to `disbursement/fee_schedule_service.go:279`; cascade `ResolveFee→GetDisbursementFeeVND→Initiate→worker` made explicit; `advance_payment` twin scoped out.
