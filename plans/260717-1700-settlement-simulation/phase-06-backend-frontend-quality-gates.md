---
phase: 6
title: "Backend & Frontend Quality Gates"
status: pending
priority: P2
effort: "S"
dependencies: [1, 2, 3, 4, 5]
---

# Phase 6: Backend & Frontend Quality Gates

## Overview

Final pass: run the full quality bar defined in `AGENTS.md` / `CLAUDE.md` and document the deliverables. No new logic — verification, regression sweep, and the post-implementation report the brief asks for (root-cause findings, files changed, API contract, validation rules, test results, edge cases that remain unverifiable).

## Requirements

- **Functional:** Every command in `AGENTS.md` "Testing Requirements" passes.
- **Non-functional:** Coverage maintained or improved; no lint regressions; PWA build clean.

## Architecture

N/A — verification phase.

## Related Code Files

- **Run-only refs:** `Makefile` (targets: `api-test`, `dev`, `deploy`), `backend/AGENTS.md`, `frontend/AGENTS.md`.

## Implementation Steps

1. **Backend unit + race + coverage:**
   ```bash
   cd backend && go test ./... -v -race -cover | tee /tmp/go-test.log
   ```
   Confirm new package coverage ≥ 80%. Confirm zero race detections.

2. **Integration suite:**
   ```bash
   make api-test
   ```
   Must include the new `SettlementSimulation` flow and the existing `SaoKê`, `ManualBulkTransfer`, `Transaction` flows unaffected.

3. **Frontend lint + type-check:**
   ```bash
   cd frontend && pnpm lint && pnpm type-check
   ```
   Zero errors. Zero new warnings.

4. **Frontend build:**
   ```bash
   cd frontend && pnpm build
   ```
   PWA bundle builds clean.

5. **Clock/timezone check:** verify the simulation uses `clock.Now()` in `Asia/Ho_Chi_Minh` for cycle resolution — grep for any `time.Now()` introduced in the new files. None should exist in domain logic.

6. **Backend cache invalidation check:** confirm the simulation does NOT publish any event that would trigger cache invalidation (it shouldn't — read-only). Grep `eventBus.Publish` in new files.

7. **Permission recheck:** confirm `POST /payrolls/simulate-settlement` is unreachable by non-admin tokens (covered by test, re-verified by curl with an employee token).

8. **Write the post-implementation report** (the brief's final deliverable). Append to this phase file under "Report" once tests are green:
   - **Root-cause findings** — the R1–R6 risks from `plan.md`, plus any new ones discovered during implementation.
   - **Files changed** — full list, grouped by create/modify.
   - **API contract** — final `simulate-settlement` + the `if_match_snapshot` extension (copy from Phase 3).
   - **Validation rules implemented** — the table from Phase 3.
   - **Test results** — paste the tail of `make api-test` and `go test -cover`.
   - **Edge cases not automatically verifiable** — at minimum: real-bank-result-upload drift, concurrent admin sessions running sim+export simultaneously, currency actually being non-VND (cannot happen by schema but worth stating), ledger entries created out-of-band by other features between sim and export.

## Success Criteria

- [ ] `go test ./... -v -race -cover` green; new package coverage ≥ 80%.
- [ ] `make api-test` green including the new `SettlementSimulation` flow.
- [ ] `pnpm lint && pnpm type-check && pnpm build` all green.
- [ ] No `time.Now()` in new domain logic (uses `clock.Now()`).
- [ ] No `eventBus.Publish` / cache-invalidation calls in new read-only code.
- [ ] Non-admin tokens rejected (curl-verified).
- [ ] Post-implementation report appended with all six required sections.

## Risk Assessment

**Risk: a pre-existing flaky test gets blamed on this change.**
Mitigation: run the suite on `main` first to establish the baseline green state. If a pre-existing flake exists, document it and don't gate this PR on it.

**Risk: coverage threshold not met because validators are simple.**
Mitigation: table-driven tests across many inputs easily clear 80% for pure functions. If genuinely under, add cases rather than lowering the bar.

**Risk: the report's "edge cases not verifiable" section is hand-wavy.**
Mitigation: be specific. Name the exact scenario, why it can't be tested in CI (e.g. requires real bank upload), and the manual verification step an admin should perform. This is the deliverable the user explicitly asked for in the brief.
