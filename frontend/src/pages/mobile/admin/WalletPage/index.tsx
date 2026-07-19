import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRightLeft, RefreshCw, Wallet as WalletIcon, Loader2, TrendingDown } from 'lucide-react';
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import WalletTransactionsList from '@/components/wallet/WalletTransactionsList';
import CreateManualDisbursementDialog from '@/components/wallet/CreateManualDisbursementDialog';
import { walletService } from '@/services/api/wallet.service';
import type { WalletBalance } from '@/types/api/wallet.types';
import { showErrorNotification } from '@/utils/error-handler';
import { formatVietnameseDateTime } from '@/utils/vietnamese';
import { formatCurrency as formatVND } from '@/utils/formatters';
import { toast } from 'sonner';

const BALANCE_QUERY_KEY = ['wallet', 'balance'] as const;


function BalanceFigure({ value }: { value: number | undefined }) {
  if (value == null) return <span>—</span>;
  const formatted = new Intl.NumberFormat('vi-VN').format(value);
  return (
    <>
      {formatted}
      <span className="text-2xl font-semibold text-neutral-content/45 ml-1">đ</span>
    </>
  );
}

export default function WalletPageMobile() {
  const queryClient = useQueryClient();
  const [disbursementOpen, setDisbursementOpen] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [adjusting, setAdjusting] = useState(false);
  const [mismatch, setMismatch] = useState<{ provider: number; local: number } | null>(null);

  const { data: balance } = useQuery<WalletBalance>({
    queryKey: BALANCE_QUERY_KEY,
    queryFn: () => walletService.getBalance(),
  });

  const invalidateAll = () => { queryClient.invalidateQueries({ queryKey: ['wallet'] }); };
  const asOf = balance?.as_of ? formatVietnameseDateTime(balance.as_of) : null;
  const diff = mismatch ? mismatch.provider - mismatch.local : 0;

  const handleSync = async () => {
    setSyncing(true);
    try {
      const result = await walletService.syncBalance();
      await queryClient.invalidateQueries({ queryKey: ['wallet'] });

      if (result.provider_balance === result.local_balance) {
        toast.success(
          result.adjusted
            ? `Đã đồng bộ và điều chỉnh: ${formatVND(result.local_balance)}`
            : `Số dư khớp: ${formatVND(result.local_balance)}`,
        );
      } else {
        setMismatch({ provider: result.provider_balance, local: result.local_balance });
        toast.warning('Đã đồng bộ, cần xử lý chênh lệch số dư');
      }
    } catch (err) {
      showErrorNotification(err, 'Không thể đồng bộ số dư');
    } finally {
      setSyncing(false);
    }
  };

  const handleAdjust = async () => {
    if (!mismatch) return;
    setAdjusting(true);
    try {
      await walletService.adjustBalance({
        amount: diff,
        reason: `Số dư nhà cung cấp: ${formatVND(mismatch.provider)}, Hệ thống: ${formatVND(mismatch.local)}, Chênh lệch: ${formatVND(diff)}`,
      });
      toast.success('Đã điều chỉnh số dư thành công');
      setMismatch(null);
      invalidateAll();
    } catch (err) {
      showErrorNotification(err, 'Điều chỉnh số dư thất bại');
    } finally {
      setAdjusting(false);
    }
  };

  return (
    <div className="ct-hero min-h-dvh bg-neutral text-neutral-content">
      {/* Dark hero zone */}
      <div className="ct-hero-content px-4 pt-[var(--mobile-header-top-padding,calc(env(safe-area-inset-top)+1rem))] pb-10">
        <div className="relative z-10 flex items-start justify-between gap-3 mb-6">
          <div className="min-w-0 flex flex-1 items-center gap-2 pt-1">
            <WalletIcon className="h-5 w-5 shrink-0 text-neutral-content/60" />
            <h1 className="truncate text-[17px] font-semibold text-neutral-content tracking-tight">Ví điện tử</h1>
          </div>
          <button
            type="button"
            onClick={handleSync}
            disabled={syncing}
            aria-label="Đồng bộ số dư ví"
            className="ct-btn ct-btn-ghost ct-btn-sm min-h-11 shrink-0 gap-1.5 rounded-lg border border-base-100/10 bg-base-100/5 px-3 text-xs font-medium normal-case text-neutral-content/80 shadow-none hover:bg-base-100/10 disabled:opacity-60"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${syncing ? 'animate-spin' : ''}`} />
            <span>{syncing ? 'Đang đồng bộ' : 'Đồng bộ'}</span>
          </button>
        </div>

        <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-neutral-content/60 mb-2">Số dư khả dụng</p>
        <p className="text-[34px] font-bold text-neutral-content tabular-nums leading-none mb-2 tracking-tight">
          <BalanceFigure value={balance?.available} />
        </p>
        {asOf && <p className="text-xs text-neutral-content/55 mb-5 tabular-nums tracking-wide">Cập nhật {asOf}</p>}

        {/* Pending-out stat tile — restores parity with desktop (was missing on mobile) */}
        <div className="ct-stat mb-4 flex-row items-center gap-3 rounded-xl border border-base-100/10 bg-base-100/5 px-3.5 py-3 shadow-none">
          <div className="ct-stat-figure flex h-9 w-9 items-center justify-center rounded-lg bg-warning/15">
            <TrendingDown className="h-4 w-4 text-warning" />
          </div>
          <div className="min-w-0 flex-1">
            <p className="ct-stat-title text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-content/55">Đang chi trả</p>
            <p className="ct-stat-value font-financial text-base font-bold tabular-nums text-neutral-content leading-tight">
              {balance ? formatVND(balance.pending_out) : '—'}
            </p>
          </div>
          <span className="text-[11px] text-neutral-content/45 leading-tight text-right max-w-[7rem]">
            Chuyển khoản đang xử lý
          </span>
        </div>

        <button
          type="button"
          onClick={() => setDisbursementOpen(true)}
          className="ct-btn ct-btn-primary ct-btn-block h-11 min-h-11 gap-2 rounded-xl text-sm font-semibold normal-case shadow-[0_12px_28px_-14px_rgba(8,120,62,0.8)]"
        >
          <ArrowRightLeft className="h-4 w-4" />Chuyển tiền
        </button>
      </div>

      {/* Light transaction panel */}
      <div className="ct-card rounded-t-3xl -mt-4 min-h-[60dvh] bg-base-100 shadow-lg">
        <div className="ct-card-body space-y-4 px-4 pt-5 pb-[calc(5rem+env(safe-area-inset-bottom))]">
          <WalletTransactionsList />
        </div>
      </div>

      <CreateManualDisbursementDialog
        open={disbursementOpen}
        onOpenChange={setDisbursementOpen}
        onSuccess={invalidateAll}
      />

      <AlertDialog open={!!mismatch} onOpenChange={(open) => !open && setMismatch(null)}>
        <AlertDialogContent className="max-w-sm">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-sm">Phát hiện chênh lệch số dư</AlertDialogTitle>
            <AlertDialogDescription className="text-xs space-y-2">
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                <div className="rounded-md bg-base-200 p-2">
                  <p className="text-[11px] text-muted-foreground">Nhà cung cấp</p>
                  <p className="font-semibold tabular-nums text-foreground">{formatVND(mismatch?.provider ?? 0)}</p>
                </div>
                <div className="rounded-md bg-base-200 p-2">
                  <p className="text-[11px] text-muted-foreground">Hệ thống</p>
                  <p className="font-semibold tabular-nums text-foreground">{formatVND(mismatch?.local ?? 0)}</p>
                </div>
                <div className={`rounded-md p-2 ${diff > 0 ? 'bg-success/10' : 'bg-error/10'}`}>
                  <p className="text-[11px] text-muted-foreground">Chênh lệch</p>
                  <p className={`font-semibold tabular-nums ${diff > 0 ? 'text-success' : 'text-error'}`}>
                    {diff > 0 ? '+' : ''}{formatVND(diff)}
                  </p>
                </div>
              </div>
              <span>Tạo bản ghi nạp tiền <strong className={diff > 0 ? 'text-success' : 'text-error'}>{diff > 0 ? '+' : ''}{formatVND(diff)}</strong> để khớp số dư?</span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="min-h-11 text-xs" disabled={adjusting}>Bỏ qua</AlertDialogCancel>
            <AlertDialogAction onClick={handleAdjust} disabled={adjusting} className="ct-btn ct-btn-primary min-h-11 h-11 border-0 bg-primary text-xs font-semibold text-primary-content shadow-none hover:bg-primary/90">
              {adjusting ? <><Loader2 className="h-3 w-3 mr-1.5 animate-spin" />Đang xử lý...</> : 'Điều chỉnh'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
