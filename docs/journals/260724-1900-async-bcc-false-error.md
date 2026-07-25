---
title: "Async BCC false-error fix"
date: "2026-07-24 19:00"
severity: "High"
component: "bcc import upload flow"
status: "Resolved"
---

## Context

Partner `cuongnv` could upload a BCC file and see the history row marked success from the admin side, but his browser still showed `Lỗi hệ thống` after submit. That meant the upload path was lying: the backend kept working after the HTTP request had already been killed by timeout.

## What Happened

We confirmed the root cause instead of papering over it. The request was taking longer than the server timeout, so Gin/Nginx dropped the response while the import continued in the background and eventually committed. A 60s wait-and-spinner patch was rejected because it only hides the mismatch between request lifetime and job lifetime; it does not make the upload contract correct.

On July 25, we closed the transport gap that had been breaking the browser path. The async commit already added `Idempotency-Key`, but CORS did not allow that header, so `localhost:3000` blocked the POST before it could reach the API. The fix now allows the header and adds a regression test. Live QA with the supplied July workbook returned `202` and polled normally with no browser errors.

## The Brutal Truth

This was maddening because the system looked half-right and half-broken at the same time. Admin history said success, the partner saw failure, and the only thing worse than a slow import is a slow import that lies about its own state. We should not have left a long-running file import bound to a request/response path in the first place.

## Technical Details

- The real failure was `upstream prematurely closed connection while reading response header` after the server hit its timeout window.
- The long-term fix is `202 Accepted` plus a durable import job row, not a bigger timeout.
- Uploads now carry an idempotency key and a request fingerprint so the same file can safely replay, while a different payload under the same key is rejected.
- CORS now includes `Idempotency-Key`, which was the missing browser-side transport permission.
- Terminal import against the local July 15-21 data set failed safely because those records were already paid, so the guard created `0` rows and preserved the paid data.
- The worker path uses a lease/fencing model, active-scope locking, transaction-context propagation, atomic terminal state updates, and protected-row replacement so partial imports do not orphan old data.

## What We Tried

- A timeout extension and loading animation was considered and rejected.
- The backend was refactored to enqueue the import after commit, then recover pending jobs so a dropped request no longer decides the final outcome.
- Critical review fixes landed for transaction context handling, claim/release flow, and atomic replacement of timesheet rows.
- Browser QA on `localhost:3000` confirmed the POST now reaches the API, and the regression test covers the missing CORS header.

## Root Cause Analysis

The original design treated a multi-minute import like a normal HTTP form submit. That was the mistake. The request lifecycle, UI state, and database commit path were not aligned, so the browser got an error while the backend legitimately finished work later.

## Lessons Learned

- If the work can outlive the request, the request must return a job handle immediately.
- A longer timeout is not a fix when the contract itself is wrong.
- Idempotency, leasing, and atomic replacement are not extras here; they are the minimum safety bar.

## Verification

- Local API calls now return quickly with `202` and a polling path for job state.
- Browser QA with the supplied July workbook returned `202`, then polled normally with no console or network errors.
- Replays with the same idempotency key return the same job; a different file with the same key is rejected.
- Focused Go tests and frontend checks passed locally, including `go test` on the touched packages, `go vet` on the changed backend areas, `pnpm lint`, `pnpm build`, and `git diff --check`.
- No production deploy happened yet.

## Next Steps

Keep the async BCC flow behind the remaining release steps, then deploy separately once the team is ready. The open work now is operational rollout, not more timeout tuning.
