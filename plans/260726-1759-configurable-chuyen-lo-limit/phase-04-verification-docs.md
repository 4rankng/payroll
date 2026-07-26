---
phase: 4
title: "Verification docs"
status: in-progress
effort: "medium"
---

# Phase 4: Verification docs

## Overview

Prove backend, settings, UI, and documentation behavior without claiming
unavailable authenticated evidence.

## Implementation Steps

1. Update `docs/qa/plan/07-bulk-transfer-payment.md` to describe the
   configurable strict threshold and 400M default.
2. Run `gofmt` and focused backend tests, then
   `cd backend && go test ./... -race`.
3. Run focused frontend tests, `pnpm lint`, `pnpm type-check`, and production
   build.
4. Start the required local services and run `make api-test`; classify
   environment or unrelated baseline failures without weakening checks.
5. Run `graphify update .`.
6. Inspect authenticated Admin Settings at 1280, 390, and 320 pixels, including
   dirty/reset/save, invalid, loading, error, and keyboard focus states. Verify
   no overflow, readable full money, and 44px targets; retain screenshots when
   the browser/auth environment permits.
7. Run mandatory tester/debugger and code-reviewer gates. Review acceptance,
   financial boundary behavior, migration safety, no-write-on-failure,
   responsive parity, and public contracts.
8. Synchronize all plan/phase statuses, update warranted docs, journal the
   result, and hand commit preparation to the required git workflow.

## Evidence

- Unit and integration command outputs.
- Authenticated desktop/mobile screenshots or an explicit authentication block.
- Final diff/status, graph update, reviewer and verification reports.

## Success Criteria

- [x] All required checks pass or skipped/baseline failures are explicitly
      classified with evidence.
- [x] QA docs and graph match the implementation.
- [x] Adversarial review finds no reachable financial/UI regression or
      unverified acceptance claim.
- [x] Plan and every phase file are synchronized to actual completion.

## Risks and rollback

- Do not use production payroll files or cross OTP/provider actions for
  verification.
- Do not commit, push, deploy, or alter unrelated user-owned work without the
  matching workflow authority.
