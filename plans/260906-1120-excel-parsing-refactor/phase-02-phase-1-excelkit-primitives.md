---
title: "Phase 1: excelkit primitives"
status: todo
priority: P1
effort: "0.5d"
dependencies: [phase-01]
---

# Phase 1: excelkit primitives

## Overview
New `internal/pkg/excelkit` package with generic Excel primitives (zero business
deps). Nothing imports it yet — additive only.

## Requirements
- [x] `open.go`: `OpenReader` with 50MiB unzip / 10MiB XML caps (port from `wallet_bulk/parser.go:19-29`)
- [x] `header.go`: `NormalizeHeader` verbatim from `normHeader` (`excel/bcc_parser.go:130`: trim+NFC+lower)
- [x] `alias.go`: `AliasTable.Resolve` verbatim from OnePay `resolveHeader` semantics (unidecode retry) — verify wallet_bulk `resolveHeader` variant identical first; if divergent, two distinctly named methods
- [x] `cell.go`: cell readers factoring `bccCell`/`dateRowNumericCell`; money parsers ONLY for proven-identical pairs (table tests), distinctly named otherwise
- [x] `date.go`: `ExcelDate(cell, loc)` — location injected, never `time.Local`
- [x] `row.go`: `IsEmptyRow`, `FindHeaderRow`, shared predicates
- [x] Additive `clock.Location()` accessor (HCM) — pkg currently lacks one
- [x] Unit tests per file

## Implementation Steps
1. Read source implementations (wallet_bulk caps, normHeader, onepay resolveHeader, bccCell, dateRowNumericCell)
2. Port verbatim + tests (table tests proving identity with originals)
3. `go test ./internal/pkg/... && go build ./...`

## Success Criteria
- [x] Package compiles standalone; zero importers outside its tests
- [x] Identity table tests green (kit fn ≡ source fn on shared inputs)

## Risk Assessment
Low — nothing adopts it. Rollback: delete package.
