/**
 * React Query hooks for Stage 1 (timesheet OnePay export).
 *
 * The mutation triggers the export, saves the .xlsx blob via triggerBlobDownload,
 * and shows a Vietnamese toast. When employees are skipped (missing bank info),
 * a warning toast prompts the admin to check the data.
 */
import { useMutation } from '@tanstack/react-query';
import { onepayExportService } from '@/services/api/onepay-export.service';
import { BulkTransferExportParams } from '@/services/api/bulk-transfer.service';
import { triggerBlobDownload } from '@/utils/file-download';
import { showErrorNotification } from '@/utils/error-handler';
import { toast } from 'sonner';

export function useExportOnePayBulk() {
  return useMutation({
    mutationFn: (params: BulkTransferExportParams) => onepayExportService.exportBulk(params),
    onSuccess: async (data) => {
      // Always trigger the download — even with skips, the file is valid.
      triggerBlobDownload(data.blob, data.filename);
      if (data.skippedCount > 0) {
        toast.warning(
          `Đã xuất ${data.totalCount} dòng, bỏ qua ${data.skippedCount} nhân viên thiếu thông tin ngân hàng`,
          { duration: 6000 },
        );
      } else {
        toast.success(`Đã xuất file OnePay (${data.totalCount} dòng)`);
      }
    },
    onError: (error) => {
      showErrorNotification(error, 'Xuất file OnePay thất bại');
    },
  });
}
