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
    gradientClass: "bg-gradient-to-b from-emerald-600 to-emerald-700",
    iconText: "text-emerald-600",
    watermark: "text-emerald-500/15",
    getValue: (p) => p.totalPaid,
    getAmount: (p) => p.totalPaidAmount,
  },
  {
    key: "pending",
    label: "Chờ xử lý",
    icon: Clock,
    gradientClass: "bg-gradient-to-b from-amber-600 to-amber-700",
    iconText: "text-amber-600",
    watermark: "text-amber-500/15",
    getValue: (p) => p.totalPending,
    getAmount: (p) => p.totalPendingAmount,
  },
  {
    key: "failed",
    label: "Thất bại",
    icon: AlertTriangle,
    gradientClass: "bg-gradient-to-b from-rose-600 to-rose-700",
    iconText: "text-rose-600",
    watermark: "text-rose-500/15",
    getValue: (p) => p.totalFailed,
    getAmount: (p) => p.totalFailedAmount,
  },
  {
    key: "cancelled",
    label: "Đã hủy",
    icon: XCircle,
    gradientClass: "bg-gradient-to-b from-slate-600 to-slate-700",
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
        "rounded-xl border border-border/70 bg-card p-4 shadow-[0_1px_2px_0_rgb(15_23_42/0.04)] sm:p-4",
        className,
      )}
    >
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h2 className="text-[14px] font-bold tracking-tight text-foreground">
            Tiến độ yêu cầu
          </h2>
          <div className="rounded-md bg-muted px-2 py-0.5 font-financial text-[12px] font-medium text-muted-foreground">
            {totalRequests}
          </div>
          {totalRequests > 0 && (
            <div className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2 py-0.5 font-financial text-[11.5px] font-medium text-foreground/70">
              <CheckCircle2 className="h-3.5 w-3.5" />
              {successRate.toFixed(0)}%
            </div>
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-7 w-full rounded-lg" />
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
            <Skeleton className="h-[58px] rounded-lg" />
            <Skeleton className="h-[58px] rounded-lg" />
            <Skeleton className="h-[58px] rounded-lg" />
            <Skeleton className="h-[58px] rounded-lg" />
          </div>
        </div>
      ) : totalRequests === 0 ? (
        <div className="flex min-h-11 items-center rounded-lg border border-dashed border-border/70 bg-muted/30 px-3 py-2.5">
          <p className="text-sm text-muted-foreground">
            Chưa có yêu cầu trong kỳ đã chọn
          </p>
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
          <div className="flex h-7 overflow-hidden rounded-lg bg-muted shadow-[inset_0_0_0_1px_theme(colors.border)]">
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
                    "relative flex h-full min-w-0 items-center font-financial text-[12px] font-semibold text-white transition-[filter] hover:brightness-110",
                    showIcon ? "px-2.5 sm:px-3" : "justify-center px-1.5",
                    cell.gradientClass,
                  )}
                  style={{ width: `${width}%` }}
                >
                  {showIcon && (
                    <div className="mr-1.5 grid h-4 w-4 shrink-0 place-items-center rounded-full bg-white/20">
                      <cell.icon className="h-[11px] w-[11px]" strokeWidth={3} />
                    </div>
                  )}
                  <span className="shrink-0 text-[13px] font-bold tabular-nums">{value}</span>
                  {showLabel && (
                    <span className="ml-1.5 truncate font-display text-[12px] font-medium opacity-90">
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
          <div className="mt-2.5 grid grid-cols-2 gap-2 lg:grid-cols-4">
            {CELLS.map((cell) => {
              const value = cell.getValue(props);
              const amount = cell.getAmount(props);

              return (
                <div
                  key={cell.key}
                  className="group relative flex min-h-[58px] flex-col overflow-hidden rounded-lg border border-border/70 bg-card p-2.5 shadow-sm transition-colors hover:bg-muted/40"
                >
                  {/* Watermark — large faint icon bleeding right, behind text */}
                  <cell.icon
                    className={cn(
                      "pointer-events-none absolute right-2 top-1/2 h-9 w-9 -translate-y-1/2",
                      "transition-transform duration-300 group-hover:scale-105",
                      cell.watermark,
                    )}
                    strokeWidth={1.5}
                  />

                  <div className="relative pr-12">
                    {/* Label row with small inline icon prefix */}
                    <div className="flex items-center gap-1.5">
                      <cell.icon className={cn("h-3 w-3 shrink-0", cell.iconText)} strokeWidth={2.2} />
                      <span className="text-[11px] font-semibold uppercase leading-tight tracking-[0.06em] text-muted-foreground">
                        {cell.label}
                      </span>
                    </div>
                    {/* Big number + optional amount */}
                    <div className="mt-1 flex flex-wrap items-baseline gap-1.5">
                      <span className="font-financial text-[18px] font-semibold leading-none tracking-[-0.01em] text-foreground tabular-nums">
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
