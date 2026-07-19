/**
 * BulkTransferBatchList — paginated history of wallet bulk-transfer batches.
 *
 * Renders one Card per batch with filename, status Badge, counts, total
 * fee, and created_at. Clicking a card opens a Sheet that embeds
 * BulkTransferProgress for that batch.
 *
 * The history is paginated (default 10/page) using the same ChevronLeft /
 * ChevronRight controls used elsewhere in the app.
 */
import { memo, useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, Inbox, Loader2 } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet';
import { useWalletBulkTransferBatches } from '@/hooks/api/useWalletBulkTransfer';
import { BulkTransferProgress } from '@/components/wallet/BulkTransferProgress';
import { formatVietnameseDateTime } from '@/utils/vietnamese';
import { formatCurrency } from '@/utils/formatters';
import {
  batchStatusToVietnamese,
  type BulkTransferBatchStatus,
} from '@/types/wallet-bulk-transfer';
import { cn } from '@/lib/utils';

const PAGE_SIZE = 10;

interface BulkTransferBatchListProps {
  /** Optional controlled page size — defaults to 10. */
  pageSize?: number;
}

/**
 * Badge variant + class for a batch status. Mirrors the tone palette used
 * by BulkTransferProgress so the two views feel like one screen.
 */
function batchStatusBadgeClass(status: BulkTransferBatchStatus): string {
  switch (status) {
    case 'completed':
      return 'bg-emerald-100 text-emerald-800 hover:bg-emerald-100';
    case 'failed':
      return 'bg-rose-100 text-rose-800 hover:bg-rose-100';
    case 'completing':
      return 'bg-blue-100 text-blue-800 hover:bg-blue-100';
    case 'processing':
      return 'bg-amber-100 text-amber-800 hover:bg-amber-100';
    case 'pending':
    default:
      return 'bg-slate-100 text-slate-800 hover:bg-slate-100';
  }
}

export const BulkTransferBatchList = memo(function BulkTransferBatchList({
  pageSize = PAGE_SIZE,
}: BulkTransferBatchListProps) {
  const [page, setPage] = useState(1);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const listQuery = useWalletBulkTransferBatches(page, pageSize);

  // Drop the user back to page 1 if their current page runs off the end
  // (e.g. after the only batch on the last page is removed/expired).
  const total = listQuery.data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  useEffect(() => {
    if (!listQuery.isLoading && page > totalPages) {
      setPage(1);
    }
  }, [page, totalPages, listQuery.isLoading]);

  const canPrev = page > 1 && !listQuery.isFetching;
  const canNext = page < totalPages && !listQuery.isFetching;

  return (
    <section aria-label="Lịch sử chuyển tiền hàng loạt" className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold text-foreground">
            Lịch sử chuyển tiền hàng loạt
          </h2>
          <p className="text-xs text-muted-foreground">
            Theo dõi các lô đã tải lên và trạng thái xử lý.
          </p>
        </div>
        {total > 0 && (
          <span className="text-xs text-muted-foreground">
            {total.toLocaleString('vi-VN')} lô
          </span>
        )}
      </div>

      {listQuery.isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full rounded-xl" />
          ))}
        </div>
      ) : listQuery.isError ? (
        <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
          Không thể tải lịch sử chuyển tiền. Vui lòng thử lại sau.
        </div>
      ) : !listQuery.data || listQuery.data.batches.length === 0 ? (
        <EmptyState />
      ) : (
        <>
          <div className="space-y-2">
            {listQuery.data.batches.map((batch) => (
              <BatchCard
                key={batch.id}
                batch={batch}
                onClick={() => setSelectedId(batch.id)}
              />
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-1 pt-1">
              <Button
                size="icon"
                variant="outline"
                className="h-8 w-8"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={!canPrev}
                aria-label="Trang trước"
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="px-2 text-xs tabular-nums text-muted-foreground">
                {page} / {totalPages}
              </span>
              <Button
                size="icon"
                variant="outline"
                className="h-8 w-8"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={!canNext}
                aria-label="Trang sau"
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </>
      )}

      {/* Detail sheet */}
      <Sheet open={selectedId !== null} onOpenChange={(o) => !o && setSelectedId(null)}>
        <SheetContent
          side="right"
          className="flex w-full flex-col gap-0 sm:max-w-2xl"
        >
          <SheetHeader className="border-b">
            <SheetTitle>Chi tiết lô chuyển tiền</SheetTitle>
            <SheetDescription>
              {selectedId !== null ? `Lô #${selectedId}` : ''}
            </SheetDescription>
          </SheetHeader>
          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            {selectedId !== null ? (
              // key forces a fresh component instance (and fresh prevStatus
              // ref) whenever the admin opens a different batch — otherwise
              // the auto-download transition guard would carry state across
              // batch selections (reviewer W2).
              <BulkTransferProgress key={selectedId} batchId={selectedId} />
            ) : null}
          </div>
        </SheetContent>
      </Sheet>
    </section>
  );
});

