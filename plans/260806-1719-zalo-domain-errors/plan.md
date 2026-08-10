# Fix Zalo 500 Errors - Convert to Domain Errors

## Status
In Progress

## Overview
Convert Zalo plain errors (`ErrNotConfigured`, `ErrRefreshFailed`) to domain errors so they return proper HTTP status codes instead of 500.

## Root Cause
- Zalo errors are plain `error` types in `internal/infra/zalo/errors.go`
- `response.HandleDomainError` treats non-domain errors as unexpected → 500

## Solution
Convert Zalo errors to domain errors at the **service layer** (`zaloconnect.Service`) to keep the infra package pure and follow DDD.

## Files to Modify

1. **`internal/app/services/zaloconnect/service.go`**
   - Convert `ErrNotConfigured` → `domain.NewValidationError`
   - Convert `ErrRefreshFailed` → `domain.NewInternalError` (wrapper)

2. **Tests**
   - Add test for proper error conversion

## Implementation Steps

### Phase 1: Add Error Conversion
- [ ] Import `api-server/internal/domain` in `zaloconnect/service.go`
- [ ] Wrap `ErrNotConfigured` from `RefreshNow` → `domain.NewValidationError`
- [ ] Wrap `ErrRefreshFailed` from `RefreshNow` → `domain.NewInternalError`
- [ ] Same for `TestSend`

### Phase 2: Update Error Messages
- Use user-friendly Vietnamese messages:
  - Not configured: "Chưa cấu hình Zalo — vui lòng nhập App ID và tokens từ Zalo OA Console"
  - Refresh failed: "Token Zalo đã hết hạn hoặc không hợp lệ. Admin vui lòng làm mới tokens từ Zalo OA Console"

### Phase 3: Test
- [ ] Unit test for error conversion
- [ ] Integration test for refresh endpoint
- [ ] Verify 400/422 instead of 500

## Acceptance Criteria
1. `/admin/zalo/refresh` returns 400 (validation) when not configured
2. `/admin/zalo/refresh` returns 500 (internal) with clear message when Zalo API fails
3. All existing tests pass
4. No breaking changes to public contracts
