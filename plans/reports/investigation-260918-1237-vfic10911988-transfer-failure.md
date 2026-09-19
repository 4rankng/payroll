# Investigation: VFIC10911988 transfer failure (prod)

**Date:** 2026-09-18 · **Scope:** prod `tingting.vip` · **Type:** disbursement transfer rejected (code 15)

## Summary

Batch 4 (2 rows, 2026-09-18 11:36) — row `VFIC10911988` (VU THI THU HA, MSB `MCOBVNVX`, 2.423.750 ₫) failed with OnePay `response_code=15` `INVALID_ACCOUNT_INFO` ("Thông tin tài khoản không hợp lệ"). The other row (VietinBank) succeeded. **Root cause: transient bank-side rejection relayed by OnePay's payout switch — not an application bug.**

## Timeline (prod DB, +0700)

| Time | Event |
|---|---|
| 11:36:15.178 | Batch 4 created |
| 11:36:16.087 | wallet_payment #498 created (`VFIC10911988`), fee 3.850 ₫ stamped |
| 11:36:17.877 | OnePay transfer endpoint sync-rejected: `error_code=15`, "Thông tin tài khoản không hợp lệ" → row `failed`, `settled_at` stamped |
| 11:36:19.5 | Batch finalized: 1 ✅ / 1 ❌, fee_booked |

## Evidence chain

1. **Failure came from the transfer endpoint, not the check step.** Fee 3.850 ₫ charged proves the account-check step passed (`FeeWaived: !check.Valid` would have zeroed it). `error_code=15` is recorded via `RecordSyncResponse(Accepted=false)` — the funds-transfer call's response code.
2. **Account is genuinely valid.** Free lookup (GET /customers) validated account 0385167516 (code 00, confirmed name "VU THI THU HA") — verified live by the user's "Tra cứu tài khoản" after the incident.
3. **Not the name-echo bug.** Fix `d9abb47b` (account-number echo in holder-name match) was committed 09:56:23 and the prod image built 09:56:58, before the 11:36 batch. Even on a pre-fix binary, an echo would have failed at the check step with `name_mismatch` (like the 80 rows on 2026-08-25), never reaching the transfer with code 15.
4. **Payload is well-formed.** The VietinBank row in the same batch succeeded on the identical code path. MCOBVNVX has 78 prior successes (2026-06-25 → 2026-09-07).
5. **Precedent: this failure mode is transient.** 2026-06-25: MSB account 2640117855252 failed with the same code 15 + fee charged; a retry **19 minutes later succeeded**, followed by 6 more successful transfers to the same account.

## Why "not counted into receivables / not linked to timesheet"

By design. `booking.go:159-160`: failed rows contribute **fee-only** to the ledger (3.850 ₫); principal never moved, so no receivable is created and nothing links to the timesheet (bảng công). The batch banner states this correctly.

## Root cause

OnePay's lookup endpoint validated the account, but its payout switch rejected the transfer with `15 INVALID_ACCOUNT_INFO` 1.8s later — a provider/bank-side transient rejection (MSB channel at that moment). The same mode occurred 2026-06-25 and succeeded on retry.

## Remediation

1. **Re-run the beneficiary in a new batch** (new VFIC — same request_id returns 409 duplicate). Expected outcome: success, as with the June precedent. Retry costs another 3.850 ₫ fee only on attempt.
2. Optional: ask OnePay support to confirm `funds_transfer_id=VFIC10911988` at 2026-09-18 11:36:17 (+0700) returned 15 INVALID_ACCOUNT_INFO on their side.

## Prevention (optional, non-urgent)

- Consider surfacing `MessageVI(15)` + "retryable" hint in the bulk-transfer UI so ops can distinguish bank-transient failures (15/19/86/9x) from data errors (name_mismatch, invalid bank code).
- The failed row's `recipient_name` is clean; no data repair needed.
