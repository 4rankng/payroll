/**
 * OnePay export service — Stage 1 of the Wallet Bulk Transfer Pipeline.
 *
 * Triggers POST /payrolls/export-onepay-bulk and returns the .xlsx blob
 * plus metadata parsed from response headers (X-Total-Count, X-Skipped-Count,
 * X-Transfer-Amount). The handler streams the bytes as a blob with custom
 * X- headers; we use the underlying axios client directly (not apiClient.post)
 * so we can read those headers — same pattern as bulkTransferService.exportBulkTransfer.
 *
 *越南文 UI strings live in the consuming hook, not here.
 */
import { apiClient } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import { BulkTransferExportParams } from './bulk-transfer.service';

export interface OnePayExportResult {
  blob: Blob;
  filename: string;
  totalCount: number;
  transferAmount: number;
  skippedCount: number;
  cycle: string;
}

export const onepayExportService = {
  /**
   * Export approved timesheets to a OnePay-compatible .xlsx.
   * Returns the blob + parsed metadata. The caller is responsible for
   * triggering the download (triggerBlobDownload) and surfacing the toast.
   */
  async exportBulk(params: BulkTransferExportParams): Promise<OnePayExportResult> {
    // Reuse the same param-shape validation as the 9Pay export.
    if (params.for_month && (params.fromDate || params.toDate)) {
      throw new Error('Cannot specify both for_month and date range parameters');
    }
    if (!params.for_month && (!params.fromDate || !params.toDate)) {
      throw new Error('Weekly export requires both fromDate and toDate');
    }

    // Use the raw axios client so we can read response headers (apiClient.post
    // normalizes to ApiResponse<T> and drops them). This mirrors
    // bulk-transfer.service.ts:166.
    const response = await apiClient['client'].post(
      API_ENDPOINTS.payrolls.exportOnePayBulk,
      params,
      { responseType: 'blob' },
    );

    const filename =
      extractFilenameFromContentDisposition(response.headers?.['content-disposition']) ??
      `Yeu_cau_chuyen_tien.xlsx`;

    const contentType =
      typeof response.headers?.['content-type'] === 'string' && response.headers['content-type'].trim()
        ? response.headers['content-type']
        : 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';

    const blob = new Blob([response.data], { type: contentType });

    return {
      blob,
      filename,
      totalCount: parseInt(response.headers?.['x-total-count'] ?? '0', 10) || 0,
      transferAmount: parseInt(response.headers?.['x-transfer-amount'] ?? '0', 10) || 0,
      skippedCount: parseInt(response.headers?.['x-skipped-count'] ?? '0', 10) || 0,
      cycle: (response.headers?.['x-cycle'] as string) ?? '',
    };
  },
};

/**
 * Parses a Content-Disposition header for the filename. Supports both
 * RFC 2183 (filename="...") and RFC 5987 (filename*=UTF-8''...) forms.
 * Returns null when no filename is present.
 */
function extractFilenameFromContentDisposition(header: unknown): string | null {
  if (typeof header !== 'string' || !header) return null;
  // RFC 5987 form first (handles UTF-8 filenames).
  const starMatch = header.match(/filename\*=([^;]+)/i);
  if (starMatch) {
    const raw = starMatch[1].trim();
    const parts = raw.split("'");
    if (parts.length === 3) {
      try {
        return decodeURIComponent(parts[2]);
      } catch {
        return parts[2];
      }
    }
    return raw;
  }
  // RFC 2183 form.
  const match = header.match(/filename="?([^";]+)"?/i);
  return match ? match[1] : null;
}
