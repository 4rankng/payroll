---
phase: 1
title: "Migration: pending check-in columns"
status: pending
priority: P1
effort: "1h"
dependencies: []
---

# Phase 1: Migration — pending check-in columns

## Overview

Add nullable pending-activation columns to `project_employees` so a deferred check-in enable survives restarts. Mirrors the existing `pending_payment_schedule` / `schedule_effective_from` pair.

## Requirements

- Functional: new columns `pending_check_in_enabled` (TINYINT NULL) and `check_in_effective_from` (DATE NULL) on `project_employees`; existing rows untouched (NULL = no pending change).
- Non-functional: additive only — no backfill, no downtime; nullable so old code paths ignore them.

## Architecture

Same shape as the payment-schedule deferral: request stores intent + effective date; a scheduled job applies due rows. The check-in toggle gains a pending state without a new table.

## Related Code Files

- Create: `backend/migrations/104_project_employees_pending_check_in.up.sql` (+ `.down.sql`) — number per existing chain (verify latest number before writing; chain was at 103 as of 2026-08-16)
- Reference: `backend/internal/domain/project_employee.go` — entity struct (`PendingPaymentSchedule`/`ScheduleEffectiveFrom` neighbors show the pattern)

## Implementation Steps

1. Check `ls backend/migrations/*.up.sql | tail -3` for the next migration number.
2. Write `.up.sql`: two `ALTER TABLE project_employees ADD COLUMN ...` statements (nullable).
3. Write `.down.sql`: drop both columns.
4. Add the two fields to `domain.ProjectEmployee` with GORM tags + JSON tags following the schedule-pending neighbors (pointer types: `*bool`, `*time.Time`).
5. `cd backend && go build ./...` — compile check.

## Success Criteria

- [x] `golang-migrate` up/down both apply cleanly against local dev DB (`make db` running)
- [x] `go build ./...` green
- [x] Existing rows show NULL/NULL in both new columns

## Risk Assessment

Low. Additive nullable columns; standing prod pattern (`ALTER ADD COLUMN` authorized as deploy). Verify column names don't collide with anything existing (`SHOW COLUMNS FROM project_employees LIKE 'check_in%'`).
