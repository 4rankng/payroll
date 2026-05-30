# Flexi Check-in / Check-out — Spec

> Phiên bản: 0.3 — 28/05/2026

---

## Tổng quan

Tính năng cho phép nhân viên flexi tại nhà máy LG Display check-in/check-out qua GPS, tích lũy tiền lương ngày và tăng hạn mức advance payment.

**Nguyên tắc cốt lõi:**
- Chấm công **KHÔNG** tạo luồng thanh toán riêng — chỉ tăng ngân sách cho advance request hiện có
- Attendance và Timesheet là hai entity riêng biệt, hai bảng riêng biệt
- Pipeline advance request (9Pay/OnePay/MBank) không thay đổi

**Công thức hạn mức:**
```
Hạn mức khả dụng = SumMaxAdvByEmployeeMonth(employee_id, for_month) - đã_ứng
```
Trong đó `đã_ứng` = tổng advance requests với status completed + approved + pending.

---

## Epic 1: Check-in / Check-out (Nhân viên)

### US-1.1 Check-in tại nhà máy

**Là** nhân viên flexi đã được admin bật check-in
**Tôi muốn** check-in bằng cách bấm nút trên app
**Để** ghi nhận thời gian bắt đầu làm việc

**Chấp nhận:**
- Frontend: Generate UUID idempotency key → store in sessionStorage
- Gửi tọa độ GPS từ trình duyệt
- Frontend validate GPS ≤ 100m (fast UX, client-side)
- Gửi API với header `X-Idempotency-Key: <uuid>`
- Backend re-validate GPS (security, server-side)
- Nếu GPS invalid → reject error 400, **không tạo record**
- Nếu hợp lệ → kiểm tra Redis cache idempotency key
  - Đã processed → return cached response (idempotent)
  - Chưa processed → tạo bản ghi attendance với `check_in_time`
- Store idempotency key in Redis with 24h TTL
- UNIQUE KEY `(employee_id, date)` ở DB level (safety net)
- Nếu nằm ngoài geofence → từ chối, hiện "Bạn cần ở trong khu vực nhà máy để check-in"
- Chỉ được check-in 1 lần/ngày — đã check-in rồi → nút đổi sang "Check out"
- Ngày công thuộc ngày check-in (ca đêm qua đêm vẫn tính cho ngày check-in)
- `project_id` lấy từ assignment của nhân viên (server-side, không do client gửi)

---

### US-1.2 Check-out cuối ca

**Là** nhân viên flexi đã check-in
**Tôi muốn** check-out bằng cách bấm nút trên app
**Để** hoàn tất bản ghi chấm công và tính tiền ngày

**Chấp nhận:**
- Frontend: Generate UUID idempotency key (nếu chưa có trong sessionStorage)
- Gửi tọa độ GPS, xác thực nằm trong 100m một cổng (giống check-in)
- Backend: Tìm attendance WHERE employee_id=X AND project_id=Y AND date=Z AND status="checked_in"
- Nếu **không tìm thấy** → reject error "Chưa có bản ghi chấm công nào cho hôm nay"
- Frontend validate GPS (client-side) → Backend re-validate (server-side)
- Nếu GPS invalid → reject error 400, **không update record**
- Ghi nhận `check_out_time`
- Hệ thống tự tính `earning_amount` dựa trên band match (xem US-5.1)
- Cập nhật hạn mức advance payment trong cùng một transaction (xem US-5.3)
- Status = "completed"
- Nếu đã check-out rồi → không cho check-out lại
- Không có giới hạn thời gian check-out (có thể check-out sau 18h, qua đêm, v.v.)

---

### US-1.3 Quên check-out & Orphan Record

**Là** nhân viên quên check-out
**Tôi muốn** biết ngày đó không được tính
**Để** tránh nhầm lẫn

**Chấp nhận:**
- Bản ghi chỉ có check-in, không có check-out → **không tính tiền**
- Sau **18 giờ** từ check-in mà chưa check-out → status = "orphaned"
- Orphan record **không tích lũy** vào advance payment quota
- Orphan record giữ trong DB (audit trail) nhưng exclude khỏi calculations
- Admin KHÔNG được sửa/xóa bản ghi (MVP)
- Nhân viên thấy bản ghi hôm nay ở trạng thái "Chưa check-out"
- Bản ghi này **không block** check-in ngày hôm sau

