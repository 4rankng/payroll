---
phase: 1
title: domain-api
status: completed
effort: medium
---

# Phase 1: domain-api

## Overview

Create a strict Admin-only filtered command and make rejection safe for every non-paid state.

## Implementation Steps

1. Add request/response contracts for `project_id`, ISO dates, and trimmed reason; reject invalid/reversed ranges.
2. Add a dedicated `/timesheets/reject-unpaid` route and Partner deny regression.
3. Implement a transaction-aware repository command with project/date predicates, `payment_status <> paid`, and a concurrent-safe affected-row count.
4. Clear active approval/edit/force-payroll state while preserving historical payment-attempt fields.
5. Publish the existing bulk-reject event with actual count and project metadata.
6. Add domain/repository/handler tests for eligible states, paid protection, isolation, validation, zero-match idempotency, and concurrent paid protection.

## Success Criteria

- [x] All non-paid states are rejected in the selected scope.
- [x] Paid rows cannot be overwritten, including a concurrent change.
- [x] Response and audit metadata use actual affected count.
- [x] Partner is denied.
