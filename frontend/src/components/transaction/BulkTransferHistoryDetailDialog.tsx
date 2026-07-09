import { memo, useCallback, useMemo, useState } from 'react';
import { Dialog, DialogContent, DialogClose } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { FileDown, ArrowLeft, CheckCircle2, XCircle, FileSpreadsheet, Loader2, Clock, X } from 'lucide-react';
import { useBulkTransferUploadHistoryById } from '@/hooks/api/usePayrolls';
import { showErrorNotification } from '@/utils/error-handler';
import { generateBulkTransferHistoryPdf } from '@/utils/pdf/bulk-transfer-history';
import { sanitizeFilename } from '@/utils/file-naming';
import { dateToString } from '@/utils/dateHelpers';
import { bulkTransferService } from '@/services/api/bulk-transfer.service';
import { cn } from '@/lib/utils';
import { formatCurrencyFromString } from '@/utils/formatters';

// --- Helpers ---

function formatDate(dateString: string): string {
  if (!dateString) return '';
  try {
    return new Date(dateString).toLocaleString('vi-VN', {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  } catch { return dateString; }
}

// --- Sub-components ---

function DetailSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-8 w-full rounded-xl" />
      {Array.from({ length: 6 }).map((_, i) => (
        <Skeleton key={i} className="h-11 w-full rounded-xl" />
      ))}
    </div>
  );
}

interface TransactionRowProps {
  detail: {
    row: number;
    payment_status: string;
    employee_name: string;
    employee_bank: string;
    employee_account_number: string;
    employee_cccd?: string;
    amount: string;
    paid_at?: string;
  };
  isLast: boolean;
}

function TransactionRow({ detail, isLast }: TransactionRowProps) {
  const isPaid = detail.payment_status === 'paid';
  return (
    <div className={cn(
      'grid grid-cols-[1.5rem_1fr_auto] items-center gap-x-3 py-2.5 px-3',
      !isLast && 'border-b border-border'
    )}>
      {/* Status icon */}
      <div className="flex justify-center">
        {isPaid
          ? <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
          : <XCircle className="w-3.5 h-3.5 text-red-500 shrink-0" />}
      </div>

      {/* Employee */}
      <div className="min-w-0">
        <p className="typography-body-small font-medium text-foreground truncate leading-tight">
          {detail.employee_name}
        </p>
        <p className="typography-label-medium text-slate-400 truncate leading-tight mt-0.5">
          {detail.employee_bank} · {detail.employee_account_number}
        </p>
      </div>

      {/* Amount */}
      <p className="typography-body-small font-semibold text-slate-800 tabular-nums text-right whitespace-nowrap">
        {formatCurrencyFromString(detail.amount)}
      </p>
    </div>
  );
}

// --- Types ---

interface BulkTransferHistoryDetailDialogProps {
  open: boolean;
  historyId: number | null;
  filename: string;
  uploadedAt?: string;
  onOpenChange: (open: boolean) => void;
  onBack: () => void;
}

// --- Main component ---

