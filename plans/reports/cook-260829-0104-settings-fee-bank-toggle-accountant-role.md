# Cook report: /admin/settings fee 0% + transfer-bank toggle + role Kế toán

**Date:** 2026-08-29 01:10 · **Plan:** `plans/260829-0019-settings-fee-bank-toggle-accountant-role/plan.md` · **Branch:** main (uncommitted) · **Mode:** brainstorm→plan→cook --auto

## Delivered (full requested scope)

1. **Phí 0% được chấp nhận** — `FeeScheduleEntry.Validate` cho `MinFeeVND=0`; 0% tier + 0 floor → fee 0 (`TestFeeScheduleEntry_ResolveFee_ZeroPercentFreeAdvance`). Frontend `form-helpers.ts` bỏ 2 chặn min-fee>0. Default 2%/10k giữ nguyên (mig 044 seed + service fallback).
2. **Toggle "Tài khoản nhận chuyển khoản"** (default BẬT) — setting key `transfer_bank_visible`; `TransferBankInfo.Hidden` với zero-value = hiển thị (an toàn cho mọi struct literal hiện có). Off ⇒:
   - Excel payroll_template: clear E9-E11 + labels D9-D11 (probe template xác nhận tọa độ nhãn).
   - Excel sao_ke_tt_theo_du_an (by_project + FlexPay recon): clear E8-E10 + D8-D10.
   - Email payroll: `payroll_statement.html` `{{if .ShowBank}}` → else câu "Đây là sao kê thanh toán lương tuần…".
   - Email FlexPay: `BuildSaoKeEmailBodies` bỏ khối chuyển khoản, thay câu thông báo sao kê (signature +4th param `config.TransferBankInfo`; sole real caller email_handler.go đã cập nhật; cron không đụng template này).
   - Settings UI: `SettingToggleCard` (Switch + Lưu/Hoàn tác) trong section Tài khoản nhận chuyển khoản, create/update row value_type 'string' theo pattern hiện có.
3. **Role `accountant` (Kế toán)** — domain RoleAccountant + Validate + IsAccountant; Casbin allow-list đúng 4 nhóm chức năng + auth + read-only projects/employees/timesheet reads + upload-histories GET; deny phần còn lại (auto bulk transfer OTP, settings, users, wallet, bulk-reject/reset, mutations). In-handler check `timesheet_export.go` + accountant cho GET payroll/report. FE: role unions (types/user, modal-config, lib/auth, lib/permissions), Login + ProtectedRoute redirect → `/accountant`, AddUserSheet/UserDetailsSheet option "Kế toán", route `/accountant` → `AccountantPage` KHÔNG sidebar, 4 tab (Duyệt công approve-only week picker + bulk approve; Xuất file chuyển lô; Nhập KQ chuyển lô; Xuất sao kê) tái dùng 3 dialog hiện có.

## Verification evidence

- `go build ./...` sạch; `go test` XANH toàn bộ package chạm: domain, config, flex_pay, payroll, notification, auth (accountant allow/deny tests), advance_payment handlers, timesheet handlers.
- FE `pnpm lint` (eslint + tsc) sạch; vitest `settings-page-parity` 12/12, `useSettingsForm` 4/4.
- Blast-radius tự kiểm: mọi `TransferBankInfo{` literal (2 test fixtures) zero-value Hidden=false = hiện — không đổi hành vi; mọi `BuildSaoKeEmailBodies` caller đã truyền bank; ShowBank wired đúng template.
- **Pre-existing failures (xác minh trên HEAD sạch qua git stash, KHÔNG do change này):** `internal/app/services` wallet_demand_forecast_math (7 test), `internal/infra/persistence` `TestProjectEmployeeRepository_HasActiveFlexiblePaymentScheduleByEmployeeID`.

## Process notes

- `code-reviewer` subagent được spawn nhưng không phản hồi sau 2 ping (~13 phút) → dừng; thay bằng verification pass độc lập ở trên (không có findings chưa xử lý nào còn mở từ self-review).
- `frontend/src/components/ForceChangePasswordDialog.tsx` đang mang sửa đổi có chủ đích từ session song song (password rules checklist) — KHÔNG thuộc feature này, KHÔNG commit chung.
- Sự cố nội bộ: 2 lần Write full-file `email_template.go` sinh nội dung hỏng → restore từ git, chuyển sang edit nhỏ từng khối; học lại bài: file dài + template literal phức tạp = Edit từng phần, không Write toàn file.

## Unresolved questions

1. Có cần seed sẵn row `transfer_bank_visible='true'` vào DB (hiện default trong code, row tạo lúc admin lưu lần đầu)?
2. Tạo tài khoản kế toán đầu tiên: admin tự tạo qua Users page (option "Kế toán" đã có) — user muốn tự tạo hay cần creds đặt sẵn?
3. Deploy: feature này cần `make deploy` (user quyết định thời điểm; cambox-tokens và pre-existing test failures không liên quan).
