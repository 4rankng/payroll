---
phase: 3
title: "Frontend UI"
status: in-progress
effort: "medium"
---

# Phase 3: Frontend UI

## Overview

Add one accessible full-VND setting control to both Admin Settings render paths
using the existing shared hook and `SettingCard`. The field is the focal
guardrail; surrounding settings remain visually quiet and flat.

## Implementation Steps

1. Make `useMultipleSettings` retrieve exact keys (by-key requests) so settings
   pagination cannot hide the new row.
2. Extend `useSettingsForm` with raw integer/original state and a save handler
   for `bulk_transfer_workbook_limit_vnd`; update the original only on success
   and expose missing/error state.
3. Extend `SettingCard` rather than creating a parallel control:
   - real `<Label htmlFor>` and stable description/error IDs;
   - `currency-vnd` display mode with digits-only canonical state, full
     `vi-VN` grouping, `inputMode="numeric"`, right-aligned tabular digits;
   - strict integer/range validation, `aria-invalid`, live error/status;
   - disabled input/actions and `Đang lưu…` state while saving;
   - existing 44px reset/save controls and flat semantic tokens.
4. Add the field first in the desktop `Giới hạn thanh toán` section and in the
   mobile `Trả lương` list. Copy:
   - title: `Giới hạn tổng tiền mỗi file Chuyển lô`
   - description: `Hệ thống tự tách file để tổng tiền mỗi file luôn nhỏ hơn giới hạn này.`
5. Update desktop skeleton/grid for three cards and preserve one-column mobile
   layout. Add focused hook/component/page-parity tests.

## Files

- `frontend/src/hooks/api/useSettings.ts`
- `frontend/src/hooks/settings/useSettingsForm.ts`
- relevant new/existing hook tests
- `frontend/src/components/settings/SettingCard.tsx`
- `frontend/src/components/settings/SettingCard.test.tsx`
- `frontend/src/pages/admin/SettingsPage/index.tsx`
- `frontend/src/pages/mobile/admin/SettingsPage/index.tsx`
- focused desktop/mobile render tests where local harness permits

## Success Criteria

- [x] Loaded `400000000` displays as `400.000.000 đ` and saves as
      canonical string `400000000`.
- [x] Blank, decimals, negatives, `< 2`, and int64 overflow cannot be saved.
- [x] Missing/load/save errors remain visible and recoverable; success is not
      optimistic.
- [x] Keyboard labels/descriptions/errors are associated and controls are 44px.
- [ ] Desktop 1280 and mobile 390/320 have full parity and no overflow.

## Risks and rollback

- Currency behavior is opt-in so percentage/text `SettingCard` consumers do not
  change.
- If exact-key loading fails, render an explicit unavailable state; never fall
  back to a blank editable value.
