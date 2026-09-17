---
phase: 2
title: "Self-healing BCC import backdate"
status: pending
priority: P1
effort: "6h"
dependencies: [1]
---

# Phase 2: Self-healing BCC import backdate

## Overview

When a BCC file contains entries earlier than an employee's assignment start,
backdate the assignment to the file's earliest entry date for that employee
BEFORE entry building — so validation passes and the import succeeds instead of
rolling back wholesale.

## Requirements

- Functional:
  - For every (employee, earliest entry date E) in a parsed file: if an active assignment exists and `StartDate > E`, set `StartDate = E`.
  - Backdate runs in its OWN committed transaction before the main import transaction; the `assignment:{projectID}:{employeeID}` Redis cache key is deleted AFTER that commit (ADR-007) so the main transaction's validation re-reads the new value.
  - Applies to all four format paths: legacy (bcc_import_process), multi-position, weekly-BCC, weekly-payment.
  - Skips employees whose failed rows are protected for other reasons (approved/paid rows) — those keep current behavior.
- Non-functional:
  - One audit-log entry per backdated assignment (`entity_type=project_employee`, action UPDATE, metadata old/new start + source file).
  - Backdate survives import failure: extending coverage backward is evidence-aligned and harmless; do NOT couple it to the import transaction.

## Architecture

New shared helper in `app/services/bcc_import_pipeline.go`:

```go
// backdateAssignmentsForImport aligns assignment start dates with the earliest
// entry each imported file proves. Own transaction per ADR-007 cache ordering.
func (s *BCCImportService) backdateAssignmentsForImport(
    ctx context.Context,
    byCCCD map[string]*domain.ProjectEmployee,   // from step 6 (assignments load)
    earliestEntryByCCCD map[string]time.Time,     // per-format adapter computes
) (backdated []BackdateRecord)
```

Each format path already parses sheets before loading assignments; add a small
adapter computing `earliestEntryByCCCD` from its parsed struct (all four
`Parsed*` types expose per-employee entries). Call the helper right after the
assignments load (step 6), before entry building (step 7). The `byCCCD` map is
mutated in place so downstream steps see new dates.

Cache ordering (critical): backdate tx → commit →
`validationService.InvalidateEmployeeAssignmentCache(projectID, employeeID)` →
main import tx. Without the delete, a cached assignment from a previous failed
upload (TTL `constants.EmployeeAssignmentCacheTTL`) keeps rejecting rows even
after the DB row changed — this exact trap would have broken the 12:43 recovery
upload had the partner's edit happened seconds earlier.

## Related Code Files

- Modify: `backend/internal/app/services/bcc_import_pipeline.go` (helper)
- Modify: `backend/internal/app/services/bcc_import_process.go`, `bcc_import_multi_position.go`, `bcc_import_weekly_bcc.go`, `bcc_import_weekly_payment.go` (adapters + calls)
- Modify: `backend/internal/domain/services/timesheet_validation_assignment.go` — no change to validation; reuse existing `InvalidateEmployeeAssignmentCache`
- Modify: `backend/tests/integration/flow_bcc_weekly_import.go` (new scenario)

## Implementation Steps

1. Implement the helper: per-employee check, own `WithTransaction`, post-commit cache delete, audit log, slog.Info per backdate.
2. Add earliest-entry adapters for the four parsed formats (pure functions — unit-testable).
3. Wire calls in the four format paths after assignments load.
4. Unit tests: adapter correctness; helper skips when `StartDate <= E`; helper skips employees absent from `byCCCD` (they get auto-created in later steps with smart start from Phase 1).
5. Integration test scenario in `flow_bcc_weekly_import.go`: create assignment starting tomorrow, upload file with yesterday's entries, expect status=completed and assignment start backdated.

## Success Criteria

- [x] Incident replay: assignment start = today, file covers 5 earlier days → import completes, assignment start = file's earliest date
- [x] Cache integration: after backdate commit + cache delete, `ValidateEmployeeAssignment` passes for the earlier dates (unit test against real Redis or cache fake)
- [x] Approved/paid protected rows still block the import (unchanged `planBCCReplacement` semantics — regression test)
- [x] `go test ./internal/app/services/... -race` green

## Risk Assessment

- **Silently rewriting a deliberately-set start date.** An admin may set 09-17 on purpose (employee contract start). Counter-view: the uploaded file is payroll evidence that the employee worked 09-10; the record should match reality, and today's incident shows partners cannot diagnose the alternative. Mitigation: audit log + slog on every backdate; the value remains editable.
- **Assignment updated concurrently** (admin edits while import runs): backdate uses a targeted `UPDATE project_employees SET start_date = ? WHERE id = ? AND start_date > ?` (compare-and-set) — a concurrent earlier value wins, a concurrent later value gets overwritten by ours; both end states are valid coverage extensions. Signal for breakage: audit rows showing start oscillating between imports; response: switch to `GREATEST`-style guarded update.
- **Phase 1 dependency**: if Phase 1 slips, this phase still works standalone (backdate is independent of the default rule).
