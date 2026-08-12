---
phase: 3
title: "Atomic persist + failure self-heal"
status: pending
priority: P2
effort: "3h"
dependencies: [2]
---

# Phase 3: Atomic persist + failure self-heal

## Overview
Close the lost-update window in credential persistence, and make a failed
refresh surface a real reason (`LastError` + Zalo code) instead of looping
silently on a dead `refresh_token`.

## Requirements
- Functional: `mutateCredentials` is an atomic read-modify-write (no lost
  update); a failed refresh writes `LastError`; admin UI shows it.
- Non-functional: cache invalidation (if any) happens AFTER commit (ADR-007).

## Architecture
- `mutateCredentials` (service.go:407) currently does `loadCredentials` then
  `writeCredentials` — non-transactional. Wrap in a GORM transaction (or a
  conditional atomic upsert keyed on the row) so two concurrent writers can't
  clobber each other.
- On `refresh()` failure, call a new `MarkRefreshFailed(ctx, zaloCode, msg)`
  that writes `LastError = "Zalo refresh failed (code <n>): <msg> — dán lại
  token từ Zalo OA Console"` (Vietnamese, per rule #8).
- Retain the `refresh_token` by default (a transient Zalo error must not wipe
  it). If Phase 1's logged code is a definitive "refresh_token invalid/expired"
  code, clear it so the next attempt surfaces `ErrNotConfigured` instead of
  looping. Decide from evidence; start retain+flag.

## Related Code Files
- Modify: `backend/internal/app/services/zaloconnect/service.go`
  (`mutateCredentials` transaction; add `MarkRefreshFailed`)
- Modify: `backend/internal/infra/zalo/provider.go` (`refresh()` failure branch
  calls `creds.MarkRefreshFailed` before returning)
- Modify: admin Zalo settings UI/handler to surface `LastError` (Vietnamese) —
  locate via `zalo_handler.go` + frontend Settings component
- Modify: tests for atomic write + `LastError` persistence

## Implementation Steps
1. Make `mutateCredentials` atomic (GORM `Transaction` around load+mutate+write,
   or `UPDATE ... WHERE updated_at = ?` optimistic lock).
2. Add `MarkRefreshFailed(ctx, code, msg)` writing `LastError`; successful
   refresh already clears `LastError` (verify).
3. In `refresh()` empty-access-token branch: before returning, call
   `MarkRefreshFailed(tok.Error, tok.Message)`; log consistently.
4. Surface `LastError` in the admin "Cấu hình kết nối" UI (banner/toast) and/or
   the `GET /admin/zalo/credentials` response.
5. Tests: two concurrent `mutateCredentials` writers → no lost update; a forced
   refresh failure → `LastError` persisted and cleared on next success.

## Success Criteria
- [ ] Concurrent credential writes do not lose updates (test proves it).
- [ ] Failed refresh persists `LastError` with the real Zalo code; success clears it.
- [ ] Admin UI shows `LastError` (Vietnamese) when set.
- [ ] No silent infinite retry loop on a dead token.
- [ ] Any cache invalidation occurs after the DB commit (ADR-007).

## Risk Assessment
- **Optimistic-lock vs transaction**: prefer a short transaction (simplest,
  matches `backend/CLAUDE.md` pattern). Avoid adding a schema version column
  unless needed (rule: prefer no migration).
- **Clearing vs retaining a dead refresh_token** depends on the real Zalo code
  (Open Question 2). Wrong choice either loops (retain on a hard-invalid code)
  or forces a needless re-paste (clear on a transient code). Default
  retain+flag; revisit after Phase 1 logs.
