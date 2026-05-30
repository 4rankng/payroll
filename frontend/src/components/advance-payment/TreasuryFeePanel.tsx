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
}: TreasuryFeePanelProps) {
  return (
    <div className={cn("flex h-full flex-col justify-between p-5 sm:p-6", className)}>
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
              <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
                Phí thu kỳ này
              </span>
            </div>

            {/* Value */}
            <div className="mt-2 font-financial text-[28px] font-semibold leading-none tracking-[-0.02em] text-foreground">
              {formatCurrency(totalFeeEarned)}
            </div>

            {/* Rate */}
            <p className="mt-1.5 text-[12.5px] text-muted-foreground">
              <span className="mr-1.5 inline-flex items-center rounded-[5px] bg-muted px-[7px] py-[2px] font-financial text-[11px] font-semibold text-foreground/70">
                {feePercentage.toFixed(1)}%
              </span>
              trên giải ngân
            </p>
          </div>

          {/* Mini grid */}
          <div className="mt-4 grid grid-cols-2 gap-2.5">
            <div className="rounded-lg border border-border/70 bg-muted/30 p-3">
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                {totalPaid} yêu cầu
              </div>
              <div className="mt-0.5 font-financial text-sm font-medium text-foreground">
                {avgFeePerRequest > 0 ? `~${formatCurrency(avgFeePerRequest)}` : "—"} /yc
              </div>
            </div>
            <div className="rounded-lg border border-border/70 bg-muted/30 p-3">
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                {totalRequests} NV
              </div>
              <div className="mt-0.5 font-financial text-sm font-medium text-foreground">
                {avgFeePerEmployee > 0 ? `~${formatCurrency(avgFeePerEmployee)}` : "—"} /nv
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
});
