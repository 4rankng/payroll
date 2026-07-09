import { memo, useMemo, useCallback, useState, useRef } from 'react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import {
  Dialog,
  DialogContent,
  DialogNavyHeader,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { CheckCircle2, Download, Loader2, FileText, AlertCircle, RefreshCw, Upload } from 'lucide-react';
import { cn } from '@/lib/utils';
import { showErrorNotification } from '@/utils/error-handler';
import { useEmailHistory, useSettleFromEmailHistory, useStandaloneSettlementUpload } from '@/hooks/api/useEmails';
import { apiClient } from '@/services/api/client';
import { API_ENDPOINTS } from '@/config/api.config';
import type { EmailHistoryRecord } from '@/types/api/email.types';

interface SaoKeHistoryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const fmt = (n: number) => new Intl.NumberFormat('vi-VN').format(n) + ' đ';
const fmtDate = (v: string) => {
  try { return format(new Date(v), 'dd/MM/yyyy HH:mm', { locale: vi }); }
  catch { return v; }
};
const fmtDateShort = (v: string) => {
  try { return format(new Date(v), 'dd/MM/yyyy', { locale: vi }); }
  catch { return v; }
};

interface RowProps {
  record: EmailHistoryRecord;
  onSettle: (id: number) => void;
  isSettling: boolean;
  localSettled: boolean;
}

