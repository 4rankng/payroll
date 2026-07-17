import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import { formatDateForAPI } from '@/utils/formatters';
import { extractFilenameFromHeaders } from '@/utils/file-download';

export interface BulkTransferExportParams {
  project_ids?: number[];
  employee_ids?: number[];
  fromDate?: string;
  toDate?: string;
  for_month?: string;
}

export interface BulkTransferResultDetail {
  row: number;
  employee_bank: string;
  employee_account_number: string;
  employee_name: string;
  employee_cccd: string;
  amount: string;
  payment_status: 'paid' | 'failed';
  paid_at: string;
}

export interface BulkTransferResultResponse {
  status: string;
  message: string;
  items: BulkTransferResultDetail[];
  total_txn: number;
  completed_txn: number;
  failed_txn: number;
}

export interface BulkTransferUploadHistory {
  id: number;
  filename: string;
  total_txn: number;
  completed_txn: number;
  failed_txn: number;
  uploaded_by: string;
  uploaded_at: string;
}

export interface BulkTransferUploadHistoriesParams {
  sortBy?: string; // created_at | uploaded_at
  sortOrder?: 'asc' | 'desc';
  fromDate?: string; // YYYY-MM-DD
  toDate?: string;   // YYYY-MM-DD
  page?: number;
  pageSize?: number; // default 20, max 100
}

export interface PaginationMeta {
  page: number;
  pageSize: number;
  totalPages: number;
  totalRecords: number;
}

export interface BulkTransferUploadHistoriesResult {
  data: BulkTransferUploadHistory[];
  pagination: PaginationMeta;
}

export interface BulkTransferHistoryDetail {
  row: number;
  employee_bank: string;
  employee_bank_code: string;
  employee_account_number: string;
  employee_name: string;
  employee_cccd: string;
  amount: string;
  payment_status: 'paid' | 'failed';
  paid_at: string;
}

export interface BulkTransferHistoryDetailResponse {
  items: BulkTransferHistoryDetail[];
  total_txn: number;
  completed_txn: number;
  failed_txn: number;
  uploaded_at?: string;
}

export interface AutoBulkTransferConfig {
  enabled: boolean;
}

export interface AutoBulkTransferResponse {
  batch_id: string;
  file_id: number;
  total_count: number;
  status: string;
}

export interface AutoBulkTransferStatusResponse {
  batch_id: string;
  total_count: number;
  completed: number;
  failed: number;
  processing: number;
  status: string;
  /**
   * Optional list of failed timesheet/payment items (only present if backend
   * exposes them — currently a TODO; consumer code must handle undefined).
   */
  failed_items?: BulkTransferFailedItem[];
}

export interface BulkTransferFailedItem {
  timesheet_id: number;
  employee_id?: number;
  employee_name?: string;
  amount: number;
  reason?: string;
  error_code?: string;
}

export interface MarkExternallyPaidRequest {
  timesheet_ids: number[];
  reference: string;
  note?: string;
}

export interface MarkExternallyPaidResponse {
  marked_count: number;
}

class BulkTransferService {
  /**
   * Download bulk transfer template file
   */
  async downloadBulkTransferTemplate(): Promise<void> {
    // Use the underlying axios client directly for blob response
    const response = await apiClient['client'].get(
      API_ENDPOINTS.payrolls.bulkTransferTemplate,
      { responseType: 'blob' }
    );

    // Extract filename from Content-Disposition header
    const filename = extractFilenameFromHeaders(response.headers) || 'bank_bulk_transfer_template.xls';

    const blob = new Blob([response.data], {
      type: 'application/vnd.ms-excel'
    });
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    // Safely remove the link
    if (link.parentNode) {
      link.parentNode.removeChild(link);
    }
    window.URL.revokeObjectURL(downloadUrl);
  }

