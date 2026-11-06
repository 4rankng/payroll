# Investigation: BCC upload "Không thể tạo hồ sơ nhân viên" (7 errors)

Date: 2026-11-02 · Scope: prod read-only diagnosis · File: `BCC_ChotLuongTuan_26_29.10.2026 (1).xlsx` (project 76, month 2026-10)

## Outcome

All 7 row errors have one cause: **the mandatory-mobile policy is ON (`employee.mobile_required = true` on prod since 25 Oct) and those 7 employees have no phone number in the Excel's STK sheet**, so `createEmployeeCore` rejected each profile creation with:

> `Số điện thoại là bắt buộc khi bật chế độ bắt buộc số điện thoại cho nhân viên mới`

The UI then masks that actionable reason behind the generic label "Không thể tạo hồ sơ nhân viên" (`frontend/src/utils/import-errors.ts:28` — the `không thể tạo nhân viên` rule fires before any more specific mapping).

## Evidence chain

1. **Prod asset 735** (19:52:32 +07, `partner_bcc_import`): `error_count: 7`, `created_count: 122`; `error_detail` holds all 7 reasons — identical mobile-required message, one per CCCD:
   `031168011277, 031183222198, 017192003558, 031097005313, 031203010176, 031185016231, 031305007574`
2. **Prod DB**: none of those 7 CCCDs exist in `employees`; no employee rows match those names either → genuinely new hires, auto-create attempted, creation blocked.
3. **Prod settings**: `employee.mobile_required = true`, `value_type: boolean`, updated `2026-10-25 06:31:02`.
4. **Backend gate**: `backend/internal/app/services/employee/service.go:192-194` — when the toggle is on, every create path (manual, Excel import, BCC import) requires a mobile. Import path builds the employee from the STK row (`bcc_import_weekly_payment.go:234-242`, `:425-440`); empty mobile → rejection, CCCD added to `blockedEmployeeCCCDs`, all their BCC rows skipped.

## Timeline of today's attempts (assets 731–735)

| Time (+07) | File | Errors | Cause |
|---|---|---|---|
| 18:46, 18:48, 19:00, 19:02 | `BCC_ChotLuongTuan_26_29.10.2026.xlsx` | 1 each | `vị trí "Ca đêm" không có trong cấu hình lương` (payrate config has: lái xe nâng, tổ trưởng, Chia chọn) — partner fixed the file between attempts |
| 19:52 | `...(1).xlsx` | 7 | position issue resolved; mobile-less new hires now surface |

## Impact

The 7 new hires have **no employee profile, no login account, and zero timesheets** for 26–29 Oct — only the other employees' 122 entries were created (confirmed: project 76 timesheet count in that window = 122).

## Resolution options (decision pending owner)

1. **Recommended:** collect mobile numbers for the 7, add them to the STK sheet, re-upload (import is per-row idempotent; existing employees are matched by CCCD). Or create the 7 profiles manually in the admin UI with mobiles, then re-upload.
2. Frontend improvement (small): pass through operator-friendly backend reasons instead of the generic label — the backend already writes clean Vietnamese ("Số điện thoại là bắt buộc…") into `error_detail`; a specific mapping like "Thiếu số điện thoại trong sheet STK — chế độ bắt buộc SĐT đang bật" would make future cases self-explanatory to partners.
3. Turning `employee.mobile_required` off would unblock, but it reverses the 25 Oct admin policy decision — owner's call only.

## Unresolved questions

- Who supplies the 7 mobile numbers (partner's STK sheet vs. HR).
- Whether the owner wants the import-errors pass-through fix now.
