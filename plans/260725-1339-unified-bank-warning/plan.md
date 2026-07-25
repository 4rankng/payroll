---
title: Unified Bank Warning on Partner Timesheet
description: >-
  Treat incomplete and confirmed-invalid bank data as one actionable category,
  keep BCC imports usable, and refresh partner warnings immediately.
status: completed
priority: P1
branch: main
tags:
  - timesheet
  - employee
  - bank-validation
  - bcc
  - frontend
  - backend
blockedBy: []
blocks: []
created: '2026-07-25T05:39:43.332Z'
createdBy: 'ck:plan'
source: skill
---

# Unified Bank Warning on Partner Timesheet

## Overview

Keep the existing `MissingBankDetailsSection` as the single remediation surface
on desktop and mobile partner timesheets. Expand its backend predicate to catch
every incomplete required bank field as well as confirmed-invalid accounts,
while excluding transient `unverified` results. BCC imports may create or
update employees with invalid bank information, persist the validation verdict,
and continue importing timesheets; strict manual create/update rejection remains
unchanged.

When a BCC job reaches a terminal state, refresh the unified employee warning
query without a page reload. A completed import with row errors must render a
concise partial-success result instead of hiding those errors behind a pure
success message.

## Acceptance Criteria

- One section titled `Thông tin ngân hàng không hợp lệ` on desktop and mobile
  `/partner/timesheet` covers missing bank, missing account number, missing
  account holder name, and confirmed-invalid accounts.
- Each row has a concise Vietnamese reason based on current data; provider
  names and internal errors are never displayed.
- `unverified` accounts caused by provider outages remain outside the warning
  category unless required bank fields are missing.
- New BCC employees with complete-but-invalid bank data are created with
  `bank_account_status=invalid` and their timesheets continue importing.
- Existing BCC employee bank updates remain allow-and-flag.
- Manual employee create/update continues rejecting confirmed-invalid bank
  changes before persistence.
- Terminal BCC completion or failure refreshes the warning list automatically.
- Completed imports with `error_count > 0` show partial-success counts and
  actionable row details.
- Existing endpoint payloads and database schema remain compatible.
- Existing partner-accessible scope is preserved; selected-project filtering
  is not introduced.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Backend warning + import semantics](./phase-01-backend-warning-import-semantics.md) | Completed |
| 2 | [Frontend parity + import UX](./phase-02-frontend-parity-import-ux.md) | Completed |
| 3 | [Verification + rollout safety](./phase-03-verification-rollout-safety.md) | Completed |

## Dependencies

- Builds on the completed async BCC import plan at
  `plans/260724-1829-async-bcc-import/`.
- Preserve unrelated uncommitted mobile partner timesheet/header/test changes.

## Scope Exclusions

- No database migration or new endpoint.
- No payment, transfer, or disbursement behavior changes.
- No selected-project filtering for the warning section.
- No relaxation of manual employee bank validation.
- No commit, push, or deployment without separate user authorization.

## Rollback

Changes are code-only. Roll back the import-specific creation path, warning
predicate/reason helper, and frontend query invalidation/result rendering
together; no data migration rollback is required.
