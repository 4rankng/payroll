import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { RefreshCw, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { walletService } from "@/services/api/wallet.service";
import { formatCurrency } from "@/utils/formatters";
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
        "bg-[#101D35]",
        compact && "p-3.5",
        className
      )}
    >
      {/* Grid pattern overlay */}
      <div 
        className="pointer-events-none absolute inset-0"
        style={{
          backgroundImage: 'linear-gradient(#1B2A47 1px, transparent 1px), linear-gradient(90deg, #1B2A47 1px, transparent 1px)',
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
                "mt-1.5 whitespace-nowrap font-financial font-semibold leading-[1.05] tracking-[-0.02em]",
                compact ? "text-[24px]" : "text-[32px]",
                isLow ? "text-red-400" : "text-white"
              )}>
                {formatCurrency(available).replace('₫', '')}
                <span className={cn("ml-1 font-medium", compact ? "text-base" : "text-lg", isLow ? "text-red-400/80" : "text-[#6FA8FF]")}>₫</span>
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
              <div>
                <div className={cn("font-semibold uppercase tracking-[0.1em] text-white/45", compact ? "text-[9.5px]" : "text-[10.5px]")}>
                  Tổng phí trả
                </div>
                <div className={cn("mt-1 font-financial font-medium text-white", compact ? "text-[13px]" : "text-[15px]")}>
                  {formatCurrency(totalProviderFee ?? 0)}
                </div>
              </div>
              <div>
                <div className={cn("font-semibold uppercase tracking-[0.1em] text-white/45", compact ? "text-[9.5px]" : "text-[10.5px]")}>
                  Phí tháng này
                </div>
                <div className={cn("mt-1 font-financial font-medium text-white", compact ? "text-[13px]" : "text-[15px]")}>
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