export const BulkTransferHistoryDetailDialog = memo(function BulkTransferHistoryDetailDialog({
  open,
  historyId,
  filename,
  uploadedAt,
  onOpenChange,
  onBack,
}: BulkTransferHistoryDetailDialogProps) {
  const { data: historyDetail, isLoading } = useBulkTransferUploadHistoryById(historyId);
  const detailItems = useMemo(() => historyDetail?.items ?? [], [historyDetail]);
  const [isDownloadingExcel, setIsDownloadingExcel] = useState(false);

  const handleDownloadPdf = useCallback(async () => {
    if (!historyId || !historyDetail) return;
    try {
      const rawName = `ket_qua_chuyen_tien_${dateToString(new Date())}.pdf`;
      await generateBulkTransferHistoryPdf(
        { ...historyDetail, uploaded_at: historyDetail.uploaded_at || uploadedAt },
        { fileName: sanitizeFilename(rawName) }
      );
    } catch (error) {
      showErrorNotification(error, 'Không thể tạo PDF');
    }
  }, [historyId, historyDetail, uploadedAt]);

  const handleDownloadExcel = useCallback(async () => {
    if (!historyId) return;
    setIsDownloadingExcel(true);
    try {
      await bulkTransferService.downloadBulkTransferHistoryExcel(historyId);
    } catch (error) {
      showErrorNotification(error, 'Tải file Excel thất bại');
    } finally {
      setIsDownloadingExcel(false);
    }
  }, [historyId]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[92dvh] w-full max-w-2xl flex-col gap-0 overflow-hidden" contentPadding="none" hideCloseButton>

        {/* Header */}
        <div className="shrink-0 bg-slate-900 px-4 pt-4 pb-3 text-white">
          <div className="flex items-start gap-3">
            {/* Back button */}
            <button
              onClick={onBack}
              className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-white/10 transition-colors hover:bg-white/20"
              aria-label="Quay lại"
            >
              <ArrowLeft className="w-3.5 h-3.5 text-white" />
            </button>

            {/* Title */}
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-white leading-tight">Chi tiết kết quả chuyển tiền</p>
              <p className="mt-0.5 break-all text-xs text-slate-400" title={filename}>{filename}</p>
            </div>

            {/* Actions */}
            <div className="grid shrink-0 grid-cols-1 gap-1.5 min-[380px]:grid-cols-3">
              <Button
                size="sm"
                onClick={handleDownloadPdf}
                disabled={!historyId || !historyDetail || isLoading}
                className="min-h-11 px-2.5 text-xs gap-1 bg-white/10 hover:bg-white/20 text-white border-0 shadow-none"
              >
                <FileDown className="w-3 h-3" />
                PDF
              </Button>
              <Button
                size="sm"
                onClick={handleDownloadExcel}
                disabled={!historyId || isDownloadingExcel}
                className="min-h-11 px-2.5 text-xs gap-1 bg-white/10 hover:bg-white/20 text-white border-0 shadow-none"
              >
                {isDownloadingExcel
                  ? <Loader2 className="w-3 h-3 animate-spin" />
                  : <FileSpreadsheet className="w-3 h-3" />}
                Excel
              </Button>
              <DialogClose className="flex h-11 w-11 items-center justify-center rounded-full bg-white/10 outline-none transition-colors hover:bg-white/20">
                <X className="w-3.5 h-3.5 text-white" />
              </DialogClose>
            </div>
          </div>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto scrollbar-thin scrollbar-thumb-muted min-h-0">
          {isLoading ? (
            <div className="p-4"><DetailSkeleton /></div>
          ) : historyDetail ? (
            <>
              {/* Inline summary strip */}
              <div className="flex items-center gap-4 px-4 py-2.5 bg-muted/50 border-b text-xs">
                <span className="text-muted-foreground">
                  Tổng: <span className="font-semibold text-slate-800">{historyDetail.total_txn}</span>
                </span>
                <span className="text-slate-300">|</span>
                <span className="text-muted-foreground flex items-center gap-1">
                  <CheckCircle2 className="w-3 h-3 text-emerald-500" />
                  Thành công: <span className="font-semibold text-emerald-700 ml-0.5">{historyDetail.completed_txn}</span>
                </span>
                <span className="text-slate-300">|</span>
                <span className="text-muted-foreground flex items-center gap-1">
                  <XCircle className="w-3 h-3 text-red-400" />
                  Thất bại: <span className={cn('font-semibold ml-0.5', historyDetail.failed_txn > 0 ? 'text-red-600' : 'text-muted-foreground')}>
                    {historyDetail.failed_txn}
                  </span>
                </span>
              </div>

              {/* Transaction list */}
              {detailItems.length > 0 ? (
                <div className="divide-y-0">
                  {/* Shared timestamp above table */}
                  {detailItems[0]?.paid_at && (
                    <div className="flex items-center gap-1.5 px-3 py-2 bg-muted/50 border-b text-xs text-slate-400">
                      <Clock className="w-3 h-3 shrink-0" />
                      Thời gian xử lý: <span className="font-medium text-muted-foreground ml-0.5">{formatDate(detailItems[0].paid_at)}</span>
                    </div>
                  )}
                  {/* Column headers */}
                  <div className="grid grid-cols-[1.5rem_1fr_auto] gap-x-3 px-3 py-1.5 bg-muted/50 border-b">
                    <div />
                    <p className="typography-label-small text-slate-400 uppercase">Nhân viên</p>
                    <p className="typography-label-small text-slate-400 uppercase text-right">Số tiền</p>
                  </div>
                  {detailItems.map((detail, index) => (
                    <TransactionRow
                      key={index}
                      detail={detail}
                      isLast={index === detailItems.length - 1}
                    />
                  ))}
                </div>
              ) : (
                <div className="flex items-center justify-center py-10">
                  <p className="text-sm text-slate-400">Không có giao dịch nào</p>
                </div>
              )}
            </>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <FileSpreadsheet className="w-8 h-8 text-slate-300 mb-2" />
              <p className="text-sm text-muted-foreground">Không tìm thấy dữ liệu</p>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
});