  /**
   * Export bulk transfer XLSX file (MBank format)
   */
  async exportBulkTransfer(params: BulkTransferExportParams): Promise<void> {
    // Validation: cannot have both for_month and date range
    if (params.for_month && (params.fromDate || params.toDate)) {
      throw new Error('Cannot specify both for_month and date range parameters');
    }

    // Validation: weekly cycle requires both fromDate and toDate
    if (!params.for_month && (!params.fromDate || !params.toDate)) {
      throw new Error('Weekly export requires both fromDate and toDate');
    }

    // Prepare request parameters
    const requestParams = {
      ...params,
      // Only format dates if they exist (for weekly cycle)
      ...(params.fromDate && { fromDate: formatDateForAPI(params.fromDate) }),
      ...(params.toDate && { toDate: formatDateForAPI(params.toDate) }),
    };

    // Use the underlying axios client directly for blob response
    const response = await apiClient['client'].post(
      API_ENDPOINTS.payrolls.exportBulkTransfer,
      requestParams,
      { responseType: 'blob' }
    );

    // Extract filename from Content-Disposition header
    const filename = extractFilenameFromHeaders(response.headers);

    if (!filename) {
      throw new Error(
        'Cannot access filename from server response. ' +
        'Backend needs to add "Access-Control-Expose-Headers: Content-Disposition" to expose the header to the frontend.'
      );
    }

    const responseContentType = response.headers['content-type'];
    const contentType = typeof responseContentType === 'string' && responseContentType.trim()
      ? responseContentType
      : 'application/octet-stream';

    // Preserve the server-provided media type so both XLSX and ZIP exports download correctly.
    const blob = new Blob([response.data], {
      type: contentType
    });
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    // Safely remove the link
    if (link.parentNode) {
      link.parentNode.removeChild(link);
    }
    window.URL.revokeObjectURL(downloadUrl);
  }

  /**
   * Import bulk transfer result from Excel file
   */
  async importBulkTransferResult(file: File): Promise<BulkTransferResultResponse> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await apiClient.post<{
      items: BulkTransferResultDetail[];
      total_txn: number;
      completed_txn: number;
      failed_txn: number;
    }>(
      API_ENDPOINTS.payrolls.importBulkTransferResult,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    );

