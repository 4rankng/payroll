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
- [ ] employee import: safe-open via `excelkit.OpenReader`; blind first-sheet/B..I offsets PRESERVED; structured errors mapped to existing `dto.RowError` JSON
- [ ] flexpay (`admin_flexpay_import.go`): triple `GetRows` → one; `getOrCreate` trio delegates to shared provisioning; `detectColumnOffset` stays; silent-skip+counters regime PRESERVED (user-visible)
- [ ] onepay fee + wallet_bulk: alias matcher → `excelkit.AliasTable` (their tables stay local)
- [ ] NEW `internal/app/services/imports/provisioning.go` (getOrCreate trio ~180 LOC ×2: `employee/import_service.go:269,300,433` ↔ `admin_flexpay_import.go:398,426,504`) + bootstrap wiring. Fallback if wiring turns ugly: leave duplication + tests (explicitly allowed)
- [ ] NEW `dto/import_issue.go`: `ImportRowIssue{Code,Message,Row,Reference}` (model: `OnePayFeeReportIssue`); islands adapt to existing external JSON — payloads unchanged
- [ ] Do NOT restructure `admin_bulk_transfer.go` (adjacent to recently-dirty worker; only upload guard in Phase 6)

## Implementation Steps
1. dto contract + provisioning package + wiring
2. Island-by-island commits: employee → flexpay → onepay/wallet_bulk
3. `go test ./...` + `make api-test` (flow_flexpay_import + settlement flows)

## Success Criteria
- [ ] getOrCreate trio exists once; flexpay reads sheet rows once
- [ ] External JSON byte-compatible (frontend untouched)

## Risk Assessment
Medium — bootstrap wiring for provisioning is the one structural change; fallback
documented. Rollback: per-island revert; provisioning extraction droppable independently.
