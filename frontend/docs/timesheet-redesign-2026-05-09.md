# Bảng công Admin Redesign — 2026-05-09

Tổng quan thiết kế lại trang `/admin/timesheet` và workflow Chuyển lô (bulk transfer).

## 1. Header / Primary Actions

### Trước

```
[TimesheetMonthSelector] [Duyệt tất cả] [⋮]
                                          ├─ Nhập công
                                          ├─ Xuất chuyển lô
                                          ├─ CK bằng 9Pay
                                          ├─ Nhập KQ chuyển lô
                                          ├─ Xuất bảng công
                                          └─ Lịch sử chuyển lô
```

### Sau

```
[TimesheetMonthSelector] [Duyệt hết] [Chuyển lô] [⋮]
                                                  ├─ Nhập công
                                                  ├─ Nhập KQ chuyển lô
                                                  ├─ ─────────────────
                                                  ├─ Xuất bảng công
                                                  └─ Lịch sử chuyển lô
```

- **"Duyệt tất cả" → "Duyệt hết"** (gọn hơn, cùng style segmented).
- **"Chuyển lô"** primary button mới — mở dialog wizard 2 bước (chọn phương thức → fill form).
- "Xuất chuyển lô" / "CK bằng 9Pay" rời menu (hợp nhất vào dialog "Chuyển lô" mới).
- Overflow menu group lại: nhóm 1 = nhập (Nhập công, Nhập KQ), nhóm 2 = báo cáo (Xuất bảng công, Lịch sử chuyển lô) — phân tách bằng `DropdownMenuSeparator`.

## 2. "Duyệt hết" Confirmation Dialog

API `POST /timesheets/approve-all` duyệt **toàn hệ thống**, không nhận filter. Dialog show context honest:

```
Duyệt N bảng công đang chờ duyệt
─────────────────────────────────
• Tổng NV liên quan:   23
• Tổng tiền:           12.345.678đ
• Phạm vi:             Toàn hệ thống (không theo bộ lọc hiện tại)

Hành động này không thể hoàn tác.
                              [Hủy] [Duyệt N bảng công]
```

Số liệu lấy từ `useTimesheetSummary({})` (no filter) để khớp với scope thật của API.

## 3. Dialog Chuyển lô (mới)

### Step 1 — Chọn phương thức

2 cards radio-style:

```
┌─────────────────────────────┐  ┌─────────────────────────────┐
│  📑 Thủ công                │  │  💳 9Pay                    │
│                             │  │                             │
│  Xuất file Excel để CK      │  │  Tự động chuyển qua ví 9Pay,│
│  qua app ngân hàng,         │  │  kết quả realtime           │
│  upload kết quả sau         │  │                             │
│                             │  │  Số dư: 25.000.000đ         │
│                             │  │  Phí: ~0.5% ước tính        │
└─────────────────────────────┘  └─────────────────────────────┘
```

- 9Pay disabled nếu wallet balance < dự kiến tổng tiền + warning text.
- 9Pay disabled nếu config `enabled === false` + warning "9Pay chưa được kích hoạt".

### Step 2A — Manual flow

#### Stage A: Xuất file (đã có form export hiện tại)

- Reuse `BulkTransferDateRangeSection` + `BulkTransferFiltersSection`.
- Submit → download Excel → success state với 2 option:
  - "Đóng" (sẽ upload kết quả sau qua menu "Nhập KQ chuyển lô")
  - "Upload kết quả ngay" → switch sang Stage B inline.
- Save tới localStorage `pendingBulkTransferExports`: `{ exportedAt, fromDate, toDate, filename, employeeIds }` để hint reminder.

#### Stage B: Upload kết quả

- Re-entry: menu "Nhập KQ chuyển lô" hoặc inline từ Stage A success.
- Hint banner ở top dialog: "Bạn vừa export N giây trước, file: …" nếu localStorage có entry < 7 ngày tuổi.
- Reuse component `BulkTransferResultUploadDialog` đã có (với upgraded hint logic).

### Step 2B — 9Pay flow

#### Confirm screen (trước khi gọi API)

```
Xác nhận chuyển lô qua 9Pay
─────────────────────────────────
Số nhân viên:        23
Tổng tiền:           12.345.678đ
Phí 9Pay:            ~0.5% (auto trừ vào ví)
Số dư hiện tại:      25.000.000đ
Số dư sau:           12.654.322đ (ước tính)

[Hủy]  [Xác nhận chuyển 23 giao dịch]
```

#### Processing / completion

Reuse existing 9Pay progress view — show summary khi xong:
- N/M thành công, K thất bại
- Failed rows: lý do (insufficient balance, invalid bank account, error_code)
- Action button **"Trả ngoài app"** trên failed rows: mở sub-dialog cho user nhập payment reference (ngân hàng, số GD), bấm submit → backend mark các failed timesheet rows = paid với note "Trả ngoài app: {reference}".
  - **Backend endpoint TODO**: `POST /payrolls/bulk-transfer-external-mark` `{ timesheet_ids: number[], reference: string, note?: string }` → set `payment_status = 'paid'`, log audit.

