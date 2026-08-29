# Code review: wave #36 — fee 0% + transfer-bank toggle + accountant role (pending diff)

**Date:** 2026-08-29 10:08 · **Target:** uncommitted diff trên main (29 file, +559/−118; excl. `ForceChangePasswordDialog.tsx` — parallel-session edit, out of scope) · **Protocol:** Stage 1 spec-compliance → Stage 2 code-quality → Final verification.

## Process deviation (important)

`code-reviewer` subagent spawn thất bại 3 lần (1 lần buổi đêm + 2 reviewer scoped `rev-be`/`rev-fe`) — agents khởi động nhưng không bao giờ trả report sau 2 ping mỗi agent; đã stop, không để mồ côi. Stage 2 được thực hiện trực tiếp với cùng bộ hard-checks đã giao cho reviewer; mọi claim dưới đây có evidence từ lệnh chạy thật. Cần kiểm tra lại teammate/mailbox infra ở session sau.

## Stage 1 — Spec compliance: PASS (sau 3 fix)

| Yêu cầu | Trạng thái | Evidence |
|---|---|---|
| 1. Lưu phí 0% (default 2%) | ✅ sau fix | Validate BE cho MinFeeVND=0 (`TestFeeScheduleEntry_ResolveFee_ZeroPercentFreeAdvance`); **tìm thấy FE `Input min={1}` còn chặn 0 → sửa `min={0}`**; form-helpers bỏ 2 check; describeRuleFragments bỏ mệnh đề "mức tối thiểu 0 ₫" khi floor=0; default 2%/10k giữ nguyên (presets + mig 044) |
| 2. Toggle TK nhận CK (default bật) + email đổi nội dung | ✅ | key `transfer_bank_visible`; Hidden zero-value-safe; 3 Excel path xóa value+label (tọa độ nhãn xác minh bằng probe template thực: payroll D9-D11/E9-E11, sao_ke D8-D10/E8-E10); payroll email `{{if .ShowBank}}` → else câu thông báo; FlexPay email swap khối. **Fix: thống nhất ngữ nghĩa parse BE/FE** — BE: ''=default-visible, chỉ 'true' (case-insens) là visible, 'false'/'0'/rác = hidden; FE mirror chính xác; test edge thêm '0'/''. Mobile settings dùng chung SettingsGeneralPanel → toggle tự có |
| 3. Role kế toán: 4 chức năng + UI không sidebar | ✅ | Casbin allow-list đúng 4 nhóm + auth + read-only projects/employees + histories GET; deny còn lại (`TestAccountantRole_AllowList/DenyList`); in-handler check timesheet_export + accountant; FE plumbing đủ (types/AppRole/permissions/ProtectedRoute/Login → /accountant, AddUser/UserDetails "Kế toán"); route + AccountantPage 4 tab tái dùng dialog hiện có. **Fix MINOR: clear selection khi đổi tuần + guard date input rỗng** (tránh duyệt id không còn hiển thị / Invalid Date) |

Extras ngoài spec: casbin `/timesheets/grouped` + `bulk-transfer-upload-histories` GET (read-only, hỗ trợ UI, vô hại — NIT giữ nguyên).

## Stage 2 — Code quality findings (tự thực hiện)

- `[FIXED-CRITICAL]` FE `min={1}` chặn nhập 0 — mâu thuanh trực tiếp spec 1. → `min={0}`.
- `[FIXED-MAJOR]` Parse toggle lệch nhau BE (''→hidden) vs FE (''→visible, '0'→visible) — trạng thái UI có thể nói dối so với export thật. → thống nhất + 3 test case mới.
- `[FIXED-MINOR]` AccountantPage stale selection sau đổi filter; date input trống → Invalid Date. → effect reset + truthy guard.
- `[NIT-kệ]` `clearBankInfoCell` giả định value-cell cột E — đúng cho cả 2 template hiện tại (probe xác minh); comment đã ghi ràng buộc.
- `[NIT-kệ]` Casbin accountant `/timesheets/:id GET` về nguyên tắc keyMatch2 khớp cả path 1-segment như `bulk-approve` ở method GET — nhưng route đó chỉ định nghĩa POST nên vô hại; partner có deny-rule tương tự vì partner có GET surface rộng hơn.
- `[NIT-kệ]` textBody hidden để dòng trống thay khối bank — plain-text, vô hại.
- Không tìm thấy slop/dead-code mới; `BankInfoForStatement` + `SettingToggleCard` đều 1-usage hợp lý.

## Final verification (sau mọi fix)

- `go build ./...` sạch; `go test` 8 package chạm: **0 FAIL** (domain, config, flex_pay, payroll, notification, auth, timesheet+advance_payment handlers).
- FE `pnpm lint` (eslint + tsc) sạch; vitest settings 16/16.
- Pre-existing failures (đã verify trên HEAD sạch, không thuộc diff này): wallet_demand_forecast_math (7), persistence `HasActiveFlexiblePaymentScheduleByEmployeeID` (1).

## Verdict: **PASS-WITH-FIXES** — 4 fix đã áp dụng và verify; không còn finding mở nàoblocking. Diff sẵn sàng commit (nhớ loại `ForceChangePasswordDialog.tsx`).

## Unresolved

1. Teammate subagent infra (spawn ok, không reply) — cần xác minh ở session sau.
2. (Kéo từ cook) seed row `transfer_bank_visible` + tạo tài khoản kế toán đầu tiên — chờ user.
