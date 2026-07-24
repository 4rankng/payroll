---
phase: 4
title: "Frontend polling and end-to-end verification"
status: completed
effort: "medium"
---

# Phase 4: Frontend polling and end-to-end verification

## Overview

Replace the request-duration spinner with durable job tracking across the shared
admin/partner modal and history surfaces.

## Implementation Steps

1. Add hook/component tests for idempotency-key reuse, accepted response,
   queued/processing polling, close/reopen, completion invalidation, and
   actionable failure rendering.
2. Generate one idempotency key per selected file/project/month and retain it
   across network retries until the server acknowledges a job.
3. Poll job detail while non-terminal, stop on completion/failure, and
   invalidate timesheet/history queries only at terminal completion.
4. Show an indeterminate waiting animation and Vietnamese queued/processing
   copy. Permit closing the modal and explain that processing continues.
5. Make import history refresh while any active job exists and allow status
   detail to resume after navigation.
6. Remove the temporary 50-second server and 55-second client timeouts; restore
   the normal 15-second global write timeout because POST is now fast.
7. Run focused tests, Go race tests, frontend lint/type/build, API integration
   tests, `git diff --check`, and `graphify update .`.

## Success Criteria

- [x] Slow processing does not produce a browser error.
- [x] Modal can close while the durable job continues.
- [x] Admin and partner use the same terminal result.
- [x] Retry does not create duplicate history.
- [x] Required quality gates pass or pre-existing failures are explicitly
      isolated with evidence.
