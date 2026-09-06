---
title: "Phase 3: dispatch collapse"
status: todo
priority: P1
effort: "0.5d"
dependencies: [phase-03]
---

# Phase 3: dispatch collapse

## Overview
Collapse the `bcc_import_process.go:69-92` switch into a router map and name the
production-contract sites as tested helpers. Pure extraction; no behavior deltas;
`BCCImportService` wiring untouched.

## Requirements
- [ ] `resolveParseRoute(formatResult)` router helper replaces the switch (incl. legacy→ParseBCCData fallback + winner-format overwrite `:90`)
- [ ] Extract `finalizeBCCImport` (kills the 3rd copy of stats marshaling)
- [ ] Extract `autoCreateEmployeesFromSTK` (the `:162-357` STK block) + `crossCheckSTKNames`
- [ ] Named predicates for contract sites: file-position precedence (`:319`), label-keyed resolution (`:381`)
- [ ] NEW `bcc_import_dispatch_test.go`: golden asserting winner-format contract + predicate behavior by name

## Implementation Steps
1. Router helper + tests (dispatch golden)
2. Extract STK block → method with own test seam
3. Extract finalizer + predicates
4. `go test ./...` + `make api-test` (flow_bcc_import critical)

## Success Criteria
- [ ] `processAssetData` reads as: route → parse → shared pipeline steps (each named)
- [ ] Dispatch golden proves winner-format contract (`:90` semantics) unchanged

## Risk Assessment
High (production gotchas live here) but diff is mechanical and Phase 0/2 test-guarded.
Rollback: revert commit.
