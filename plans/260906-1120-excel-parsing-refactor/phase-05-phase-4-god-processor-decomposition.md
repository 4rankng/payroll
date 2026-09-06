---
title: "Phase 4: god-processor decomposition"
status: todo
priority: P1
effort: "1d"
dependencies: [phase-04]
---

# Phase 4: god-processor decomposition

## Overview
Split the three ~500–680-line `process*Upload` functions. Smallest first: multiPosition
(482) → weeklyPayment (642) → weeklyBCC (684). One format per commit. Exported method
signatures unchanged.

## Requirements
- [ ] `bcc_import_multi_position.go`: ≤ ~250 LOC format-specific; shared boilerplate (fail/stats, `prepareImportContext`, employee resolution, replacement planning) reused from Phase 3 helpers
- [ ] `bcc_import_weekly.go` (1,324 LOC, 2 funcs) splits into `bcc_import_weekly_bcc.go` + `bcc_import_weekly_payment.go`
- [ ] Each processor: parse → validate/provision → persist phases as named functions
- [ ] After EACH format commit: `go test ./...` + `make api-test` (weekly flows especially)

## Implementation Steps
1. multiPosition: extract blocks → verify → commit
2. weeklyPayment: extract + split file → verify → commit
3. weeklyBCC: extract + split file → verify → commit

## Success Criteria
- [ ] No function > ~250 LOC in the BCC import pipeline
- [ ] All suites green after each commit

## Risk Assessment
High (LOC volume) but per-commit scope = one format; goldens + api-test guard.
Rollback: revert that format's commit only (formats independent).