**Cron job (mỗi giờ):**
```sql
UPDATE attendances SET status = "orphaned"
WHERE status = "checked_in" 
  AND check_in_time < NOW() - INTERVAL 18 HOUR
```

---

### US-1.4 Xem bản ghi chấm công hôm nay

**Là** nhân viên đã enrolled
**Tôi muốn** xem trạng thái chấm công hôm nay
**Để** biết đã check-in/check-out chưa, bao nhiêu giờ, bao nhiêu tiền

**Chấp nhận:**
- API trả về bản ghi attendance raw, frontend suy ra UI state từ các field null:
  - `check_in_time == null` → chưa check-in, hiện nút "Check in"
  - `check_out_time == null` → đã check-in, hiện nút "Check out" + "Chưa check-out"
  - Cả hai có giá trị → hiện summary (giờ, tiền, cổng)
- Không có status field riêng

---

### US-1.5 Xem tất cả bản ghi kỳ lương

**Là** nhân viên đã enrolled
**Tôi muốn** xem danh sách tất cả bản ghi chấm công trong kỳ lương hiện tại
**Để** theo dõi số ngày làm và tiền tích lũy

**Chấp nhận:**
- Danh sách theo ngày, giảm dần
- Mỗi dòng: ngày, giờ check-in, giờ check-out, số giờ, số tiền
- Lọc theo kỳ lương hiện tại (dựa trên `SalaryPeriodFrom`/`SalaryPeriodTo` của project)

---

### US-1.6 Xem hạn mức còn lại

**Là** nhân viên đã enrolled
**Tôi muốn** xem hạn mức advance payment còn lại
**Để** biết mình có thể yêu cầu ứng thêm bao nhiêu

**Chấp nhận:**
- `Hạn mức = SumMaxAdvByEmployeeMonth(for_month) - đã_ứng`
- Trước khi admin upload bảng lương: hạn mức = tiền chấm công tích lũy
- Sau khi admin upload: hạn mức = giá trị Excel (ghi đè)
- Nếu hạn mức ≤ 0 → hiện "Đã hết hạn mức"
- **Frontend hiển thị nhiều hạn mức theo thứ tự:**
  - Tháng cũ nhất在前 (FIFO pattern)
  - Ví dụ (hôm nay 15/6):
    ```
    Chọn hạn mức để rút:
    ├─ Tháng 5: 2,000,000 VND (hết hạn sau 5 ngày) ⚠️
    └─ Tháng 6: 1,500,000 VND
    ```
  - User chọn hạn mức muốn rút → system trừ từ month đó

---

### US-1.7 Yêu cầu ứng trước (không đổi)

**Là** nhân viên đã enrolled
**Tôi muốn** yêu cầu ứng trước như bình thường
**Để** nhận tiền qua 9Pay/OnePay hoặc MBank

**Chấp nhận:**
- Luồng advance request hiện tại KHÔNG thay đổi
- Hạn mức được tăng nhờ tiền chấm công tích lũy (xem US-1.6)
- Pipeline thanh toán (9Pay/OnePay/MBank) KHÔNG đổi
- Nhân viên chưa enrolled vẫn dùng advance request bình thường (không ảnh hưởng)

---

## Epic 2: Cấu hình Ca làm (Admin)

Ca làm được lưu trong Payrate JSON với key format `position.ngày thường.HH:MM-HH:MM`.
Quản lý qua endpoint hiện có `PUT /payrates/:id`. Overlap validation được thêm vào `UpdatePayrate`.

**Phạm vi áp dụng:** Chỉ hiển thị UI ca làm cho project **flexible**. Một project là weekly/monthly OR flexible — không mixed. Project flexible dùng ca làm thay vì payrate theo loại ngày/giờ của timesheet system.

### US-2.1 Tạo ca làm

