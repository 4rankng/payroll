---
phase: 4
title: "Frontend Upload UI"
status: pending
priority: P1
dependencies: [3]
---

# Phase 4: Frontend Upload UI

## Overview

Add a "Tải lên chuyển tiền" (Upload Transfer Request) entry point on the `/admin/wallet` desktop page. Clicking opens a dialog with the upload dropzone, validation, post-upload progress tracking (per-row status polling), batch history list, and KQ Excel download. Mobile gets the same dialog (responsive), optimized for click-to-browse rather than drag-and-drop.

## Requirements

- **Functional**: Upload `.xlsx` via drag-and-drop or click. Show parsing/validation errors inline. After successful upload, show a batch progress card with per-row status (`Đang xử lý` / `Thành công` / `Thất bại`), counts, total amount, total fee, and a "Tải KQ" download button. Show recent batch history below.
- **Non-functional**: Poll batch status every 5s while status is `processing`, stop polling on terminal status. Files ≤10MB (existing `FILE_LIMITS.excel.maxSize`). Vietnamese UI text throughout. Optimistic UI where possible; no blocking toasts for long ops.

## Architecture

### New components (all under `frontend/src/components/wallet/`)

```
WalletPage/index.tsx
  └── <BulkTransferUploadDialog>           (NEW trigger + dialog)
        ├── <FileDropZone>                  (existing, reused — accepts .xlsx only, see step 5)
        ├── <BulkTransferProgress>          (NEW — polls + shows per-row status)
        │     ├── SummaryStats (total, success, failed, amount, fee)
        │     ├── Progress bar (success+failed / total)
        │     └── Row status Table (STT, account, name, amount, status badge)
        └── [Tải KQ Excel] button           (always enabled; tooltip "Sẽ hiển thị kết quả hiện tại" while processing)

  └── <BulkTransferBatchList>              (NEW — recent batches, below transactions)
        └── Batch row card (filename, date, counts, totals, status badge, download button)
```

### New service: `walletBulkTransfer.service.ts`

```typescript
// frontend/src/services/api/wallet-bulk-transfer.service.ts
export const walletBulkTransferService = {
  upload(file: File, onProgress?): Promise<UploadResponse>,
  getBatch(id: number): Promise<BatchDetail>,
  listBatches(params: {page, page_size}): Promise<PaginatedBatches>,
  downloadKQ(id: number): Promise<{blob, filename}>,
};
```

Follows the exact patterns of `wallet.service.ts` and `bulk-transfer.service.ts`:
- `upload`: `FormData` with `'file'` key, `Content-Type: multipart/form-data`, `apiClient.upload<T>()` for progress events.
- `downloadKQ`: GET with `responseType: 'blob'`, extract filename from `Content-Disposition` via `extractFilenameFromHeaders`, trigger anchor download.

### New TanStack Query hooks

File: `frontend/src/hooks/api/useWalletBulkTransfer.ts`

| Hook | Type | Purpose |
|------|------|---------|
| `useUploadWalletBulkTransfer()` | `useMutation` | Upload file; on success invalidate `['wallet','bulk-transfer','batches']` |
| `useWalletBulkTransferBatch(id)` | `useQuery` | Poll single batch; `refetchInterval: (data) => data?.status === 'processing' ? 5000 : false` |
| `useWalletBulkTransferBatches(params)` | `useQuery` | Paginated history list; `refetchInterval: 30000` if any batch is `processing` |
| `useDownloadWalletBulkTransferKQ()` | `useMutation` | Trigger blob download |

### Query key additions

In `frontend/src/lib/queryKeys/index.ts`:

```typescript
wallet: {
  all: ['wallet'] as const,
  balance: () => [...QueryKeys.wallet.all, 'balance'] as const,
  bulkTransfer: {
    all: [...QueryKeys.wallet.all, 'bulk-transfer'] as const,
    batches: (params) => [...QueryKeys.wallet.bulkTransfer.all, 'batches', params] as const,
    batch: (id) => [...QueryKeys.wallet.bulkTransfer.all, 'batch', id] as const,
  },
}
```

### Trigger UI on WalletPage

Add a new button next to the existing "Chuyển tiền" and "Đồng bộ" buttons in `WalletPage/index.tsx:210-248`:

```tsx
<Button variant="default" onClick={() => setUploadDialogOpen(true)}>
  <Upload className="h-4 w-4 mr-2" />
  Tải lên chuyển tiền
</Button>
```

Followed by the dialog component. Use `lucide-react` `Upload` icon (already used elsewhere in the app).

### File validation (client-side, before upload)

