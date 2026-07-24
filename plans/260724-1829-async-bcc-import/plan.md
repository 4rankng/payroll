---
title: "Durable Asynchronous BCC Timesheet Import"
description: "Replace the timeout-prone synchronous BCC upload with a durable, idempotent background import and a resumable polling UI."
status: completed
priority: P1
branch: "main"
tags: [bugfix, timesheet, backend, frontend, database, api, critical]
blockedBy: []
blocks: []
created: "2026-07-24T10:29:05.221Z"
createdBy: "ck:plan"
source: skill
---

# Durable Asynchronous BCC Timesheet Import

## Overview

Production completes Cuong's EVA BCC import after the HTTP server's 15-second
write deadline, so the partner sees a system error while history later shows
success. The fix is an asynchronous accepted-job contract: persist the upload
and job before returning, process it through Asynq, expose durable status, and
let every admin/partner surface poll or resume the result.

The existing endpoint paths, project authorization, import history, downloads,
statistics, and completed/failed representations remain compatible. The
temporary 50/55-second timeout workaround is removed after the async contract
is covered.

## Acceptance Criteria

- POST `/api/v1/timesheets/partner-import` returns `202` with a durable job ID
  before spreadsheet parsing begins.
- Retrying the same file with the same idempotency key returns the same job;
  reusing a key for different content returns a validation error.
- At most one active import can replace a project/month at a time, including
  after worker crashes or retries.
- Existing timesheets remain intact if the replacement transaction fails.
- Pending jobs survive API/worker restarts and are re-enqueued safely.
- The shared upload modal shows queued/processing state, may be closed safely,
  and resumes through history/status polling.
- Partner ownership and EVA project authorization remain enforced.
- Backend race/unit/integration tests and frontend lint/type/build gates pass,
  with unrelated baseline failures documented rather than hidden.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Durable job model and API contract](./phase-01-durable-job-model-and-api-contract.md) | Completed |
| 2 | [Asynq orchestration and recovery](./phase-02-asynq-orchestration-and-recovery.md) | Completed |
| 3 | [Atomic BCC apply pipeline](./phase-03-atomic-bcc-apply-pipeline.md) | Completed |
| 4 | [Frontend polling and end-to-end verification](./phase-04-frontend-polling-and-end-to-end-verification.md) | Completed |

## Dependencies

No cross-plan dependency. The unfinished partner visual plan does not own the
shared BCC import contract or modal behavior.

## Rollback

The schema addition is additive. The preferred operational rollback is to roll
back application code while retaining the job table; the migration down file
is destructive and must not be run against production history without separate
approval. Production deployment and any cleanup of duplicate historical assets
require separate user approval.
