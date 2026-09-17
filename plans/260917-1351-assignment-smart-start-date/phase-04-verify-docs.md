---
phase: 4
title: "End-to-end verification + docs"
status: pending
priority: P2
effort: "2h"
dependencies: [1, 2, 3]
---

# Phase 4: End-to-end verification + docs

## Overview

Prove the incident cannot recur on any import format, then document the new
`start_date` default contract.

## Requirements

- Functional:
  - Incident replay E2E on local dev: fresh employee → assignment today → upload weekly-BCC file covering earlier dates → import completes AND assignment start equals file's earliest date.
  - Same replay for legacy BCC format (cheap sanity; formats share the pipeline).
  - Regression: explicit start dates, protected (approved/paid) rows, and end-date invariants from plan `260910-1435-assignment-end-date-extend` unchanged.
- Docs (per documentation-management rules — only user-visible contract changed):
  - `docs/api.md`: note under project-employees that omitted `start_date` is computed (last timesheet + 1 day, else 1st of current month).
  - `docs/lessons/`: short lesson entry for the 2026-09-17 incident linking the report, if the lessons dir convention fits (check existing lesson format first).

## Related Code Files

- Modify: `docs/api.md` (one paragraph)
- Create: `docs/lessons/` entry (follow existing naming)
- No schema migration: no `migrations/` change (pure service logic)

## Implementation Steps

1. `make dev` with seeded data; run the two replay scenarios via UI + `make api-test`.
2. `make api-test` full suite (30 flow files).
3. `cd backend && go test ./... -race -cover` final pass.
4. Update `docs/api.md`; write lesson entry.
5. Commit phases separately per repo convention: `feat(assignment): ...`, `fix(bcc-import): ...`, `feat(frontend): ...` — no AI references (repo rule 6).

## Success Criteria

- [x] Both replay scenarios complete with expected assignment backdate
- [x] `make api-test` green; backend race tests green; frontend lint/type-check green
- [x] `docs/api.md` updated; lesson recorded
- [x] Plan phases marked completed via `ak plan check`

## Risk Assessment

- **api-test env drift** (flows depend on seeded project/payrate): if a flow fails for unrelated seed reasons, run the weekly-BCC flow file alone and record the unrelated failures in the phase report rather than blocking.
- None structural: no contract break, no schema change.
