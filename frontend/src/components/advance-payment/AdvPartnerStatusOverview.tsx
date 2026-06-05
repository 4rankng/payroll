import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { CheckCircle2, Clock, AlertTriangle, XCircle } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";

export interface AdvPartnerStatusOverviewProps {
  totalPaid: number;
  totalPending: number;
  totalFailed: number;
  totalCancelled: number;
  totalRequests: number;
  totalPaidAmount?: number;
  totalPendingAmount?: number;
  totalFailedAmount?: number;
  totalCancelledAmount?: number;
  /** From backend — totalPaid / (totalRequests - totalCancelled) * 100 */
  successRate: number;
  isLoading?: boolean;
  className?: string;
}

interface StatusCellDef {
  key: string;
  label: string;
  icon: typeof CheckCircle2;
  gradientClass: string;
  /** Color tokens for the meta card: small inline icon + faint watermark icon. */
  iconText: string;
  watermark: string;
  getValue: (props: AdvPartnerStatusOverviewProps) => number;
  getAmount: (props: AdvPartnerStatusOverviewProps) => number | undefined;
}

const CELLS: StatusCellDef[] = [
  {
    key: "completed",
    label: "Hoàn tất",
    icon: CheckCircle2,
    gradientClass: "bg-gradient-to-b from-[#10A876] to-[#0E9F6E]",
    iconText: "text-emerald-600",
    watermark: "text-emerald-500/15",
    getValue: (p) => p.totalPaid,
    getAmount: (p) => p.totalPaidAmount,
  },
  {
    key: "pending",
    label: "Chờ xử lý",
    icon: Clock,
    gradientClass: "bg-gradient-to-b from-[#D89000] to-[#C58200]",
    iconText: "text-amber-600",
    watermark: "text-amber-500/15",
    getValue: (p) => p.totalPending,
    getAmount: (p) => p.totalPendingAmount,
  },
  {
    key: "failed",
    label: "Thất bại",
    icon: AlertTriangle,
    gradientClass: "bg-gradient-to-b from-[#D34D3F] to-[#C0382B]",
    iconText: "text-rose-600",
    watermark: "text-rose-500/15",
    getValue: (p) => p.totalFailed,
    getAmount: (p) => p.totalFailedAmount,
  },
  {
    key: "cancelled",
    label: "Đã hủy",
    icon: XCircle,
    gradientClass: "bg-gradient-to-b from-[#6E7891] to-[#5B6478]",
    iconText: "text-slate-600",
    watermark: "text-slate-500/15",
    getValue: (p) => p.totalCancelled,
    getAmount: (p) => p.totalCancelledAmount,
  },
];

