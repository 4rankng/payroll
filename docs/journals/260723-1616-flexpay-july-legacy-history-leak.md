---
title: "FlexPay July legacy history leak"
date: "2026-07-23 16:16"
severity: "High"
component: "frontend/src/utils/advancePaymentHelpers.ts"
status: "Resolved"
---

## Context

On July 23, the live API returned top-level zeros, `quotas: []`, and a pile of all-time completed history entries with no `forMonth`. The last number was the trap: `11,340,000` looked like usable July usage when it was just legacy history that had lost its month tag.

## What Happened

`getAdvanceQuotaSummaryForMonth` treated those untagged records as if they belonged to July, then rendered the employee state as exhausted at 100%. The UI was confidently wrong: July was not open, but it also was not consumed.

## The Brutal Truth

This was a stale-history leak disguised as a quota calculation. The page looked alive and deterministic, which made the bug harder to notice and more annoying to debug. We shipped a path that trusted old data more than the API’s explicit month signal.

## Evidence / Red-Green Verification

- New component regression reproduces the bad case with `11,340,000` in legacy history.
- The helper tests now pin the rule set around explicit quota rows and legacy fallback handling.
- `177` tests pass.
- `tsc`, `lint`, and `build` pass.
- Live browser verification now shows `Chưa mở / Chưa có hạn mức / Chờ bảng lương tháng 07/2026` instead of an exhausted July state.

## Root Cause

The helper mixed two sources of truth. It let untagged history act like current-month usage even when the API had already provided explicit quota structure. That was the wrong default. Legacy history is only valid when the quota list is omitted entirely, or when it can be tied to the exact resolved quota row.

## Decision

We made explicit quota arrays authoritative. Untagged history now only counts as legacy fallback when the API omits the quota list, or when it matches the single exact quota row that the helper has already resolved. Anything else stays out of the July calculation.

## Impact

The employee portal stopped showing July as exhausted/100% when it was really waiting on the July payroll upload. That prevented a false sense of already-used quota and restored the intended blocked-but-not-consumed state.

## Next Steps

Keep the regression in the frontend test suite and watch for any future API shape drift around `quotas` and legacy history payloads. If the backend ever changes the history contract again, the helper needs another explicit rule, not another inference.
