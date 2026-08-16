---
title: Standardize displayed VND units to the dong symbol
status: completed
priority: P2
effort: small
branch: main
tags: [currency, typography, frontend]
created: 2026-08-15
---

## Scope

- [x] Make shared VND formatters and their fallbacks emit `₫`.
- [x] Update the loan KPI's explicit VND unit and direct monetary display literals.
- [x] Add regression coverage and validate affected frontend output.