**Là** admin
**Tôi muốn** tạo ca làm cho dự án + vị trí
**Để** hệ thống tính lương theo khung giờ

**Chấp nhận:**
- Nhập: khung giờ bắt đầu, kết thúc, đơn giá/giờ (VND)
- Phạm vi: per project + per position (ví dụ: LG Display + "phổ thông")
- Khung giờ không được chồng lấn với ca khác cùng project + position
- Ví dụ: ca ngày `07:00-17:00` = 30,000đ/h, ca tăng ca `17:00-22:00` = 45,000đ/h

---

### US-2.2 Xem danh sách ca làm

**Là** admin
**Tôi muốn** xem tất cả ca làm theo dự án và vị trí
**Để** kiểm tra cấu hình

**Chấp nhận:**
- Lọc theo project, position
- Hiển thị: khung giờ, đơn giá

---

### US-2.3 Sửa ca làm

**Là** admin
**Tôi muốn** sửa khung giờ hoặc đơn giá của ca làm
**Để** điều chỉnh khi thay đổi chính sách

**Chấp nhận:**
- Sửa start time, end time, hourly rate
- Kiểm tra không chồng lấn sau khi sửa
- Chỉ ảnh hưởng bản ghi attendance mới (không tính lại bản ghi cũ)

---

### US-2.4 Xóa ca làm

**Là** admin
**Tôi muốn** xóa ca làm không dùng
**Để** giữ cấu hình sạch

**Chấp nhận:**
- Xóa ca làm khỏi Payrate JSON
- Nếu có nhân viên đang enrolled mà không còn ca làm → cảnh báo

---

## Epic 3: Enrollment (Admin)

### US-3.1 Bật check-in cho nhân viên

**Là** admin
**Tôi muốn** bật tính năng check-in/check-out cho nhân viên flexi
**Để** nhân viên đó tham gia luồng chấm công

**Chấp nhận:**
- **Individual:**
  - API: `PATCH /api/v1/projects/:projectId/employees/:employeeId/checkin-enabled`
  - Toggle `check_in_enabled = true` trên bản ghi `project_employees`
  - Chỉ áp dụng cho nhân viên có `payment_schedule = "flexible"`
  - Nhân viên được bật → thấy nút Check-in/Check-out trong PWA
- **Bulk:**
  - API: `PATCH /api/v1/projects/:projectId/employees/checkin-enabled/bulk`
  - Body: `{ "employee_ids": [1, 2, 3], "enabled": true }`
  - Toggle multiple employees at once
  - UI: Employee list → select multiple → "Enable check-in" button

---

### US-3.2 Tắt check-in cho nhân viên

**Là** admin
**Tôi muốn** tắt tính năng check-in/check-out cho một nhân viên
**Để** nhân viên đó quay lại chỉ dùng advance request

**Chấp nhận:**
- **Individual:**
  - API: `PATCH /api/v1/projects/:projectId/employees/:employeeId/checkin-enabled`
  - Body: `{ "enabled": false }`
  - Toggle `check_in_enabled = false`
  - **Reset quota to 0** cho current month onward:
    ```sql
    UPDATE advance_payments SET accumulated_amount = 0
    WHERE employee_id = ? AND project_id = ?
      AND for_month >= CURRENT_MONTH
    ```
  - Past months (đã settled) giữ nguyên
  - Bản ghi attendance đã có vẫn giữ nguyên (không xóa, historical data)
  - Future check-in requests: Reject với error "Check-in not enabled"
  - Advance payment requests: Reject "No quota available" (cho đến khi admin upload file)
  - Nhân viên không thấy nút Check-in/Check-out nữa
- **Bulk:**
  - API: `PATCH /api/v1/projects/:projectId/employees/checkin-enabled/bulk`
  - Body: `{ "employee_ids": [1, 2, 3], "enabled": false }`
  - Reset quota for all selected employees (current month onward)

---

## Epic 4: Quản lý chấm công (Admin)

### US-4.1 Xem danh sách bản ghi chấm công

**Là** admin
**Tôi muốn** xem tất cả bản ghi check-in/check-out
**Để** theo dõi nhân viên làm việc

