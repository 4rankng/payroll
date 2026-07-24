---
phase: 3
title: "Atomic BCC apply pipeline"
status: completed
effort: "large"
---

# Phase 3: Atomic BCC apply pipeline

## Overview

Make project/month replacement transactional so the old approved/protected data
survives any parse, validation, database, or worker failure.

## Implementation Steps

1. Add regression tests that inject failures after stale-row selection,
   deletion, insertion, and job-result persistence.
2. Refactor BCC processing into parse/validate and apply stages. Parse outside
   the transaction; perform protected-row checks, stale deletion, replacement
   insertion, and terminal job update in one explicit transaction.
3. Add transaction-aware repository/service helpers instead of nesting the
   current `BulkCreateTimesheets` transaction.
4. Apply the same atomic boundary to legacy, weekly, and multi-position BCC
   formats; keep employee/assignment enrichment idempotent.
5. Publish cache invalidations, domain events, and audit records only after
   commit.

## Success Criteria

- [x] A replacement transaction rollback restores deleted rows and removes inserts.
- [x] Re-running the same job is a no-op after completion.
- [x] Approved/protected rows are never deleted.
- [x] All supported BCC formats preserve existing result counts and errors.
