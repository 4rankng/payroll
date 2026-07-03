# Plan: Enforce "one OPEN attendance per employee/day" (fix duplicate check-in race)

Status: implementing · Branch: main · 2026-07-03

## Problem
`CheckIn` read-then-write (GetByEmployeeAndDate → Create) has a TOCTOU. Mig 078 dropped `UNIQUE uq_employee_date`, so concurrent first-check-ins can both insert → duplicate open rows. Bounded (no double pay; checkout orders open-first), but breaks the 1/day invariant, adds reporting noise, double-fires the auto-reject asynq task. Confirmed by passing reproduction test.

## Solution (option A — partial unique via stored generated column)
Declaratively enforce "at most one OPEN row per employee/day" at the DB. `open_key` is non-NULL only for open rows; MySQL allows multiple NULLs in a unique index, so closed/rejected rows (re-check-in after no-salary checkout) coexist. Stored generated column auto-recomputes on INSERT/UPDATE → checkout/auto-reject free the slot with no app code.

## Changes
1. **Migration 081** (`081_add_attendance_open_row_unique.{up,down}.sql`): add `open_key DATE GENERATED ALWAYS AS (CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NULL THEN date ELSE NULL END) STORED` + `UNIQUE(employee_id, open_key)`.
2. **`attendance_repository.go` `Create`**: translate duplicate-key → `domain.NewValidationError("Bạn đã vào làm trong ngày hôm nay rồi")` (matches existing sequential dup-check UX; repo precedent: `user_repository.go`).
3. **Invert reproduction test** (`attendance_checkin_concurrency_test.go`): fake now enforces the open-row unique; assert exactly 1 row + one friendly validation error.

## Acceptance
- Concurrent first-check-ins → exactly 1 row; loser gets friendly "đã vào làm" validation error; only 1 auto-reject task enqueued.
- Re-check-in after confirmed-no-salary checkout still works (closed row open_key=NULL, new open row open_key=date).
- `go test -race ./internal/app/services/attendance/` green; no other package broken.
- Mig applies on demo+prod MySQL (version ≥ 5.7.6 confirmed read-only on prod).

## Out of scope
- Frontend, deploy execution (verify-only on prod; deploy to demo/prod is a separate user-approved step).
- Removing the sequential dup-check (kept as fast-path friendly error).

## Pre-flight (before applying migration anywhere)
`SELECT employee_id, date, COUNT(*) c FROM attendances WHERE check_out_time IS NULL AND salary_reject_reason IS NULL GROUP BY employee_id, date HAVING c>1;` — must be empty, else `ADD UNIQUE` fails (surfacing existing corruption to reconcile).

## Rollback
`DROP INDEX uq_attendances_employee_open; DROP COLUMN open_key;` — non-destructive.
