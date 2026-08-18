---
title: "Track invalid bank info employees on bang cong"
description: >-
  Make the invalid-bank-info warning list on the bảng công (timesheet) pages
  always show employees whose bank account OnePay confirmed invalid, for both
  Admin and Partner roles, desktop and mobile. Today the list is gated on
  "has pending timesheet or PENDING advance request", so confirmed-invalid
  employees without pending work disappear from view.
status: completed
priority: P1
effort: 1d
tags:
  - feature
  - backend
  - frontend
  - employee
  - timesheet
  - bank-validation
blockedBy: []
blocks: []
branch: main
created: '2026-08-18'
createdBy: 'ak:plan (fast mode, HOLD scope)'
---

# Track invalid bank info employees on bang cong

## Overview

The payroll system already validates employee bank info against OnePay on every
create/update/import (`BankAccountValidator`, migration 095) and already
surfaces a "Thông tin ngân hàng không hợp lệ" warning section
(`MissingBankDetailsSection`) on all four bảng công pages (admin/partner ×
desktop/mobile). It also already excludes `unverified` status from the list so
OnePay outages don't flood admins with false alarms.

The gap: `buildMissingBankDetailsBaseQuery`
(`backend/internal/infra/persistence/query_builders/employee_project_query_builder.go:240`)
requires every listed employee to have a **pending timesheet or PENDING advance
request**. An employee whose account OnePay confirmed invalid, but whose work
is already approved/paid, vanishes from the list — so admins/partners stop
being reminded to fix the bank info until new pending work appears. This plan
makes **confirmed-invalid** rows always visible (subject only to the
active-project assignment rule), while **missing-info** rows keep the existing
pending-work condition. The UI also gains an explicit row-kind distinction
("Sai thông tin" vs "Thiết" — see phase 3) so the two populations are not
conflated in one amber box.

Out of scope (locked by scope challenge, 2026-08-18):

- Persisting wallet "Tra cứu tài khoản" verdicts onto employees — yesterday's
  completed plan `260818-1500-admin-wallet-employee-bank-lookup` decided
  lookups stay read-only. Not reversed here.
- Row-level invalid badges on individual timesheet rows, notification nudges,
  or any re-verification cron. The list is the single reminder surface.
- New migrations — `bank_account_status` / `bank_account_invalid_reason` /
  `bank_account_validated_at` columns already exist (migration 095, applied
  prod).

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Employees with `bank_account_status = 'invalid'` always appear in the bảng công warning list (admin + partner, desktop + mobile) — no pending-work gate | P1 |
| 2 | Missing-info employees keep the current pending-work + active-project gating (no behavior change) | P1 |
| 3 | Row-level distinction between "sai thông tin" (OnePay-confirmed invalid) and "thiếu thông tin" (missing fields) in the warning UI | P2 |
| 4 | No regressions: partner scoping, unverified exclusion, and existing list tests keep passing | P1 |

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Backend: query relaxation](./phase-01-backend-query-relaxation.md) | Completed |
| 2 | [Frontend: distinguish invalid vs missing rows](./phase-02-frontend-row-kind-distinction.md) | Completed |
| 3 | [Validation and release readiness](./phase-03-validation-and-release-readiness.md) | Completed |

## Success Criteria

- [x] An employee with `bank_account_status = 'invalid'`, zero pending
  timesheets, zero PENDING advance requests, currently assigned to an active
  project, appears in `GET /employees/missing-bank-details` results
- [x] The same employee does NOT appear when their active-project assignment
  ends (`last_date` set) — the active-project rule still applies to invalid rows
- [x] An employee with merely missing bank fields and zero pending work still
  does NOT appear (unchanged behavior)
- [x] `bank_account_status = 'unverified'` employees never appear (unchanged)
- [x] Partner requesters only see employees in their accessible set
  (unchanged; covered by `applyFilters` → `AccessibleBy`)
- [x] Warning section title/badge on bảng công pages distinguishes counts:
  invalid vs missing, e.g. "Sai thông tin ngân hàng: 3 · Thiếu thông tin: 1"
- [x] `make api-test` passes; `pnpm lint && pnpm type-check` pass; targeted
  Go/TS unit tests pass

