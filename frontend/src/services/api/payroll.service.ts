import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import { extractFilenameFromHeaders } from '@/utils/file-download';
import type {
  PaymentHistoryFilters,
  PaymentHistoryListResponse,
} from '@/types/api/payroll.types';

export interface PaymentHistoryExportParams {
  fromDate: string;
  toDate: string;
}

class PayrollService {
  /**
   * Get paginated list of payment histories
   */
  async getPaymentHistories(filters?: PaymentHistoryFilters): Promise<PaymentHistoryListResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<PaymentHistoryListResponse>(
      `${API_ENDPOINTS.payrolls.histories}${queryString}`
    );
    return response as unknown as PaymentHistoryListResponse;
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
