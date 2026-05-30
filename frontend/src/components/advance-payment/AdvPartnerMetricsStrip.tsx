import { memo } from "react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { CheckCircle2, Clock, DollarSign } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

export interface AdvPartnerMetricsStripProps {
  totalPaid: number;
  totalRequests: number;
  totalCancelled: number;
  avgProcessingTimeSecs: number;
  completedUnder30s: number;
  /** From backend — fee earned / paid amount * 100 */
  feePercentage: number;
  /** From backend — fee earned / total paid */
  avgFeePerRequest: number;
  /** From backend — totalPaid / (totalRequests - totalCancelled) * 100 */
  successRate: number;
  isLoading?: boolean;
  className?: string;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function formatSeconds(secs: number): string {
  if (secs === 0) return "—";
  if (secs < 60) return `${Math.round(secs)}s`;
  const mins = Math.floor(secs / 60);
  const rem = Math.round(secs % 60);
  return rem > 0 ? `${mins}m ${rem}s` : `${mins}m`;
}

/* ------------------------------------------------------------------ */
/*  Metric card                                                        */
/* ------------------------------------------------------------------ */

interface MetricCardProps {
  icon: typeof CheckCircle2;
  label: string;
  value: string;
  unit?: string;
  footer: React.ReactNode;
  iconText: string;
  watermark: string;
  isLoading: boolean;
}

function MetricCard({
  icon: Icon,
  label,
  value,
  unit,
  footer,
  iconText,
  watermark,
  isLoading,
}: MetricCardProps) {
  if (isLoading) {
    return (
      <div className="rounded-xl border border-border/50 bg-card p-3 sm:p-4">
        <div className="space-y-2">
          <Skeleton className="h-2.5 w-24" />
          <Skeleton className="h-7 w-20" />
          <Skeleton className="h-3 w-36" />
        </div>
      </div>
    );
  }

  return (
    <div className="group relative rounded-xl border border-border/50 bg-card p-3 sm:p-4 overflow-hidden transition-colors hover:bg-muted/40">
      {/* Watermark — large faint icon decoration on the right side */}
      <Icon
        className={cn(
          "absolute right-3 top-1/2 -translate-y-1/2 h-14 w-14 pointer-events-none",
          "transition-transform duration-300 group-hover:scale-105",
          watermark,
        )}
        strokeWidth={1.5}
      />

      <div className="relative pr-14">
        {/* Label */}
        <div className="flex items-center gap-1.5">
          <Icon className={cn("h-3.5 w-3.5 shrink-0", iconText)} strokeWidth={2.2} />
          <span className="text-[10px] font-semibold uppercase tracking-[0.08em] leading-tight text-muted-foreground/80">
            {label}
          </span>
        </div>

        {/* Value */}
        <div className="mt-2 font-financial text-xl font-semibold tracking-[-0.02em] text-foreground leading-none sm:text-2xl">
          {value}
          {unit && <span className="ml-0.5 text-sm font-normal text-muted-foreground">{unit}</span>}
        </div>

        {/* Footer */}
        <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-muted-foreground">
          {footer}
        </div>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Main component                                                     */
/* ------------------------------------------------------------------ */

export const AdvPartnerMetricsStrip = memo(function AdvPartnerMetricsStrip({
  totalPaid,
  totalRequests,
  totalCancelled,
  avgProcessingTimeSecs,
  completedUnder30s,
  feePercentage,
  avgFeePerRequest,
  successRate,
  isLoading = false,
  className,
}: AdvPartnerMetricsStripProps) {
  const effectiveTotal = totalRequests - totalCancelled;

  return (
    <section
      aria-label="Chỉ số hiệu suất"
      className={cn("grid grid-cols-1 gap-3 sm:grid-cols-3", className)}
    >
      <MetricCard
        icon={CheckCircle2}
        label="Tỉ lệ thành công"
        value={successRate.toFixed(1)}
        unit="%"
        iconText="text-emerald-600"
        watermark="text-emerald-500/15"
        footer={
          <>
            <span className="rounded bg-muted px-1.5 py-px font-financial text-[10px] font-medium text-foreground/70">
              {totalPaid}/{effectiveTotal} HT
            </span>
            <span>{totalCancelled} hủy</span>
          </>
        }
        isLoading={isLoading}
      />

      <MetricCard
        icon={Clock}
        label="Thời gian xử lý TB"
        value={formatSeconds(avgProcessingTimeSecs)}
        iconText="text-amber-600"
        watermark="text-amber-500/15"
        footer={
          <>
            <span className="rounded bg-muted px-1.5 py-px font-financial text-[10px] font-medium text-foreground/70">
              {completedUnder30s}/{totalPaid} &lt;30s
            </span>
            <span className="whitespace-nowrap">{completedUnder30s < totalPaid ? "Cần tối ưu" : "Tốt"}</span>
          </>
        }
        isLoading={isLoading}
      />

      <MetricCard
        icon={DollarSign}
        label="Phí thu trung bình"
        value={formatCurrency(avgFeePerRequest)}
        iconText="text-violet-600"
        watermark="text-violet-500/15"
        footer={
          <>
            <span className="whitespace-nowrap rounded bg-muted px-1.5 py-px font-financial text-[10px] font-medium text-foreground/70">
              / yêu cầu
            </span>
            <span className="whitespace-nowrap">{feePercentage.toFixed(1)}% GN</span>
          </>
        }
        isLoading={isLoading}
      />
    </section>
  );
});
