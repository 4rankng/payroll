---
phase: 2
title: "OnePay IPN and fee hardening"
status: pending
priority: P1
dependencies: []
---

# Phase 2: OnePay IPN and fee hardening

## Overview
Harden the OnePay money-in path: the IPN webhook doesn't cross-check the amount against the persisted request and has an attacker-controllable replay window (M8/M9); the fee schedule fails open to `fee=0` when a OnePay schedule is missing (M12 — silent revenue loss).

## Requirements
- Functional: an IPN whose amount ≠ the persisted request is rejected; replayed IPNs beyond a tight window are rejected; a missing fee schedule surfaces as an error/alert, not `fee=0`.
- Non-functional: nonce store is cheap (Redis TTL); fee fail-closed must not break legitimate *waived* schedules (distinguish "missing" from "waived").

## Architecture
- **M8:** in the IPN handler, after parsing, assert `event.Amount == persistedRequest.Amount` (and currency); reject + log on mismatch. Surface the `Int64()` parse error instead of discarding it.
- **M9:** replace the ±900s / `X-OP-Expires`-driven replay window with server-clock ±60s (no negative skew), cap `expires`; add a nonce/JTI TTL store in Redis so a replayed IPN is rejected on second sight.
- **M12:** `ResolveFee` returns `(fee, waived, err)`; missing schedule → `err` (caller rejects disbursement + alerts); waived → `fee=0, waived=true` (honest zero). Preserve `feature_fee_zero_on_preflight` (zero-on-preflight, honest at transfer time) — the fix targets *missing schedule*, not *waived*.

## Related Code Files
- Modify: `backend/internal/adapters/onepay/webhook.go` (M8 87-94, M9 60-70)
- Modify: `backend/internal/app/services/.../fee_schedule_service.go` (M12 — `ResolveFee` 279-302) + its caller
- Reference: `backend/internal/infra/cache` for Redis TTL nonce store; `feature_fee_zero_on_preflight` (impl 2026-06-25, NOT deployed — coordinate if this plan deploys first)
- Tests: webhook amount-mismatch rejection, replay rejection, missing-schedule error

## Implementation Steps
1. **M8 — amount cross-check.** Load the persisted request by IPN reference; compare `amount` (and currency). On mismatch, return non-2xx + alert log.
2. **M8 — surface parse errors.** Replace the discarded `Int64()` error with a logged rejection.
3. **M9 — window.** Server-clock ±60s; reject negative skew; cap `expires`. Make the threshold env-configurable.
4. **M9 — nonce store.** `SETNX ipn:nonce:<jti> 1 EX 600`; reject if already exists.
5. **M12 — fail-closed.** Change `ResolveFee` to `(fee, waived, err)`; missing schedule for a chargeable route → `err`; waived → honest zero.
6. **Tests.** Amount-mismatch (M8), replay within TTL (M9), missing-schedule error path (M12).

## Success Criteria
- [ ] M8: a forged IPN with a tampered amount is rejected and logged.
- [ ] M9: a replayed IPN within TTL is rejected; legitimate first-sight succeeds.
- [ ] M12: a missing OnePay schedule for a chargeable route no longer silently yields `fee=0`.
- [ ] `feature_fee_zero_on_preflight` semantics preserved (preflight `fee=0`, transfer-time honest).

## Risk Assessment
- **M8 false rejection on legitimate OnePay retry:** retries carry the same amount for the same reference — safe. *Mitigation:* idempotency key is the IPN reference, not the amount.
- **M9 window too tight:** OnePay clock skew > 60s would reject legit IPNs. *Mitigation:* env-configurable threshold + rejection-rate monitor before tightening further.
- **M12 fail-closed blocks disbursements:** if a schedule legitimately doesn't exist for a new route. *Mitigation:* explicit allowlist of zero-fee routes + alert; stage behind a flag for broad rollout.