export const AdvPartnerStatusOverview = memo(function AdvPartnerStatusOverview({
  totalPaid,
  totalPending,
  totalFailed,
  totalCancelled,
  totalRequests,
  totalPaidAmount,
  totalPendingAmount,
  totalFailedAmount,
  totalCancelledAmount,
  successRate,
  isLoading = false,
  className,
}: AdvPartnerStatusOverviewProps) {
  const props = {
    totalPaid, totalPending, totalFailed, totalCancelled,
    totalRequests, totalPaidAmount, totalPendingAmount,
    totalFailedAmount, totalCancelledAmount,
    successRate,
  };

  return (
    <section
      aria-label="Tiến độ yêu cầu"
      className={cn(
        "rounded-xl border border-border/70 bg-card p-5 shadow-[0_1px_2px_0_rgb(15_23_42/0.04)] sm:p-6",
        className,
      )}
    >
      <div className="mb-[18px] flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h2 className="text-[14px] font-bold tracking-tight text-foreground">
            Tiến độ yêu cầu
          </h2>
          <div className="rounded-md bg-muted px-2 py-0.5 font-financial text-[12px] font-medium text-muted-foreground">
            {totalRequests}
          </div>
          <div className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2 py-0.5 font-financial text-[11.5px] font-medium text-foreground/70">
            <CheckCircle2 className="h-3.5 w-3.5" />
            {successRate.toFixed(0)}%
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-9 w-full rounded-lg" />
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <Skeleton className="h-[68px] rounded-lg" />
            <Skeleton className="h-[68px] rounded-lg" />
            <Skeleton className="h-[68px] rounded-lg" />
            <Skeleton className="h-[68px] rounded-lg" />
          </div>
        </div>
      ) : (
        <>
          {/* Pipe Bar — progressive disclosure based on segment width so the
              number is always visible (was being clipped behind the icon when
              the segment was tight). Thresholds tuned for a typical container:
                < 8%   : value-only, centered, minimal padding
                8–18%  : icon-bubble + value, left-aligned
                > 18%  : icon-bubble + value + label
          */}
          <div className="flex h-9 overflow-hidden rounded-lg bg-muted shadow-[inset_0_0_0_1px_theme(colors.border)]">
            {CELLS.map((cell) => {
              const value = cell.getValue(props);
              const width = totalRequests > 0 ? (value / totalRequests) * 100 : 0;
              if (width === 0) return null;

              const showIcon = width >= 8;
              const showLabel = width > 18;

              return (
                <div
                  key={cell.key}
                  // title attr gives mobile users tap-to-see and desktop hover tooltip
                  // so the segment is never ambiguous when the inline label is hidden.
                  title={`${cell.label}: ${value}`}
                  aria-label={`${cell.label}: ${value}`}
                  className={cn(
                    "relative flex h-full items-center font-financial text-[13px] font-semibold text-white transition-[filter] hover:brightness-110 min-w-0",
                    showIcon ? "px-3 sm:px-3.5" : "px-1.5 justify-center",
                    cell.gradientClass,
                  )}
                  style={{ width: `${width}%` }}
                >
                  {showIcon && (
                    <div className="mr-2 grid h-[18px] w-[18px] shrink-0 place-items-center rounded-full bg-white/20">
                      <cell.icon className="h-[11px] w-[11px]" strokeWidth={3} />
                    </div>
                  )}
                  <span className="text-[14px] font-bold tabular-nums shrink-0">{value}</span>
                  {showLabel && (
                    <span className="ml-1.5 font-display text-[12.5px] font-medium opacity-90 truncate">
                      {cell.label}
                    </span>
                  )}
                </div>
              );
            })}
          </div>

          {/* Pipe Meta Grid — watermark icon style (large faint icon on right,
              small inline icon next to the label). Matches the dashboard
              StatTile / KpiHeroCard treatment so the visual language is
              consistent across stat cards. */}
          <div className="mt-3.5 grid grid-cols-2 gap-3 lg:grid-cols-4">
            {CELLS.map((cell) => {
              const value = cell.getValue(props);
              const amount = cell.getAmount(props);

              return (
                <div
                  key={cell.key}
                  className="group relative flex flex-col rounded-lg border border-border/70 bg-card p-3 overflow-hidden shadow-sm transition-colors hover:bg-muted/40"
                >
                  {/* Watermark — large faint icon bleeding right, behind text */}
                  <cell.icon
                    className={cn(
                      "absolute right-2 top-1/2 -translate-y-1/2 h-12 w-12 pointer-events-none",
                      "transition-transform duration-300 group-hover:scale-105",
                      cell.watermark,
                    )}
                    strokeWidth={1.5}
                  />

                  <div className="relative pr-12">
                    {/* Label row with small inline icon prefix */}
                    <div className="flex items-center gap-1.5">
                      <cell.icon className={cn("h-3 w-3 shrink-0", cell.iconText)} strokeWidth={2.2} />
                      <span className="text-[10.5px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight">
                        {cell.label}
                      </span>
                    </div>
                    {/* Big number + optional amount */}
                    <div className="mt-1 flex items-baseline gap-1.5 flex-wrap">
                      <span className="font-financial text-[20px] font-semibold tracking-[-0.01em] leading-none text-foreground tabular-nums">
                        {value}
                      </span>
                      {amount !== undefined && (
                        <span className="font-financial text-[11px] text-muted-foreground">
                          {formatCurrency(amount)}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}
    </section>
  );
});
