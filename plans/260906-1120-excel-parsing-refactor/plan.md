---
title: "excel-parsing-refactor"
description: "Behavior-preserving refactor of all server-side Excel parsing: one registry extension point, shared excelkit primitives, decomposed god processors, unified error contract, hardened uploads."
status: completed
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
| 2 | [Phase 1: excelkit primitives](./phase-02-phase-1-excelkit-primitives.md) | Completed |
| 3 | [Phase 2: registry-driven detection](./phase-03-phase-2-registry-driven-detection.md) | Completed |
| 4 | [Phase 3: dispatch collapse](./phase-04-phase-3-dispatch-collapse.md) | Completed |
| 5 | [Phase 4: god-processor decomposition](./phase-05-phase-4-god-processor-decomposition.md) | Completed |
| 6 | [Phase 5: satellite adoption](./phase-06-phase-5-satellite-adoption.md) | Completed |
| 7 | [Phase 6: upload guard](./phase-07-phase-6-upload-guard.md) | Completed |
| 8 | [Phase 7: cleanup and docs](./phase-08-phase-7-cleanup-and-docs.md) | Completed |

## Success Criteria

- [ ] All 8 phases land; each verified by `cd backend && go test ./...` + `make api-test`
- [ ] Adding a hypothetical new template requires only: parser file, registry line, test file, fixture
- [ ] Golden + routing characterization tests permanently green in CI-suite
- [ ] Contracts preserved: winner-format (`bcc_import_process.go:90`) → `:162/:319/:381`; rateless/label-keyed resolution; explicit-0 delete; STK auto-create + upsert; "BCC"-named date-row (T09); day-number-only payrate; `clock.Now()` everywhere
- [ ] External JSON payloads (error_detail, DTOs) unchanged; frontend untouched

## Deviations from original phase text (verified during execution)

1. **parseForMonth NOT unified** — the two implementations have divergent
   contracts (accepted formats + user-visible error text); both kept,
   cross-documented. "Single home" failed the identity test.
2. **Parameterized header-map builder dropped** — the 7 builders use 4
   different stop-column predicates and 2 header-normalization idioms;
   unifying matching semantics is exactly the frozen-behavior boundary.
   shared.go took only verified-identical helpers (isSummaryRow ×5 sites,
   headerContainsAny ×3, sheet-name rules, excelkit delegates).
3. **Provisioning trio fallback invoked** (allowed by plan) — getOrCreate
   variants genuinely diverged (bank validation, StartDate/PaymentSchedule
   parsing live only in the employee island); documented at both sites.
4. **includeFlexibleEmployees removal DEFERRED** — the flag is hashed into
   the idempotency fingerprint and persisted on TimesheetImportJob; removal
   changes stored-fingerprint comparison for in-flight retries. Follow-up.
5. **ResultStrategyFactory NOT deleted** — grep shows live callers
   (result_processor.go:55,349); it is vestigial in shape but wired and
   functional. Audit finding corrected.
6. **Weekly split = one atomic commit** (vs per-format commits) — the file
   split cannot land half-way; verification was still per-suite green.
7. **clock.Location() not needed** — clock.DefaultLocation already exported.
8. **"type:" consts cross-referenced, not unified** — a shared const would
   add a package edge for two strings; both sites now document the contract.
9. **dto.ImportRowIssue removed post-review** — it shipped with zero
   consumers (adversarial review blocker); re-introduce at first adoption.
10. **Upload guard sweep completed post-review** — the first wave guarded 6
    endpoints but claimed "all"; review found 5 more (flexible-employee
    list, advance-payment result, wallet reconciliation, template import,
    bulk-transfer result incl. `.xls` via allowXLS). All 11 now guarded.
11. **AliasTable is deliberately a superset** — wallet-bulk gained the
    unidecode retry and OnePay gained whole-cell/newline steps; strictly
    more permissive on pathological headers, canonical mappings unchanged
    (review disclosure, accepted).

## Non-goals (YAGNI)

No unified tabular framework · no Parse-type unification · no name-normalizer merge
(bank-recon keys) · no BCCImportService dependency diet · employee alias-headers /
bulktransfer Local-tz fix / flexpay structured errors = documented follow-ups.

<!-- slug: excel-parsing-refactor -->
