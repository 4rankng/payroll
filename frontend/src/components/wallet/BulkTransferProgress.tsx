/**
 * BulkTransferProgress — per-row live status of a wallet bulk-transfer batch.
 *
 * Polls GET /wallet/bulk-transfer/batches/:id every 5s while the batch is
 * non-terminal (the hook handles refetchInterval gating). Renders a
 * summary header + a Table of rows with status Badges and an FT-pending
 * tooltip. The "Tải KQ Excel" button uses useDownloadWalletBulkTransferKQ.
 *
 * Mobile-friendly: cards fall back to a stacked layout via useIsMobile.
 */
import { memo, useCallback } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Clock3, Download, Loader2 } from 'lucide-react';
import { useWalletBulkTransferBatch, useDownloadWalletBulkTransferKQ } from '@/hooks/api/useWalletBulkTransfer';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { formatCurrency } from '@/utils/formatters';
import {
  batchStatusToVietnamese,
  isFTPending,
  paymentStatusToVietnamese,
  type WalletBulkPaymentRow,
  type WalletPaymentStatus,
} from '@/types/wallet-bulk-transfer';
import { cn } from '@/lib/utils';

interface BulkTransferProgressProps {
  batchId: number;
}

/**
 * Badge variant + class for a wallet_payment status. completed=green,
 * failed=red, processing states=yellow. Uses outline variant + tailwind
 * bg classes so the palette matches WalletTransactionsList.
 */
function statusBadgeClass(status: WalletPaymentStatus): string {
  switch (status) {
    case 'completed':
      return 'bg-emerald-100 text-emerald-800 hover:bg-emerald-100';
    case 'failed':
    case 'reversed':
      return 'bg-rose-100 text-rose-800 hover:bg-rose-100';
    case 'pending':
    case 'verified':
    case 'authorised':
    default:
      return 'bg-amber-100 text-amber-800 hover:bg-amber-100';
  }
}

export const BulkTransferProgress = memo(function BulkTransferProgress({
  batchId,
}: BulkTransferProgressProps) {
  const isMobile = useIsMobile();
  const batchQuery = useWalletBulkTransferBatch(batchId);
  const downloadKQ = useDownloadWalletBulkTransferKQ();

  const handleDownloadKQ = useCallback(() => {
    downloadKQ.mutate(batchId);
  }, [downloadKQ, batchId]);

  if (batchQuery.isLoading) {
    return <ProgressSkeleton />;
  }

  if (batchQuery.isError || !batchQuery.data) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
        Không thể tải thông tin lô chuyển tiền. Vui lòng thử lại sau.
      </div>
    );
  }

  const batch = batchQuery.data;
  const total = batch.total_count || 0;
  const success = batch.success_count || 0;
  const failed = batch.failed_count || 0;
  const processing = Math.max(0, total - success - failed);
  const pct = total > 0 ? Math.round(((success + failed) / total) * 100) : 0;
  const isRunning =
    batch.status === 'pending' ||
    batch.status === 'processing' ||
    batch.status === 'completing';

  return (
    <TooltipProvider delayDuration={150}>
      <div className="space-y-4">
        {/* Summary */}
        <div className="space-y-3 rounded-xl border border-border/60 bg-muted/40 p-3 sm:p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <Badge className={statusBadgeClass(batch.status)}>
                {batchStatusToVietnamese(batch.status)}
              </Badge>
              <span className="text-xs text-muted-foreground">
                Lô #{batch.id} · {batch.filename}
              </span>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={handleDownloadKQ}
              disabled={downloadKQ.isPending}
              className="h-8 gap-1.5"
            >
              {downloadKQ.isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Download className="h-3.5 w-3.5" />
              )}
              Tải KQ Excel
            </Button>
          </div>

          <Progress value={pct} className="h-2" />

          <div className="grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
            <Stat label="Tổng" value={total} tone="default" />
            <Stat label="Thành công" value={success} tone="success" />
            <Stat label="Thất bại" value={failed} tone="failed" />
            <Stat label="Đang xử lý" value={processing} tone="processing" />
          </div>

          {isRunning && (
            <p className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              Đang xử lý — cập nhật mỗi 5 giây.
            </p>
          )}
        </div>

        {/* Rows */}
        {isMobile ? (
          <MobileRowList rows={batch.rows} />
        ) : (
          <div className="overflow-hidden rounded-xl border border-border/60">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12 text-right">STT</TableHead>
                  <TableHead>Số TK</TableHead>
                  <TableHead>Tên</TableHead>
                  <TableHead>Ngân hàng</TableHead>
                  <TableHead className="text-right">Số tiền</TableHead>
                  <TableHead className="text-center">Trạng thái</TableHead>
                  <TableHead className="text-center">FT</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {batch.rows.map((row) => (
                  <ProgressRow key={row.id} row={row} />
                ))}
                {batch.rows.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="py-6 text-center text-xs text-muted-foreground">
                      Chưa có dòng nào trong lô này.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </TooltipProvider>
  );
});

interface ProgressRowProps {
  row: WalletBulkPaymentRow;
}