    return {
      status: response.status,
      message: response.message || '',
      items: response.data?.items || [],
      total_txn: response.data?.total_txn || 0,
      completed_txn: response.data?.completed_txn || 0,
      failed_txn: response.data?.failed_txn || 0,
    };
  }

  /**
   * Get list of bulk transfer upload histories
   */
  async getBulkTransferUploadHistories(
    params?: BulkTransferUploadHistoriesParams
  ): Promise<BulkTransferUploadHistoriesResult> {
    const queryString = params
      ? buildQueryString({
          sortBy: params.sortBy ?? 'uploaded_at',
          sortOrder: params.sortOrder ?? 'desc',
          fromDate: params.fromDate,
          toDate: params.toDate,
          page: params.page,
          pageSize: params.pageSize,
        })
      : '';

    const response = await apiClient.get<BulkTransferUploadHistory[]>(
      `${API_ENDPOINTS.payrolls.bulkTransferUploadHistories}${queryString}`
    );

    const list = response.data ?? [];
    const pagination = response.pagination ?? {
      page: params?.page ?? 1,
      pageSize: params?.pageSize ?? 20,
      totalPages: 1,
      totalRecords: list.length,
    };

    return {
      data: list,
      pagination,
    };
  }

  /**
   * Get detailed results of a specific bulk transfer upload
   */
  async getBulkTransferUploadHistoryById(id: number): Promise<BulkTransferHistoryDetailResponse> {
    // API response shape: { status, data: { completed_txn, failed_txn, total_txn, items: [...] } }
    type HistoryPayload = {
      items?: BulkTransferHistoryDetail[];
      total_txn?: number;
      completed_txn?: number;
      failed_txn?: number;
      uploaded_at?: string;
    };

    const response = await apiClient.get<HistoryPayload>(
      API_ENDPOINTS.payrolls.bulkTransferUploadHistoryById(id)
    );

    // apiClient.get returns ApiResponse<T>, so actual payload is response.data
    const payload = response.data ?? ({} as HistoryPayload);

    return {
      items: Array.isArray(payload.items) ? payload.items : [],
      total_txn: payload.total_txn ?? 0,
      completed_txn: payload.completed_txn ?? 0,
      failed_txn: payload.failed_txn ?? 0,
      uploaded_at: payload.uploaded_at,
    };
  }

  /**
   * Export PDF for a specific bulk transfer upload history
   * API: POST /api/v1/payrolls/bulk-transfer-upload-histories/:id/export-pdf
   * Response: application/pdf (binary)
   */
  async downloadBulkTransferHistoryPDF(id: number): Promise<void> {
    const response = await apiClient['client'].post(
      API_ENDPOINTS.payrolls.bulkTransferUploadHistoryExportPdf(id),
      {},
      { responseType: 'blob' }
    );

    // Prefer filename from headers; fallback to chuyen_tien_YYYY-MM-DD.pdf
    const headerFilename = extractFilenameFromHeaders(response.headers);
    const today = formatDateForAPI(new Date());
    const filename = headerFilename || `chuyen_tien_${today}.pdf`;

    const blob = new Blob([response.data], { type: 'application/pdf' });
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    if (link.parentNode) {
      link.parentNode.removeChild(link);
    }
    window.URL.revokeObjectURL(downloadUrl);
  }

  /**
   * Download the original uploaded Excel file for a bulk transfer history
   * API: GET /api/v1/payrolls/bulk-transfer-upload-histories/:id/download
   * Response: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet (binary)
   */
  async downloadBulkTransferHistoryExcel(id: number): Promise<void> {
    const response = await apiClient['client'].get(
      API_ENDPOINTS.payrolls.bulkTransferUploadHistoryDownload(id),
      { responseType: 'blob' }
    );

    // Extract filename from Content-Disposition header
    const filename = extractFilenameFromHeaders(response.headers);

    if (!filename) {
      throw new Error(
        'Cannot access filename from server response. ' +
        'Backend needs to add "Access-Control-Expose-Headers: Content-Disposition" to expose the header to the frontend.'
      );
    }

    // Create blob with XLSX content type
    const blob = new Blob([response.data], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    });
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    if (link.parentNode) {
      link.parentNode.removeChild(link);
    }
    window.URL.revokeObjectURL(downloadUrl);
  }

  async getAutoBulkTransferConfig(): Promise<AutoBulkTransferConfig> {
    const response = await apiClient.get<AutoBulkTransferConfig>(
      API_ENDPOINTS.payrolls.autoBulkTransferConfig
    );
    return response.data ?? { enabled: false };
  }

  async initiateAutoBulkTransfer(params: BulkTransferExportParams): Promise<AutoBulkTransferResponse> {
    const requestParams = {
      ...params,
      ...(params.fromDate && { fromDate: formatDateForAPI(params.fromDate) }),
      ...(params.toDate && { toDate: formatDateForAPI(params.toDate) }),
    };
    const response = await apiClient.post<AutoBulkTransferResponse>(
      API_ENDPOINTS.payrolls.initiateAutoBulkTransfer,
      requestParams
    );
    return response.data!;
  }

  async getAutoBulkTransferStatus(batchId: string): Promise<AutoBulkTransferStatusResponse> {
    const response = await apiClient.get<AutoBulkTransferStatusResponse>(
      API_ENDPOINTS.payrolls.autoBulkTransferStatus(batchId)
    );
    return response.data!;
  }

  /**
   * Mark a list of timesheets as paid externally (e.g. after a provider batch
   * partially fails and admin pays those rows out-of-band).
   *
   * Backend TODO: endpoint `POST /payrolls/bulk-transfer-external-mark`
   * is not implemented yet. The frontend wires this method end-to-end so
   * the UI is ready as soon as the server side ships.
   */
  async markAsPaidExternally(payload: MarkExternallyPaidRequest): Promise<MarkExternallyPaidResponse> {
    const response = await apiClient.post<MarkExternallyPaidResponse>(
      API_ENDPOINTS.payrolls.markBulkTransferExternallyPaid,
      payload
    );
    return response.data ?? { marked_count: payload.timesheet_ids.length };
  }
}

export const bulkTransferService = new BulkTransferService();
