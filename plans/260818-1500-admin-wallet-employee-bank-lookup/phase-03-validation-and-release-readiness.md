---
phase: 3
title: Validation and release readiness
status: completed
priority: P1
dependencies:
  - 1
  - 2
effort: medium
---

# Phase 3: Validation and release readiness

## Overview

Prove endpoint semantics, responsive parity, regression safety, and the absence of unintended persistence or sensitive logging. This phase does not authorize deployment.

## Validation Matrix

- Backend focused: handler/service/provider classification tests with race detection.
- Frontend focused: new dialog tests plus desktop/mobile wallet action-presence tests.
- Static: `cd frontend && pnpm lint && pnpm type-check`.
- Backend: narrow package tests, then `cd backend && make lint` and relevant broader Go tests.
- Integration: `make api-test` against the running local backend; classify environment failures separately.
- Browser: authenticated Admin at 1280px, 390px, and 320px; verify search, successful/failed result, retry, close/focus restoration, no overflow, console, and failed network requests.
- Authorization: non-Admin request cannot use the endpoint.
- Security: inspect captured logs/errors for full account-number leakage.
- Knowledge graphs: `graphify update .`; after a code commit, run the repository-required incremental `/understand` update.

## Implementation Steps

1. Run the narrowest tests first and repair only in-scope regressions.
2. Run lint/type/build and backend quality gates affected by the public API change.
3. Run integration and authenticated multi-viewport QA when the local environment supports them.
4. Perform production-readiness review for authorization, provider classification, privacy, and public-contract stability.
5. Record exact pass/fail/skipped evidence; do not equate local green with deployment.

## Success Criteria

- [x] Every acceptance criterion has named evidence.
- [x] No existing manual-disbursement or employee-selection contract regresses.
- [x] No full bank account appears in logs or error telemetry introduced by this change.
- [x] Desktop/mobile Admin parity passes source and component checks; rendered production widths remain a post-deploy verification.
- [x] Skipped environment-dependent checks are explicit.

## Rollback

Remove the new route, handler dependency/method, frontend contract/hook/dialog/buttons, and focused tests. No database or persisted-state rollback is required.
