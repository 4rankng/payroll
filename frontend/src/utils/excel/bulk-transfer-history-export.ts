import { format } from "date-fns";
import { exportToXLSWithMapping, type XLSColumnMapping } from '@/utils/xls-export';
import type { BulkTransferHistoryDetail } from '@/services/api/bulk-transfer.service';

/**
 * Export bulk transfer history details to Excel file
 */
export async function exportBulkTransferHistoryToExcel(
  data: BulkTransferHistoryDetail[],
  originalFilename: string,
  summary: {
    total_txn: number;
    completed_txn: number;
    failed_txn: number;
  }
): Promise<void> {
  if (!data || data.length === 0) {
    throw new Error('Không có dữ liệu để xuất');
  }

  // Define column mappings
  const columns: XLSColumnMapping[] = [
    {
      key: 'row',
      header: 'STT',
      width: 8,
    },
    {
      key: 'employee_bank',
      header: 'Ngân hàng',
      width: 15,
    },
    {
      key: 'employee_account_number',
      header: 'Số tài khoản',
      width: 18,
    },
    {
      key: 'employee_name',
      header: 'Tên nhân viên',
      width: 25,
    },
    {
      key: 'employee_cccd',
      header: 'CCCD',
      width: 15,
    },
    {
      key: 'amount',
      header: 'Số tiền',
      width: 18,
      formatter: (value: unknown) => {
        if (!value) return '';
        const strValue = String(value);
        // Remove commas and format as number
        const numValue = Number(strValue.replace(/,/g, ''));
        return isNaN(numValue) ? strValue : `${numValue.toLocaleString('vi-VN')} VND`;
      },
    },
    {
      key: 'payment_status',
      header: 'Trạng thái',
      width: 15,
      formatter: (value: unknown) => {
        return value === 'paid' ? 'Thành công' : 'Thất bại';
      },
    },
    {
      key: 'paid_at',
      header: 'Ngày thanh toán',
      width: 20,
      formatter: (value: unknown) => {
        if (!value) return '';
        try {
          const date = new Date(value as string);
          return format(date, 'dd/MM/yyyy');
        } catch {
          return String(value);
        }
      },
    },
  ];

  // Generate filename
  const cleanFilename = originalFilename.replace(/\.[^/.]+$/, ''); // Remove extension
  const fileName = `lich_su_chuyen_tien_${cleanFilename}.xls`;

  // Export to Excel
  await exportToXLSWithMapping(data, columns, {
    fileName,
    sheetName: 'Lịch sử chuyển tiền',
  });
}
