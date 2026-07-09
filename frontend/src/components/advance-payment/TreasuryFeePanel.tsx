import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/utils/formatters";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

export interface TreasuryFeePanelProps {
  totalFeeEarned: number;
  totalPaid: number;
  totalRequests: number;
  /** From backend — fee earned / paid amount * 100 */
  feePercentage: number;
  /** From backend — fee earned / total paid */
  avgFeePerRequest: number;
  /** From backend — fee earned / total requests */
  avgFeePerEmployee: number;
  isLoading?: boolean;
  className?: string;
  compact?: boolean;
}

/* ------------------------------------------------------------------ */
/*  Main component                                                     */
/* ------------------------------------------------------------------ */

export const TreasuryFeePanel = memo(function TreasuryFeePanel({
  totalFeeEarned,
  totalPaid,
  totalRequests,
  feePercentage,
  avgFeePerRequest,
  avgFeePerEmployee,
  isLoading = false,
  className,
  compact = false,
}: TreasuryFeePanelProps) {
  return (
    <div className={cn("flex h-full flex-col justify-between", compact ? "p-3.5" : "p-5 sm:p-6", className)}>
      {isLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-8 w-36" />
          <Skeleton className="h-3 w-32" />
          <div className="grid grid-cols-2 gap-2.5 pt-3">
            <Skeleton className="h-14 rounded-lg" />
            <Skeleton className="h-14 rounded-lg" />
          </div>
        </div>
      ) : (
        <>
          <div>
            {/* Label */}
            <div className="flex items-center gap-1.5">
              <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/40" />
              <span className={cn("font-bold uppercase tracking-[0.12em] text-muted-foreground", compact ? "text-[9.5px]" : "text-[10.5px]")}>
                Phí thu kỳ này
              </span>
            </div>

            {/* Value */}
            <div className={cn("mt-1.5 max-w-full break-words font-financial font-semibold leading-[1.08] tracking-normal text-foreground tabular-nums", compact ? "text-[clamp(1.375rem,7vw,1.625rem)]" : "text-[clamp(1.625rem,5vw,1.75rem)]")}>
              {formatCurrency(totalFeeEarned)}
            </div>

            {/* Rate */}
            <p className={cn("mt-1.5 text-muted-foreground", compact ? "text-[11px]" : "text-[12.5px]")}>
              <span className="mr-1.5 inline-flex items-center rounded-[5px] bg-muted px-[7px] py-[2px] font-financial text-[11px] font-semibold text-foreground/70">
                {feePercentage.toFixed(1)}%
              </span>
              trên giải ngân
            </p>
          </div>

          {/* Mini grid */}
          <div className={cn("grid grid-cols-2", compact ? "mt-3 gap-2" : "mt-4 gap-2.5")}>
            <div className={cn("rounded-lg border border-border/70 bg-muted/30", compact ? "p-2.5" : "p-3")}>
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                {totalPaid} yêu cầu
              </div>
              <div className="mt-0.5 break-words font-financial text-sm font-medium leading-snug text-foreground tabular-nums">
                {avgFeePerRequest > 0 ? `~${formatCurrency(avgFeePerRequest)}` : "—"} /yc
              </div>
            </div>
            <div className={cn("rounded-lg border border-border/70 bg-muted/30", compact ? "p-2.5" : "p-3")}>
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                {totalRequests} NV
              </div>
              <div className="mt-0.5 break-words font-financial text-sm font-medium leading-snug text-foreground tabular-nums">
                {avgFeePerEmployee > 0 ? `~${formatCurrency(avgFeePerEmployee)}` : "—"} /nv
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
});
