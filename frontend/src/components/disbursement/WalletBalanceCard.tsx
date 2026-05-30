import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { RefreshCw, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { walletService } from "@/services/api/wallet.service";
import { formatCurrency } from "@/lib/validation";
import { useDisbursementSettings } from "@/hooks/useDisbursementSettings";

interface WalletBalanceCardProps {
  monthlyProviderFee?: number;
  totalProviderFee?: number;
  className?: string;
}

export function WalletBalanceCard({ monthlyProviderFee, totalProviderFee, className }: WalletBalanceCardProps) {
  const queryClient = useQueryClient();
  const [isSyncing, setIsSyncing] = useState(false);

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

  const handleSync = async () => {
    setIsSyncing(true);
    try {
      await walletService.syncBalance();
      await queryClient.refetchQueries({ queryKey: ["disbursement-settings"] });
    } finally {
      setIsSyncing(false);
    }
  };

  return (
    <section
      aria-label="Ví tiền"
      className={cn(
        "relative overflow-hidden p-[22px] px-6 text-white",
        "bg-[radial-gradient(120%_80%_at_0%_0%,rgba(29,78,216,0.06)_0%,transparent_60%),linear-gradient(180deg,#0E1729_0%,#1A2542_100%)]",
        className
      )}
    >
      {/* Glow effect top right */}
      <div className="pointer-events-none absolute -right-10 -top-10 h-[180px] w-[180px] rounded-full bg-[radial-gradient(circle,rgba(110,168,255,0.18)_0%,transparent_70%)]" />
      
      {/* Grid pattern overlay */}
      <div 
        className="pointer-events-none absolute inset-0"
        style={{
          backgroundImage: 'linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px)',
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
                  <span className="h-1.5 w-1.5 rounded-full bg-[#6FA8FF] shadow-[0_0_0_3px_rgba(111,168,255,0.18)]" />
                  <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-[#6FA8FF]">
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
                "mt-1.5 whitespace-nowrap font-financial text-[32px] font-semibold leading-[1.05] tracking-[-0.02em]",
                isLow ? "text-red-400" : "text-white"
              )}>
                {formatCurrency(available).replace('₫', '')}
                <span className={cn("ml-1 text-lg font-medium", isLow ? "text-red-400/80" : "text-[#6FA8FF]")}>₫</span>
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
            <div className="mt-[22px] grid grid-cols-2 gap-3.5">
              <div>
                <div className="text-[10.5px] font-semibold uppercase tracking-[0.1em] text-white/45">
                  Tổng phí trả
                </div>
                <div className="mt-1 font-financial text-[15px] font-medium text-white">
                  {formatCurrency(totalProviderFee ?? 0)}
                </div>
              </div>
              <div>
                <div className="text-[10.5px] font-semibold uppercase tracking-[0.1em] text-white/45">
                  Phí tháng này
                </div>
                <div className="mt-1 font-financial text-[15px] font-medium text-white">
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
    </section>
  );
}
