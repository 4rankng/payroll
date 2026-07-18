# Bank transfer history was the right product, not reconciliation theater

**Date**: 2026-07-18 20:21
**Severity**: Medium
**Component**: Payroll bank-transfer history
**Status**: Resolved

## What Happened

The request started as a complaint about workers saying the payment amount was wrong. The useful clarification came later and was blunt: the screen should show bank-transfer history only, not reconciliation status, not payment logic, and not another monthly payroll workflow. We built a read-only Admin/Partner view for completed transfers, grouped by employee and fixed weekly cycle, with every bank reference number and amount visible inline.

The tricky part was historical data. Some batches were uploaded late, some transfers were split across multiple references, and some records only had enough metadata to reconstruct the employee-cycle grouping after the fact. We had to make the read path honest about that mess instead of pretending the data was clean.

## The Brutal Truth

The frustrating part is that the first version of this problem statement was too broad. If we had kept chasing “payment correctness” as a general reconciliation feature, we would have dragged payment behavior, statuses, and human dispute handling into a screen that should have stayed read-only. That would have been slower, riskier, and wrong.

What actually mattered was giving admins and partners a place to inspect what was already transferred, with enough evidence to answer the inevitable “where did this amount come from?” question without mutating anything.

## Technical Details

- Added `GET /api/v1/payrolls/bank-transfer-histories`.
- Grouping is by employee + work month + fixed weekly cycle:
  - Kỳ 1: days 1–7, paid on day 10
  - Kỳ 2: days 8–14, paid on day 17
  - Kỳ 3: days 15–21, paid on day 24
  - Kỳ 4: days 22–28, paid on the 1st of the next month
- Only completed transfers are shown.
- Bank references are shown inline, including split payments like:
  - `FT26198846619959 : 1.548.000`
  - `FT26198940380850 : 450K`
- Late uploads are still discoverable because the read path does not rely on a single “current month only” assumption.
- Partner scope is enforced server-side against accessible projects.

Focused verification passed on the backend service/handler path and the frontend screen. Broader Go and integration runs still have unrelated failures elsewhere in the repo; those were left alone because they are not caused by this feature.

## What We Tried

- We did not add reconciliation statuses or a payment-accuracy workflow.
- We did not change payment percentages, cycle math, or transfer execution.
- We kept the UI read-only and made the history list the only visible surface.

## Root Cause Analysis

The real problem was scope confusion. The user needed evidence of transfers, not a new accounting system. Once we stopped trying to solve “payment correctness” as a general business process, the feature got much smaller and much safer.

Historical imports made the backend harder than the UI. Late uploads and split bank references meant the aggregation layer had to dedupe by reference, preserve amounts, and reconstruct grouping from imperfect metadata instead of trusting a neat one-row-per-payout model that does not exist.

## Lessons Learned

- Do not turn a history screen into a reconciliation engine unless the user explicitly asks for that.
- Bank transfer history must be read-only by default.
- Historical payroll data is messy; assume late uploads and split references will happen again.
- Partner access has to be enforced in the backend, not by UI convention.

## Next Steps

- Keep this endpoint and screen read-only unless product explicitly reopens reconciliation scope.
- If disputes keep coming up, define a separate reconciliation workflow instead of overloading history.
- Owner for follow-up bug reports: payroll/backend and payroll/frontend, same cycle.