const ProgressRow = memo(function ProgressRow({ row }: ProgressRowProps) {
  const order = typeof row.bulk_transfer_order === 'number' ? row.bulk_transfer_order : row.id;
  const ftPending = isFTPending(row);
  const invoiceLabel =
    row.invoice_no && !ftPending
      ? row.invoice_no
      : ftPending
        ? 'Đang chờ FT'
        : '—';

  return (
    <TableRow>
      <TableCell className="text-right tabular-nums text-muted-foreground">
        {order}
      </TableCell>
      <TableCell className="font-mono text-xs">{row.recipient_account_no || '—'}</TableCell>
      <TableCell className="max-w-[180px] truncate" title={row.recipient_name}>
        {row.recipient_name || '—'}
      </TableCell>
      <TableCell className="max-w-[140px] truncate" title={row.recipient_bank}>
        {row.recipient_bank || '—'}
      </TableCell>
      <TableCell className="text-right font-financial tabular-nums">
        {formatCurrency(row.requested_amount)}
      </TableCell>
      <TableCell className="text-center">
        <Badge className={statusBadgeClass(row.status)}>
          {paymentStatusToVietnamese(row.status)}
        </Badge>
      </TableCell>
      <TableCell className="text-center">
        {ftPending ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="inline-flex items-center gap-1 text-xs text-amber-700 hover:underline"
              >
                <Clock3 className="h-3 w-3" />
                <span className="tabular-nums">{invoiceLabel}</span>
              </button>
            </TooltipTrigger>
            <TooltipContent>
              FT chưa về — đang chờ OnePay trả về số FT qua IPN.
            </TooltipContent>
          </Tooltip>
        ) : (
          <span className="font-mono text-xs tabular-nums">{invoiceLabel}</span>
        )}
      </TableCell>
    </TableRow>
  );
});

interface MobileRowListProps {
  rows: WalletBulkPaymentRow[];
}

const MobileRowList = memo(function MobileRowList({ rows }: MobileRowListProps) {
  if (rows.length === 0) {
    return (
      <p className="rounded-xl border border-border/60 bg-muted/30 px-4 py-6 text-center text-xs text-muted-foreground">
        Chưa có dòng nào trong lô này.
      </p>
    );
  }
  return (
    <div className="space-y-2">
      {rows.map((row) => (
        <MobileRow key={row.id} row={row} />
      ))}
    </div>
  );
});

interface MobileRowProps {
  row: WalletBulkPaymentRow;
}

const MobileRow = memo(function MobileRow({ row }: MobileRowProps) {
  const order = typeof row.bulk_transfer_order === 'number' ? row.bulk_transfer_order : row.id;
  const ftPending = isFTPending(row);
  return (
    <div className="rounded-xl border border-border/60 bg-card p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-muted-foreground">Dòng {order}</span>
        <Badge className={statusBadgeClass(row.status)}>
          {paymentStatusToVietnamese(row.status)}
        </Badge>
      </div>
      <div className="mt-1 truncate text-sm font-medium text-foreground" title={row.recipient_name}>
        {row.recipient_name || '—'}
      </div>
      <div className="mt-0.5 truncate text-xs text-muted-foreground" title={row.recipient_bank}>
        {row.recipient_bank || '—'} · {row.recipient_account_no || '—'}
      </div>
      <div className="mt-2 flex items-center justify-between gap-2">
        <span className="font-financial text-sm font-semibold tabular-nums">
          {formatCurrency(row.requested_amount)}
        </span>
        {ftPending ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex items-center gap-1 text-xs text-amber-700">
                <Clock3 className="h-3 w-3" />
                Đang chờ FT
              </span>
            </TooltipTrigger>
            <TooltipContent>FT chưa về — đang chờ OnePay trả về số FT qua IPN.</TooltipContent>
          </Tooltip>
        ) : (
          row.invoice_no && (
            <span className="font-mono text-xs tabular-nums text-muted-foreground">
              {row.invoice_no}
            </span>
          )
        )}
      </div>
    </div>
  );
});

/* ----------------------------- helpers ----------------------------- */

type StatTone = 'default' | 'success' | 'failed' | 'processing';

interface StatProps {
  label: string;
  value: number;
  tone: StatTone;
}

const STAT_TONE: Record<StatTone, string> = {
  default: 'text-foreground',
  success: 'text-emerald-700',
  failed: 'text-rose-700',
  processing: 'text-amber-700',
};

function Stat({ label, value, tone }: StatProps) {
  return (
    <div className="min-w-0">
      <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
        {label}
      </p>
      <p className={cn('mt-0.5 font-financial text-sm font-bold tabular-nums', STAT_TONE[tone])}>
        {value.toLocaleString('vi-VN')}
      </p>
    </div>
  );
}

function ProgressSkeleton() {
  return (
    <div className="space-y-4">
      <div className="space-y-3 rounded-xl border border-border/60 bg-muted/40 p-3 sm:p-4">
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-2 w-full" />
        <div className="grid grid-cols-4 gap-2">
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
        </div>
      </div>
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    </div>
  );
}

export default BulkTransferProgress;
