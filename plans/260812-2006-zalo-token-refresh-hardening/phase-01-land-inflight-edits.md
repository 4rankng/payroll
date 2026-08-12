---
phase: 1
title: "Land in-flight rotation edits"
status: pending
priority: P1
effort: "2h"
dependencies: []
---

# Phase 1: Land in-flight rotation edits

## Overview
The "diagnose" half of the fix already exists locally but uncommitted. Verify
it is correct, race-clean, and commit it. This alone replaces the misleading
prod message with the real Zalo error code.

## Requirements
- Functional: `refresh()` parses Zalo `error`/`message`; keeps the old
  `refresh_token` when Zalo doesn't rotate; logs the real error code.
- Non-functional: zalo unit tests pass under `-race`.

## Architecture
No structural change. Review-only of the existing diff in `provider.go`,
`service.go`, `zalo_test.go`, then commit.

## Related Code Files
- Modify (verify): `backend/internal/infra/zalo/provider.go`
- Modify (verify): `backend/internal/app/services/zaloconnect/service.go`
- Modify (verify): `backend/internal/infra/zalo/zalo_test.go`

## Implementation Steps
1. `git diff backend/internal/infra/zalo/provider.go
   backend/internal/app/services/zaloconnect/service.go` — review.
2. Confirm `oauthTokenResponse` has `Error int` + `Message string` and the
   empty-access-token branch logs `zalo_error`/`zalo_message`/`body`.
3. Confirm the old `missing access_token or refresh_token` string is gone from
   the tree: `rg "missing access_token or refresh_token" backend` → no hits.
4. Run tests:
   `cd backend && go test ./internal/infra/zalo/... -race` and
   `go test ./internal/app/services/zaloconnect/... -race`.
5. Stage **only the three zalo files** (NOT the frontend chunk-reload files).
6. Commit: `fix(zalo): surface refresh error code and keep valid refresh_token`.

## Success Criteria
- [ ] Old "missing access_token or refresh_token" string absent from source.
- [ ] `refresh()` logs the real Zalo `error` code on failure.
- [ ] zalo + zaloconnect tests pass under `-race`.
- [ ] Commit contains only the three zalo files.

## Risk Assessment
- **Shared `service.go`** with plan `260806-1719-zalo-domain-errors`. Commit
  this first; the domain-error plan layers on top. Low risk if staged narrowly.
- If tests reveal the in-flight edits are incomplete, do NOT extend here —
  carry the gap into Phase 2/3 where the lock/atomic work lives.
