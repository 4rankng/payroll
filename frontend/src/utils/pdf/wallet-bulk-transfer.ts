import type { WalletBulkBatchDetail } from '@/types/wallet-bulk-transfer';
import { loadPdfMake } from '@/utils/pdf/pdfmake';
import { sanitizeFilename } from '@/utils/file-naming';

function formatVND(value: number): string {
  return `${value.toLocaleString('vi-VN')} ₫`;
}

function formatDate(value: string | null): string {
  if (!value) return '';
  return new Date(value).toLocaleString('vi-VN', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });
}

export async function generateWalletBulkTransferPdf(batch: WalletBulkBatchDetail): Promise<void> {
  const rows = batch.rows.filter((row) => row.status === 'completed');
  if (rows.length === 0) {
    throw new Error('Không có giao dịch thành công để xuất PDF');
  }
  const pdfMake = await loadPdfMake();
  if (!pdfMake) throw new Error('Không thể tải thư viện PDF');

  const total = rows.reduce((sum, row) => sum + row.requested_amount, 0);
  const body = [
    ['STT', 'Mã VFIC', 'Người nhận', 'Số tài khoản', 'Ngân hàng', 'Số tiền', 'Mã FT', 'Hoàn tất']
      .map((text) => ({ text, style: 'th' })),
    ...rows.map((row, index) => [
      String(index + 1), row.request_id, row.recipient_name,
      row.recipient_account_no, row.recipient_bank,
      { text: formatVND(row.requested_amount), alignment: 'right' },
      row.invoice_no || 'Đang chờ FT', formatDate(row.settled_at),
    ]),
  ];

  pdfMake.createPdf({
    pageSize: 'A4',
    pageOrientation: 'landscape',
    pageMargins: [20, 20, 20, 20],
    defaultStyle: { fontSize: 9 },
    content: [
      { text: 'Danh sách giao dịch chuyển tiền thành công', style: 'title' },
      { text: `Lô #${batch.id} · ${batch.filename}`, style: 'muted', margin: [0, 2, 0, 2] },
      { text: `${rows.length} giao dịch · Tổng tiền: ${formatVND(total)}`, style: 'summary', margin: [0, 0, 0, 10] },
      { table: { headerRows: 1, widths: [28, 80, '*', 90, 75, 80, 80, 90], body } },
    ],
    styles: {
      title: { fontSize: 14, bold: true },
      muted: { color: '#475569', fontSize: 9 },
      summary: { color: '#047857', bold: true },
      th: { bold: true, fillColor: '#f1f5f9' },
    },
  }).download(sanitizeFilename(`KQ_Thanh_Cong_${batch.id}.pdf`));
}
