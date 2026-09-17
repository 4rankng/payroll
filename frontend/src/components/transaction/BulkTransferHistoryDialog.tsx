import { memo, useCallback, useMemo, useState, useEffect, useRef } from 'react';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { FileSpreadsheet, Clock, User, X, Calendar as CalendarIcon, CheckCircle2, ChevronRight } from 'lucide-react';
import { useBulkTransferUploadHistories } from '@/hooks/api/usePayrolls';
import type { BulkTransferUploadHistory } from '@/services/api/bulk-transfer.service';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetClose } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { cn } from '@/lib/utils';
import { EmptyState as SharedEmptyState } from '@/components/shared/EmptyState';
import { dateToString } from '@/utils/dateHelpers';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';

interface BulkTransferHistoryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelectHistory: (id: number, filename: string, uploadedAt: string) => void;
  shouldFetchHistories: boolean;
}

const getYearToDateDefaults = () => {
  const today = new Date();
  return {
    fromDate: new Date(today.getFullYear(), 0, 1),
    toDate: today,
  };
};

function formatDate(dateString: string): string {
  try {
    return new Date(dateString).toLocaleString('vi-VN', {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  } catch { return dateString; }
}

function HistorySkeleton() {
  return (
    <div className="divide-y" aria-label="Đang tải lịch sử giao dịch">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="px-4 py-3">
          <Skeleton className="h-4 w-3/4 mb-2 rounded" />
          <Skeleton className="h-3 w-1/2 rounded" />
        </div>
      ))}
    </div>
  );
}

function EmptyState() {
  return (
    <SharedEmptyState
      title="Chưa có lịch sử"
      description="Chưa có file kết quả chuyển tiền nào được tải lên."
      size="sm"
    />
  );
}

interface HistoryRowProps {
  history: BulkTransferUploadHistory;
  onClick: (history: BulkTransferUploadHistory) => void;
  isLast: boolean;
}

