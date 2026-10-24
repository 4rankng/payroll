---
title: "Payrate conflict recovery — safe delete & re-apply khi trùng cấu hình mức lương"
description: "Partner bị kẹt khi sửa cấu hình lương vì các config rác sinh ra do nhập nhiều lần (split-update chain) không hiển thị trong UI. Plan thêm: structured conflict error + resolve_conflicts flag trên PUT payrate, dialog khôi phục phía frontend, hiển thị config ẩn, và xử lý dứt điểm incident project 76 trên prod."
status: pending
priority: P1
effort: "4 phases — prod unblock (~30 phút, cần duyệt), backend (~1 ngày), frontend (~1 ngày), verification (~0.5 ngày)"
tags: [feature, backend, frontend, payrate, incident, partner-ux]
created: 2026-10-22
blockedBy: []
blocks: []
---

# Payrate conflict recovery — safe delete & re-apply

## Bối cảnh incident (evidence từ prod, 2026-10-22)

Partner nhập cấu hình lương nhiều lần → mỗi lần sửa đẩy `from_date` về **muộn hơn**,
`UpdateEffectiveDatedPayrate` chạy `splitUpdateTx` (update-as-create): đóng config cũ,
tạo config mới mở. Nhập 3 lần → chuỗi config:

| payrate id | project | from_date | to_date | ghi chú |
|---|---|---|---|---|
| 74 | 76 | 2026-08-01 | 2026-10-21 | config gốc, bị đóng 16:08 do split tạo #87 |
| 87 | 76 | 2026-10-22 | NULL | soft-delete 16:17 (lần xoá của partner) |
| 88 | 76 | 2026-10-15 | 2026-10-15 | **rác 1 ngày**, không timesheet nào trỏ vào |
| 89 | 76 | 2026-10-16 | 2026-10-16 | **rác 1 ngày**, không timesheet nào trỏ vào |
| 91 | 76 | 2026-10-17 | NULL | config mở duy nhất còn lại |

- Partner muốn: **một** config hiệu lực 15/10. Bị chặn bởi `ensureLatestConfigTx`
  (`payrate_temporal_service.go:330`) vì luôn tồn tại sibling `from_date >=` ngày nhập.
- **Timesheet:** project 76 có 0 timesheet từ 15/10 trở đi; lần trả lương cuối 14/10
  (approved + paid, trỏ vào #74). Cả #88/#89/#91 không có timesheet liên kết → xoá an toàn.
- **UI gap:** `PayrateConfigTab.tsx` chỉ render config *active* + *upcoming* → 2 config
  rác 1-ngày **vô hình**, partner không thể tự xoá → kẹt chết.

## Phases

| Phase | File | Nội dung | Phụ thuộc |
|---|---|---|---|
| 1 | `phase-01-prod-unblock.md` | Mở khoá project 76 trên prod qua API (cần duyệt, backup trước) | — |
| 2 | `phase-02-backend-resolve-api.md` | Structured conflict error + `resolve_conflicts` trên PUT payrate | — |
| 3 | `phase-03-frontend-recovery-ux.md` | Dialog khôi phục + hiện config ẩn + nút xoá (desktop/mobile, admin/partner) | Phase 2 |
| 4 | `phase-04-verification.md` | Integration tests + UI verification theo anti-lying contract | Phases 2–3 |

## Acceptance criteria

1. Partner gặp lỗi "đã có cấu hình mức lương hiệu lực từ ngày này hoặc muộn hơn" → thấy
   dialog liệt kê config xung đột + **1 nút** "Xoá cấu hình xung đột và cập nhật"; thao tác
   hoàn tất trong một lần xác nhận, timeline payrate nhất quán sau đó.
2. Config xung đột có timesheet **approved hoặc paid** liên kết → vẫn bị chặn cứng, dialog
   chỉ giải thích, không có nút xoá.
3. Config đã kết thúc/không active-upcoming hiển thị trong section "Cấu hình khác" và xoá
   được khi không ràng buộc (server vẫn chặn `IsUsedByTimesheets`).
4. Prod project 76 kết thúc ở timeline: #74 (01/08 → 14/10) + #91 (15/10 → mở); partner tự
   sửa được mức lương 15/10 từ UI.
5. `make lint` + `make api-test` (backend), `pnpm type-check` + `pnpm lint` (frontend) xanh.

## Quyết định thiết kế đã chốt

- **Một endpoint, flag-gated:** `PUT /api/v1/payrates/:id` nhận `resolve_conflicts: true`
  (không thêm endpoint riêng) — xoá sibling + áp update trong **một transaction** đã có
  `lockProjectTimelineTx`, không có cửa sổ race giữa 2 lần gọi.
- **Hard block** khi sibling có timesheet bất biến (`timesheet_status='approved'` OR
  `payment_status='paid'`). Timesheet mutable (pending/rejected) liên kết vào sibling bị
  xoá là an toàn: `recalculateMutableTimesheets` re-link + re-price lại toàn bộ timesheet
  mutable `date >= from_date` trong cùng transaction — đúng tinh thần "chưa duyệt thì xoá
  hết rồi cập nhật".
- Không cần đổi Casbin: partner đã có `PUT /api/v1/payrates/*` (`casbin_policy.csv:75`).

## Cross-plan scan

Không plan nào đang mở đụng payrate temporal logic. Soft note: `261004-1812-uu-full-migration`
(migrating Radix→Base UI) — khi làm Phase 3, ưu tiên component đang có trong repo, không
giới thiệu primitive mới.
