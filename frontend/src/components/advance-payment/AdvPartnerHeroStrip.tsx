import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/utils/formatters";
import { AnimatedCurrency } from "@/components/ui/animated-currency";

export interface AdvPartnerHeroStripProps {
  /** Gross total of completed advance requests this period
   *  (SUM(request_amount)) — the actual amount disbursed. */
  totalAmount: number;
  /** Completed requests this period. Raw counts belong to the pipeline band
   *  (AdvPartnerStatusOverview) — here the count only derives the footer's
   *  average disbursement per request so no metric repeats across bands. */
  totalPaid?: number;
  isLoading?: boolean;
  className?: string;
  compact?: boolean;
}

export const AdvPartnerHeroStrip = memo(function AdvPartnerHeroStrip({
  totalAmount,
  totalPaid = 0,
  isLoading = false,
  className,
  compact = false,
}: AdvPartnerHeroStripProps) {
  // Footer companion: typical advance size. Em dash before the first
  // completed request of the period.
  const avgPerRequest = totalPaid > 0 ? totalAmount / totalPaid : 0;
  return (
    <section
      aria-label="Giải ngân"
      className={cn(
        "treasury-panel flex flex-col",
        compact ? "p-3.5" : "p-[22px] px-7",
        "rounded-xl border border-[#D8E2EE] bg-white shadow-[0_1px_2px_0_rgb(15_23_42/0.04)]",
        className,
      )}
    >
      {/* Measurement mesh — faint dot grid under the panel label. */}
      <div className="treasury-mesh" aria-hidden />

      {isLoading ? (
        <div className="flex h-full flex-col space-y-4">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="mt-2 h-12 w-48" />
          <div className="mt-auto">
            <Skeleton className="h-8 w-36" />
          </div>
        </div>
      ) : (
        <>
          <div className="relative z-[1] flex items-center gap-2 text-[10.5px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
            <span className="treasury-signal bg-emerald-500" />
            Giải ngân kỳ này
          </div>
          <div className={cn("treasury-value relative z-[1] mt-1.5 max-w-full break-words font-financial font-semibold leading-[1.08] tracking-normal text-foreground [text-shadow:0_0_36px_rgba(16,185,129,0.22)]", compact ? "text-[clamp(1.5rem,7.5vw,1.75rem)]" : "text-[clamp(1.875rem,7vw,2.25rem)]")}>
            <AnimatedCurrency target={totalAmount} />
          </div>

          {/* Footer rail — pinned via mt-auto to the shared band baseline so the
              wallet, disbursement and fee panels read as one instrument strip. */}
          <div className="mt-auto pt-3">
            <div className="treasury-rail pt-2.5">
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                Trung bình /yc
              </div>
              <div className="mt-0.5 truncate font-financial text-[15px] font-semibold leading-snug text-foreground tabular-nums">
                {avgPerRequest > 0 ? `~${formatCurrency(avgPerRequest)}` : "—"}
              </div>
            </div>
          </div>
        </>
      )}
    </section>
  );
});
