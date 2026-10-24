---
phase: 3
title: "Frontend — recovery dialog + hiện config ẩn"
status: pending
priority: P1
dependencies: [1]
---

# Phase 3: Frontend — recovery dialog + hiện config ẩn

## Overview
1) Khi PUT payrate trả `PAYRATE_CONFLICTING_CONFIG` → mở dialog liệt kê config xung đột,
nút "Xoá cấu hình xung đột và cập nhật" retry với `resolve_conflicts: true`.
2) Thêm section "Cấu hình khác" trong `PayrateConfigTab` hiển thị các config không còn
active/upcoming (như #88/#89 trong incident) với nút Xoá.

## Requirements
- Functional:
  - Dialog: list config (từ ngày → đến ngày), câu giải thích rõ timesheet sau ngày hiệu
    lực (nếu có) sẽ được **tính lại theo mức lương mới** nếu chưa duyệt; nếu có
    timesheet approved/paid liên quan → hiển thị thông báo bị chặn, KHÔNG có nút xoá.
  - Gọi update lại với `resolve_conflicts: true` sau khi partner xác nhận; thành công →
    invalidate queries (hook `useUpdatePayRate` đã làm) + toast thành công.
  - Section "Cấu hình khác": collapsible, mỗi row: badge "Đã kết thúc", from→to, nút Xoá
    (confirm dialog) → `useDeletePayRate`. Server vẫn chặn khi `IsUsedByTimesheets`.
- Non-functional:
  - Text 100% tiếng Việt; tap target ≥ 44px mobile; desktop + mobile parity cho cả
    Admin và Partner (component dùng chung, route phân `/admin` vs `/partner` đã có).
  - `.tsx` chỉ render, logic đặt hook/`.ts` theo file-responsibility rule.

## Architecture
- Bắt lỗi trong save handler của trang edit payrate (nơi hiện toast lỗi trong screenshot):
  đọc `error.response.data.code` (hoặc shape lỗi chuẩn hoá bởi `error-handler`).
- State dialog giữ ở hook mới `usePayrateConflictRecovery` (thư mục
  `frontend/src/components/payrates/hooks/`) — nhận mutation update, trả
  `{ conflictInfo, open, confirmResolve, close }`.

## Related Code Files
- Modify: `frontend/src/components/payrates/PayrateConfigTab.tsx` — thêm section
  "Cấu hình khác" (data đã có sẵn từ `useProjectPayRates`, chỉ đang lọc mất).
- Create: `frontend/src/components/payrates/hooks/usePayrateConflictRecovery.ts`.
- Create: `frontend/src/components/payrates/components/PayrateConflictDialog.tsx`.
- Modify: `frontend/src/types/api/payrate.types.ts` — `UpdatePayRateRequest`.
  `resolve_conflicts?: boolean`; kiểu conflict context.
- Modify: `frontend/src/hooks/api/usePayRates.ts` — `useUpdatePayRate` mutationFn truyền
  flag; KHÔNG tự showError khi code là conflict (để caller mở dialog) — xử lý bằng
  option `silentErrorCodes`.
- Modify: trang edit payrate lưu save-flow (tìm qua route
  `payrates/:id/edit` — `frontend/src/pages/...`, xác định khi làm).
- Modify (nếu có): `frontend/src/services/api/payrate.service.ts` update signature.

## Implementation Steps
1. Types + service: thêm field/kiểu; cập nhật `UpdatePayRateRequest`.
2. `usePayrateConflictRecovery` + `PayrateConflictDialog` (shadcn AlertDialog; copy VN:
   tiêu đề "Trùng cấu hình mức lương", mô tả danh sách config, nút chính
   "Xoá cấu hình xung đột và cập nhật", nút phụ "Để nguyên").
3. Nối vào save handler trang edit: lỗi conflict → mở dialog thay vì toast.
4. `PayrateConfigTab`: section "Cấu hình khác" — logic chọn: mọi payrate không thoả
   (active OR upcoming) theo đúng predicate hiện có (line 38–63); row xoá dùng
   `useDeletePayRate` + confirm.
5. Chạy `pnpm type-check` + `pnpm lint`.

## Success Criteria
- [ ] Reproduce được luồng lỗi → dialog → xác nhận → update thành công, danh sách config
      refresh đúng.
- [ ] Config rác 1-ngày nhìn thấy và xoá được trong tab.
- [ ] Desktop 1280 + mobile 390/320: không tràn ngang, parity Admin/Partner.

## Risk Assessment
- Double-submit khi partner bấm nhanh: disable nút khi `isPending`.
- Partner sợ mất dữ liệu: copy phải nói rõ timesheet chưa duyệt sẽ được tính lại (không
  mất chấm công), timesheet đã duyệt/trả thì không bao giờ bị xoá.
