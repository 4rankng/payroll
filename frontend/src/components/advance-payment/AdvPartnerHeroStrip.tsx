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
    <span className="font-financial tabular-nums tracking-tight">
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
          <div className={cn("flex items-center gap-2 font-bold uppercase tracking-[0.12em] text-muted-foreground", compact ? "text-[9.5px]" : "text-[10.5px]")}>
            <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/40" />
            Giải ngân kỳ này
          </div>
          <div className={cn("mt-1.5 font-financial font-semibold leading-none tracking-tight text-foreground", compact ? "text-[24px]" : "text-4xl")}>
            <AnimatedCurrency target={totalAmount} />
          </div>
        </div>
      )}
    </section>
  );
});