Use existing `validateFile(file, options)` from `frontend/src/utils/file-upload.ts`:
- maxSize: `FILE_LIMITS.excel.maxSize` (10MB).
- allowedFormats: `['.xlsx']` — **`.xls` is excluded** because the backend parser uses `excelize.OpenReader` which only handles the `.xlsx` (OOXML) format. A `.xls` upload would parse-fail at the server.

Reject with a Vietnamese toast: `"File phải nhỏ hơn 10MB và định dạng .xlsx"`.

### Error mapping (server → UI)

| Server response | UI behavior |
|-----------------|-------------|
| 409 `duplicate_batch` | Toast error: `"File này đã được tải lên trước đó (lô #N)"` |
| 409 `duplicate_vfic` | Dialog shows conflict list: `"Các mã VFIC đã tồn tại: VFICxxx, VFICyyy"` |
| 400 `unresolved_bank` | Toast: `"Dòng N: Không nhận diện được ngân hàng '...'"` |
| 400 `invalid_amount` / `missing_vfic` | Toast: `"Dòng N: Số tiền/ mã VFIC không hợp lệ"` |
| 400 `invalid_excel_format` | Toast: `"File Excel không đúng định dạng Yêu cầu chuyển tiền"` |
| 400 `empty_file` | Toast: `"File không có giao dịch nào"` |

## Related Code Files

- **Create**: `frontend/src/components/wallet/BulkTransferUploadDialog.tsx` — main dialog.
- **Create**: `frontend/src/components/wallet/BulkTransferProgress.tsx` — progress card with row table.
- **Create**: `frontend/src/components/wallet/BulkTransferBatchList.tsx` — batch history list.
- **Create**: `frontend/src/services/api/wallet-bulk-transfer.service.ts`.
- **Create**: `frontend/src/hooks/api/useWalletBulkTransfer.ts`.
- **Create**: `frontend/src/types/wallet-bulk-transfer.ts` — TypeScript interfaces (`UploadResponse`, `BatchDetail`, `BulkTransferRow`, `BatchSummary`).
- **Modify**: `frontend/src/pages/admin/WalletPage/index.tsx` — add upload button + dialog mount + batch list section.
- **Modify**: `frontend/src/lib/queryKeys/index.ts` — add `wallet.bulkTransfer.*` keys.
- **Reuse**: `frontend/src/components/advance-payment/FileDropZone.tsx` — file dropzone (accepts `.xlsx,.xls`).
- **Reuse**: `frontend/src/utils/file-upload.ts` — `validateFile`, `createFileFormData`.
- **Reuse**: `frontend/src/utils/file-download.ts` — `triggerBlobDownload`, `extractFilenameFromHeaders`.
- **Reuse**: shadcn `Dialog`, `Table`, `Progress`, `Badge`, `Button`, `Card`.

## Implementation Steps

1. **Define TypeScript types** in `types/wallet-bulk-transfer.ts`:
   ```typescript
   export type BatchStatus = 'pending' | 'processing' | 'completed' | 'failed';
   export type RowStatus = 'pending' | 'verified' | 'authorised' | 'completed' | 'failed' | 'reversed';

   export interface UploadResponse { batch_id: number; total_count: number; transfer_amount: number; estimated_fee_total: number; }
   export interface BulkTransferRow { order_no: number; account_no: string; account_name: string; bank: string; amount: number; fee: number; status: RowStatus; ft_number: string|null; error_message: string|null; }
   export interface BatchDetail { id: number; filename: string; status: BatchStatus; total_count: number; success_count: number; failed_count: number; transfer_amount: number; total_fee: number; created_at: string; completed_at: string|null; ledger_txn_id: number|null; rows: BulkTransferRow[]; }
   export interface BatchSummary extends Omit<BatchDetail, 'rows'> {}
   ```

2. **Write the service** (`wallet-bulk-transfer.service.ts`):
   ```typescript
   upload(file, onProgress) {
     const fd = createFileFormData(file);
     return apiClient.upload<UploadResponse>('/wallet/bulk-transfer/upload', fd, onProgress);
   },
   getBatch(id) { return apiClient.get<BatchDetail>(`/wallet/bulk-transfer/batches/${id}`); },
   listBatches(params) { return apiClient.get<PaginatedBatches>('/wallet/bulk-transfer/batches', { params }); },
   downloadKQ(id) {
     return apiClient['client'].get(`/wallet/bulk-transfer/batches/${id}/kq`, { responseType: 'blob' })
       .then(res => ({ blob: res.data, filename: extractFilenameFromHeaders(res.headers) ?? `KQ_${id}.xlsx` }));
   },
   ```

