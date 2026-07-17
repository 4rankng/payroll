# Phase 1: Correct forecast composition

## Change

Remove the summary-reader dependency. Compose the target-Kỳ forecast from
approved value observed in the current target Kỳ plus projected approvals from
the current cycle-day through its pay date.

## Validation

- Unit test injects a large outstanding-payment summary and proves it is never read.
- Current target-Kỳ rows are excluded from historical basis but included in the observed floor.
