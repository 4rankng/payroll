---
title: "excel-parsing-refactor"
description: "Behavior-preserving refactor of all server-side Excel parsing: one registry extension point, shared excelkit primitives, decomposed god processors, unified error contract, hardened uploads."
status: in-progress
priority: P1
effort: "8 phases"
tags: [refactor, excel, bcc-import, backend]
created: 2026-09-06
---

# excel-parsing-refactor

## Overview

Excel is parsed at ~10 places in the backend (BCC timesheet family ×5 formats + STK,
OnePay fee, settlement sao-kê, employee, FlexPay, MBank results, eMB wallet bulk).
A new partner template today touches 4–6 files across 3 stacked routing layers.
This refactor makes "add a template" = 1 parser file + 1 registry line + 1 test +
fixture, with a real-fixture characterization net locking every production quirk.

Source plan (approved 2026-09-06): `~/.claude/plans/see-how-we-can-toasty-turtle.md`.
Audit evidence: `plans/reports/excel-audit-260906-1025.md`.
Mode: behavior-preserving EXCEPT Phase 6 (upload guard — user-approved behavior
change). No DB migrations. No frontend contract changes. Work on `main`.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | New template = 1 parser + 1 registry line + 1 test + fixture | P1 |
| 2 | Zero behavior change in parsing/routing outcomes (except Phase 6 rejects) | P1 |
| 3 | Real-file golden corpus covering all 5 BCC formats + routing table | P1 |
| 4 | Shared primitives (excelkit) adopted by all parsing islands | P2 |
| 5 | God processors ≤ ~250 LOC format-specific each | P2 |
| 6 | One internal row-issue error contract (external JSON unchanged) | P2 |

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Phase 0: Characterization safety net](./phase-01-start.md) | In Progress |
| 2 | [Phase 1: excelkit primitives](./phase-02-phase-1-excelkit-primitives.md) | Pending |
| 3 | [Phase 2: registry-driven detection](./phase-03-phase-2-registry-driven-detection.md) | Pending |
| 4 | [Phase 3: dispatch collapse](./phase-04-phase-3-dispatch-collapse.md) | Pending |
| 5 | [Phase 4: god-processor decomposition](./phase-05-phase-4-god-processor-decomposition.md) | Pending |
| 6 | [Phase 5: satellite adoption](./phase-06-phase-5-satellite-adoption.md) | Pending |
| 7 | [Phase 6: upload guard](./phase-07-phase-6-upload-guard.md) | Pending |
| 8 | [Phase 7: cleanup and docs](./phase-08-phase-7-cleanup-and-docs.md) | Pending |

## Success Criteria

- [ ] All 8 phases land; each verified by `cd backend && go test ./...` + `make api-test`
- [ ] Adding a hypothetical new template requires only: parser file, registry line, test file, fixture
- [ ] Golden + routing characterization tests permanently green in CI-suite
- [ ] Contracts preserved: winner-format (`bcc_import_process.go:90`) → `:162/:319/:381`; rateless/label-keyed resolution; explicit-0 delete; STK auto-create + upsert; "BCC"-named date-row (T09); day-number-only payrate; `clock.Now()` everywhere
- [ ] External JSON payloads (error_detail, DTOs) unchanged; frontend untouched

## Non-goals (YAGNI)

No unified tabular framework · no Parse-type unification · no name-normalizer merge
(bank-recon keys) · no BCCImportService dependency diet · employee alias-headers /
bulktransfer Local-tz fix / flexpay structured errors = documented follow-ups.

<!-- slug: excel-parsing-refactor -->
