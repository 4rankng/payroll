---
phase: 5
title: "Frontend UI (Timesheet + Wallet)"
status: pending
priority: P1
dependencies: [2, 4]
---

# Phase 5: Frontend UI (Timesheet + Wallet)

## Overview

Two UI additions spanning both stages of the workflow:
1. **Timesheet page** (`/admin/timesheet`): new "Chuyển OnePay" dropdown item that triggers the export (Phase 2).
2. **Wallet page** (`/admin/wallet`): upload dialog for the OnePay file, progress polling, KQ download, batch history (consumes Phase 3+4 APIs).

## Requirements

- **Functional**:
  - Timesheet: "Chuyển OnePay" dropdown item triggers `POST /payrolls/export-onepay-bulk` with current filters → downloads `.xlsx` + shows skipped-employees warning if any.
  - Wallet: upload `.xlsx` via drag/drop, show parsing/validation errors inline, show batch progress card with per-row status (polling 5s), KQ download button, batch history list.
- **Non-functional**: Files ≤10MB. Vietnamese UI. KQ download via `apiClient.downloadBlob` (NOT private-field access).

## Architecture

### Stage 1 — Timesheet page (`/admin/timesheet`)

Add "Chuyển OnePay" item to the existing dropdown next to "Chuyển lô" and "Xuất bảng công":

```
TimesheetPage/index.tsx (existing)
  └── existing DropdownMenu
       ├── existing: "Chuyển lô" (handleChuyenLo)
       ├── existing: "Tải lên kết quả BCC" (handleBulkTransferResultUpload)
       ├── existing: "Xuất bảng công" (handleApprovedTimesheetsExport)
       └── NEW: "Chuyển OnePay" (handleChuyenOnePay)
              ↓
            useExportOnePayBulk() mutation
              ↓
            POST /payrolls/export-onepay-bulk
              ↓
            Blob download + skipped-employees toast
```

**Skipped-employees UX** (Validation Decision V5): read `X-Skipped-Count` header. If >0, show a warning toast: `"Đã bỏ qua N nhân viên thiếu thông tin ngân hàng. Vui lòng kiểm tra và xuất lại sau khi cập nhật."` with a "Xem danh sách" button that opens a `SkippedEmployeesDialog` listing them with names + reasons (`missing_bank_info`, `unresolved_swift`). The dialog is dismissible; the file still downloads successfully.

### Stage 2 — Wallet page (`/admin/wallet`)

```
WalletPage/index.tsx
  └── <BulkTransferUploadDialog>
        ├── <FileDropZone accept=".xlsx">
        ├── <BulkTransferProgress>
        │     ├── SummaryStats
        │     ├── Progress bar
        │     └── Row status Table
        └── [Tải KQ Excel] button (always enabled)
  └── <BulkTransferBatchList>
```

### Services

```typescript
// frontend/src/services/api/onepay-export.service.ts (NEW)
export const onepayExportService = {
  async exportBulk(params: ExportBulkTransferRequest): Promise<{ blob: Blob; filename: string; skippedCount: number }> {
    const res = await apiClient['client'].post('/payrolls/export-onepay-bulk', params, {
      responseType: 'blob',
    });
    const filename = extractFilenameFromHeaders(res.headers) ?? `Yeu_cau_chuyen_tien.xlsx`;
    const skippedCount = parseInt(res.headers['x-skipped-count'] ?? '0', 10);
    return { blob: res.data, filename, skippedCount };
  },
};

// frontend/src/services/api/wallet-bulk-transfer.service.ts (Stage 2 — from Phase 4)
export const walletBulkTransferService = {
  upload(file, onProgress) { /* apiClient.upload */ },
  getBatch(id, summary) { /* apiClient.get */ },
  listBatches(params) { /* apiClient.get */ },
  async downloadKQ(id) {
    const blob = await apiClient.downloadBlob(`/wallet/bulk-transfer/batches/${id}/kq`);
    return { blob, filename: `KQ_Chuyen_Tien_${id}.xlsx` };
  },
};
```

