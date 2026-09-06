---
phase: 2
title: "Integration flow"
status: pending
priority: P1
effort: "3h"
dependencies: [1]
---

# Phase 2: Integration flow

## Overview
One flow file proving the full loop against the live local backend, wired into the
shared runner.

## Requirements
- Functional: admin creates targeted campaign → targeted employee GET returns it →
  untargeted employee gets null → click records a row → admin list shows the count →
  campaign past `ends_at` disappears from the employee response.
- Non-functional: flow registered in `backend/tests/integration/main.go` (beside
  `runEmployeeSelfServiceTests`); uses the shared client/reporter/config helpers.

## Related Code Files
- Create: `backend/tests/integration/flow_ad_banner.go`
- Modify: `backend/tests/integration/main.go`

## Implementation Steps
1. Apply migration locally: `go run cmd/migrate/main.go up` (chain at 105 → 106).
2. Write `runAdBannerTests(client, data, reporter, cfg)` following
   `flow_employee_self_service.go` (employee client via `client.WithToken(...)`
   hitting `/api/v1/me*`).
3. Register the call in main.go's flow list.
4. Run `make api-test` from `backend/`.

## Success Criteria
- [ ] All 30+ existing flows still pass (no regression)
- [ ] New flow passes end-to-end, which also proves the Casbin POST line

## Risk Assessment
If the runner rejects the new flow (helper drift), align with the newest flow file,
not the oldest — the suite's shared helpers evolve. Live-backend dependency: if the
local server is stale, restart via `make dev` before concluding anything failed.