const HistoryRow = memo(function HistoryRow({ history, onClick, isLast }: HistoryRowProps) {
  const allSuccess = history.failed_txn === 0;

  return (
    <button
      type="button"
      onClick={() => onClick(history)}
      className={cn(
        'w-full text-left px-4 py-3 flex items-start gap-3 transition-colors',
        'hover:bg-muted/50 active:bg-slate-100',
        'focus-visible:outline-none focus-visible:bg-muted/50',
        'touch-manipulation group',
        !isLast && 'border-b border-border'
      )}
    >
      {/* Icon */}
      <div className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-100">
        <FileSpreadsheet className="w-4 h-4 text-muted-foreground" />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="mb-1 flex items-start gap-2">
          <p className="min-w-0 flex-1 break-all text-sm font-medium text-foreground" title={history.filename}>
            {history.filename}
          </p>
          {allSuccess ? (
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
          ) : (
            <Badge variant="destructive" className="h-4 px-1.5 text-xs shrink-0">
              {history.failed_txn} lỗi
            </Badge>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-slate-400">
          <span className="tabular-nums">
            {history.completed_txn}/{history.total_txn} thành công
          </span>
          <span className="text-slate-200">·</span>
          <span className="flex items-center gap-1">
            <Clock className="w-3 h-3 shrink-0" />
            {formatDate(history.uploaded_at)}
          </span>
          <span className="hidden items-center gap-1 sm:flex">
            <span className="text-slate-200">·</span>
            <User className="w-3 h-3 shrink-0" />
            <span className="max-w-[140px] break-words">{history.uploaded_by}</span>
          </span>
        </div>
      </div>

      <ChevronRight className="w-3.5 h-3.5 text-slate-300 shrink-0 transition-transform group-hover:translate-x-0.5" />
    </button>
  );
});

export const BulkTransferHistoryDialog = memo(function BulkTransferHistoryDialog({
  open,
  onOpenChange,
  onSelectHistory,
  shouldFetchHistories,
}: BulkTransferHistoryDialogProps) {
  const isMobile = useIsMobile();
  const shouldFetchData = open && shouldFetchHistories;
  const yearToDate = useMemo(() => getYearToDateDefaults(), []);

  const [page, setPage] = useState(1);
  const [allHistories, setAllHistories] = useState<BulkTransferUploadHistory[]>([]);
  const [fromDate, setFromDate] = useState<Date | undefined>(yearToDate.fromDate);
  const [toDate, setToDate] = useState<Date | undefined>(yearToDate.toDate);
  const [hasMore, setHasMore] = useState(true);
  const [fromDateOpen, setFromDateOpen] = useState(false);
  const [toDateOpen, setToDateOpen] = useState(false);

  const loadMoreRef = useRef<HTMLDivElement>(null);
  const pageSize = 20;

  const fromDateStr = fromDate ? dateToString(fromDate) : undefined;
  const toDateStr = toDate ? dateToString(toDate) : undefined;

  const { data, isLoading, isFetching } = useBulkTransferUploadHistories(
    { sortBy: 'uploaded_at', sortOrder: 'desc', fromDate: fromDateStr, toDate: toDateStr, page, pageSize },
    shouldFetchData
  );

  useEffect(() => {
    setPage(1);
    setAllHistories([]);
    setHasMore(true);
  }, [fromDate, toDate]);

  useEffect(() => {
    if (!data?.data || !Array.isArray(data.data)) return;
    // Use functional update so we always operate on the latest state,
    // avoiding the race where the reset and append effects fire together
    setAllHistories(prev => page === 1 ? data.data : [...prev, ...data.data]);
    if (data.pagination) setHasMore(data.pagination.page < data.pagination.totalPages);
  }, [data, page]);

  useEffect(() => {
    if (!loadMoreRef.current || !hasMore || isFetching) return;
    const el = loadMoreRef.current;
    const observer = new IntersectionObserver(
      (entries) => { if (entries[0].isIntersecting && hasMore && !isFetching) setPage(p => p + 1); },
      { threshold: 0.1 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasMore, isFetching, fromDate, toDate]);

  const handleRowClick = useCallback((history: BulkTransferUploadHistory) => {
    onSelectHistory(history.id, history.filename, history.uploaded_at);
  }, [onSelectHistory]);

  const histories = Array.isArray(allHistories) ? allHistories : [];
  const isEmpty = !isLoading && histories.length === 0;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side={isMobile ? "bottom" : "right"} className={cn("w-full sm:w-[540px] sm:max-w-[540px] p-0 flex flex-col h-full gap-0", isMobile && "rounded-t-2xl max-h-[94dvh] shadow-[0_-4px_24px_rgba(0,0,0,0.08)]")}>

        {/* Mobile drag handle */}
        {isMobile && (
          <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
            <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
          </div>
        )}

        {/* Header */}
        <div className="shrink-0 px-4 pt-4 pb-3 border-b">
          <div className="flex items-start justify-between gap-3">
            <SheetHeader className="text-left space-y-0">
              <SheetTitle className="text-sm font-semibold">Lịch sử chuyển lô</SheetTitle>
              <SheetDescription className="text-xs text-slate-400">
                Danh sách file kết quả đã tải lên
              </SheetDescription>
            </SheetHeader>
            <SheetClose asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-11 w-11 shrink-0 rounded-xl text-slate-400 hover:text-foreground hover:bg-slate-100"
                aria-label="Đóng"
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </SheetClose>
          </div>

          {/* Date filters — calendar popovers */}
          <div className="mt-3 grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
            <Popover open={fromDateOpen} onOpenChange={setFromDateOpen}>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  className={cn(
                    "h-11 w-full justify-start px-3 text-left text-xs font-normal",
                    !fromDate && "text-muted-foreground"
                  )}
                >
                  <CalendarIcon className="mr-1.5 h-3 w-3 shrink-0" />
                  {fromDate ? format(fromDate, "dd/MM/yyyy", { locale: vi }) : "Từ ngày"}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0" align="start">
                <Calendar
                  mode="single"
                  selected={fromDate}
                  onSelect={(date) => {
                    setFromDate(date);
                    setFromDateOpen(false);
                  }}
                  initialFocus
                  locale={vi}
                />
              </PopoverContent>
            </Popover>

            <Popover open={toDateOpen} onOpenChange={setToDateOpen}>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  className={cn(
                    "h-11 w-full justify-start px-3 text-left text-xs font-normal",
                    !toDate && "text-muted-foreground"
                  )}
                >
                  <CalendarIcon className="mr-1.5 h-3 w-3 shrink-0" />
                  {toDate ? format(toDate, "dd/MM/yyyy", { locale: vi }) : "Đến ngày"}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0" align="start">
                <Calendar
                  mode="single"
                  selected={toDate}
                  onSelect={(date) => {
                    setToDate(date);
                    setToDateOpen(false);
                  }}
                  initialFocus
                  locale={vi}
                  disabled={(date) => fromDate ? date < fromDate : false}
                />
              </PopoverContent>
            </Popover>
          </div>
        </div>

        {/* List */}
        <div className="flex-1 overflow-y-auto scrollbar-thin scrollbar-thumb-muted min-h-0">
          {isLoading && page === 1 ? (
            <HistorySkeleton />
          ) : isEmpty ? (
            <EmptyState />
          ) : (
            <>
              {histories.map((history, index) => (
                <HistoryRow
                  key={history.id}
                  history={history}
                  onClick={handleRowClick}
                  isLast={index === histories.length - 1 && !hasMore}
                />
              ))}

              {hasMore && (
                <div ref={loadMoreRef} className="py-3 flex justify-center">
                  {isFetching && (
                    <div className="flex items-center gap-1.5 text-xs text-slate-400">
                      <div className="w-3 h-3 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                      Đang tải thêm...
                    </div>
                  )}
                </div>
              )}

              {!hasMore && histories.length > 0 && (
                <p className="py-3 text-center text-xs text-slate-400">
                  {histories.length} kết quả
                </p>
              )}
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
});
