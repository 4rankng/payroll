import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/utils/formatters";
import { AnimatedCurrency } from "@/components/ui/animated-currency";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

export interface TreasuryFeePanelProps {
  totalFeeEarned: number;
  /** From backend — fee earned / paid amount * 100 */
  feePercentage: number;
  /** From backend — fee earned / completed requests */
  avgFeePerRequest: number;
  /** From backend — fee earned / employees in the period */
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
  feePercentage,
  avgFeePerRequest,
  avgFeePerEmployee,
  isLoading = false,
  className,
  compact = false,
}: TreasuryFeePanelProps) {
  // Footer companions: the two fee averages. Request/employee counts stay in
  // the pipeline band — repeating them here would duplicate adjacent metrics.
  const footerCols = [
    { label: "Phí /yc", value: avgFeePerRequest },
    { label: "Phí /nv", value: avgFeePerEmployee },
  ];

  return (
    <div
      className={cn(
        "treasury-panel flex h-full flex-col",
        compact ? "p-3.5" : "p-5 sm:p-6",
        className,
      )}
    >
      {/* Measurement mesh — faint dot grid under the panel label. */}
      <div className="treasury-mesh" aria-hidden />

      {isLoading ? (
        <div className="flex h-full flex-col">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="mt-2.5 h-6 w-40" />
          <div className="mt-auto pt-3">
            <div className="grid grid-cols-2 gap-3 border-t border-border/60 pt-2.5">
              <Skeleton className="h-8 w-20" />
              <Skeleton className="h-8 w-20" />
            </div>
          </div>
        </div>
      ) : (
        <>
          {/* Label */}
          <div className="relative z-[1] flex items-center gap-1.5">
            <span className="treasury-signal bg-emerald-500" />
            <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
              Phí thu kỳ này
            </span>
          </div>

          {/* Value + rate chip */}
          <div className="relative z-[1] mt-1.5 flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
            <span className="treasury-value max-w-full break-words font-financial text-[22px] font-semibold leading-[1.08] tracking-normal text-foreground tabular-nums">
              <AnimatedCurrency target={totalFeeEarned} />
            </span>
            <span className="rounded bg-emerald-50 px-1.5 py-px font-financial text-[11px] font-semibold text-emerald-700 ring-1 ring-emerald-200/60 shadow-[0_0_14px_rgba(16,185,129,0.18)]">
              {feePercentage.toFixed(1)}%
            </span>
            <span className="text-[11px] text-muted-foreground">trên giải ngân</span>
          </div>

          {/* Footer rail — pinned via mt-auto to the shared band baseline. */}
          <div className="mt-auto pt-3">
            <div className="treasury-rail grid grid-cols-2 gap-3 pt-2.5">
              {footerCols.map((col) => (
                <div key={col.label} className="min-w-0">
                  <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                    {col.label}
                  </div>
                  <div className="mt-0.5 truncate font-financial text-[15px] font-semibold leading-snug text-foreground tabular-nums">
                    {col.value > 0 ? `~${formatCurrency(col.value)}` : "—"}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
});