Note: `onepayExportService` uses `apiClient['client'].post` directly because the response is a blob with custom headers (the existing `apiClient.downloadPost` doesn't expose response headers). This is the same pattern used by `bulkTransferService.exportBulkTransfer` (`bulk-transfer.service.ts:166`). Document this exception to the "no private field access" rule (red-team v1 A10 doesn't apply when we need response headers).

### Hooks

```typescript
// frontend/src/hooks/api/useOnePayExport.ts (NEW)
export function useExportOnePayBulk() {
  return useMutation({
    mutationFn: onepayExportService.exportBulk,
    onSuccess: async (data) => {
      triggerBlobDownload(data.blob, data.filename);
      if (data.skippedCount > 0) {
        toast.warning(`Đã bỏ qua ${data.skippedCount} nhân viên thiếu thông tin ngân hàng`);
      } else {
        toast.success(`Đã xuất ${data.filename}`);
      }
    },
    onError: () => toast.error('Xuất file OnePay thất bại'),
  });
}

// frontend/src/hooks/api/useWalletBulkTransfer.ts (Stage 2 — from Phase 4)
// useUploadWalletBulkTransfer, useWalletBulkTransferBatch, useWalletBulkTransferBatches,
// useDownloadWalletBulkTransferKQ
```

### File validation (Stage 2)

`validateFile(file, options)` from `utils/file-upload.ts`:
- maxSize: 10MB.
- allowedFormats: `['.xlsx']`.

Server also enforces 10MB + ZIP magic + multipart cap.

### Error mapping (Stage 2 → UI)

| Server response | UI |
|-----------------|----|
| 409 `duplicate_batch` | Toast `"File này đã được tải lên trước đó (lô #N)"` |
| 409 `duplicate_vfic` | Inline conflict list |
| 400 `missing_swift_column` | Toast `"File thiếu cột SWIFT. Vui lòng dùng nút 'Chuyển OnePay' trên trang Bảng công để xuất file."` |
| 400 `invalid_swift` | Toast `"Dòng N: Mã SWIFT không hợp lệ"` |
| 400 `invalid_row` | Toast `"Dòng N: <detail>"` |
| 400 `invalid_excel_format` | Toast `"File không đúng định dạng"` |
| 400 `invalid_file_type` | Toast `"File phải là .xlsx hợp lệ"` |
| 400 `file_too_large` | Toast `"File vượt quá 10MB"` |
| 400 `empty_file` | Toast `"File không có giao dịch nào"` |
| 400 `too_many_rows` | Toast `"File vượt quá 5,000 dòng"` |

## Related Code Files

### Stage 1 — Timesheet

- **Create**: `frontend/src/services/api/onepay-export.service.ts`.
- **Create**: `frontend/src/hooks/api/useOnePayExport.ts`.
- **Modify**: `frontend/src/pages/admin/TimesheetPage/index.tsx` — add "Chuyển OnePay" dropdown item + handler.
- **Create** (optional): `frontend/src/components/payroll/SkippedEmployeesDialog.tsx` — shows list of skipped employees after export.
- **Reuse**: existing dropdown menu structure + filter state in TimesheetPage.
- **Reuse**: `triggerBlobDownload`, `extractFilenameFromHeaders` from `utils/file-download.ts`.

### Stage 2 — Wallet

- **Create**: `frontend/src/components/wallet/BulkTransferUploadDialog.tsx`.
- **Create**: `frontend/src/components/wallet/BulkTransferProgress.tsx`.
- **Create**: `frontend/src/components/wallet/BulkTransferBatchList.tsx`.
- **Create**: `frontend/src/services/api/wallet-bulk-transfer.service.ts`.
- **Create**: `frontend/src/hooks/api/useWalletBulkTransfer.ts`.
- **Create**: `frontend/src/types/wallet-bulk-transfer.ts`.
- **Modify**: `frontend/src/pages/admin/WalletPage/index.tsx` — add upload button + dialog + batch list.
- **Modify**: `frontend/src/lib/queryKeys/index.ts` — add `wallet.bulkTransfer.*` keys.
- **Reuse**: `FileDropZone`, `validateFile`, `createFileFormData`, `triggerBlobDownload`.
- **Reuse**: `apiClient.downloadBlob` at `client.ts:306-311`.
- **Reuse**: shadcn `Dialog`, `Table`, `Progress`, `Badge`, `Button`, `Card`, `DropdownMenu`.

