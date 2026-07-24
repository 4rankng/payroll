---
title: "Async BCC import completion"
date: "2026-07-24"
status: completed
---

# Async BCC import completion

## Summary

| Area | Result |
|---|---|
| Intake | `202 Accepted` with durable asset/job ID |
| Processing | Asynq worker with database lease, attempt fence, and recovery sweep |
| Consistency | Project/month serialization and atomic replacement + terminal state |
| Retry | Same key/payload returns the same job; changed payload returns `422` |
| UI | Queued/processing polling, close-safe waiting state, active history refresh |
| Deployment | Not performed |

## Verification

- Exact EVA workbook: `202` in 12.8 ms; completed with 83 source rows,
  900 created timesheets, and zero errors.
- Cuong local account (`cuongnv`, user 4) can see EVA KCN Vsip and received
  `202` from the same upload endpoint.
- Mobile browser: waiting state rendered, modal was close-safe, no horizontal
  overflow, and no console errors.
- Focused Go tests and race detector passed.
- Go vet, frontend lint/typecheck, frontend production build, and
  `git diff --check` passed.
- Full integration harness: 254 passed, 4 unrelated baseline failures, 15
  skipped; BCC fixture selection remains data-dependent in that broad harness.

## Review

Final adversarial review found no remaining blocker. Production migration,
commit, and deployment require separate approval.
