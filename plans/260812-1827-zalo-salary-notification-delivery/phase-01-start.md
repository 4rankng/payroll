---
title: "Phase 1: Durable delivery and Settings"
status: completed
---

# Phase 1: Start

## Overview

Finish the flexible-pay ZNS path with a persisted outbox-like delivery record, bounded Asynq retries, and recovery without changing BCC semantics or the public generic Zalo test API.

## Requirements

- [x] Wire delivery to `IsFlexPayZNSEnabled`, not the password-reset toggle.
- [x] Persist one notification per asset/project/employee/template before marking the import complete; use a claimed, low-concurrency Asynq job and recovery sweep.
- [x] Build the recipient list only after reading the project assignment; exclude `CheckInEnabled` and any non-flexible assignment.
- [x] Add `619686` parameter limits and a salary-template test preset using the existing protected test-send endpoint.
- [x] Update the shared Admin Settings UI and focused tests.

## Implementation Steps

1. Add regression tests for the independent toggle, recipient eligibility, and exact template data.
2. Correct service wiring and recipient collection in the flexible payroll import worker/service.
3. Persist and deliver each recipient through a claimed retryable worker; record sent, failed, and suppressed outcomes.
4. Have the shared Settings action pass the salary template and its approved sample parameters.
5. Run focused tests, frontend lint/type-check/build, relevant backend tests, and `make api-test` when the local API is available.

## Todo

- [x] Backend delivery tests
- [x] Shared Settings interaction tests
- [x] Regression checks

## Success Criteria

The migration creates a durable ZNS delivery ledger. Existing generic `POST /admin/zalo/test` keeps its OTP default for callers that omit a template; the Settings salary test explicitly supplies template `619686`. Zalo does not expose a demonstrated provider-side idempotency lookup, so an ambiguous external response is tracked and retried as durable at-least-once delivery rather than claimed as exactly once.