**Chấp nhận:**
- Tab "Chấm công" trên trang AdvancePaymentsPage
- Danh sách: tên nhân viên, ngày, giờ check-in, giờ check-out, số giờ, số tiền, cổng
- Lọc theo: nhân viên, ngày, project
- Sắp xếp theo ngày giảm dần

---

### US-4.2 Xem chi tiết bản ghi

**Là** admin
**Tôi muốn** xem chi tiết một bản ghi chấm công
**Để** kiểm tra GPS và cách tính tiền

**Chấp nhận:**
- Hiển thị: thời gian check-in/out, tọa độ GPS, cổng
- Chi tiết tính tiền: từng ca làm → giờ tròn → đơn giá → thành tiền
- KHÔNG có nút sửa/xóa (MVP)

---

## Epic 5: Tính lương ngày (Backend)

### US-5.1 Tính tiền từ check-in/check-out

**Là** hệ thống
**Tôi muốn** tự động tính tiền ngày khi nhân viên check-out
**Để** cập nhật tiền tích lũy và hạn mức

**Chấp nhận:**
- Lấy ca làm từ Payrate JSON: keys `position.ngày thường.HH:MM-HH:MM`, đọc từ Redis cache `payrate:{projectID}`
- Mỗi ca là **flat rate per band** (không tính theo giờ)
- Format payrate: `"phổ thông.ngày thường.07:00-17:00": 300000` → 300k FLAT cho cả ca (không phải 30k/giờ)
- Làm tròn check-in **LÊN** giờ tròn tiếp theo (`ceil`)
- Làm tròn check-out **XUỐNG** giờ tròn trước đó (`floor`)
- Tìm band đầu tiên được cover bởi check-in/check-out (theo thứ tự JSON)
- Chỉ **một band duy nhất** — nếu cover nhiều band → dùng band đầu tiên
- Phải cover **đầy đủ band** (check_in ≤ band.start && check_out ≥ band.end) → mới được tính tiền
- Lưu `hours_worked`, `earning_amount` vào bản ghi attendance
- Cập nhật `advance_payments` trong cùng transaction (xem US-5.3)
- Reject nếu không cover band nào → earning_amount = 0
- Reject shift > 24h

**Thuật toán tìm band:**
```
for each band in payrate JSON (in order) {
  if checkIn <= band.start && checkOut >= band.end {
    // Found first matching band → use it
    earning_amount = band.rate  // Full flat rate
    return success
  }
}

// No band matched → 0 earnings
earning_amount = 0
```

**Ví dụ 1 — cover đầy đủ band:** check-in 06:50, check-out 17:00
- Band `07:00-17:00` trong JSON: 300,000đ (flat rate)
- checkIn(06:50) ≤ band.start(07:00) ✅
- checkOut(17:00) ≥ band.end(17:00) ✅
- → earning_amount = **300,000đ** (toàn bộ band, không tính giờ)

**Ví dụ 2 — cover nhiều band (dùng band đầu tiên):** check-in 06:50, check-out 21:30
- Band `07:00-17:00`: 300,000đ ✅ cover đầy đủ → **DÙNG BAND NÀY**
- Band `17:00-21:00`: 450,000đ ✅ cover đầy đủ → BỎ QUA (đã có band đầu)
- → earning_amount = **300,000đ** (chỉ band đầu, không cộng dồn)

**Ví dụ 3 — không cover band → 0 đồng:** check-in 08:00, check-out 16:00
- Band `07:00-17:00`: checkIn(08:00) > band.start(07:00) ❌ không cover đầu
- → earning_amount = **0đ** (phải cover đầy đủ mới tính)

**Ví dụ 4 — ca đêm:** check-in 21:50, check-out 06:10 (hôm sau)
- Band `22:00-06:00`: 480,000đ (flat rate cho ca đêm)
- checkIn(21:50) ≤ 22:00 ✅
- checkOut(06:10) ≥ 06:00 ✅
- → earning_amount = **480,000đ**

---

### US-5.2 Kiểm tra GPS geofence