## 4. Bulk Select trên Table

- Checkbox column added at the leftmost (before status strip).
- Per-employee group level: chọn 1 row group = chọn tất cả entries của nhóm đó.
- "Chọn tất cả" trên header row select tất cả groups visible.

### Sticky Action Bar

Khi count > 0, slide-down từ trên (sticky bên dưới page header):

```
┌────────────────────────────────────────────────────────────────┐
│  Đã chọn 5 nhân viên     [Duyệt] [Chuyển lô (5)] [Bỏ chọn]   │
└────────────────────────────────────────────────────────────────┘
```

- "Duyệt": gọi `bulkApprove(timesheet_ids[])` từ entries của các nhóm đã chọn.
- "Chuyển lô (5)": mở Chuyển lô dialog với pre-filled `employee_ids` của các nhóm đã chọn.

## 5. Row Click Navigation

- **Body row click → navigate** `/admin/timesheet/employees/:employeeId?period=YYYY-MM`.
- **Chevron click only → expand/collapse** group.
- Stop event propagation rõ ràng giữa 2 zones.

### Employee Detail Page (scaffold)

`/admin/timesheet/employees/:employeeId`:
- Placeholder page: Back button + "Đang phát triển" message.
- TODO comment in code để track implementation.

## 6. Polling for 9Pay Auto-update

- Hook `useAutoBulkTransferStatus` đã có `refetchInterval: 3000` khi chưa completed.
- Khi batch active, additionally invalidate `['timesheets']` cache mỗi tick để table reflect status changes.
- Stop khi `status.status === 'completed'`.

## 7. Empty States (3 cases)

### a) Không có timesheet kỳ này
```
[icon: Calendar+]
Chưa có dữ liệu chấm công cho kỳ này
[Nhập công]
```

### b) Tất cả đã duyệt
```
[icon: CheckCircle2 green]
Tất cả N bảng công đã được duyệt!
```

### c) Filter trả về rỗng
```
[icon: Search]
Không tìm thấy kết quả phù hợp với bộ lọc
[Xóa bộ lọc]
```

Logic: dùng `summary.totalEntries` để phân biệt case (a) vs (c). Case (b) khi `summary.pendingApproval === 0 && summary.totalEntries > 0`.

## 8. Files Touched

### Modified
- `src/pages/admin/TimesheetPage/index.tsx` — header buttons, new dialog wiring, navigation handler.
- `src/components/timesheet/TimesheetListTable.tsx` — bulk select column, sticky action bar, chevron-only expand.
- `src/components/timesheet/TimesheetSummaryCards.tsx` — empty state hooks.

### New
- `src/components/timesheet/ChuyenLoDialog/index.tsx` — main wrapper.
- `src/components/timesheet/ChuyenLoDialog/MethodPickerStep.tsx` — Step 1.
- `src/components/timesheet/ChuyenLoDialog/ManualExportStep.tsx` — Stage A wrapper.
- `src/components/timesheet/ChuyenLoDialog/NinePayConfirmStep.tsx` — confirm + balance check.
- `src/components/timesheet/ChuyenLoDialog/NinePayResultStep.tsx` — completion summary + failed rows + Trả ngoài app trigger.
- `src/components/timesheet/MarkExternallyPaidDialog.tsx` — sub-dialog cho "Trả ngoài app".
- `src/components/timesheet/TimesheetBulkActionBar.tsx` — sticky action bar.
- `src/components/timesheet/TimesheetEmptyState.tsx` — 3 empty state variants.
- `src/pages/admin/EmployeeTimesheetDetailPage/index.tsx` — placeholder page.

## 9. Backend TODOs (out of scope, flag for future)

1. **Selective approval**: `approveAll` should accept filter (project_ids, employee_ids, date range) so confirm dialog có thể show filtered scope chính xác.
2. **External mark-paid**: `POST /payrolls/bulk-transfer-external-mark` — body `{ timesheet_ids, reference, note }`, used by "Trả ngoài app" flow.
3. **9Pay batch retry endpoint**: optional; alternatively, the current "re-trigger from filter" is OK since failed rows revert to `pending_payment`.
4. **Wallet fee preview**: API to estimate 9Pay fee before initiation (currently shown as % estimate).
5. **Persisted bulk_transfers state**: track exports pending upload so re-entry not just localStorage hint.

## 10. Conventions

- All commits conventional commits format with `feat(timesheet)` / `refactor(timesheet)` / `fix(timesheet)` prefix.
- Keep existing dialogs (`BulkTransferExportDialog`, `BulkTransferResultUploadDialog`) functional during transition — new dialog replaces invocation site, not the components.
