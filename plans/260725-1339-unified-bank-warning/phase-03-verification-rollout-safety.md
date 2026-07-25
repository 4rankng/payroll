---
phase: 3
title: Verification + rollout safety
status: completed
effort: medium
---

# Phase 3: Verification + rollout safety

## Overview

Prove the unified category, BCC behavior, manual-write safety, UI refresh, and
contract compatibility across focused and shared quality gates.

## Implementation Steps

1. Run focused employee bank-validation/query tests and BCC service tests.
2. Run focused frontend Vitest suites for warning UI and BCC polling/result
   states.
3. Run backend package tests, frontend lint, type-check, and production build.
4. Run `make api-test` and explicitly separate new regressions from known
   baseline failures; do not weaken or skip tests.
5. Run mandatory tester and debugger agents, then the code-reviewer with
   acceptance, blast-radius, and public-contract checks.
6. Validate manual invalid create/update remains atomic and no invalid manual
   values reach persistence.
7. Run `git diff --check`, inspect the final dirty-worktree boundary, and update
   the graph index.
8. Perform authenticated local browser QA on desktop and mobile partner
   timesheet when the local services are available; otherwise report the
   precise environment limitation.
9. Sync all plan checkboxes/statuses, update docs only if durable user-facing or
   architectural behavior warrants it, and write the required journal.

## Success Criteria

- [x] Focused backend and frontend tests pass.
- [x] Lint, type-check, and production build introduce no new failures.
- [x] Shared/integration gates pass or unrelated baselines are evidenced.
- [ ] Reviewer finds no regression in manual writes, payment eligibility,
      partner scoping, or BCC atomicity.
- [x] Public API, database schema, and environment contracts remain unchanged.
- [x] Final worktree preserves all pre-existing unrelated edits.
