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
    <div
      className={cn(
        "relative flex h-full flex-col justify-center",
        compact ? "p-3.5" : "p-5 sm:p-6",
        "bg-[radial-gradient(90%_70%_at_50%_0%,rgba(8,120,62,0.05),transparent_70%)]",
        className,
      )}
    >
      {isLoading ? (
        <div className="space-y-2.5">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-3 w-44" />
        </div>
      ) : (
        <>
          {/* Label */}
          <div className="flex items-center gap-1.5">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.15)]" />
            <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
              Phí thu kỳ này
            </span>
          </div>

          {/* Value + rate chip */}
          <div className="mt-1.5 flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
            <span className="max-w-full break-words font-financial text-[22px] font-semibold leading-[1.08] tracking-normal text-foreground tabular-nums">
              {formatCurrency(totalFeeEarned)}
            </span>
            <span className="rounded bg-emerald-50 px-1.5 py-px font-financial text-[11px] font-semibold text-emerald-700 ring-1 ring-emerald-200/60">
              {feePercentage.toFixed(1)}%
            </span>
            <span className="text-[11px] text-muted-foreground">trên giải ngân</span>
          </div>

          {/* Micro stats — per-request / per-employee averages with counts */}
          <div className="mt-1.5 flex flex-wrap items-baseline gap-x-3 gap-y-0.5 text-[11px] leading-snug text-muted-foreground">
            <span className="whitespace-nowrap">
              <span className="font-financial font-semibold text-foreground/75 tabular-nums">
                {avgFeePerRequest > 0 ? `~${formatCurrency(avgFeePerRequest)}` : "—"}
              </span>{" "}
              /yc · {totalPaid} yc
            </span>
            <span className="whitespace-nowrap">
              <span className="font-financial font-semibold text-foreground/75 tabular-nums">
                {avgFeePerEmployee > 0 ? `~${formatCurrency(avgFeePerEmployee)}` : "—"}
              </span>{" "}
              /nv · {totalRequests} NV
            </span>
          </div>
        </>
      )}
    </div>
  );
});