/* ------------------------------ pieces ------------------------------ */

interface BatchCardProps {
  batch: {
    id: number;
    filename: string;
    status: BulkTransferBatchStatus;
    total_count: number;
    success_count: number;
    failed_count: number;
    transfer_amount: number;
    total_fee: number;
    created_at: string;
  };
  onClick: () => void;
}

const BatchCard = memo(function BatchCard({ batch, onClick }: BatchCardProps) {
  const processing = Math.max(
    0,
    (batch.total_count || 0) - (batch.success_count || 0) - (batch.failed_count || 0),
  );
  return (
    <Card
      variant="outlined"
      onClick={onClick}
      className={cn(
        'cursor-pointer p-3 transition-colors hover:bg-muted/40',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
      )}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onClick();
        }
      }}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-foreground" title={batch.filename}>
            {batch.filename || `Lô #${batch.id}`}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {formatVietnameseDateTime(batch.created_at)}
          </p>
        </div>
        <Badge className={batchStatusBadgeClass(batch.status)}>
          {batchStatusToVietnamese(batch.status)}
        </Badge>
      </div>

      <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
        <Counter label="Tổng" value={batch.total_count} tone="default" />
        <Counter label="Thành công" value={batch.success_count} tone="success" />
        <Counter label="Thất bại" value={batch.failed_count} tone="failed" />
        {processing > 0 && (
          <Counter label="Đang xử lý" value={processing} tone="processing" />
        )}
      </div>

      <div className="mt-2 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 border-t border-border/60 pt-2 text-xs text-muted-foreground">
        <span>
          Tổng tiền:{' '}
          <span className="font-financial font-semibold tabular-nums text-foreground">
            {formatCurrency(batch.transfer_amount)}
          </span>
        </span>
        <span>
          Phí:{' '}
          <span className="font-financial font-semibold tabular-nums text-foreground">
            {formatCurrency(batch.total_fee)}
          </span>
        </span>
      </div>
    </Card>
  );
});

interface CounterProps {
  label: string;
  value: number;
  tone: 'default' | 'success' | 'failed' | 'processing';
}

const COUNTER_TONE: Record<CounterProps['tone'], string> = {
  default: 'text-muted-foreground',
  success: 'text-emerald-700',
  failed: 'text-rose-700',
  processing: 'text-amber-700',
};

function Counter({ label, value, tone }: CounterProps) {
  return (
    <span className={cn('inline-flex items-center gap-1', COUNTER_TONE[tone])}>
      <span className="font-financial font-semibold tabular-nums">
        {value.toLocaleString('vi-VN')}
      </span>
      <span className="text-muted-foreground">{label}</span>
    </span>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border/60 bg-muted/20 px-4 py-10 text-center">
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
        <Inbox className="h-5 w-5 text-muted-foreground" />
      </div>
      <p className="text-sm font-medium text-foreground">Chưa có lô chuyển tiền nào</p>
      <p className="max-w-xs text-xs text-muted-foreground">
        Tải lên file Yêu cầu chuyển tiền (.xlsx) từ Bảng công để tạo lô đầu tiên.
      </p>
    </div>
  );
}

export default BulkTransferBatchList;
