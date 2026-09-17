# Debug: "Cần nạp thêm" 2.417.404.960 ₫ trên dashboard ứng lương

- **Ngày:** 2026-09-17
- **Commit:** `40bda95b` — `fix(wallet): stop single-outlier cycle from exploding demand-forecast top-up`
- **Trạng thái:** Đã fix, test, push origin/main

## Triệu chứng

Card "CẦN NẬP THÊM" trên trang Quản lý ứng lương (admin) hiển thị **2.417.404.960 ₫**
mặc dù: ví có 48,8M, kỳ này chỉ giải ngân 1,85M, kỳ xấu nhất từ trước tới nay
(kỳ 08/2026) mới ~882M nhu cầu — sao kê 9/9 cho thấy giải ngân thực ~700M/kỳ.

## Nguyên nhân gốc (đã tái hiện bằng code production + dữ liệu DB thật)

Ngày 9–19 hàng tháng là **locked gap** → forecast nhắm kỳ kế tiếp (2026-10),
`todayCycleDay = 0` → phân phối "nhu cầu còn lại" = toàn bộ tổng các kỳ lịch sử:

| for_month | Tổng trong window (net) |
|---|---|
| 2026-05 | 134.967.030 |
| 2026-06 | 121.420.920 |
| 2026-07 | 181.292.965 |
| **2026-08** | **881.756.800** (outlier 1 lần, 280 request) |
| 2026-09 | 0 (4 request tạo 11/09 rơi cycle_day −8 → bị pivot loại) |

Chuỗi lỗi:

1. Outlier kỳ 08 → `trendRatio = max/min = 7.26 ≥ 3` → code kết luận "tăng trưởng định hướng".
2. `growthEWMA = 3.03` (rate [0.90, 1.49, 4.86] bị chi phối bởi spike đơn lẻ 4.86×)
   **nhân cả phân phối**: gamma p95 (953M — đã chứa outlier trong đuôi) → 2.89 tỷ.
   Outlier bị đếm kép: 1 lần trong gamma variance, 1 lần trong growth factor.
3. Quota ceiling (`max_adv_amount`) không áp dụng trên path này (chỉ chạy khi
   `paceMethod == "cohort-median"`, cần `todayCycleDay ≥ 5`); kỳ 2026-10 cũng
   chưa upload quota. Và quota kỳ 08 (2.76 tỷ) còn lỏng hơn cả số thống kê.

Replica chính xác (Go test dùng hàm production + row DB thật): shortfall
**2.840.845.067** hôm nay; biến thể window 12 tháng **2.487.041.858** ≈ đúng
số screenshot (chênh 2,8% do prod tiến triển sau snapshot local 11/09).

## Fix

`wallet_demand_forecast_math.go` — 2 guard cho growth adjustment:

1. **Consistency gate:** max rate trong EWMA window > 2× median rate → spike
   đơn lẻ là variance, không phải direction → bỏ re-centering.
2. **Cap factor ở `defaultPaceScaleCap` (2.0)** — cùng triết lý pace cap đã có
   ("outlier cycle easily produces a 5-8x ratio... compounding two safety margins").

Ramp bền vững thật (Ky-2, rates 1.0–1.9, factor 1.63) vẫn được re-center như cũ.

## Kết quả trên dữ liệu hiện tại

| | Trước | Sau |
|---|---|---|
| Locked-gap shortfall | 2.840.845.067 ₫ (screenshot: 2.417.404.960 ₫) | **904.888.111 ₫** |

p95 ≈ 953,7M ≈ chu kỳ xấu nhất từng quan sát (881,8M + margin 8%) — đúng bản chất
"mức cần giữ để an toàn 95%".

## Verification

- `go build ./...` ✅; `go test ./...` (backend, toàn bộ) ✅
- 3 test mới: math gate/cap + service regression (dữ liệu hình dạng prod) ✅
- 2 test growth hiện có (Ky-2 ramp, stationary) không đổi hành vi ✅
- **`make api-test` KHÔNG chạy:** backend chưa từng start từ checkout này (không
  `.env`), và `payroll-mysql` đang giữ bản restore production thật — 30 flows
  sẽ ghi dữ liệu test vào DB đó. Thay đổi là pure-math một hàm, đã cover bằng
  unit + service regression với đúng dữ liệu prod. Cần chạy lại api-test khi có
  env test dùng DB dùng một lần.

## Ghi chú

- Có session khác đang làm việc đồng thời trong repo (BCC upload debug —
  `config.go`, `Makefile`, `.env.example`, excel/* modified). Commit này chỉ
  gồm 3 file forecast; các file kia không được đụng tới.
- Phát hiện phụ (chưa xử lý, ngoài scope): request của kỳ 09 tạo ngày 11/09
  (locked gap) mang `for_month=2026-09` nhưng cycle_day −8 → rơi ra ngoài
  window cohort, không bao giờ vào lịch sử forecast.

## Câu hỏi còn mở

1. ~~Request tạo trong locked gap (9–19) nên gắn `for_month` kỳ nào để cohort
   không mất dữ liệu?~~ **ĐÃ QUYẾT (17/09/2026, product owner):** request tạo
   trong locked gap bị cohort loại là hành vi mong muốn — bỏ qua, không sửa.
   Ghi chú đã thêm vào `pivotCohort`.
2. Có nên hiển thị nhãn phương pháp/confidence ("growth-adjusted",
   "monte-carlo") trên card để admin tự đánh giá độ tin cậy của con số này?
