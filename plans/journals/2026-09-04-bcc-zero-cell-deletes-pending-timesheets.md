---
title: BCC zero-cell deletes pending timesheets
date: 2026-09-04
summary: "All five BCC parsers dropped explicit-0 cells at parse time, so deletion intent never reached a pipeline that already understood it (planBCCReplacement stale-deletes requested pending rows; BulkCreateTimesheets treats 0 as delete/skip). Fixed parsers to emit zero-hour entries, decoupled zeros from payrate probing, guarded draft filters with positive-hours. Verified: unit suite green, committed BUMHAN fixture + 8-case live matrix with DB assertions, code review findings all fixed, api-test 273/273 executed 0 failed. Shipped b0d78947, deployed prod 2026-09-04 23:10 +07, healthy."
---

# BCC zero-cell deletes pending timesheets

All five BCC parsers dropped explicit-0 cells at parse time, so deletion intent never reached a pipeline that already understood it (planBCCReplacement stale-deletes requested pending rows; BulkCreateTimesheets treats 0 as delete/skip). Fixed parsers to emit zero-hour entries, decoupled zeros from payrate probing, guarded draft filters with positive-hours. Verified: unit suite green, committed BUMHAN fixture + 8-case live matrix with DB assertions, code review findings all fixed, api-test 273/273 executed 0 failed. Shipped b0d78947, deployed prod 2026-09-04 23:10 +07, healthy.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
