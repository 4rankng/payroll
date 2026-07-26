---
title: "Settlement reconciliation false delta"
date: "2026-07-26 11:09"
severity: "High"
component: "settlement simulation reconciliation"
status: "Resolved"
---

# Settlement reconciliation false delta

## Context

The settlement simulation was reporting a massive mismatch because it compared the wrong ledger scope against the wrong export scope. The bug lived in the reconciliation path only: no database writes changed, no API shape changed, and the settlement execution path itself stayed intact.

## What happened

The old comparator produced a false delta of `-352,097,029 VND` by mixing a date-window ledger total with paid principal from the wrong scope. The corrected path now compares transaction-scoped receivable values instead of the wage principal. With that fix, the expected receivable is `2,074,172,959` and the ledger value is `2,074,172,974`, so the delta is `+15`. That delta is accepted by the existing `1,000 VND` rounding tolerance.

One remainder of `340,000 VND` stays outside the numeric reconciliation scope as a separate remainder row. Shared partial transactions are excluded from the comparison entirely, and the simulation now emits an explicit warning when a covered timesheet is missing a transaction link.

## Reflection / root cause

The root cause was a bad comparison boundary, not a math bug. I let a ledger query built around a date window bleed into a transaction-level reconciliation check, then compounded it by including partially covered transactions in a place that only makes sense for whole transactions. The false delta looked dramatic enough to feel like a data corruption issue, but it was really a scope mismatch dressed up as arithmetic.

## Decisions

- Keep the fix inside the settlement simulation service and tests.
- Reconcile on transaction IDs only.
- Exclude partial transactions from both sides of the numeric comparison and warn instead of pretending the number is comparable.
- Treat missing transaction links as explicit evidence gaps, not silent zeroes.
- Accept the `+15` delta under the existing rounding tolerance rather than invent a new tolerance rule.

## Verification

The settlement simulation tests now cover the missing-link case, the `15 VND` tolerance case, and the partial-transaction exclusion case. The settlement flow passes with the corrected scope. Broader backend integration baselines still have unrelated failures, but they are not part of this fix and I did not pad this entry with their noise.

## Next / residual risk

The fix is stable as long as reconciliation stays transaction-scoped. The remaining risk is future code reintroducing a mixed-granularity comparator or quietly treating missing links as comparable values again. That would recreate the same lie with a different number, which is the kind of bug that wastes time because it looks plausibly correct until someone checks the actual money path.
