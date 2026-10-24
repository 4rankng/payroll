---
phase: 1
title: "Prod Unblock — project 76"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Prod Unblock — project 76

## Overview
Mở khoá partner ngay lập tức bằng cách xoá 2 config rác (#88, #89) và dời #91 về 15/10
**qua backend API** (không đụng SQL trực tiếp — giữ event/audit nhất quán).

## YÊU CẦU DUYỆT
Bước này **đột biến dữ liệu prod**. Phải có xác nhận của user trước khi chạy. Không tự chạy.

## Related Code Files
- Chỉ gọi API, không sửa code. Endpoint: `DELETE /api/v1/payrates/:id`, `PUT /api/v1/payrates/:id`.

## Implementation Steps
1. **Backup trước:** `make backup` (root Makefile) — dump MySQL prod về OneDrive. Phải
   thấy "Backup completed successfully".
2. Kết nối: SSH tunnel hoặc curl trực tiếp trên droplet (`https://tingting.vip/api/v1`).
   Dùng tài khoản admin (frankng) lấy JWT qua `/api/v1/auth/login`.
3. Xoá theo thứ tự tăng dần (mỗi lần check response 200):
   - `DELETE /api/v1/payrates/88` — `IsUsedByTimesheets(88)=0` đã xác minh qua SQL.
   - `DELETE /api/v1/payrates/89`.
   - `DeletePayrate` sẽ tự tìm predecessor theo `to_date = from_date-1`; đã suy luận cả
     hai thứ tự xoá đều an toàn (không có row nào có `to_date=2026-10-14` hay `=10-15`
     sau các bước).
4. Dời config mở về 15/10:
   - `PUT /api/v1/payrates/91` body: `{ from_date: "2026-10-15", payrate_json: <giữ nguyên
     giá trị hiện tại của #91> }`.
   - Kỳ vọng đi đường in-place update: `validateEffectiveDateTx` pass (sàn trả lương
     14/10), `ensureLatestConfigTx` pass (không còn sibling), predecessor sync tự đóng
     #74 tại 14/10.
5. **Xác minh DB (read-only):**
   ```sql
   SELECT id, DATE(from_date), DATE(to_date), deleted_at
   FROM payrates WHERE project_id = 76 ORDER BY from_date, id;
   ```
   Kỳ vọng: #74 (01/08 → 14/10), #91 (15/10 → NULL), #87/#88/#89 soft-deleted.
6. Báo partner đăng nhập lại và thử sửa cấu hình 15/10 → thành công (rung 3 xác minh UI
   nếu tiện; tối thiểu rung 2 API+DB).

## Success Criteria
- [ ] Backup prod tồn tại trên OneDrive trước khi đụng dữ liệu.
- [ ] Truy vấn xác minh trả đúng timeline kỳ vọng.
- [ ] Partner sửa được mức lương 15/10 từ UI không còn lỗi.

## Risk Assessment
- Rủi ro thấp nhất trong các phương án (SQL tay, restore backup). Rollback: restore từ
  backup Step 1 nếu sai — nhưng không nên cần vì mọi mutation đều qua domain logic.
