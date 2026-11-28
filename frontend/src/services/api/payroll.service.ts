import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import { extractFilenameFromHeaders } from '@/utils/file-download';
import type {
  PaymentHistoryFilters,
  PaymentHistoryListResponse,
  BankTransferHistoryFilters,
  BankTransferHistoryListResponse,
} from '@/types/api/payroll.types';

export interface PaymentHistoryExportParams {
  fromDate: string;
  toDate: string;
}

class PayrollService {
  /**
   * The bank-transfer/payment-history envelope must carry the rows array at
   * `data` with pagination/summary beside it. Guard against shape drift (rows
   * nested under `data.rows` / `data.data`, or pagination living inside the
   * nested object) so a mismatch renders an empty ledger instead of crashing
   * the page with "Cannot read properties of undefined".
   */
  private normalizeHistoryPayload<
    T extends { data: unknown; rows?: unknown; pagination?: unknown; summary?: unknown },
  >(payload: unknown): T {
    const envelope = (payload ?? {}) as Record<string, unknown>;
    const nested =
      envelope.data && typeof envelope.data === 'object' && !Array.isArray(envelope.data)
        ? (envelope.data as Record<string, unknown>)
        : null;
    const rows: unknown[] = Array.isArray(envelope.data)
      ? envelope.data
      : Array.isArray(nested?.rows)
        ? nested.rows
        : Array.isArray(nested?.data)
          ? nested.data
          : [];

    return {
      ...envelope,
      status: typeof envelope.status === 'string' ? envelope.status : 'success',
      message: typeof envelope.message === 'string' ? envelope.message : '',
      data: rows,
      pagination: (envelope.pagination ?? nested?.pagination) as T['pagination'],
      summary: (envelope.summary ?? nested?.summary) as T['summary'],
    } as unknown as T;
  }

  async getBankTransferHistories(
    filters: BankTransferHistoryFilters,
  ): Promise<BankTransferHistoryListResponse> {
    const queryString = buildQueryString(filters);
    const response = await apiClient.get<BankTransferHistoryListResponse>(
      `${API_ENDPOINTS.payrolls.bankTransferHistories}${queryString}`,
    );
    return this.normalizeHistoryPayload<BankTransferHistoryListResponse>(response);
  }

  /**
   * Get paginated list of payment histories
   */
  async getPaymentHistories(filters?: PaymentHistoryFilters): Promise<PaymentHistoryListResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<PaymentHistoryListResponse>(
      `${API_ENDPOINTS.payrolls.histories}${queryString}`
    );
    return this.normalizeHistoryPayload<PaymentHistoryListResponse>(response);
  }

  /**
   * Export payment histories to Excel
   */
  async exportPaymentHistories(params: PaymentHistoryExportParams): Promise<void> {
    // Use the underlying axios client directly for blob response
    const response = await apiClient['client'].post(
      API_ENDPOINTS.payrolls.exportHistories,
      params,
      { responseType: 'blob' }
    );

    // Extract filename from Content-Disposition header or generate default
    const filename = extractFilenameFromHeaders(response.headers) ||
      `lich_su_thanh_toan_${params.fromDate}_${params.toDate}.xlsx`;

    // Create blob with XLSX content type
    const blob = new Blob([response.data], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    });

    // Create download link and trigger download
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();

    // Cleanup
    if (link.parentNode) {
      link.parentNode.removeChild(link);
    }
    window.URL.revokeObjectURL(downloadUrl);
  }
}

export const payrollService = new PayrollService();
