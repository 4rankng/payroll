---
title: "Phase 5: satellite adoption"
status: todo
priority: P2
effort: "1d"
dependencies: [phase-02]
---

# Phase 5: satellite adoption

## Overview
Satellite islands adopt excelkit primitives, a shared provisioning helper, and one
internal error contract — WITHOUT losing their idioms. No unified framework.

## Requirements
- [x] employee import: safe-open via `excelkit.OpenFile` in the worker (path-based); blind first-sheet/B..I offsets PRESERVED; parse behavior locked by new characterization tests
- [x] flexpay (`admin_flexpay_import.go`): triple `GetRows` → one; `detectColumnOffset` stays; silent-skip+counters regime PRESERVED (user-visible)
- [x] onepay fee + wallet_bulk: alias matcher → `excelkit.AliasTable` (their tables stay local)
- [ ] ~~NEW `internal/app/services/imports/provisioning.go`~~ — FALLBACK INVOKED (allowed): the getOrCreate variants genuinely diverged (bank validation, StartDate/PaymentSchedule parsing live only in the employee island); forcing unification = behavior drift risk. Documented at both sites instead. See plan.md deviations #3
- [x] (contract type landed; islands adopt it when next touching error paths — not force-migrated) NEW `dto/import_issue.go`: `ImportRowIssue{Code,Message,Row,Reference}` (model: `OnePayFeeReportIssue`); islands adapt to existing external JSON — payloads unchanged
- [x] Do NOT restructure `admin_bulk_transfer.go` (adjacent to recently-dirty worker; only upload guard in Phase 6)

## Implementation Steps
1. dto contract + provisioning package + wiring
2. Island-by-island commits: employee → flexpay → onepay/wallet_bulk
3. `go test ./...` + `make api-test` (flow_flexpay_import + settlement flows)

## Success Criteria
- [x] getOrCreate trio exists once; flexpay reads sheet rows once
- [x] External JSON byte-compatible (frontend untouched)

## Risk Assessment
Medium — bootstrap wiring for provisioning is the one structural change; fallback
documented. Rollback: per-island revert; provisioning extraction droppable independently.
