---
title: "Phase 2: registry-driven detection"
status: todo
priority: P1
effort: "1d"
dependencies: [phase-01, phase-02]
---

# Phase 2: registry-driven detection

## Overview
Kill in-package duplication; make one ordered registry the single routing truth for
the excel package. `DetectFormat` becomes a loop over it (exported signature
unchanged). One parser migration per commit, golden-green after each.

## Requirements
- [x] NEW `excel/bcc_formats.go`: ordered 5-entry registry `[legacy, weeklyBCC, weeklyPayment, multiPosition, dateRow]`; each entry `Name/Format/Match(f)` embedding today's fingerprint body verbatim (sheet-name rules, hidden-sheet skip, `"STK "` trim, date-row gating `format_detector.go:104`)
- [x] `format_detector.go` `DetectFormat` = loop over registry producing identical `FormatDetectionResult`
- [x] `bcc_strategy.go` Match methods delegate to the same fingerprint funcs (one fingerprint source)
- [x] NEW `excel/shared.go`: parameterized header-map builder replacing ~7 per-parser builders; shared stop-row (`"Tổng cộng"`), positive-hours predicates. Any parser whose golden can't be reproduced keeps its local builder (allowed)
- [x] Single `ParseForMonth` home (delete `services/bcc_import_helpers.go:38` copy, switch call sites)
- [x] Delete dead `buildWeekdayToDayMap` + its test
- [x] `cmd/bccinspect/main.go` routed via registry + `ParseBCCData` (no longer lies on date-row files)
- [x] Do NOT extend Accept-fall-through beyond legacy→dateRow (misdetected multi-position must keep hard-failing)

## Implementation Steps
1. Create registry with fingerprints moved (not rewritten)
2. Rewire DetectFormat + strategy Match to it; run routing goldens
3. Migrate parsers to shared.go builder one per commit (legacy → dateRow → weeklyBCC → weeklyPayment → multiPosition)
4. ParseForMonth dedup; dead code; bccinspect
5. `go test ./...` + `make api-test` after each commit

## Success Criteria
- [x] Routing characterization goldens byte-identical
- [x] All parser goldens green; `go test ./...` + `make api-test` green

## Risk Assessment
Medium — highest-judgment mechanical work (can change which files match). Fully
mitigated by Phase 0 goldens. Rollback: revert per-parser commits.
