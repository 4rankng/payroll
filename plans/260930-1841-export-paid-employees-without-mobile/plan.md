---
title: Export Paid Employees Without Mobile
description: >-
  Admin-only Excel export of employees who received salary or FlexPay payment
  in the last N months but have no mobile number on file, to judge which
  unreachable employees are still active.
status: in-progress
priority: P2
branch: main
tags:
  - feature
  - backend
  - frontend
  - payroll
  - advance-payment
  - export
blockedBy: []
blocks: []
created: '2026-09-30T18:45:00+07:00'
createdBy: 'ak:plan'
source: skill---

# Export Paid Employees Without Mobile

## Overview

Admin-only Excel export answering: "which employees did we still pay recently (salary or FlexPay advance) but cannot reach because no mobile number is on file?" Received payment = at least one paid timesheet (`payment_status='paid'`, `paid_at` in window) OR at least one COMPLETED advance payment request (`advance_payment_requests`, `paid_at` in window) within the last N months. Employees with an account-active status are included regardless of working status so the admin can judge activity from payment recency.

## Scope

- In: admin-only export endpoint, combined salary+advance activity query (window bounded 1–24 months via `months`), xlsx with Vietnamese headers, per-source totals/counts/last-paid dates, working status, projects, admin Employees page months dialog + export action, export audit event.
- Out: partner/accountant access, invalid (non-empty) mobile detection, app-login activity metrics, changes to payment calculations or history pages.

## Acceptance Criteria

- `POST /api/v1/employees/export-paid-without-mobile` body `{months: 1..24}` returns an .xlsx; non-admin roles are denied.
- Window computed from `clock.Now()` (ADR-006); an employee appears once, with salary totals/count/last-paid and advance net totals/count/last-paid for the window.
- Rows: `employees.deleted_at IS NULL` and (`mobile IS NULL OR mobile = ''`); includes inactive employees with a Trạng thái column (working/unassigned per project-assignment rule).
- Columns: Mã NV, Họ tên, CCCD, Trạng thái, Tổng lương đã nhận, Số đợt trả lương, Lần trả lương cuối, Tổng tạm ứng đã nhận, Số lần tạm ứng, Lần tạm ứng cuối, Hoạt động gần nhất, Dự án.
- Export emits the standard file-export audit event with data_type/record_count/file_name.
- `go build ./...` and `go test ./...` pass; `pnpm lint && pnpm type-check` pass; `make api-test` executed against the running dev backend.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | Backend query + endpoint + Excel | Completed |
| 2 | Frontend export action on admin Employees page | Completed |
| 3 | Verification | Completed |

## Dependencies

- Phase 2 depends on Phase 1's endpoint contract (`{months}` → xlsx blob).
- Phase 3 depends on Phases 1–2.