**Là** hệ thống
**Tôi muốn** xác thực tọa độ GPS nhân viên nằm trong phạm vi cổng nhà máy
**Để** đảm bảo nhân viên thực sự ở nhà máy

**Chấp nhận:**
- Geofence config lưu trong `project_settings` JSON column:
  ```json
  {
    "geofence_lat": 20.8628815,
    "geofence_lng": 106.5653889,
    "geofence_radius": 100
  }
  ```
- Tính khoảng cách Haversine từ `(lat, lng)` gửi lên đến tâm geofence
- Nếu khoảng cách ≤ radius → hợp lệ
- Cổng gần nhất được ghi vào `check_in_gate` / `check_out_gate`
- Nếu không hợp lệ → trả lỗi "Bạn cần ở trong khu vực nhà máy"

**Frontend validation (client-side):**
- Validate GPS trước khi gọi API (UX tốc độ)
- Nếu invalid → show error, không gọi API

**Backend validation (server-side):**
- Re-validate GPS **trước khi tạo/update attendance record**
- Nếu invalid → return 400 error, không DB operation

**Cổng LG Display Hải Phòng (hardcode cho MVP):**
| Cổng | Lat | Lng | Radius |
|------|-----|-----|--------|
| A | 20.8628815 | 106.5653889 | 100m |
| B | 20.8666595 | 106.5666170 | 100m |
| C | 20.8679818 | 106.5711738 | 100m |
| D | 20.8648142 | 106.5705864 | 100m |

---

### US-5.3 Tích lũy ngân sách advance payment

**Là** hệ thống
**Tôi muốn** cập nhật hạn mức advance payment sau mỗi checkout thành công
**Để** nhân viên có thể yêu cầu ứng ngay khi có tiền tích lũy

**Chấp nhận:**

**Khi check-out thành công** (trong cùng transaction với attendance update):
- Tính `for_month = ForMonthFromDate(check_in_date, project.SalaryPeriodFrom)`
  - Nếu `check_in_date.Day() >= SalaryPeriodFrom` → `for_month` = tháng hiện tại
  - Nếu `check_in_date.Day() < SalaryPeriodFrom` → `for_month` = tháng trước
  - `SalaryPeriodFrom = 0` → xử lý như = 1 (calendar month)
- Upsert vào `advance_payments` với `upload_date = for_month`:
  - Chưa tồn tại → INSERT `max_adv_amount = earning_amount`
  - Đã tồn tại → cộng dồn `max_adv_amount += earning_amount`
- `project_id` và `SalaryPeriodFrom` đọc từ Redis cache `employee_project:{employeeID}`

**Khi admin upload FlexPay Excel** (nhân viên enrolled):
- Import pre-load enrolled employee IDs: `SELECT employee_id FROM project_employees WHERE check_in_enabled = true AND project_id IN (?)`
- Nhân viên enrolled: dùng `upload_date = for_month` → UNIQUE KEY trùng → ghi đè tự nhiên qua `ON DUPLICATE KEY UPDATE`
- Nhân viên không enrolled: hành vi cũ (`upload_date = calendar month of upload`, stacking)
- Giá trị Excel ghi đè trực tiếp — không dùng GREATEST. Admin upload là giá trị cuối cùng
- Idempotency qua `last_applied_asset_id`: cùng file → no-op
- Nếu nhân viên đã ứng vượt hạn mức mới → xử lý qua sao kê, không block

**Công thức hạn mức không đổi:**
- `SumMaxAdvByEmployeeMonth` tự động tính đúng — một dòng duy nhất cho enrolled employee
- Kỳ cũ đóng băng tự nhiên — kỳ mới có `for_month` khác, không ảnh hưởng lẫn nhau

---

### US-5.4 Kiểm tra 1 check-in/ngày

**Là** hệ thống
**Tôi muốn** ngăn nhân viên check-in lần thứ 2 trong ngày
**Để** đảm bảo chỉ 1 cặp check-in/check-out mỗi ngày

