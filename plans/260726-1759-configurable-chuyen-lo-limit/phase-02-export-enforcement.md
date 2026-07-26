---
phase: 2
title: "Export enforcement"
status: completed
effort: "medium"
---

# Phase 2: Export enforcement

## Overview

Capture the setting once, preserve strict-below partition semantics, and ensure
generation failures cannot leave transaction-code persistence behind.

## Implementation Steps

1. Extend the Excel service's narrow settings interface with the typed getter.
2. Capture the threshold once during `ExportPlanner.Plan`, store it beside
   `PaymentPercentage` in `ExportPlan`, and pass it to workbook generation.
   Preserve existing exported methods with a default wrapper where practical.
3. Replace all fixed-limit comparisons in row building and partitioning with
   the captured `int64`. Use subtraction-based capacity checks to avoid
   `currentTotal + row` overflow.
4. Preserve strict semantics: a row must be `< limit`; a new partition starts
   when adding a row would make the total `>= limit`.
5. In `ExportService.Export`, generate/preflight the workbook before `Persist`.
   Publish audit and return the download only after both generation and
   persistence succeed.
6. Extend Excel/planner/export tests for custom limits, the default 400M
   boundaries, deterministic row preservation, immutable capture, overflow,
   and zero transaction-code writes after generation failure.

## Files

- `backend/internal/app/services/payroll/excel/service.go`
- `backend/internal/app/services/payroll/excel/service_test.go`
- `backend/internal/app/services/payroll/bulktransfer/planner.go`
- `backend/internal/app/services/payroll/bulktransfer/planner_test.go`
- `backend/internal/app/services/payroll/bulktransfer/export_service.go`
- focused `export_service_test.go` if no suitable test exists

## Success Criteria

- [x] `399_999_999` is valid at the default; `400_000_000` and above is not.
- [x] Every generated workbook total is strictly below the captured setting.
- [x] Mid-export setting changes cannot mix thresholds.
- [x] Generation failure performs no writes or audit publication.
- [x] Excel and bulktransfer package tests pass with race detection.

## Risks and rollback

- Reordering generation before persistence changes only failure ordering; the
  successful response, filename, transaction-code, and audit contracts remain
  unchanged.
- Revert to the default wrapper if a hidden internal caller cannot accept the
  captured threshold; do not restore persist-before-generation.
