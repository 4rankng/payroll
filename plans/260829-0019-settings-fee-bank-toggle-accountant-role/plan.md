# Plan: Admin settings (phí 0%, toggle TK nhận CK) + vai trò Kế toán

**Status:** IMPLEMENTED 2026-08-29 (all phases done; verification evidence below)

**Date:** 2026-08-29 · **Mode:** brainstorm→plan→cook `--auto` (chained) · **Branch:** main
**Contract:** outcome/constraints/non-goals/acceptance documented below; scouted codebase evidence inline.

## Contract

- **Outcome**
  1. Trên /admin/settings, admin chỉnh sửa được bảng phí ứng lương (fee schedule, đã có sẵn UI) và **lưu được phí 0%** (min fee 0 cho phép) — 0% = công nhân không mất phí. Default vẫn 2%/10k.
  2. Trên /admin/settings, admin bật/tắt được "Tài khoản nhận chuyển khoản" (mặc định bật). Khi tắt: sao kê Excel + email sao kê (payroll + FlexPay reconciliation) không còn thông tin tài khoản nhận; email chỉ còn là thông báo sao kê.
  3. Role mới "Kế toán" (`accountant`): login → `/accountant` (không sidebar), 4 tab: Duyệt công (chỉ duyệt, không từ chối), Xuất file chuyển lô, Nhập KQ chuyển lô, Xuất sao kê.
- **Constraints:** main branch; Casbin CSV + in-handler check là 2 lớp; không CHECK constraint DB; không migration (setting row tạo khi save, default trong code); Vietnamese UI; VND format; exports đã có audit sẵn.
- **Non-goals:** không đổi logic tính phí; không cho kế toán bulk-reject / settings / users / projects CRUD / wallet; không đụng provider flow; không redesign settings page.
- **Acceptance criteria:**
  1. `go build ./...` + unit tests pass; test chứng minh Validate nhận MinFeeVND=0 và 0% → fee 0.
  2. Test chứng minh: exporters ẩn dòng ngân hàng khi toggle off; email không còn khối chuyển khoản khi off; default bật.
  3. Casbin test: accountant cho phép đúng 4 nhóm endpoint + reads cần cho UI; từ chối phần còn lại.
  4. `pnpm lint` + `pnpm type-check` pass; /accountant render 4 tab; role plumbing hoàn chỉnh.

## Evidence (scouted)

- Fee: `domain/advance_payment_fee_schedule.go` Validate (`min_fee_vnd must be greater than 0` chặn 0); service fallback 2%/10k; UI sẵn trên SettingsPage (FeeScheduleSection); mig 044 seed 2%/10k.
- Toggle: keys `transfer_bank_account_holder/number/name` + `GetTransferBankInfo` (settings_config.go); consumers: `payroll/report_exporter.go:165` (E9-E11), `payroll/report_by_project_exporter.go:552` (E8-E10), `flex_pay/flex_pay_reconciliation_exporter.go:363-371` (E8-E10), `notification/email_service.go renderPayrollTemplate` (bankInfo param), `flex_pay/email_template.go BuildSaoKeEmailBodies` (hardcoded bank). Settings UI: `SettingsGeneralPanel.tsx` + `useSettingsForm.ts` (pattern update/create row, value_type 'string').
- Role: `domain/user.go` (RoleAdmin/Partner/Employee/AdvPartner, Validate :133), `configs/casbin_policy.csv`, `auth.CanAccess` (casbin enforcer from CSV at runtime), in-handler check `timesheet_export.go:126` (admin|partner only), `payroll.go` Export/Import chỉ cần user_id (Casbin là gate), GetTimesheets chỉ filter khi partner.
- Endpoints 4 chức năng: `POST /api/v1/timesheets/bulk-approve`; `POST /api/v1/payrolls/export-bulk-transfer` (+template); `POST /api/v1/payrolls/bulk-transfer-result`; `GET /api/v1/timesheets/payroll/report`.
- Frontend: role unions ở `types/user.ts`, `modal-config.types.ts`, `ProtectedRoute.tsx`; redirect `pages/Login.tsx:86-91`; role selects `AddUserSheet.tsx:262`, `UserDetailsSheet.tsx:200`; route tree `App.tsx:235+`; dialogs có sẵn: `BulkTransferExportDialog`, `BulkTransferResultUploadDialog`, `PayrollReportExportDialog`.

## Phases

### Phase 1 — Backend: fee 0% + toggle + consumers
1. `advance_payment_fee_schedule.go`: Validate cho MinFeeVND=0 (bỏ check >0, giữ >=0; cập nhật comment).
2. `settings_config.go`: key `transfer_bank_visible` (default true), `TransferBankInfo.Visible`, getTransferBankBool.
3. Exporters: khi !Visible → clear value cells + label cells (D9-E11 / D8-E10 ×2).
4. renderPayrollTemplate + BuildSaoKeEmailBodies: khi !Visible → bỏ khối "Thông tin chuyển khoản", thêm câu "Đây là sao kê…".
5. Tests: fee validate/resolve zero; settings default + off; exporter hidden rows; 2 email templates.

