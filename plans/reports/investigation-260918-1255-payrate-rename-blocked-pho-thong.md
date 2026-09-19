# Investigation: PA BUMHAN payrate rename "phổ thông" → "750" fails to save

**Date:** 2026-09-18 · **Project:** PA BUMHAN (id 78) · **Type:** payrate config save rejected

## Summary

Saving a rename of position "phổ thông" → "750" in the active payrate config (effective 2026-09-08) fails with "Đã tồn tại mức lương với ngày hiệu lực 2026-09-08". **This is the versioning model working as designed, not a data bug** — and the position is additionally locked by 22 already-paid+approved timesheet rows.

## Why the save fails (mechanism)

1. Position names are **keys inside the config's `rates` JSON** — a rename changes the JSON.
2. The edit screen (`frontend/src/pages/admin/PayrateEditPage/index.tsx:205-233`) treats any rates-JSON change as **append-only versioning**: rates changed → **CREATE** a new config; rates unchanged → in-place UPDATE.
3. CREATE is sent with the **same** `effective_from` (2026-09-08). Backend `CreateEffectiveDatedPayrate` → `manageActivePayrates` (`payrate_temporal_service.go:483-487`) rejects a new config whose from_date equals an active config's from_date → the toast error.

So no rate/structure edit of the current config can save with the same date — the new version must carry a **later** effective date (backend floor: day after the latest paid timesheet work date; the UI pre-clamps to it).

## Timesheet lock (anh's question: "đã up file chấm công chưa?")

**Yes.** BCC file "BCC BUMHAN T09.2026 Mẫu mức lương tháng 9.xlsx" (assets 536/538) imported successfully 2026-09-17 14:42/14:49 (+0700), creating 554 rows for 2026-09.

Prod evidence (project 78):

- `phổ thông.*` timesheets: **22 rows, all `paid` + `approved`** (dates 2026-09-08 → 2026-09-14, payrate_id=84) — immutable history per the ruling "nếu có rồi thì ko đổi được".
- Payrate configs: id 81 (2026-08-21 → 2026-09-07, closed, positions 600/700/800) and id 84 (2026-09-08 → open, positions 600/700/800/**phổ thông**, created 2026-09-17 14:33 — 9 min before the BCC import).

## Remedy (matches anh's plan)

One new config version does it all — edit the matrix, rename/add "750" (remove "phổ thông" in the same pass), set effective date **later than 2026-09-08** (UI clamps to the paid floor, 2026-09-15+ since latest paid work date is 09-14):

1. Save → config 84 closes the day before the new date; new open config holds 600/700/750/800.
2. Re-upload the BCC excel with the position label corrected ("750").
3. History intact: the 22 paid `phổ thông` rows stay priced under closed config 84.

Note: mutable (unpaid/unapproved) timesheets on/after the new date with paytype "phổ thông" would fail recalculation ("loại lương không tồn tại trong cấu hình mới") — currently none exist (all 22 are paid+approved), so the rename version is safe today.

## Prevention (optional)

- The edit UI could show a hint when editing a live config's positions/rates: "thay đổi mức lương/vị trí sẽ tạo phiên bản mới — chọn ngày hiệu lực mới".
- The create-path error for same-date saves could suggest the fix directly ("chọn ngày hiệu lực khác") instead of the raw duplicate message.