3. **Write hooks** (`useWalletBulkTransfer.ts`) — follow the patterns in `useManualDisbursement.ts`:
   - `useUploadWalletBulkTransfer`: mutation, on success invalidate batches key + open the new batch in the progress dialog.
   - `useWalletBulkTransferBatch(id)`: query with `refetchInterval` callback that returns 5000 while status is `processing`, else `false`. `staleTime: 0`.
   - `useWalletBulkTransferBatches`: paginated query, refetch every 30s if any batch in the page is processing.
   - `useDownloadWalletBulkTransferKQ`: mutation calling `triggerBlobDownload(blob, filename)`.

4. **Add query keys** to `lib/queryKeys/index.ts`.

5. **Write `BulkTransferUploadDialog`**:
   - State: `phase: 'select' | 'uploading' | 'progress' | 'error'`, `selectedFile`, `batchId`.
   - Select phase: render `<FileDropZone accept=".xlsx" onFileSelected={...} />` + a "Tải lên" button + a help text linking to the template (or explaining the expected format).
   - Uploading phase: show progress bar via `onProgress` callback.
   - On upload success: transition to progress phase, set `batchId`, start polling.
   - Progress phase: render `<BulkTransferProgress batchId={batchId} />`. Show "Đóng" button (closes dialog but polling continues in background via the list query).
   - Error phase: render mapped error message + "Thử lại" button.

6. **Write `BulkTransferProgress`**:
   - Calls `useWalletBulkTransferBatch(batchId)`.
   - Top: summary stats (total_count, success_count, failed_count, transfer_amount, total_fee) in a 5-column grid.
   - Middle: `<Progress value={(success+failed)/total*100} />` with label `"Đã xử lý X/Y"`.
   - Bottom: `<Table>` with columns STT | Tài khoản | Người thụ hưởng | Số tiền | Trạng thái | FT/Lý do. Status cell uses `<Badge>` with color: completed=green, failed=red, others=yellow.
   - Footer: "Tải KQ Excel" button (always enabled; tooltip "Sẽ hiển thị kết quả hiện tại" if still processing).

7. **Write `BulkTransferBatchList`**:
   - Calls `useWalletBulkTransferBatches({page, page_size: 10})`.
   - Renders a list of cards (one per batch): filename, created_at, status badge, success/failed counts, total amount, "Tải KQ" button, "Xem chi tiết" (expands to show rows or links to a detail view).
   - Pagination controls at the bottom.

8. **Modify `WalletPage/index.tsx`**:
   - Import the new dialog and list components.
   - Add the "Tải lên chuyển tiền" button next to existing actions.
   - Add `<BulkTransferUploadDialog open={} onOpenChange={} />` near the existing dialogs (around line 272).
   - Add a new section below `<WalletTransactionsList />` (around line 268): `<BulkTransferBatchList />` with a heading "Lịch sử tải lên chuyển tiền".

9. **Run `pnpm lint && pnpm type-check`** — must pass cleanly.

## Success Criteria

- [ ] "Tải lên chuyển tiền" button visible on `/admin/wallet` next to existing actions.
- [ ] Clicking opens dialog with drag-and-drop + click-to-browse file selection.
- [ ] Files >10MB or wrong format rejected client-side with Vietnamese toast.
- [ ] Successful upload transitions dialog to progress view; per-row status updates every 5s.
- [ ] Duplicate-file upload (content hash conflict) shows the existing-batch toast.
- [ ] Duplicate-VFIC upload shows the conflict list inline.
- [ ] "Tải KQ Excel" button downloads the generated `.xlsx` with correct filename.
- [ ] Batch history list shows below the transactions list with pagination.
- [ ] All UI text is Vietnamese; matches the app's existing tone (see `WalletTransactionsList.tsx` for reference).
- [ ] `cd frontend && pnpm lint && pnpm type-check` pass.
- [ ] Mobile view (`WalletPageMobile`) renders the dialog and list without horizontal scroll.

## Risk Assessment

- **Risk**: Polling batch detail for a large batch (500 rows) every 5s causes heavy payload. **Mitigation**: Backend `GetBatch` returns full rows; for >200 rows consider adding a `?summary=true` query param that returns only counts (deferred — YAGNI for now since typical batches are 30-80 rows).
- **Risk**: Admin closes dialog mid-processing → loses track of progress. **Mitigation**: Polling continues via the batch list query (every 30s); reopening the dialog on the same batch resumes the 5s poll. Batch list always shows current status.
- **Risk**: User uploads wrong file type (CSV or `.xls` instead of `.xlsx`). **Mitigation**: Dropzone `accept=".xlsx"` filters in the file picker; server also validates and returns 400 (parser is xlsx-only via `excelize`).
- **Risk**: Dialog gets stuck on "uploading" if the network drops. **Mitigation**: Mutation has a 60s timeout (axios default); on error, transition to error phase with retry button.