**Chấp nhận:**
- UNIQUE KEY `(employee_id, date)` trên bảng `attendances` đảm bảo constraint ở DB level
- Check-in lần 2 trong ngày → từ chối, hiện "Bạn đã check-in hôm nay rồi"
- Bản ghi không có check-out (quên check-out) → **không block** check-in ngày hôm sau

---

## Epic 6: Phân tách kỳ lương

### US-6.1 Phân tách tự nhiên qua for_month

**Là** hệ thống
**Tôi muốn** phân tách dữ liệu chấm công giữa các kỳ lương tự động
**Để** không cần cơ chế expiry phức tạp

**Chấp nhận:**
- Mỗi bản ghi attendance liên kết với ngày check-in cụ thể
- `for_month` trên `advance_payments` phân tách ngân sách theo kỳ — kỳ mới có key khác, không ảnh hưởng kỳ cũ
- Bảng `attendances` KHÔNG có cột `status` hay `expired`
- Khi nhân viên xem summary → lọc theo `SalaryPeriodFrom`/`SalaryPeriodTo` của project

---

## Phạm vi bỏ qua (MVP)

| Tính năng | Lý do bỏ qua |
|-----------|--------------|
| Notification check-in/check-out | MVP |
| Admin sửa/xóa bản ghi attendance | MVP |
| Giao diện cấu hình geofence | Hardcode LG Display |
| Chống giả mạo GPS | Chấp nhận rủi ro |
| Phân biệt ngày thường/cuối tuần/lễ trong ca làm | MVP — chỉ `ngày thường` |
| Thay đổi sao kê | Giữ nguyên định dạng |

---

## Domain Glossary

### Attendance
Bản ghi chấm công GPS cho nhân viên flexi. Ghi nhận thời điểm nhân viên có mặt tại cổng nhà máy. **Không kích hoạt thanh toán** — chỉ tăng ngân sách advance payment. Thuộc về Project + Employee. Không có cột `status` hay `expired`.

### Timesheet
Bản ghi công của nhân viên cho một ngày. Có workflow phê duyệt (pending → approved/rejected) và pipeline thanh toán (pending → paid). **Kích hoạt payout thực sự.** Tách biệt hoàn toàn với Attendance.

### Ca làm
Entry trong Payrate JSON với key format `position.ngày thường.HH:MM-HH:MM` → VND/giờ. Ví dụ: `"phổ thông.ngày thường.07:00-17:00": 30000`. Day type cố định là `ngày thường` (MVP). Parsed một lần, cache Redis `payrate:{projectID}`. Ca đêm: `end < start` (vd `22:00-06:00`) được hiểu là qua đêm hôm sau.

### Payrate
Cấu hình JSON linh hoạt scoped theo Project. Dùng cho cả Timesheet system (key path `position.dayType.hourType`) và Attendance system (key path `position.ngày thường.HH:MM-HH:MM`). Phân biệt bằng format của segment cuối.

### Advance Payment
Ngân sách hạn mức ứng lương theo tháng. Với nhân viên enrolled: **đúng một dòng** per `(employee, project, for_month)` với `upload_date = for_month`. Checkout tích lũy vào dòng này; admin upload ghi đè dòng này. Với nhân viên không enrolled: nhiều dòng per `for_month` (một per lần upload), summed bởi `SumMaxAdvByEmployeeMonth`.

### Advance Payment Request
Yêu cầu ứng trước của nhân viên. PENDING → APPROVED → COMPLETED/FAILED/CANCELLED. Budget-check: `SumMaxAdvByEmployeeMonth - already_requested`.

### Flexi Employee
Nhân viên có `payment_schedule = "flexible"` trên `project_employees`. Gán tại thời điểm tạo assignment — không thể thay đổi qua schedule-change workflow. MVP: một project duy nhất (LG Display).

### Enrolled
Nhân viên flexi có `check_in_enabled = true` trên `project_employees`. Admin toggle on/off. Tắt enrollment: bản ghi attendance cũ giữ nguyên, tiền tích lũy vẫn tính vào hạn mức.

