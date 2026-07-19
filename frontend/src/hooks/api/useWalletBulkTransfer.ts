/**
 * React Query hooks for Stage 2 (wallet bulk transfer upload + progress).
 *
 * Patterns:
 *  - useUploadWalletBulkTransfer: useMutation that calls the upload endpoint.
 *    On conflict/format errors, the global MutationCache.onError shows a
 *    toast; here we also invalidate the batch list.
 *  - useWalletBulkTransferBatch: useQuery with 5s polling while the batch
 *    is in a non-terminal status.
 *  - useWalletBulkTransferBatches: paginated list for the history view.
 *  - useDownloadWalletBulkTransferKQ: useMutation that triggers the KQ
 *    .xlsx download.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { walletBulkTransferService } from '@/services/api/wallet-bulk-transfer.service';
import { QueryKeys } from '@/lib/queryKeys';
import { triggerBlobDownload } from '@/utils/file-download';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';
import type { WalletBulkKQScope } from '@/types/wallet-bulk-transfer';

export function useUploadWalletBulkTransfer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ file, onProgress }: { file: File; onProgress?: (p: number) => void }) =>
      walletBulkTransferService.upload(file, onProgress),
    onSuccess: (data) => {
      showSuccessNotification(`Đã tải lên ${data.total_count} dòng — đang xử lý`);
      qc.invalidateQueries({ queryKey: QueryKeys.wallet.bulkTransfer.all() });
    },
    onError: (error) => {
      showErrorNotification(error, 'Tải lên thất bại');
    },
  });
}

/**
 * Fetches a single batch + its rows. Polls every 5s while the batch is
 * non-terminal (pending/processing/completing). Stops polling once
 * completed/failed.
 */
export function useWalletBulkTransferBatch(id: number | null) {
  return useQuery({
    queryKey: QueryKeys.wallet.bulkTransfer.detail(id ?? 0),
    queryFn: () => walletBulkTransferService.getBatch(id as number),
    enabled: id !== null,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (!data) return 5000;
      if (data.status === 'completed' || data.status === 'failed') return false;
      return 5000;
    },
  });
}

export function useWalletBulkTransferBatches(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: QueryKeys.wallet.bulkTransfer.list({ page, pageSize }),
    queryFn: () => walletBulkTransferService.listBatches(page, pageSize),
  });
}

export function useDownloadWalletBulkTransferKQ() {
  return useMutation({
    mutationFn: (input: number | { id: number; scope: WalletBulkKQScope }) => {
      const request = typeof input === 'number' ? { id: input, scope: 'all' as const } : input;
      return walletBulkTransferService.downloadKQ(request.id, request.scope);
    },
    onSuccess: async (data) => {
      triggerBlobDownload(data.blob, data.filename);
      showSuccessNotification('Đã tải file KQ');
    },
    onError: (error) => {
      showErrorNotification(error, 'Tải KQ thất bại');
    },
  });
}