## Implementation Notes (sync-back, 2026-08-18)

- **Deviation 1 — no `bank_warning_kind` DTO field.** The endpoint response
  already carries `bank_account_status` per row (`buildEmployeeListResponse` →
  `bankAccountStatusFields`), so the frontend derives the kind client-side via
  the new `getBankInformationWarningKind` util. KISS/DRY: zero duplicated
  contract, deploy-order safe both directions.
- **Deviation 2 — Employee type unchanged.** `bank_account_status?` already
  existed at `frontend/src/types/api/employee.types.ts:64`.
- **Verified live (local dev, employee 1129)**: invalid + complete + active
  project + zero pending work → now listed (previously hidden). Admin sees 7
  rows; partner `cuongnv` correctly sees only 2 (scoping intact).
- **api-test**: 271/298 passed; the 4 failures (3× advance-payment cutoff
  clock, 1× loan interest) are byte-identical to the pre-change baseline run
  of 2026-08-17 22:28 (269/296) — pre-existing, unrelated to this change.
- **Code review**: APPROVE_WITH_NOTES; both optional hardening
  recommendations applied (DryRun test now pins invalid-branch-before-pending
  EXISTS ordering; direct unit cases added for
  `getBankInformationWarningKind`).
- **Known cosmetic (accepted)**: an invalid+missing-fields row (manual DB
  edit) shows "Sai thông tin" badge beside a "Thiếu …" reason — both true;
  pre-existing reason-util ordering.

## Key Decisions (Locked)

| Decision | Choice | Rationale |
|---|---|---|
| Which rows escape the pending gate | `status = 'invalid'` only | User decision: invalid → always, missing → keep current rule. Unverified stays excluded (fail-open design) |
| Wallet lookup persistence | Read-only, unchanged | Approved decision from completed plan 260818-1500; not reversed |
| Row-kind distinction | Derived in backend DTO/handler from same fields the frontend util uses | Keeps single source of truth in the SQL predicate + backend response; frontend renders label |

## Related Files

**Backend (modify only):**

- `backend/internal/infra/persistence/query_builders/employee_project_query_builder.go` — restructure `missingBankDetailsPredicate` + base query conditions
- `backend/internal/transport/http/handlers/employee/employee_list.go` — expose `bank_warning_kind` (or reuse existing fields) in list responses
- `backend/internal/app/services/employee/bank_validation.go` — no change expected; reference for status semantics

**Backend (tests):**

- `backend/internal/infra/persistence/query_builders/employee_project_query_builder_test.go` — new cases for relaxed invalid visibility
- `builder tests for predicate shape` via integration tests `backend/tests/integration/flow_employee_crud.go` if contract changes

**Frontend (modify only):**

- `frontend/src/components/employees/MissingBankDetailsSection.tsx` — row-kind badge/label rendering
- `frontend/src/utils/bank-information-warning.ts` — extend util (and its test) with a kind-classifier
- `frontend/src/components/employees/MissingBankDetailsSection.test.tsx`

**No changes:** bank validation pipeline, wallet lookup dialog, OnePay adapters, migrations, routes (endpoint already exists at `GET /employees/missing-bank-details`).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Invalid employees without pending work accumulate forever | List grows over years; admin fatigue | Out of scope per user decision; the fix path (admin corrects bank info → next save re-validates → status flips to valid) is the existing intended lifecycle |
| Row count growth adds frontend render cost | Minor | Existing pagination (`limit`/`offset` in builder) still applies; section renders `max-h-80` scrollable table |
| Query shape change breaks partner scoping | Data leak | Scoping lives in `applyFilters` before the predicate; tests in phase 3 assert partner isolation explicitly |
| Mislabeling unverified as invalid in UI | False alarms | Kind-classifier keys off `status === 'invalid'` exactly, same as the SQL predicate |

## Plan Context

- Follows completed plan `260818-1500-admin-wallet-employee-bank-lookup` (same domain, read-only decision preserved)
- Predecessor feature: migration 095 + `BankAccountValidator` (shipped; see `bank_validation.go` doc comments)
- Backend detail: `backend/CLAUDE.md`, `backend/AGENTS.md`
- Frontend detail: `frontend/CLAUDE.md`, `frontend/AGENTS.md`
