import type { BulkTransferResultResponse } from '@/services/api/bulk-transfer.service';
import { loadPdfMake } from '@/utils/pdf/pdfmake';
import { sanitizeFilename } from '@/utils/file-naming';

// Generate a real PDF and trigger browser download (no print dialog)
export async function generateBulkTransferResultPdf(
  data: BulkTransferResultResponse,
  options?: { fileName?: string }
): Promise<void> {
  const pdfMake = await loadPdfMake();

  const formatCurrency = (amountStr?: string): string => {
    const n = amountStr ? Number(String(amountStr).replace(/[,\s]/g, '')) : 0;
    return new Intl.NumberFormat('vi-VN').format(Number.isFinite(n) ? n : 0) + ' VND';
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return '';
    try {
      const date = new Date(dateString);
      return date.toLocaleString('vi-VN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return String(dateString);
    }
  };

  const fileName = sanitizeFilename(options?.fileName || 'ket_qua_chuyen_tien.pdf');

  const body = [
    [
      { text: 'STT', style: 'th' },
      { text: 'Nhân viên', style: 'th' },
      { text: 'Ngân hàng', style: 'th' },
      { text: 'Số tài khoản', style: 'th' },
      { text: 'CCCD', style: 'th' },
      { text: 'Số tiền', style: 'th' },
      { text: 'Trạng thái', style: 'th' },
      { text: 'Ngày thanh toán', style: 'th' },
    ],
    ...data.items.map((d, i) => [
      { text: String(i + 1) },
      { text: d.employee_name || '' },
      { text: d.employee_bank || '' },
      { text: d.employee_account_number || '' },
      { text: d.employee_cccd || '' },
      { text: formatCurrency(d.amount), alignment: 'right' },
      { text: d.payment_status === 'paid' ? 'Thành công' : 'Thất bại' },
      { text: formatDate(d.paid_at) },
    ]),
  ];

  const docDefinition = {
    pageSize: 'A4',
    pageOrientation: 'landscape' as const,
    pageMargins: [20, 20, 20, 20],
    defaultStyle: { fontSize: 10 },
    content: [
      { text: 'Kết quả chuyển tiền', style: 'title' },
      { text: `Thời điểm tạo: ${new Date().toLocaleString('vi-VN')}`, style: 'muted', margin: [0, 2, 0, 8] },
      {
        columns: [
          { width: 'auto', text: `Tổng số bản ghi: ${data.total_txn}`, style: 'summary' },
          { width: 'auto', text: `Thành công: ${data.completed_txn}`, style: 'summarySuccess' },
          { width: 'auto', text: `Thất bại: ${data.failed_txn}`, style: 'summaryFail' },
        ],
        columnGap: 24,
        margin: [0, 0, 0, 10],
      },
      {
        table: {
          headerRows: 1,
          widths: [30, '*', 90, 100, 90, 80, 70, 110],
          body,
        },
        layout: {
          fillColor: (rowIndex: number) => (rowIndex === 0 ? '#f1f5f9' : undefined),
        },
      },
    ],
    styles: {
      title: { fontSize: 14, bold: true },
      muted: { color: '#475569', fontSize: 9 },
      summary: { fontSize: 10 },
      summarySuccess: { fontSize: 10, color: '#16a34a' },
      summaryFail: { fontSize: 10, color: '#dc2626' },
      th: { bold: true },
    },
  };

  pdfMake.createPdf(docDefinition).download(fileName);
}
