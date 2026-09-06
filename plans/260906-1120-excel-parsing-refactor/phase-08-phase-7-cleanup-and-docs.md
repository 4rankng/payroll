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
- [x] Delete `scripts/diagnose_bcc.go` (hardcoded local path)
- [ ] ~~Remove `includeFlexibleEmployees` from service chain~~ — DEFERRED: the flag is hashed into the idempotency fingerprint and persisted on TimesheetImportJob; removal breaks stored-fingerprint comparison for in-flight retries (behavior change beyond the approved Phase 6 exception). See plan.md deviations #4
- [x] Deleted grep-verified dead code: `EnqueueJob` no-op + `bccMetadata`/`bccImportMetadata` aliases (→ `BCCImportStats` direct). `ResultStrategyFactory` NOT deleted — audit finding wrong, it has live callers (result_processor.go:55,349); see plan.md deviations #5
- [x] NEW `internal/app/services/excel/AGENTS.md`: "how to add a template" (fixture → parser → registry line → test)
- [x] Fix stale docs: `docs/codebase-summary.md:143` (wrong worker mapping), `frontend/src/components/timesheet/AGENTS.md:76` (wrong flow description)
- [x] `"type:"` writer/reader contract cross-documented at both sites (a shared const would add a package edge for two strings); see plan.md deviations #8
- [x] Document recommended follow-ups: employee alias-headers, bulktransfer Local-tz fix, flexpay structured errors, name-normalizer unification (needs product input)

## Implementation Steps
1. Verify payroll zone committed (`git status`)
2. Deletions (grep zero non-test refs first)
3. AGENTS.md + doc fixes
4. `go test ./... && go vet` + grep sweeps + `make api-test`

## Success Criteria
- [x] Zero refs to deleted symbols; `go vet` clean; suites green
- [x] New-template walkthrough documented and matches reality

## Risk Assessment
Low. Rollback: revert.
