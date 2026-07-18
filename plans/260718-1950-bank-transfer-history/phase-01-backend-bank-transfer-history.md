---
phase: 1
title: Backend bank-transfer history
status: completed
priority: P1
dependencies: []
---

# Phase 1: Backend bank-transfer history

## Overview

Create a protected read endpoint that aggregates completed bulk-transfer rows by employee and fixed weekly cycle while retaining all bank-reference/amount pairs.

## Requirements

- Functional: monthly work-month filter, optional cycle/project/employee/search filters, pagination, completed rows only.
- Security: Admin sees all; Partner is intersected with canonical accessible project IDs in the backend.
- Compatibility: keep `/payrolls/histories` unchanged.

## Architecture

Read filtered `bulk_transfer_files`, parse stored transaction rows, accept only completed rows with a non-empty bank reference, map each transfer to one of the four fixed weekly windows, then group by employee + work month + cycle. Preserve project metadata as a list and sort transfers deterministically.

## Related Code Files

- Modify: `backend/internal/app/dto/payroll.go`
- Modify: `backend/internal/domain/bulk_transfer_file.go`
- Modify: `backend/internal/infra/persistence/bulk_transfer_file_repository.go`
- Modify: `backend/internal/app/services/payroll/service.go`
- Modify: `backend/internal/transport/http/handlers/payroll.go`
- Modify: `backend/internal/app/bootstrap/routes_disbursement.go`
- Modify: `backend/configs/casbin_policy.csv`
- Add/modify focused backend and integration tests.

## Implementation Steps

1. Define request and response DTOs for employee-cycle history and constituent bank transfers.
2. Add the narrow repository query required to retrieve weekly bulk-transfer files for a work month and accessible projects.
3. Aggregate completed transaction rows in the payroll service without changing transfer data or status.
4. Enforce canonical Partner project access before aggregation.
5. Register `GET /api/v1/payrolls/bank-transfer-histories` and Partner GET permission.
6. Add unit/integration coverage for split transfers, cycle boundaries, exclusion states, pagination, and role scope.

## Success Criteria

- [x] One employee with two completed references returns one record with two transfers and the correct total.
- [x] Kỳ 4 includes days 22–28 only.
- [x] Non-completed and reference-less transactions are absent.
- [x] Partner cannot access unrelated projects.
- [x] Existing payroll-history contract remains unchanged.

## Risk Assessment

Historical batch JSON is the source of truth for this read-only view. Avoid treating timesheet `payment_status` alone as bank evidence. Limit batch selection before in-memory aggregation to keep the query bounded.
