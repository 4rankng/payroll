---
phase: 1
title: "Durable job model and API contract"
status: completed
effort: "large"
---

# Phase 1: Durable job model and API contract

## Overview

Define the durable import state machine and change intake from a long-running
request to an accepted job without breaking history/detail/download clients.

## Implementation Steps

1. Add failing domain/repository/handler tests for job lifecycle,
   idempotency replay, mismatched fingerprints, active project/month
   serialization, partner ownership, and `202 Accepted`.
2. Add an additive migration for `timesheet_import_jobs`, keyed by the existing
   asset ID and carrying requester, project/month, fingerprint, idempotency key,
   status, lease/version, counts, error detail, and lifecycle timestamps.
3. Add the domain model and repository contract/implementation with
   compare-and-set transitions and terminal-state invariants.
4. Split upload intake from spreadsheet processing: validate access and file,
   hash/store the immutable upload, persist asset + pending job, then return the
   accepted representation immediately.
5. Extend partner import response/status types with
   `pending|processing|completed|failed` while preserving existing terminal
   fields and legacy asset history.

## Success Criteria

- [x] Contract and repository tests cover the implemented boundaries.
- [x] POST returns `202` quickly with a queryable durable ID.
- [x] Same idempotency key and fingerprint is replay-safe.
- [x] Different content under the same key is rejected.
- [x] Partner cannot query another uploader's job.
