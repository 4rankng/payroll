---
phase: 2
title: "Backend — structured conflict error + resolve_conflicts"
status: pending
priority: P1
dependencies: []
---

# Phase 2: Backend — structured conflict error + resolve_conflicts

## Overview
Biến lỗi chết "đã có cấu hình mức lương hiệu lực từ ngày này hoặc muộn hơn" thành lỗi có
cấu trúc (code + context liệt kê config xung đột và mức độ an toàn), và thêm flag
`resolve_conflicts` cho phép xoá sibling + áp update trong cùng một transaction.

## Requirements
- Functional:
  - PUT `/api/v1/payrates/:id` khi conflict → HTTP 422 kèm `code: PAYRATE_CONFLICTING_CONFIG`
    và `context`: danh sách sibling (`id`, `from_date`, `to_date`, `has_linked_timesheets`,
    `has_immutable_timesheets`).
  - Body `resolve_conflicts: true` → transaction xoá sibling `from_date >= from_date_mới`
    (khác target) rồi áp update như bình thường.
  - Hard block (giữ lỗi, không xoá gì) nếu bất kỳ sibling nào có timesheet bất biến:
    `timesheet_status='approved' OR payment_status='paid'`. Message nêu rõ id config chặn.
  - Timesheet mutable liên kết sibling bị xoá: OK — được re-link/re-price bởi
    `recalculateMutableTimesheets` (đã chạy trong cùng tx, điều kiện `date >= from_date`
    và mutable).
- Non-functional: mọi mutation vẫn trong transaction có `lockProjectTimelineTx`; không đổi
  contract của các path không flag.

## Architecture

```
UpdatePayrate (handler) ──> PayrateService.UpdatePayrate ──> TemporalService.UpdateEffectiveDatedPayrate
                                                                ├── validateEffectiveDateTx (sàn paid — chạy TRƯỚC, không đổi)
                                                                ├── (mới) nếu conflict và !resolve → trả lỗi có context
                                                                ├── (mới) nếu conflict và resolve → deleteConflictingSiblingsTx
                                                                │     └── guard: mỗi sibling: immutable linked = 0
                                                                └── split / in-place update như cũ
```

Thứ tự quan trọng: `validateEffectiveDateTx` đã chạy trước mọi nhánh conflict
(`payrate_temporal_service.go:194`) → sàn "sau lần trả lương cuối" luôn được tôn trọng
trước khi xoá bất cứ gì.

## Related Code Files
- Modify: `backend/internal/pkg/services/payrate_temporal_service.go`
  (`ensureLatestConfigTx` → trả chi tiết; thêm `deleteConflictingSiblingsTx`; thêm method
  đọc-readonly `DescribeUpdateConflicts` dùng chung để build context).
- Modify: `backend/internal/domain/errors.go` — dùng sẵn `NewValidationErrorWithCode` +
  `WithContext` (đã tồn tại, không cần thêm constructor).
- Verify/Modify: `backend/internal/transport/http/response/` — bảo đảm serializer đưa
  `code` + `context` của DomainError ra JSON body (kiểm tra handler update hiện tại).
- Modify: `backend/internal/app/dto/` — `UpdatePayRateRequest` thêm `ResolveConflicts bool`.
- Modify: `backend/internal/transport/http/handlers/payrate.go` (`UpdatePayrate` line 367).
- Modify: `backend/internal/constants/messages.go` — message hard-block mới có id config.
- Modify: `backend/internal/infra/persistence/payrate_repository.go` + port
  `domain/payrate.go` — query đếm immutable linked timesheets per payrate (một query
  `GROUP BY payrate_id` cho list sibling, tránh N+1).

## Implementation Steps
1. Thêm helper đọc conflict: input `projectID, excludeID, fromDate` → list sibling active
   (`deleted_at IS NULL`, `from_date >= fromDate`, `id <> excludeID`) kèm 2 cột đếm
   (total linked, immutable linked) bằng LEFT JOIN timesheets (tôn trọng soft-delete).
2. `ensureLatestConfigTx`: khi `count > 0`, build `*domain.DomainError` code
   `PAYRATE_CONFLICTING_CONFIG` kèm context (dùng helper bước 1). Khi caller truyền
   `resolve` thì thay vì trả lỗi → gọi `deleteConflictingSiblingsTx`.
3. `deleteConflictingSiblingsTx`: trong tx hiện tại — nếu bất kỳ sibling có
   `immutable_linked > 0` → trả `NewValidationError` mới (không xoá gì). Ngược lại
   soft-delete các sibling. Không cần xử lý unique open constraint: target đang sửa luôn
   là row mở (row đã kết thúc bị chặn ở `MsgCannotUpdateEndedPayrateVN`), sibling đều đã
   đóng.
4. Handler: parse `resolve_conflicts`, truyền xuống service; bảo đảm response lỗi expose
   `code` + `context`.
5. Log cấu trúc cho force-delete (số sibling xoá, ids) qua `observability.GetLogger()`;
   emit `PayrateDeletedEvent` cho từng sibling xoá (pattern non-blocking như
   `DeletePayrate`).

## Success Criteria
- [ ] Lỗi conflict trả JSON có `code` + `context.conflicting_configs[]`.
- [ ] `resolve_conflicts: true` + toàn sibling an toàn → update thành công, timeline
      nhất quán (predecessor tự đóng ở `from_date - 1`).
- [ ] Sibling có timesheet approved/paid → 422, không row nào bị xoá.
- [ ] Không flag → hành vi như cũ (không phá partner hiện tại).

## Risk Assessment
- Đụng vùng tài chính (payrate/timesheet linkage) → mọi đường xoá nằm sau guard immutable
  + trong transaction có lock; test Integration phủ cả 3 nhánh (block / resolve / old behavior).
- Context payload phình: sibling per project rất ít (thực tế ≤ vài row) — chặn上限 50 row
  trong context cho an toàn.
