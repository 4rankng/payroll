import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { RefreshCw, AlertTriangle, Loader2, CheckCircle2 } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { walletService } from "@/services/api/wallet.service";
import { formatCurrency } from "@/utils/formatters";
import { showErrorNotification } from "@/utils/error-handler";
import { useDisbursementSettings } from "@/hooks/useDisbursementSettings";

interface WalletBalanceCardProps {
  monthlyProviderFee?: number;
  totalProviderFee?: number;
  className?: string;
  compact?: boolean;
}

export function WalletBalanceCard({ monthlyProviderFee, totalProviderFee, className, compact = false }: WalletBalanceCardProps) {
  const queryClient = useQueryClient();
  const [isSyncing, setIsSyncing] = useState(false);
  const [mismatch, setMismatch] = useState<{ provider: number; local: number } | null>(null);
  const [adjusting, setAdjusting] = useState(false);

  const { data: settings, isLoading } = useDisbursementSettings();

  const available = settings?.internal_balance?.available ?? 0;
  const isLow = available < 1_000_000;
  const showProviderBalance = !!settings?.active_provider?.capabilities?.balance_inquiry;
  const providerBalance = settings?.provider_balance;
  const divergence =
    showProviderBalance && providerBalance
      ? Math.abs(providerBalance.amount - available)
      : 0;
  const hasDivergence = divergence > 100_000;
  const diff = mismatch ? mismatch.provider - mismatch.local : 0;

  // Refresh this card's data source and invalidate the canonical wallet query
  // so any concurrently-open WalletPage/tab stays consistent after a sync.
  // Fire-and-forget with errors swallowed: a refetch failure here must not be
  // reported as a sync/adjust failure (the mutation already succeeded).
  const refreshBalances = () => {
    queryClient.refetchQueries({ queryKey: ["disbursement-settings"] }).catch(() => {});
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
  };

  const handleSync = async () => {
    setIsSyncing(true);
    try {
      const result = await walletService.syncBalance();
      if (result.provider_balance === result.local_balance) {
        toast.success(`Số dư khớp: ${formatCurrency(result.local_balance)}`);
        refreshBalances();
      } else {
        // Backend skipped auto-adjust (e.g. in-flight payments). Surface the
        // mismatch so the user can reconcile — mirrors the WalletPage behavior.
        setMismatch({ provider: result.provider_balance, local: result.local_balance });
      }
    } catch (err) {
      showErrorNotification(err, "Không thể đồng bộ số dư");
    } finally {
      setIsSyncing(false);
    }
  };

  const handleAdjust = async () => {
    if (!mismatch) return;
    setAdjusting(true);
    try {
      await walletService.adjustBalance({
        amount: diff,
        reason: `Số dư nhà cung cấp: ${formatCurrency(mismatch.provider)}, Hệ thống: ${formatCurrency(mismatch.local)}, Chênh lệch: ${formatCurrency(diff)}`,
      });
      toast.success("Đã điều chỉnh số dư thành công");
      setMismatch(null);
      refreshBalances();
    } catch (err) {
      showErrorNotification(err, "Điều chỉnh số dư thất bại");
    } finally {
      setAdjusting(false);
    }
  };

  return (
    <section
      aria-label="Ví tiền"
      className={cn(
        "relative overflow-hidden p-[22px] px-6 text-white",
        "bg-[#06452E]",
        compact && "p-3.5",
        className
      )}
    >
      {/* Grid pattern overlay */}
      <div 
        className="pointer-events-none absolute inset-0"
        style={{
          backgroundImage: 'linear-gradient(rgba(183,228,202,0.10) 1px, transparent 1px), linear-gradient(90deg, rgba(183,228,202,0.10) 1px, transparent 1px)',
          backgroundSize: '28px 28px'
        }}
      />

      <div className="relative flex h-full flex-col justify-between">
        {isLoading ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="h-3 w-24 animate-pulse rounded bg-white/10" />
              <div className="h-7 w-7 animate-pulse rounded-lg bg-white/10" />
            </div>
            <div className="h-9 w-44 animate-pulse rounded bg-white/10" />
            <div className="h-3 w-36 animate-pulse rounded bg-white/10" />
            <div className="grid grid-cols-2 gap-3 pt-4">
              <div className="h-12 animate-pulse rounded-lg bg-white/10" />
              <div className="h-12 animate-pulse rounded-lg bg-white/10" />
            </div>
          </div>
        ) : settings ? (
          <>
            {/* Header */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <div className="flex items-center gap-2">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-300 shadow-[0_0_0_3px_rgba(110,231,183,0.18)]" />
                  <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-emerald-200">
                    Ví tiền
                  </span>
                </div>
                <button
                  onClick={handleSync}
                  disabled={isSyncing}
                  title="Đồng bộ số dư với nhà cung cấp"
                  aria-label="Đồng bộ số dư"
                  className="flex h-7 w-7 items-center justify-center rounded-lg border border-white/10 bg-white/5 text-white/70 transition-colors hover:bg-white/10 disabled:opacity-50"
                >
                  <RefreshCw className={cn("h-3.5 w-3.5", isSyncing && "animate-spin")} />
                </button>
              </div>

              {/* Main Balance */}
              <div className={cn(
                "mt-1.5 max-w-full break-words font-financial font-semibold leading-[1.08] tracking-normal tabular-nums",
                compact ? "text-[clamp(1.5rem,7.5vw,1.75rem)]" : "text-[clamp(1.75rem,8vw,2rem)]",
                isLow ? "text-red-400" : "text-white"
              )}>
                {formatCurrency(available).replace('₫', '')}
                <span className={cn("ml-1 font-medium", compact ? "text-base" : "text-lg", isLow ? "text-red-400/80" : "text-emerald-200")}>₫</span>
              </div>

              {hasDivergence && showProviderBalance && providerBalance && (
                <div className="mt-2 inline-flex items-center gap-1 rounded border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">
                  <AlertTriangle className="h-3.5 w-3.5" />
                  NCC: {formatCurrency(providerBalance.amount)}
                </div>
              )}

              {isLow && !hasDivergence && (
                <div className="mt-1.5 text-xs text-red-400/80">
                  Số dư thấp — cân nhắc nạp thêm
                </div>
              )}
            </div>

            {/* Meta Grid */}
            <div className={cn("grid grid-cols-2", compact ? "mt-3 gap-2" : "mt-[22px] gap-3.5")}>
              <div className="min-w-0">
                <div className={cn("font-semibold uppercase tracking-[0.1em] text-white/45", compact ? "text-[9.5px]" : "text-[10.5px]")}>
                  Tổng phí trả
                </div>
                <div className={cn("mt-1 break-words font-financial font-medium leading-snug text-white tabular-nums", compact ? "text-[13px]" : "text-[15px]")}>
                  {formatCurrency(totalProviderFee ?? 0)}
                </div>
              </div>
              <div className="min-w-0">
                <div className={cn("font-semibold uppercase tracking-[0.1em] text-white/45", compact ? "text-[9.5px]" : "text-[10.5px]")}>
                  Phí tháng này
                </div>
                <div className={cn("mt-1 break-words font-financial font-medium leading-snug text-white tabular-nums", compact ? "text-[13px]" : "text-[15px]")}>
                  {formatCurrency(monthlyProviderFee ?? 0)}
                </div>
              </div>
            </div>
          </>
        ) : (
          <div className="flex h-24 items-center justify-center">
            <p className="text-sm text-white/50">Không thể tải số dư</p>
          </div>
        )}
      </div>

      {/* Mismatch reconciliation — mirrors WalletPage */}
      <AlertDialog open={!!mismatch} onOpenChange={(open) => !open && setMismatch(null)}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogHeader>
            <div className="flex items-center gap-2.5 mb-1">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-50 border border-amber-200">
                <RefreshCw className="h-3.5 w-3.5 text-amber-600" />
              </div>
              <AlertDialogTitle className="text-sm">Phát hiện chênh lệch số dư</AlertDialogTitle>
            </div>
            <AlertDialogDescription asChild>
              <div className="space-y-3">
                <div className="grid grid-cols-3 gap-2">
                  <div className="rounded-lg border border-border/60 bg-muted/40 p-2.5">
                    <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">Nhà cung cấp</p>
                    <p className="mt-0.5 font-financial text-sm font-bold tabular-nums text-foreground">
                      {formatCurrency(mismatch?.provider ?? 0)}
                    </p>
                  </div>
                  <div className="rounded-lg border border-border/60 bg-muted/40 p-2.5">
                    <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">Hệ thống</p>
                    <p className="mt-0.5 font-financial text-sm font-bold tabular-nums text-foreground">
                      {formatCurrency(mismatch?.local ?? 0)}
                    </p>
                  </div>
                  <div
                    className={cn(
                      "rounded-lg border p-2.5",
                      diff > 0 ? "border-emerald-200 bg-emerald-50" : "border-rose-200 bg-rose-50",
                    )}
                  >
                    <p className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">Chênh lệch</p>
                    <p
                      className={cn(
                        "mt-0.5 font-financial text-sm font-bold tabular-nums",
                        diff > 0 ? "text-emerald-700" : "text-rose-700",
                      )}
                    >
                      {diff > 0 ? "+" : ""}
                      {formatCurrency(diff)}
                    </p>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  Tạo bản ghi điều chỉnh{" "}
                  <strong className={cn("font-semibold", diff > 0 ? "text-emerald-700" : "text-rose-700")}>
                    {diff > 0 ? "+" : ""}
                    {formatCurrency(diff)}
                  </strong>{" "}
                  để khớp số dư với nhà cung cấp?
                </p>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="h-9 text-xs" disabled={adjusting}>
              Bỏ qua
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                // Prevent Radix from auto-closing the dialog on click so the
                // in-progress spinner shows and an adjust error stays
                // recoverable in-place. Close is driven by setMismatch(null)
                // on success instead.
                e.preventDefault();
                handleAdjust();
              }}
              disabled={adjusting}
              className="h-9 text-xs gap-1.5"
            >
              {adjusting ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin" />
                  Đang xử lý...
                </>
              ) : (
                <>
                  <CheckCircle2 className="h-3 w-3" />
                  Điều chỉnh
                </>
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
