import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRightLeft, RefreshCw, Wallet as WalletIcon, Loader2 } from 'lucide-react';
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
import { toast } from 'sonner';

const BALANCE_QUERY_KEY = ['wallet', 'balance'] as const;

function formatVND(value: number): string {
  return new Intl.NumberFormat('vi-VN').format(value) + '\u00a0đ';
}

function BalanceFigure({ value }: { value: number | undefined }) {
  if (value == null) return <span>—</span>;
  const formatted = new Intl.NumberFormat('vi-VN').format(value);
  return (
    <>
      {formatted}
      <span className="text-2xl font-semibold text-slate-400 ml-1">đ</span>
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
      if (result.provider_balance === result.local_balance) {
        toast.success(`Số dư khớp: ${formatVND(result.local_balance)}`);
        invalidateAll();
      } else {
        setMismatch({ provider: result.provider_balance, local: result.local_balance });
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
    <div className="min-h-dvh bg-[#0f172a]">
      {/* Dark hero zone */}
      <div className="px-4 pt-5 pb-10">
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2">
            <WalletIcon className="h-5 w-5 text-slate-400" />
            <h1 className="text-[17px] font-semibold text-slate-100 tracking-tight">Ví điện tử</h1>
          </div>
          <button
            type="button"
            onClick={handleSync}
            disabled={syncing}
            className="flex items-center gap-1.5 px-3 h-9 rounded-lg bg-white/5 border border-white/10 text-slate-300 text-xs font-medium hover:bg-white/10 active:bg-white/[0.04] transition-colors"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${syncing ? 'animate-spin' : ''}`} />
            Đồng bộ
          </button>
        </div>

        <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400 mb-2">Số dư khả dụng</p>
        <p className="text-[34px] font-bold text-white tabular-nums leading-none mb-2 tracking-tight">
          <BalanceFigure value={balance?.available} />
        </p>
        {asOf && <p className="text-xs text-slate-400 mb-5 tabular-nums tracking-wide">Cập nhật {asOf}</p>}

        <button
          type="button"
          onClick={() => setDisbursementOpen(true)}
          className="w-full h-11 flex items-center justify-center gap-2 rounded-xl bg-white/5 border border-white/10 text-slate-300 hover:bg-white/10 text-sm font-medium transition-colors"
        >
          <ArrowRightLeft className="h-4 w-4" />Chuyển tiền
        </button>
      </div>

      {/* Light transaction panel */}
      <div className="bg-background rounded-t-3xl -mt-4 min-h-[60dvh] shadow-[0_-8px_32px_rgba(0,0,0,0.2)]">
        <div className="px-4 pt-5 pb-20">
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
              <div className="grid grid-cols-3 gap-2">
                <div className="rounded-md bg-slate-50 p-2">
                  <p className="text-[10px] text-slate-500">Nhà cung cấp</p>
                  <p className="font-semibold tabular-nums text-slate-900">{formatVND(mismatch?.provider ?? 0)}</p>
                </div>
                <div className="rounded-md bg-slate-50 p-2">
                  <p className="text-[10px] text-slate-500">Hệ thống</p>
                  <p className="font-semibold tabular-nums text-slate-900">{formatVND(mismatch?.local ?? 0)}</p>
                </div>
                <div className={`rounded-md p-2 ${diff > 0 ? 'bg-emerald-50' : 'bg-rose-50'}`}>
                  <p className="text-[10px] text-slate-500">Chênh lệch</p>
                  <p className={`font-semibold tabular-nums ${diff > 0 ? 'text-emerald-700' : 'text-rose-700'}`}>
                    {diff > 0 ? '+' : ''}{formatVND(diff)}
                  </p>
                </div>
              </div>
              <span>Tạo bản ghi nạp tiền <strong className={diff > 0 ? 'text-emerald-700' : 'text-rose-700'}>{diff > 0 ? '+' : ''}{formatVND(diff)}</strong> để khớp số dư?</span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="text-xs h-9" disabled={adjusting}>Bỏ qua</AlertDialogCancel>
            <AlertDialogAction onClick={handleAdjust} disabled={adjusting} className="bg-[#2a3b58] hover:bg-[#1e293b] text-xs h-9">
              {adjusting ? <><Loader2 className="h-3 w-3 mr-1.5 animate-spin" />Đang xử lý...</> : 'Điều chỉnh'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
