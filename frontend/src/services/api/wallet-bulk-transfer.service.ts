/**
 * Wallet Bulk Transfer service — Stage 2 client.
 *
 * Wraps the four /wallet/bulk-transfer/* endpoints:
 *   POST   /upload                → parse + enqueue per-row tasks
 *   GET    /batches               → paginated history
 *   GET    /batches/:id           → batch + rows (progress view)
 *   GET    /batches/:id/kq        → result .xlsx download
 *
 * Mirrors the existing bulk-transfer.service.ts patterns (class-based
 * singleton, apiClient.upload for multipart, apiClient.downloadBlob for
 * KQ, custom error extraction).
 */
import { apiClient } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import {
  WalletBulkBatchDetail,
  WalletBulkBatchListResponse,
  WalletBulkUploadError,
  WalletBulkUploadResponse,
} from '@/types/wallet-bulk-transfer';

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024; // 10 MiB — must match backend MaxBulkUploadBytes

class WalletBulkTransferService {
  /**
   * Upload a "Yêu cầu chuyển tiền" .xlsx. Returns the batch_id + counts on
   * success (HTTP 202). Throws a typed error on validation/conflict failures
   * so the caller can map to a helpful Vietnamese message.
   */
  async upload(
    file: File,
    onProgress?: (percent: number) => void,
  ): Promise<WalletBulkUploadResponse> {
    if (file.size > MAX_UPLOAD_BYTES) {
      throw { error: 'file_too_large', max_bytes: MAX_UPLOAD_BYTES } as WalletBulkUploadError;
    }
    const formData = new FormData();
    formData.append('file', file);
    // apiClient.upload returns ApiResponse<T> on 2xx; on 4xx it throws
    // the normalized error body which we surface as WalletBulkUploadError.
    const res = await apiClient.upload<WalletBulkUploadResponse>(
      API_ENDPOINTS.walletBulkTransfer.upload,
      formData,
      onProgress,
    );
    return res.data as WalletBulkUploadResponse;
  }

  /** GET /batches/:id — full batch + rows for the progress view. */
  async getBatch(id: number): Promise<WalletBulkBatchDetail> {
    const res = await apiClient.get<WalletBulkBatchDetail>(
      API_ENDPOINTS.walletBulkTransfer.batchById(id),
    );
    return res.data as WalletBulkBatchDetail;
  }

  /** GET /batches — paginated list for the history view. */
  async listBatches(page = 1, pageSize = 20): Promise<WalletBulkBatchListResponse> {
    const res = await apiClient.get<WalletBulkBatchListResponse>(
      `${API_ENDPOINTS.walletBulkTransfer.batches}?page=${page}&page_size=${pageSize}`,
    );
    return res.data as WalletBulkBatchListResponse;
  }

  /**
   * GET /batches/:id/kq — download the result .xlsx. Uses apiClient.downloadBlob
   * (the public method on ApiClient that handles blob responses correctly).
   * Returns { blob, filename } — caller triggers the download.
   */
  async downloadKQ(id: number): Promise<{ blob: Blob; filename: string }> {
    const blob = await apiClient.downloadBlob(API_ENDPOINTS.walletBulkTransfer.batchKQ(id));
    return {
      blob,
      filename: `KQ_Chuyen_Tien_${id}.xlsx`,
    };
  }
}

export const walletBulkTransferService = new WalletBulkTransferService();