### for_month
Label của kỳ lương mà ngày check-in thuộc về. Công thức:
```
if check_in_date.Day() >= SalaryPeriodFrom → for_month = tháng hiện tại
else → for_month = tháng trước
SalaryPeriodFrom = 0 → xử lý như = 1
```
Dùng `ForMonthFromDate(date, salaryPeriodFrom)` — **không dùng** `AdvanceMonthFromTime` (hardcode day-21).

---

## Resolved Design Decisions

1. **Attendance ≠ Timesheet** — hai entity riêng, hai bảng riêng
2. **Ca làm trong Payrate JSON** — key `position.ngày thường.HH:MM-HH:MM`; parse + cache Redis; invalidate khi update payrate
3. **Flexible schedule immutable** — không thể đổi sang/từ flexible qua schedule-change workflow
4. **Flexi = 1 project (MVP)** — LG Display only
5. **Không có status/expiry trên attendance** — `for_month` phân tách kỳ tự nhiên
6. **Admin upload ghi đè** — không GREATEST, Excel là giá trị cuối cùng
7. **Over-advance qua sao kê** — không block khi nhân viên đã ứng vượt hạn mức mới
8. **`for_month` từ kỳ lương, không phải calendar month** — dùng `ForMonthFromDate(date, SalaryPeriodFrom)` mới
9. **Toggle enrollment qua API mới** — `PATCH /api/v1/projects/:projectId/employees/:employeeId/checkin-enabled`
10. **`salary` column dropped** — xóa khỏi `advance_payments` trong migration 064
11. **`upload_date = for_month` cho enrolled employees** — không sentinel string; UNIQUE KEY tự enforce 1 dòng/kỳ
12. **Project info cache** — `employee_project:{employeeID}` cache `{project_id, salary_period_from}`; invalidate khi assignment hoặc project salary period thay đổi
13. **Checkout transaction** — attendance update + advance_payments upsert trong cùng một DB transaction
14. **`project_id` server-side** — derive từ `ProjectEmployee` assignment, client không gửi
15. **Today's status: raw record** — API trả attendance record, frontend suy ra UI state từ null fields
16. **FlexPay import branching** — pre-load enrolled IDs trước loop; enrolled → overwrite `upload_date = for_month`; không enrolled → path cũ
17. **Reuse `PUT /payrates/:id`** — ca làm CRUD đi qua endpoint payrate hiện có; thêm overlap validation vào `UpdatePayrate`
18. **Day type cố định `ngày thường`** (MVP) — không phân biệt cuối tuần/lễ
19. **Project là weekly/monthly HOẶC flexible** — `projects.is_flexible BOOLEAN NOT NULL DEFAULT FALSE`. UI ca làm và enrollment chỉ hiện cho `is_flexible = true`. Migration 065 thêm cột + set LGD `is_flexible = true`.
20. **Advance payment quota = Per-band flat rate** — `"phổ thông.ngày thường.07:00-17:00": 300000` → 300k FLAT (không phải 30k/giờ). Khác với weekly/monthly (per-hour).
21. **Single-band only** — check-in/check-out phải cover đầy đủ 1 band → mới tính tiền. Cover nhiều band → dùng band đầu tiên. Không split/join bands.
22. **18h orphan rule** — check-in > 18h chưa check-out → status = "orphaned". Orphan record không accumulate quota. Giữ trong DB (audit trail).
23. **UUID idempotency key** — Frontend generate UUID, send `X-Idempotency-Key`. Backend cache Redis 24h. Prevent multi-tap duplicates. UNIQUE KEY DB safety net.
24. **GPS validation before DB** — Frontend validate + backend re-validate. Invalid GPS → reject error 400, no DB operation. Defense in depth.
25. **Geofence in project_settings JSON** — `{"geofence_lat": 10.8, "geofence_lng": 106.6, "geofence_radius": 100}`. Not hardcoded columns.
26. **Quota reset on disable** — `check_in_enabled = false` → reset quota to 0 for current month onward. Past months untouched.
27. **Frontend quota display: FIFO** — Oldest month first. "Tháng 5: 2M (hết hạn sau 5 ngày) ⚠️" then "Tháng 6: 1.5M".
28. **Bulk enrollment toggle** — Both individual (edit employee) + bulk (select list → enable/disable). Two APIs.
29. **Band ordering** — Bands stored in start-time order, sorted before matching for deterministic behavior.
30. **Enrollment disable = immediate shutdown** — Existing checked_in records become orphan immediately, all check-in/check-out operations blocked.
31. **Orphan quota reversion** — When record becomes orphaned, set `earning_amount = 0` AND subtract contribution from `advance_payments` (set `max_adv_amount = 0` for current month onward).
32. **Idempotency key scope** — UUID for same-session retries only. UNIQUE KEY `(employee_id, date)` is ultimate duplicate prevention across sessions.
33. **Standard rounding** — Ceil(07:00:00) = 07:00, Floor(17:00:00) = 17:00 (standard mathematical behavior).
34. **Band overlap validation** — Validate across ALL payrates for same project + position to prevent accidental overlaps.
35. **Check-in validation order** — Check Redis idempotency cache → Validate GPS → Check UNIQUE KEY constraint.
36. **Orphan record retention** — Keep as historical audit data, exclude from calculations via status filter.
37. **for_month boundary** — If `date.Day() < SalaryPeriodFrom` → use previous month (strict inequality).
38. **GPS boundary** — Use ≤ comparison (distance ≤ radius = valid).
39. **Pending requests after disable** — Requests made before disable continue through pipeline; quota was valid at request time.
40. **Overnight shift date** — Use check-in date for `for_month` calculation (consistent with "date belongs to check-in day" rule).
41. **Gate selection** — First gate in list (A, B, C, D) that passes distance check.
42. **Enrollment prerequisites** — Require GPS geofence AND payrate bands configured before enabling check-in. Reject if missing.
43. **Over-advance handling** — Allow negative quota when admin upload overwrites accumulated quota below already-withdrawn amount. Completed withdrawals stay completed.
44. **Strict GPS validation** — Both frontend and backend use strict 100m threshold. No tolerance buffers.
45. **Flexible employee = single project** — Flexible employee can only have ONE active assignment. Reject new assignments if existing exists.
46. **Assignment disable quota behavior** — Zero out quota (set `max_adv_amount = 0`) for current month onward when assignment disabled.
47. **Project switch quota** — Old project quota zeroed out (not left inaccessible). Employee starts fresh in new project.
48. **Pending request cancellation on disable** — Cancel all PENDING requests immediately when assignment disabled. Let APPROVED/processing requests complete.
49. **Month selection for advance** — Employee explicitly selects which month's quota to withdraw from (FIFO display order).
50. **SalaryPeriodFrom freeze at check-in** — Store `salary_period_from` at check-in time. Use stored value (not current config) for `for_month` calculation at check-out.
51. **for_month calculation timing** — Calculate at check-out using stored `salary_period_from` + check-in date (don't store `for_month` at check-in).
52. **Excel upload idempotency** — Use `last_applied_asset_id`. Same file = no-op. Different file = overwrite. Admin is source of truth.

---

## Attendance Table Schema

```sql
CREATE TABLE attendances (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  employee_id      BIGINT UNSIGNED NOT NULL,
  project_id       BIGINT UNSIGNED NOT NULL,
  date             DATE NOT NULL,            -- ngày check-in (ca đêm = ngày bắt đầu ca)
  check_in_time    DATETIME(3) NOT NULL,
  check_out_time   DATETIME(3) NULL,
  check_in_lat     DECIMAL(10,7) NOT NULL,
  check_in_lng     DECIMAL(10,7) NOT NULL,
  check_out_lat    DECIMAL(10,7) NULL,
  check_out_lng    DECIMAL(10,7) NULL,
  check_in_gate    VARCHAR(50) NOT NULL,
  check_out_gate   VARCHAR(50) NULL,
  hours_worked     DECIMAL(5,2) NULL DEFAULT 0,
  earning_amount   BIGINT NULL DEFAULT 0,    -- VND
  created_at       DATETIME,
  updated_at       DATETIME,
  UNIQUE KEY uq_employee_date (employee_id, date),
  INDEX idx_project_date (project_id, date)
);
```