### Phase 2 — Backend: accountant role
1. `domain/user.go`: + RoleAccountant, Validate, IsAccountant.
2. `configs/casbin_policy.csv`: accountant section (cho phép: /api/v1/auth/me GET, /api/v1/auth/change-password POST, /api/v1/timesheets GET, /timesheets/:id GET, /timesheets/summary GET, /timesheets/bulk-approve POST, /payrolls/export-bulk-transfer POST, /payrolls/bulk-transfer-template GET, /payrolls/bulk-transfer-result POST, /timesheets/payroll/report GET, /api/v1/projects GET, /api/v1/employees GET, /api/v1/payrolls/bulk-transfer-upload-histories* GET).
3. `timesheet_export.go`: role check thêm accountant.
4. `authorization_service_test.go`: accountant allow/deny (pattern adv_partner tests).

### Phase 3 — Frontend: settings + fee form + role plumbing
1. useSettingsForm + SettingsGeneralPanel: transfer_bank_visible (Switch, default on).
2. FeeScheduleFormDialog: bỏ chặn min fee > 0 (cho 0).
3. Role unions (types/user.ts, modal-config.types.ts, ProtectedRoute, Login redirect, UsersPage filter types).
4. AddUserSheet + UserDetailsSheet: option "Kế toán".

### Phase 4 — Frontend: /accountant page
1. `src/pages/accountant/AccountantPage.tsx`: shell không sidebar, 4 tab dùng lại 3 dialog có sẵn + bảng duyệt công (approve-only) bằng hooks có sẵn.
2. Route + ProtectedRoute(["admin","accountant"]).

### Phase 5 — Verify
1. `go build ./...`, `go test` (fee schedule, config, payroll, flex_pay, notification, auth).
2. `pnpm lint` + `pnpm type-check`.
3. code-reviewer subagent (bắt buộc) → fix findings.
4. Report + journal (config-dependent).

## Risks
- **Label cells của template** (D9-E11 vs D8-E10) — xác minh khi sửa; test e2e bank_info sẽ bắt sai.
- **UI duyệt công kế toán** tái sử dụng hooks admin — nếu hook yêu cầu role/param admin-only thì build list riêng; Casbin vẫn là ranh giới thật.
- **Email history đã gửi** không đổi retroactively — toggle chỉ ảnh hưởng email/Excel phát sinh sau khi đổi.

## Outcome record (2026-08-29)

All 5 phases implemented and verified:

1. **Fee 0%:** `FeeScheduleEntry.Validate` now allows `MinFeeVND=0` (0% + 0 floor = free advance, unit-tested `TestFeeScheduleEntry_ResolveFee_ZeroPercentFreeAdvance`). Frontend `form-helpers.ts` min-fee>0 checks removed. Default 2%/10k unchanged (mig 044 seed + fallback intact).
2. **Transfer-bank toggle:** setting key `transfer_bank_visible` (default visible). `TransferBankInfo.Hidden` (zero-value = shown, backward compatible). Consumers: `report_exporter` E9-E11 + labels D9-D11 cleared; `report_by_project_exporter` E8-E10 via same helper; `flex_pay_reconciliation_exporter` E8-E10 + D8-D10; `payroll_statement.html` `{{if .ShowBank}}` (else informational line); `flex_pay.BuildSaoKeEmailBodies` bank block swap. Unit tests: settings hidden/default, email hidden.
3. **Accountant role:** `RoleAccountant` + Validate + IsAccountant; casbin_policy.csv allow-list (auth me/logout/change-password; timesheets list/:id/summary/grouped GET + bulk-approve POST; payroll/report GET; payrolls bulk-transfer-template GET, export-bulk-transfer POST, bulk-transfer-result POST, upload-histories GET; projects/employees GET); `timesheet_export.go` role check + accountant. Tests: `TestAccountantRole_AllowList/DenyList`.
4. **Frontend:** settings toggle card (SettingToggleCard), role plumbing (types/user, modal-config, lib/auth AppRole, lib/permissions, ProtectedRoute + Login redirect → /accountant, AddUserSheet/UserDetailsSheet "Kế toán", UsersPage filter unions), route `/accountant` → `AccountantPage` (no sidebar, 4 tabs: Duyệt công approve-only week picker + bulk approve; Xuất file chuyển lô → BulkTransferExportDialog; Nhập KQ → BulkTransferResultUploadDialog; Xuất sao kê → PayrollReportExportDialog).

**Verification:** `go build ./...` clean; touched-package `go test` green (domain, config, flex_pay, payroll, notification, auth, advance_payment handlers, timesheet handlers); frontend `pnpm lint` (eslint+tsc) clean; vitest `settings-page-parity` 12/12 + `useSettingsForm` 4/4 pass.

**Known pre-existing failures (verified on clean HEAD via git stash — NOT from this change):** `internal/app/services` wallet_demand_forecast_math tests; `internal/infra/persistence` TestProjectEmployeeRepository_HasActiveFlexiblePaymentScheduleByEmployeeID.

**Note:** `frontend/src/components/ForceChangePasswordDialog.tsx` carries an unrelated parallel-session modification (password rules checklist) — left untouched, must NOT be committed together with this feature.
