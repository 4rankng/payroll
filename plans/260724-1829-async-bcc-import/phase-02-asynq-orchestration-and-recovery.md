---
phase: 2
title: "Asynq orchestration and recovery"
status: completed
effort: "large"
---

# Phase 2: Asynq orchestration and recovery

## Overview

Run persisted jobs through the existing Asynq infrastructure with at-least-once
delivery, bounded leases, and restart recovery.

## Implementation Steps

1. Add failing tests for enqueue, duplicate delivery, lease expiry, retryable
   infrastructure failures, terminal validation failures, and stale-pending
   recovery.
2. Define a BCC task payload containing only the durable job/asset ID. Register
   a stable task ID, queue, timeout, retry policy, handler, and bootstrap wiring.
3. Claim jobs with a database compare-and-set transition and fencing version;
   treat Redis/Asynq uniqueness as an optimization, not correctness authority.
4. Persist terminal results before acknowledging the task. Retry transient
   failures; record deterministic spreadsheet/business errors as failed.
5. Add a periodic reconciler that re-enqueues pending or expired-processing jobs
   so a crash between database commit and enqueue cannot strand an upload.
6. Move audit/event/cache work after successful commit and make it safe to run
   more than once.

## Success Criteria

- [x] Duplicate task delivery cannot apply an import twice.
- [x] Worker restart resumes pending/expired jobs.
- [x] Terminal failures retain actionable error details.
- [x] No correctness dependency remains on the old 30-second Redis lock.
