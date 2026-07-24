# Durable BCC Import Design Decision

## Problem

The synchronous partner-import request has a five-minute application context
but the HTTP server stops writing after 15 seconds. A 900-row EVA import needs
roughly 17–22 seconds, so the browser reports failure even though database work
continues and history later reports success. Increasing timeouts only moves the
failure threshold and leaves retries, crashes, and partial replacement unsafe.

## Chosen Direction

Use a durable database-backed job with an immediate `202 Accepted` response,
process it through the existing Asynq worker system, and poll status from the
shared frontend flow.

- The existing Asset remains the immutable file/history record and public ID.
- A dedicated job row owns lifecycle, idempotency, lease/fencing, results, and
  project/month serialization.
- The database is correctness authority; Asynq provides delivery and retries.
- A reconciler closes the database-to-queue crash window.
- Parsing occurs outside a transaction; replacement and terminal success commit
  atomically.
- Frontend waiting state is resumable and does not depend on one HTTP request.

## Alternatives Rejected

- **Raise client/server timeout to 60 seconds:** quickest mitigation, but larger
  files can still exceed it and lost responses still create duplicate retries.
- **Detach a goroutine from the request:** avoids the immediate timeout but loses
  work on process restart and has no durable status or retry contract.
- **Redis-only progress:** restart-sensitive and insufficient as the source of
  truth for a destructive project/month replacement.

## Human Checkpoint

The user approved implementing and locally testing the complete long-term
solution. Production deployment and deletion/cleanup of existing duplicate
history remain separate approval gates.
