---
title: "Phase 7: cleanup and docs"
status: todo
priority: P2
effort: "0.5d"
dependencies: [phase-03, phase-04, phase-05, phase-06, phase-07]
---

# Phase 7: cleanup and docs

## Overview
Dead code gone, tools tell the truth, docs describe the new extension path.
(Dirty-zone precondition CLEARED 2026-09-06 — payroll files committed; verify at start.)

## Requirements
- [ ] Delete `scripts/diagnose_bcc.go` (hardcoded local path)
- [ ] Remove `includeFlexibleEmployees` from service chain (handler keeps its 400s — param is double-dead)
- [ ] Delete grep-verified dead code: `EmployeeImportProgressService.EnqueueJob` no-op, `bccMetadata`/`bccImportMetadata` alias, vestigial `ResultStrategyFactory`
- [ ] NEW `internal/app/services/excel/AGENTS.md`: "how to add a template" (fixture → parser → registry line → test)
- [ ] Fix stale docs: `docs/codebase-summary.md:143` (wrong worker mapping), `frontend/src/components/timesheet/AGENTS.md:76` (wrong flow description)
- [ ] Unify settlement `"type:"` consts with `payroll/report_internal_sheet.go` (verify payroll files committed first)
- [ ] Document recommended follow-ups: employee alias-headers, bulktransfer Local-tz fix, flexpay structured errors, name-normalizer unification (needs product input)

## Implementation Steps
1. Verify payroll zone committed (`git status`)
2. Deletions (grep zero non-test refs first)
3. AGENTS.md + doc fixes
4. `go test ./... && go vet` + grep sweeps + `make api-test`

## Success Criteria
- [ ] Zero refs to deleted symbols; `go vet` clean; suites green
- [ ] New-template walkthrough documented and matches reality

## Risk Assessment
Low. Rollback: revert.
