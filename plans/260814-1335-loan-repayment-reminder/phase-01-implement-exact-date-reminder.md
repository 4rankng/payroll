---
phase: 1
title: "Implement exact-date reminder"
status: done
effort: ""
---

# Phase 1: Implement exact-date reminder

## Overview

Implement the reminder without changing financial-processing or public contracts.

## Implementation Steps

1. Extend `backend/internal/domain/loan_repayment_schedule.go` with a reminder projection and a dedicated exact-date repository method; retain `GetDueSchedules` unchanged.
2. Implement one bounded, eager-loaded query in `backend/internal/infra/persistence/loan_repayment_schedule_repository.go` for pending schedules in `[tomorrow 00:00, following day 00:00)`, including loan code and lender name without N+1 calls.
3. Add a small reminder orchestration helper/service near the existing notification scheduler path that formats one Vietnamese plain-text body plus safe HTML email content, gathers Admin emails via `ListByRole`, and invokes the two channels independently.
4. Register `loan_repayment_reminder` in `backend/internal/app/bootstrap/scheduler_jobs.go` at `0 9 * * *`; wire only the required repositories/services through `backend/internal/app/bootstrap/container.go`.
5. Use `clock.Now()` for business dates and `NotificationTypeLoanInterestDue` for stored Admin notifications/Web Push.
6. Add focused unit/repository/bootstrap tests using existing test utilities and patterns.

## Success Criteria

- [x] Exact tomorrow-only and pending-only selection is proven, including month/year rollover. (`loan_repayment_schedule_reminder_test.go` — window bounds, end-boundary exclusion, paid/today/later/deleted rows out, month rollover)
- [x] Multiple rows are consolidated and formatted with full `vi-VN`-style VND digits. (`loan_repayment_reminder_service_test.go` — consolidation, sorted loan codes, FormatVND amounts + total)
- [x] Admins with nil/blank email still receive in-app/Web Push; only valid non-empty addresses enter the email request. (same test — nil/blank/duplicate emails trimmed and deduped)
- [x] Zero qualifying rows cause no notification or email send. (`TestLoanRepaymentReminderService_NoSchedulesSendsNothing`)
- [x] No schema, frontend, route, response, or financial transaction behavior changes. (diff scope: domain projection + repo method + service + scheduler/DI wiring only)
