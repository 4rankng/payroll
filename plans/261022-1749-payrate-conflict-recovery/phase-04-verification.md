---
phase: 4
title: "Verification — integration tests + UI verification"
status: pending
priority: P1
dependencies: [2, 3]
---

# Phase 4: Verification

## Overview
Chống "nói dối xác minh": mọi claim UI phải rung 3 (UI DRIVEN) với artifact; backend
phủ integration test đủ 3 nhánh conflict.

## Backend (rung: tests)
Thêm scenario vào `backend/tests/integration/` (pattern `flow_payrate_crud.go`):
1. Update trùng sibling không flag → 422, JSON có `code=PAYRATE_CONFLICTING_CONFIG` +
   `context.conflicting_configs` đúng nội dung.
2. `resolve_conflicts: true`, sibling không timesheet → 200; DB: sibling soft-deleted,
   target mở với `from_date` mới, predecessor đóng đúng `from_date-1`.
3. `resolve_conflicts: true`, sibling có timesheet approved → 422, KHÔNG row nào đổi.
4. `resolve_conflicts: true`, sibling có timesheet pending_approval → 200; timesheet đó
   được re-link `payrate_id` mới + `amount` tính lại theo mức mới.
5. Regression: update thường không flag vẫn hành vi cũ; `from_date < sàn paid` vẫn chặn.

Chạy: `make lint` rồi `make api-test`.

## Frontend
- `pnpm type-check && pnpm lint`.
- UI DRIVEN (rung 3) — Playwright/agent-browser trên dev stack (`make dev`), tài khoản
  partner `cuongnv/Admin123`:
  1. Dựng dữ liệu fixture: project test + 2 payrate (mô phỏng sibling conflict).
  2. Vào edit payrate → nhập ngày trùng → thấy lỗi → bấm nút recovery → chụp screenshot
     sau click; assert DOM text kết quả (toast/list config); query MySQL xác nhận
     sibling soft-delete (DB side-effect); lưu driver log.
  3. Section "Cấu hình khác": hiện config kết thúc, xoá thành công (screenshot + DB).
  4. Parity: lặp lại bước 2 với admin `frankng`; mobile 390px và 320px (dialog + section
     không tràn ngang, nút ≥44px).

## Coverage report bắt buộc (kết thúc phase)
Điền bảng theo UI verification contract — cột "Not covered" không được rỗng
(các mục tối thiểu: role còn lại, mobile viewport, env local-vs-prod, đường lỗi/rollback).

## Success Criteria
- [ ] `make api-test` xanh với 5 scenario mới.
- [ ] Bảng coverage có rung label + artifact path cho từng claim.
- [ ] Phase 1 (prod) xác minh lại sau deploy: partner tự thao tác được.

## Risk Assessment
- Fixture payrate temporal dễ_ghost data → xoá sạch fixture sau test; dùng project test
  riêng, không đụng project 76/15/68.
