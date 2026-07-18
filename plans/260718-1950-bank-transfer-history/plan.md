---
title: Bank Transfer History
description: >-
  Show completed bank transfers grouped by employee and fixed weekly payroll
  cycle for Admin and Partner.
status: completed
priority: P2
branch: main
tags:
  - feature
  - backend
  - frontend
  - payroll
  - payments
blockedBy: []
blocks: []
created: '2026-07-18T11:50:24.005Z'
createdBy: 'ck:plan'
source: skill
---

# Bank Transfer History

## Overview

Add a read-only Vietnamese bank-transfer history for Admin and Partner. It uses completed bulk-transfer evidence, groups results by employee and weekly cycle, and displays every bank reference with its amount and an aggregate total. It does not change payment calculations, percentages, schedules, or transfer execution.

## Scope

- In: completed transfers, employee-cycle aggregation, bank references and amounts, monthly/cycle/search/project filters, Admin and Partner access, responsive UI.
- Out: reconciliation statuses, amount validation, monthly payroll, payment calculation changes, retry/refund/correction controls.

## Acceptance Criteria

- Split payments appear as separate reference/amount lines under one employee-cycle total.
- Only completed transfers with bank references appear.
- Kỳ 1–4 use work windows 1–7, 8–14, 15–21, and 22–28.
- Partner data is restricted server-side to accessible projects.
- Existing payroll and transfer behavior is unchanged.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Backend bank-transfer history](./phase-01-backend-bank-transfer-history.md) | Completed |
| 2 | [Frontend history screen](./phase-02-frontend-history-screen.md) | Completed |
| 3 | [Verification](./phase-03-verification.md) | Completed |

## Dependencies

- Phase 2 depends on Phase 1's API contract.
- Phase 3 depends on Phases 1 and 2.
- No overlap with the active Resend webhook plan.
