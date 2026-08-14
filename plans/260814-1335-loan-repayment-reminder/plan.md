---
title: "Loan repayment reminder"
description: "Remind every Admin one day before a pending loan repayment by in-app/Web Push and email, then release safely to production."
status: in_progress
priority: P1
branch: "main"
tags: []
blockedBy: []
blocks: []
created: "2026-08-14T05:35:40.591Z"
createdBy: "ck:plan"
source: skill
---

# Loan repayment reminder

## Overview

At 09:00 Asia/Ho_Chi_Minh each day, find pending repayment schedules due on the next calendar day and send one consolidated Vietnamese reminder through the existing Admin notification and email channels. Use the persisted schedule amount, keep delivery at-least-once, and avoid frontend, API, or schema changes.

## Acceptance criteria

- The cron job is named `loan_repayment_reminder`, uses `0 9 * * *`, and evaluates dates with the configured Asia/Ho_Chi_Minh business clock.
- Only `pending` schedules whose `due_date` equals tomorrow are included; today, overdue, later, and paid rows are excluded.
- The reminder includes loan code, lender, full persisted scheduled amount, and due date, consolidated into one message per Admin and one email send to all trimmed non-empty Admin email addresses.
- In-app notification persistence and Web Push use `NotificationTypeLoanInterestDue`; no channel is invoked when no schedules qualify.
- Existing public APIs, database schema, financial transaction creation, and `GetDueSchedules` semantics remain unchanged.
- Focused tests, backend unit/race checks, lint, `make api-test`, code review, graph updates, backup validation, deployment, and production health checks complete successfully.

## Scope

**In scope:** exact-date query/projection, scheduler/DI wiring, consolidated Vietnamese content, focused automated coverage, full-worktree commit/push, backup, production deploy, and independent verification.

**Out of scope:** exactly-once delivery state, new migrations, frontend changes, public API changes, lender notifications, overdue reminders, and automatic loan payment.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Implement exact-date reminder](./phase-01-implement-exact-date-reminder.md) | In Progress |
| 2 | [Verify and review](./phase-02-verify-and-review.md) | Pending |
| 3 | [Backup deploy and verify](./phase-03-backup-deploy-and-verify.md) | Pending |

## Dependencies

- Existing `NotificationService`, `EmailService`, scheduler, Admin role repository, and loan repayment schedules.
- Production scheduler timezone must remain `Asia/Ho_Chi_Minh`.
- Release follows `make backup` -> validate `.sql.gz` -> `make deploy` after commit and source push.

## Risks and rollback

- Manual/concurrent cron execution may resend because durable deduplication is intentionally out of scope.
- Push and email are best-effort independent channels; one failure must be logged without mutating loan state.
- Roll back application code to the prior commit and redeploy the prior healthy image if production verification fails; the feature has no migration to reverse.
