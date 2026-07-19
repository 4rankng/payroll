import { Skeleton } from "@/components/ui/skeleton";
import { formatVND } from "@/utils/loanHelpers";
import { DollarSign, TrendingDown, Percent, CreditCard } from "lucide-react";
import { cn } from "@/lib/utils";

interface LoansSummary {
  total_borrowed: number;
  total_outstanding: number;
  total_interest_paid: number;
  active_loans_count: number;
}

interface LoanSummaryCardsProps {
  summary?: LoansSummary;
  isLoading?: boolean;
}

// Watermark tokens — small inline icon + large faint icon decoration.
const colorMap = {
  blue:   { iconText: "text-blue-600",    watermark: "text-blue-500/15" },
  red:    { iconText: "text-rose-600",    watermark: "text-rose-500/15" },
  amber:  { iconText: "text-amber-600",   watermark: "text-amber-500/15" },
  teal:   { iconText: "text-teal-600",    watermark: "text-teal-500/15" },
} as const;

export function LoanSummaryCards({ summary, isLoading }: LoanSummaryCardsProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="relative rounded-xl border border-border/60 bg-card px-3 py-2.5 overflow-hidden">
            <div className="space-y-1.5">
              <Skeleton className="h-2.5 w-20" />
              <Skeleton className="h-4 w-16" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (!summary) return null;

  const cards = [
    { title: "Tổng vay", value: formatVND(summary.total_borrowed), icon: DollarSign, color: "blue" as const },
    { title: "Dư nợ hiện tại", value: formatVND(summary.total_outstanding), icon: TrendingDown, color: "red" as const },
    { title: "Lãi đã trả", value: formatVND(summary.total_interest_paid), icon: Percent, color: "amber" as const },
    { title: "Khoản vay", value: summary.active_loans_count.toLocaleString("vi-VN"), icon: CreditCard, color: "teal" as const },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
      {cards.map(({ title, value, icon: Icon, color }) => {
        const c = colorMap[color];
        return (
          <div
            key={title}
            className="group relative rounded-xl border border-border/60 bg-card px-3 py-2.5 overflow-hidden shadow-sm transition-colors hover:bg-muted/40"
          >
            <Icon
              className={cn(
                "absolute right-2 top-1/2 -translate-y-1/2 h-10 w-10 pointer-events-none",
                "transition-transform duration-300 group-hover:scale-105",
                c.watermark,
              )}
              strokeWidth={1.5}
            />
            <div className="relative min-w-0 pr-10">
              <div className="flex items-center gap-1.5">
                <Icon className={cn("h-3 w-3 shrink-0", c.iconText)} strokeWidth={2.2} />
                <span className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight line-clamp-2">
                  {title}
                </span>
              </div>
              <p className="mt-1 break-words text-[15px] font-semibold tabular-nums text-foreground leading-tight">
                {value}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
