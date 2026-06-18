import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/utils/formatters";
import { useCountUp } from "@/hooks/useCountUp";

export interface AdvPartnerHeroStripProps {
  totalPaidAmount: number;
  totalAmount: number;
  totalFeeEarned: number;
  totalPaid: number;
  totalRequests: number;
  totalCancelled: number;
  avgProcessingTimeSecs: number;
  completedUnder30s: number;
  /** From backend — totalPaidAmount / totalAmount * 100.
   *  Optional: if missing/NaN, computed locally to avoid "NaN%" rendering. */
  disbursementPercentage?: number;
  isLoading?: boolean;
  className?: string;
  compact?: boolean;
}

const AnimatedCurrency = memo(function AnimatedCurrency({
  target,
}: {
  target: number;
}) {
  const animated = useCountUp(target, 700);
  return (
    <span className="font-financial tabular-nums tracking-tight">
      {formatCurrency(animated)}
    </span>
  );
});

export const AdvPartnerHeroStrip = memo(function AdvPartnerHeroStrip({
  totalPaidAmount,
  totalAmount,
  disbursementPercentage,
  isLoading = false,
  className,
  compact = false,
}: AdvPartnerHeroStripProps) {
  // Compute pct safely: prefer backend-provided value, fall back to local
  // calculation, clamp to [0,100], and guard against NaN/undefined which
  // previously rendered as "NaN%" on the mobile advance-payments page.
  const computedPct =
    typeof disbursementPercentage === 'number' && Number.isFinite(disbursementPercentage)
      ? disbursementPercentage
      : totalAmount > 0
        ? (totalPaidAmount / totalAmount) * 100
        : 0;
  const pct = Math.max(0, Math.min(computedPct, 100));

  return (
    <section
      aria-label="Giải ngân"
      className={cn(
        "relative flex flex-col justify-between",
        compact ? "p-3.5" : "p-[22px] px-7",
        "rounded-xl border border-[#D8E2EE] bg-white shadow-[0_1px_2px_0_rgb(15_23_42/0.04)]",
        className,
      )}
    >
      {isLoading ? (
        <div className="flex h-full flex-col space-y-4">
          <div className="flex justify-between">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-6 w-16" />
          </div>
          <Skeleton className="mt-2 h-12 w-48" />
          <Skeleton className="mt-1 h-4 w-40" />
          <div className="mt-auto">
            <Skeleton className="h-2 w-full rounded-full" />
            <div className="mt-2 flex justify-between">
              <Skeleton className="h-3 w-10" />
              <Skeleton className="h-3 w-10" />
            </div>
          </div>
        </div>
      ) : (
        <>
          <div>
            <div className="flex items-center justify-between">
              <div className={cn("flex items-center gap-2 font-bold uppercase tracking-[0.12em] text-muted-foreground", compact ? "text-[9.5px]" : "text-[10.5px]")}>
                <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/40" />
                Giải ngân kỳ này
              </div>
              <div className={cn("rounded-md bg-muted font-financial font-semibold text-foreground/70", compact ? "px-1.5 py-0.5 text-[11px]" : "px-2 py-1 text-xs")}>
                {pct.toFixed(1)}%
              </div>
            </div>
            <div className={cn("mt-1.5 font-financial font-semibold leading-none tracking-tight text-foreground", compact ? "text-[24px]" : "text-4xl")}>
              <AnimatedCurrency target={totalPaidAmount} />
            </div>

            <div className={cn("mt-1 text-muted-foreground", compact ? "text-[11px]" : "text-[12.5px]")}>
              Hạn mức <span className="font-financial font-medium text-foreground">{formatCurrency(totalAmount)}</span>
            </div>
          </div>
          <div className={cn(compact ? "mt-3" : "mt-[18px]")}>
            <div className={cn("relative overflow-hidden rounded-full bg-muted", compact ? "h-1.5" : "h-2")}>
              <div
                className="relative h-full rounded-full bg-gradient-to-r from-emerald-600 to-emerald-400 transition-all duration-700 ease-out"
                style={{ width: `${pct}%` }}
              >
                <div className="absolute -right-0.5 -top-[2px] h-3 w-1 rounded-full bg-white shadow-[0_0_0_1px_#10b981]" />
              </div>
            </div>
            <div className={cn("flex justify-between font-financial text-[10.5px] text-muted-foreground", compact ? "mt-1.5" : "mt-2")}>
              <span>0%</span>
              <span>100%</span>
            </div>
          </div>
        </>
      )}
    </section>
  );
});