## Implementation Steps

### Stage 1 — Timesheet

1. **Write `onepayExportService`** per pseudocode above.

2. **Write `useExportOnePayBulk` hook**.

3. **Modify `TimesheetPage/index.tsx`**:
   - Import `useExportOnePayBulk`.
   - Add handler `handleChuyenOnePay` that calls the mutation with current filters (cycle, project, period — same shape as `handleChuyenLo` uses).
   - Add new `<DropdownMenuItem onClick={handleChuyenOnePay} disabled={exportOnePayMutation.isPending}>` between "Chuyển lô" and "Xuất bảng công".
   - Label: `"Chuyển OnePay"` with `Banknote` icon (lucide-react).

4. **(Optional) Write `SkippedEmployeesDialog`** — opens when `skippedCount > 0`, shows list with employee names + reasons.

5. **Verify**: existing "Chuyển lô" and "Xuất bảng công" still work.

### Stage 2 — Wallet

6. **Define TypeScript types** in `types/wallet-bulk-transfer.ts` (BatchStatus, RowStatus, UploadResponse, BulkTransferRow, BatchDetail, BatchSummary).

7. **Write `walletBulkTransferService`** per Phase 4 pseudocode.

8. **Write hooks** (`useWalletBulkTransfer.ts`).

9. **Add query keys** to `lib/queryKeys/index.ts`.

10. **Write `BulkTransferUploadDialog`** (phases: select / uploading / progress / error).

11. **Write `BulkTransferProgress`**: uses `<Badge>` with color mapping. For completed rows with `ft_pending=true`, show "Đang chờ FT" tooltip.

12. **Write `BulkTransferBatchList`**: paginated cards.

13. **Modify `WalletPage/index.tsx`**: add upload button + dialog + batch list section.

14. **Run `pnpm lint && pnpm type-check`**.

## Success Criteria

### Stage 1 — Timesheet
- [ ] "Chuyển OnePay" dropdown item visible on `/admin/timesheet` between "Chuyển lô" and "Xuất bảng công".
- [ ] Clicking it triggers blob download of `Yeu_cau_chuyen_tien_<cycle>_<timestamp>.xlsx`.
- [ ] Downloaded file has 7 columns including "Mã SWIFT".
- [ ] If employees are skipped (missing bank info), warning toast shows count.
- [ ] Existing "Chuyên lô" and "Xuất bảng công" continue to work.

### Stage 2 — Wallet
- [ ] "Tải lên chuyển tiền" button visible on `/admin/wallet`.
- [ ] Files >10MB or wrong format rejected client-side.
- [ ] Progress view updates every 5s.
- [ ] Duplicate-file shows existing-batch toast.
- [ ] Duplicate-VFIC shows conflict list inline.
- [ ] Missing-SWIFT-column error shows helpful Vietnamese message pointing to the timesheet export button.
- [ ] KQ download uses `apiClient.downloadBlob`.
- [ ] FT column shows "Đang chờ FT" tooltip for completed rows still pending FT.
- [ ] Batch history list renders below transactions with pagination.
- [ ] All UI text Vietnamese.
- [ ] `pnpm lint && pnpm type-check` pass.
- [ ] Mobile renders without horizontal scroll.

## Risk Assessment

- **Risk**: Export response headers not accessible via standard apiClient methods. **Mitigation**: Use `apiClient['client'].post` with `responseType: 'blob'` (same pattern as `bulkTransferService.exportBulkTransfer`); document the exception to red-team v1 A10.
- **Risk**: Polling heavy payloads on 500-row batches. **Mitigation**: `summary=true` query for >200 rows.
- **Risk**: Dialog closed mid-processing. **Mitigation**: Batch list query continues.
- **Risk**: `.xls` upload. **Mitigation**: Dropzone `accept=".xlsx"` + server ZIP magic.
- **Risk**: Network drop mid-upload. **Mitigation**: 60s timeout; retry button.
- **Risk**: User uploads a non-OnePay-export file (e.g., the 9Pay bulk-transfer template). **Mitigation**: Parser checks for SWIFT column; missing → `missing_swift_column` error with helpful message.
