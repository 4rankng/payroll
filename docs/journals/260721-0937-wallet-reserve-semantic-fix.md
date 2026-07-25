# Wallet reserve semantic fix

**Date**: 2026-07-21 09:37
**Severity**: High
**Component**: Wallet demand forecast / reserve guidance
**Status**: Resolved

## What Happened

The wallet reserve recommendation was using the wrong time window. We had built it around the 2-day request-created-at lead window, which made sense for a reminder, but it was the wrong semantic boundary for the reserve itself. That model could return zero or near-zero in periods where demand was still coming later in the flexible-pay cycle.

The fix changed the recommendation to a remaining-cycle model: include everything left in the current cycle, plus any residual amount still unsatisfied on the current day, then add the all-time net payable obligation for requests that are still actionable. `PENDING` and `APPROVED` requests now count as live obligations even when they fall outside the bounded forecast history.

## The Brutal Truth

This was a semantic bug, not a math tweak. We were answering the right question in the UI and the wrong question in the backend. That is the kind of mistake that feels small during implementation and then wastes time because the numbers look plausible while still being wrong.

What makes it annoying is that the old behavior was easy to rationalize as conservative. It wasn’t conservative. It was truncating the reserve to a narrow lead window and calling that the wallet target.

## Technical Details

- `GetDemandForecast` now uses `GetTotalPayableAmount()` to pull the all-time net obligation for `PENDING` + `APPROVED` requests.
- The forecast math switched from a lead-window-only distribution to `forecastRemainingCycleDistribution()`, which keeps same-day residual demand and all future cycle-day demand.
- `LeadDays` and `HorizonCycleDay` remain in the API as metadata, but they no longer cap `RecommendedBalance`.
- Regression coverage was added for:
  - remaining-cycle behavior before period start
  - residual demand on cutoff day
  - non-double-counting of current-day payable demand
  - `PENDING`/`APPROVED` inclusion only
  - all-time carryover outside the forecast history window

## What We Tried

- We first kept the old lead-window framing and patched around the zero cases. That was the wrong fix because it preserved the wrong boundary.
- We rejected widening the lead window because it still tied the reserve to request timing instead of remaining cycle demand.

## Root Cause Analysis

The root cause was a design mistake: the reserve logic had been coupled to the request-created-at window instead of the remaining flexible-pay period. Once the forecast question became “what must stay in the wallet for the rest of this cycle?”, the old 2-day horizon was obviously incomplete.

## Lessons Learned

- Reserve guidance and reminder windows are not the same thing.
- If a target represents money that may still be paid out, count actionable `PENDING` and `APPROVED` obligations across the full history, not just the statistical lookback.
- Add regression tests around semantic boundaries, not just numeric outputs.

## Next Steps

- Keep an eye on cached PWA clients: the new copy and behavior land with the next bundle/service-worker refresh, so stale clients may still show the old lead-window language until they reload.
- If this regresses again, the first thing to check is whether the reserve target is being bounded by display metadata instead of actual remaining-cycle demand.

## Follow-up: active-cycle state conditioning (2026-07-25)

The remaining-cycle boundary was correct, but the distribution was still
unconditional once a cycle was underway. It removed the historical amount
observed on the current day, yet ignored the cumulative request pace already
visible in the active cycle. This let the card keep showing a historical-only
reserve after the payroll entitlement had been uploaded and employees had
started requesting and receiving advances.

The active-cycle forecast now:

- re-centres the historical remaining-demand distribution on the observed
  current-cycle pace once enough cycle data exists for the cohort-median model;
- subtracts completed and currently payable demand before projecting genuinely
  new requests, avoiding double counting;
- caps future projected requests at the unused uploaded `max_adv_amount`
  capacity without treating that capacity as guaranteed demand;
- preserves the historical same-day residual on the cutoff day, where the
  current day is necessarily incomplete;
- drives the chart's current line from the backend p50 forecast instead of an
  independent historical mean; and
- refreshes desktop and mobile every 15 seconds during an active cycle, with an
  immediate invalidation after a successful payroll import.
