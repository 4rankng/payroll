---
phase: 1
title: "Backend setting"
status: completed
effort: "medium"
---

# Phase 1: Backend setting

## Overview

Establish the persisted whole-VND setting and the typed backend contract used
by exports. Keep the read fresh by avoiding the unwired in-process typed cache
for this key; the existing settings service already invalidates Redis detail
and list caches after successful writes.

## Implementation Steps

1. Add `096_seed_bulk_transfer_workbook_limit.up.sql` using idempotent
   `INSERT ... ON DUPLICATE KEY UPDATE key = key`, value `400000000`, and type
   `number`. Add a narrow down migration only if the runner supports paired
   downs; never overwrite an existing Admin value on up.
2. Add the same `FirstOrCreate` default to `backend/internal/seed/settings.go`.
3. In `settings_config.go`, define the key/default/minimum, parse the value with
   `strconv.ParseInt(..., 10, 64)`, and expose
   `GetBulkTransferWorkbookLimit(ctx) int64`. Missing/corrupt/out-of-range data
   logs a warning and returns `400_000_000`.
4. In `settings_service.go`, validate this known key after applying updates:
   number type, non-null canonical integer string, range `[2, math.MaxInt64]`.
   Return a Vietnamese domain validation error before persistence.
5. Add focused unit tests for valid, minimum/maximum, missing, corrupt, decimal,
   negative, overflow, and fallback behavior.

## Files

- `backend/migrations/096_seed_bulk_transfer_workbook_limit.up.sql`
- `backend/migrations/096_seed_bulk_transfer_workbook_limit.down.sql` if paired
- `backend/internal/seed/settings.go`
- `backend/internal/app/services/config/settings_config.go`
- `backend/internal/app/services/config/settings_config_test.go`
- `backend/internal/app/services/config/settings_service.go`
- `backend/internal/app/services/config/settings_service_test.go`

## Success Criteria

- [x] Default row is idempotent and preserves prior edits.
- [x] Whole-VND validation is authoritative on the backend.
- [x] A successful update is visible to the next getter call without restart.
- [x] Config package tests pass with `go test ./internal/app/services/config`.

## Risks and rollback

- Generic number validation is intentionally not broadened; only the new
  money-moving key gets strict rules.
- Roll back code and delete only the new key. Do not mutate other settings.