const SaoKeRow = memo(function SaoKeRow({ record, onSettle, isSettling, localSettled }: RowProps) {
  const meta = record.payrollMeta!;
  const settled = !!record.settledAt || localSettled;
  const [downloading, setDownloading] = useState(false);
  const isAdvancePayment = record.type === 'advance_payment_report';

  const toList = record.recipients?.map((r) => r.address) ?? [];
  const ccList = meta.cc ?? [];
  const bccList = meta.bcc ?? [];

  const handleDownload = useCallback(async () => {
    if (!meta.saoKeAssetId) return;
    setDownloading(true);
    try {
      await apiClient.download(
        API_ENDPOINTS.assets.download(meta.saoKeAssetId),
        `sao_ke_${meta.reportAtDate ?? record.sentAt.slice(0, 10)}.xlsx`,
      );
    } catch (error) {
      showErrorNotification(error);
    } finally { setDownloading(false); }
  }, [meta, record.sentAt]);

  return (
    <div className={cn(
      'rounded-xl border text-sm',
      settled
        ? 'border-emerald-200 bg-emerald-50/40'
        : 'border-border bg-card',
    )}>
      <div className="px-3 py-2.5 space-y-2">

        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="text-xs font-medium text-muted-foreground tabular-nums shrink-0">
            {fmtDate(record.sentAt)}
          </span>

          <span className={cn(
            'inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-semibold shrink-0',
            isAdvancePayment
              ? 'bg-blue-50 text-blue-700 border border-blue-200'
              : 'bg-slate-100 text-slate-600 border border-slate-200',
          )}>
            {isAdvancePayment ? 'Ứng lương' : 'Bảng công'}
          </span>

          {settled ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-semibold text-emerald-700 shrink-0">
              <CheckCircle2 className="h-3 w-3" />
              Đã đối soát{record.settledAt ? ` · ${fmtDateShort(record.settledAt)}` : ''}
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 text-xs font-semibold text-amber-700 shrink-0">
              Chờ đối soát
            </span>
          )}

          <div className="ml-auto flex items-center gap-1.5 shrink-0">
            {meta.saoKeAssetId && (
              <Button
                variant="ghost"
                size="sm"
                className="h-6 gap-1 px-2 text-xs text-muted-foreground hover:text-foreground"
                onClick={handleDownload}
                disabled={downloading}
              >
                {downloading
                  ? <Loader2 className="h-3 w-3 animate-spin" />
                  : <Download className="h-3 w-3" />}
                Tải
              </Button>
            )}
            {!settled && (
              <Button
                size="sm"
                className="h-6 gap-1 bg-emerald-600 px-2.5 text-xs text-white hover:bg-emerald-700"
                onClick={() => onSettle(record.id)}
                disabled={isSettling}
              >
                {isSettling
                  ? <Loader2 className="h-3 w-3 animate-spin" />
                  : <CheckCircle2 className="h-3 w-3" />}
                Đối soát
              </Button>
            )}
          </div>
        </div>

        <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
          <span className="text-xs uppercase tracking-wide text-muted-foreground">
            {isAdvancePayment ? 'Tổng ứng' : 'Đã trả'} <span className="text-sm font-semibold normal-case tracking-normal text-foreground tabular-nums">{fmt(meta.totalAmount)}</span>
          </span>
          {!isAdvancePayment ? (
            meta.feePercentage > 0 && (
              <span className="text-xs uppercase tracking-wide text-muted-foreground">
                Phí {(meta.feePercentage * 100).toFixed(0)}% <span className="text-sm font-semibold normal-case tracking-normal text-foreground tabular-nums">{fmt(meta.feeAmount)}</span>
              </span>
            )
          ) : meta.feeAmount > 0 ? (
            <span className="text-xs uppercase tracking-wide text-muted-foreground">
              Phí dịch vụ <span className="text-sm font-semibold normal-case tracking-normal text-foreground tabular-nums">{fmt(meta.feeAmount)}</span>
            </span>
          ) : null}
          <span className="text-xs uppercase tracking-wide text-muted-foreground">
            Phải thu <span className="text-sm font-semibold normal-case tracking-normal text-primary tabular-nums">{fmt(meta.totalWithFee)}</span>
          </span>
        </div>

        <div className="space-y-0.5">
          {toList.length > 0 && (
            <div className="flex flex-wrap items-baseline gap-x-1.5 gap-y-0.5 text-xs">
              <span className="shrink-0 font-medium text-muted-foreground">To:</span>
              {toList.map((a) => (
                <span key={a} className="text-foreground/80">{a}</span>
              ))}
            </div>
          )}
          {ccList.length > 0 && (
            <div className="flex flex-wrap items-baseline gap-x-1.5 gap-y-0.5 text-xs">
              <span className="shrink-0 font-medium text-muted-foreground">CC:</span>
              {ccList.map((a) => (
                <span key={a} className="text-foreground/80">{a}</span>
              ))}
            </div>
          )}
          {bccList.length > 0 && (
            <div className="flex flex-wrap items-baseline gap-x-1.5 gap-y-0.5 text-xs">
              <span className="shrink-0 font-medium text-muted-foreground">BCC:</span>
              {bccList.map((a) => (
                <span key={a} className="text-foreground/80">{a}</span>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});

function RowSkeleton() {
  return (
    <div className="rounded-xl border border-border px-3 py-2.5 space-y-2">
      <div className="flex items-center gap-2">
        <Skeleton className="h-3.5 w-32" />
        <Skeleton className="h-4 w-20 rounded-full" />
        <div className="ml-auto flex gap-1.5">
          <Skeleton className="h-6 w-10 rounded-xl" />
          <Skeleton className="h-6 w-16 rounded-xl" />
        </div>
      </div>
      <div className="flex gap-4">
        <Skeleton className="h-4 w-28" />
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-4 w-28" />
      </div>
      <Skeleton className="h-3 w-48" />
    </div>
  );
}

export const SaoKeHistoryDialog = memo(function SaoKeHistoryDialog({
  open,
  onOpenChange,
}: SaoKeHistoryDialogProps) {
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const [localSettled, setLocalSettled] = useState<Set<number>>(() => new Set());
  const standaloneUploadRef = useRef<HTMLInputElement>(null);

  const { data, isLoading, isError, refetch, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useEmailHistory({ enabled: open, pageSize: 30 });

  const settleMutation = useSettleFromEmailHistory();
  const standaloneUploadMutation = useStandaloneSettlementUpload();

  const records = useMemo<EmailHistoryRecord[]>(() => {
    if (!data?.pages) return [];
    return data.pages
      .flatMap((p) => p.data ?? [])
      .filter((r) =>
        (r.type === 'payroll_report' || r.type === 'advance_payment_report') &&
        r.payrollMeta?.saoKeAssetId
      );
  }, [data]);

  const isEmpty = !isLoading && !isError && records.length === 0;

  const handleConfirm = useCallback(() => {
    if (confirmId == null) return;
    const id = confirmId;
    settleMutation.mutate(id, {
      onSuccess: () => setLocalSettled((prev) => new Set(prev).add(id)),
      onSettled: () => setConfirmId(null),
    });
  }, [confirmId, settleMutation]);

  const handleStandaloneUpload = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      standaloneUploadMutation.mutate(file);
      e.target.value = '';
    }
  }, [standaloneUploadMutation]);

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="flex max-h-[92dvh] w-full max-w-xl flex-col gap-0 overflow-hidden" contentPadding="none" hideCloseButton>

          <DialogNavyHeader
            title="Lịch sử sao kê"
            description="Xác nhận nhận tiền · tải file · đối soát"
            action={
              <>
                <Button
                  variant="outline"
                  size="sm"
                  className="min-h-11 gap-1.5 border-white/20 bg-white/10 text-xs text-white hover:bg-white/20 hover:text-white"
                  onClick={() => standaloneUploadRef.current?.click()}
                  disabled={standaloneUploadMutation.isPending}
                >
                  {standaloneUploadMutation.isPending
                    ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    : <Upload className="h-3.5 w-3.5" />}
                  Nhập đối soát
                </Button>
                <input
                  ref={standaloneUploadRef}
                  type="file"
                  accept=".xlsx"
                  className="hidden"
                  onChange={handleStandaloneUpload}
                />
              </>
            }
          />

          <div className="flex-1 overflow-y-auto">
            <div className="space-y-1.5 p-3 pt-1">

              {isLoading && [0, 1, 2, 3].map((i) => <RowSkeleton key={i} />)}

              {!isLoading && isError && (
                <div className="flex flex-col items-center gap-2 py-10 text-center">
                  <AlertCircle className="h-7 w-7 text-destructive/60" />
                  <p className="text-sm text-muted-foreground">Không thể tải dữ liệu</p>
                  <Button variant="outline" size="sm" className="min-h-11 gap-1.5" onClick={() => refetch()}>
                    <RefreshCw className="h-3.5 w-3.5" /> Thử lại
                  </Button>
                </div>
              )}

              {isEmpty && (
                <div className="flex flex-col items-center gap-2 py-10 text-center">
                  <FileText className="h-8 w-8 text-muted-foreground/30" />
                  <p className="text-sm font-medium">Chưa có sao kê nào</p>
                  <p className="text-xs text-muted-foreground">
                    Nhấn <span className="font-medium text-foreground">Gửi sao kê</span> để tạo báo cáo
                  </p>
                </div>
              )}

              {!isLoading && !isError && records.map((r) => (
                <SaoKeRow
                  key={r.id}
                  record={r}
                  onSettle={(id) => setConfirmId(id)}
                  isSettling={settleMutation.isPending && confirmId === r.id}
                  localSettled={localSettled.has(r.id)}
                />
              ))}

              {hasNextPage && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="min-h-11 w-full text-xs text-muted-foreground"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage
                    ? <><Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />Đang tải...</>
                    : 'Xem thêm'}
                </Button>
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog open={confirmId != null} onOpenChange={(v) => !v && setConfirmId(null)}>
        <AlertDialogContent className="max-w-xs rounded-xl">
          <AlertDialogHeader>
            <AlertDialogTitle>Xác nhận đã nhận tiền?</AlertDialogTitle>
            <AlertDialogDescription className="text-sm">
              Hệ thống sẽ xử lý đối soát cho tất cả timesheets trong sao kê này. Không thể hoàn tác.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={settleMutation.isPending} className="min-h-11">Hủy</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirm}
              disabled={settleMutation.isPending}
              className="min-h-11 bg-emerald-600 hover:bg-emerald-700"
            >
              {settleMutation.isPending
                ? <><Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />Đang xử lý...</>
                : 'Xác nhận'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
});
