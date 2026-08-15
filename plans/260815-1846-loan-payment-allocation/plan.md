---
title: Correct loan installment principal and interest allocation
status: completed
priority: P1
effort: medium
branch: main
tags: [loan, accounting, data-repair]
created: 2026-08-15
---

## Scope

- [x] Persist a principal/interest allocation for each loan schedule and its transaction.
- [x] Correct future payment aggregates and ledger entries.
- [x] Repair legacy generated schedules, loan aggregates, transaction allocations, and ledger entries.
- [x] Add regression tests and run focused, integration, and static verification (the local API sandbox remains on its pre-migration schema).
