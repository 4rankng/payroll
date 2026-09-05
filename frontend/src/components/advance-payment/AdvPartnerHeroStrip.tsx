import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/utils/formatters";
import { useCountUp } from "@/hooks/useCountUp";

export interface AdvPartnerHeroStripProps {
  /** Gross total of completed advance requests this period
   *  (SUM(request_amount)) — the actual amount disbursed. */
  totalAmount: number;
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
    <span className="font-financial tabular-nums tracking-normal">
      {formatCurrency(animated)}
    </span>
  );
});

export const AdvPartnerHeroStrip = memo(function AdvPartnerHeroStrip({
  totalAmount,
  isLoading = false,
  className,
  compact = false,
}: AdvPartnerHeroStripProps) {
  return (
    <section
      aria-label="Giải ngân"
      className={cn(
        "relative flex flex-col justify-between",
        compact ? "p-3.5" : "p-[22px] px-7",
        "rounded-xl border border-[#D8E2EE] bg-white shadow-[0_1px_2px_0_rgb(15_23_42/0.04)]",
        "bg-[radial-gradient(90%_70%_at_50%_0%,rgba(8,120,62,0.05),transparent_70%)]",
        className,
      )}
    >
      {isLoading ? (
        <div className="flex h-full flex-col space-y-4">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="mt-2 h-12 w-48" />
        </div>
      ) : (
        <div>
          <div className="flex items-center gap-2 text-[10.5px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.15)]" />
            Giải ngân kỳ này
          </div>
          <div className={cn("mt-1.5 max-w-full break-words font-financial font-semibold leading-[1.08] tracking-normal text-foreground", compact ? "text-[clamp(1.5rem,7.5vw,1.75rem)]" : "text-[clamp(1.875rem,7vw,2.25rem)]")}>
            <AnimatedCurrency target={totalAmount} />
          </div>
        </div>
      )}
    </section>
  );
});
