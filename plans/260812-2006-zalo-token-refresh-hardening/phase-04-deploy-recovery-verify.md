---
phase: 4
title: "Deploy + recovery + verify"
status: pending
priority: P1
effort: "2h"
dependencies: [1, 2, 3]
---

# Phase 4: Deploy + recovery + verify

## Overview
Ship the hardened refresh path to prod (amd64, OnePay-unrelated), re-paste one
fresh token pair (the stored `refresh_token` is already dead), and verify the
chain survives an expiry boundary under concurrency without admin intervention.

## Requirements
- Functional: prod ZNS sends succeed and keep succeeding across token expiry.
- Non-functional: no regressions; no unrelated frontend files in the deploy.

## Architecture
No code change. Build + deploy + operational verification.

## Related Code Files
- Verify only: `backend/internal/infra/zalo/*`, `zaloconnect/service.go`
- Operational: Zalo OA Console (fresh token pair), admin Settings UI.

## Implementation Steps
1. Confirm Phases 1-3 are committed and `go test ... -race` is green locally.
2. `make deploy` (builds amd64 — rule #2; SSH to `tingting.vip`).
3. Admin: open Zalo OA Console → copy fresh `access_token` + `refresh_token`
   → paste into `/admin/settings?tab=zalo` → Save. (One-time recovery.)
4. Admin "Kiểm tra kết nối" / test send → expect success.
5. Verify durability: trigger a ZNS send around an expiry boundary (or wait);
   watch logs for exactly one refresh call and a successful retry.
6. Regression: `make api-test`; frontend unaffected (do NOT deploy the dirty
   frontend chunk-reload files — leave them for their own effort).
7. Spot-check `LastError` is empty after a successful refresh.

## Success Criteria
- [ ] Prod ZNS test send succeeds after admin re-paste.
- [ ] A refresh at the expiry boundary produces one Zalo call, no 500.
- [ ] `make api-test` green; no new regressions.
- [ ] Only the three zalo backend files shipped (no frontend chunk-reload files).

## Risk Assessment
- **Forgetting the re-paste**: the fix can't recover an already-consumed
  `refresh_token`. If sends still fail right after deploy, the re-paste was
  skipped or the OA tokens were revoked — check logs for the real Zalo code.
- **Deploy-time restart overlap** is precisely the race Phase 2 fixes; the first
  send after deploy may still race once if a second process is briefly alive.
  The lock absorbs it; if it does not, revisit Phase 2 TTL/poll tuning.
