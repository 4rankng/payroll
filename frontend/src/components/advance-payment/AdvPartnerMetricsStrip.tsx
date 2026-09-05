import { memo } from 'react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';
import { CheckCircle2, Clock, DollarSign, type LucideIcon } from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';

export interface AdvPartnerMetricsStripProps {
  totalPaid: number;
  totalRequests: number;
  totalCancelled: number;
  avgProcessingTimeSecs: number;
  completedUnder30s: number;
  /** From backend — completed with processing time 30s–2m */
  completed30sTo2m: number;
  /** From backend — completed with processing time 2m–5m */
  completed2mTo5m: number;
  /** From backend — completed with processing time 5m–15m */
  completed5mTo15m: number;
  /** From backend — completed with processing time >15m */
  completedOver15m: number;
  /** From backend — fee earned / paid amount * 100 */
  feePercentage: number;
  /** From backend — fee earned / total paid */
  avgFeePerRequest: number;
  /** From backend — totalPaid / (totalRequests - totalCancelled) * 100 */
  successRate: number;
  isLoading?: boolean;
  className?: string;
}

function formatSeconds(secs: number): string {
  if (secs === 0) return '—';
  if (secs < 60) return `${Math.round(secs)}s`;
  const mins = Math.floor(secs / 60);
  const rem = Math.round(secs % 60);
  return rem > 0 ? `${mins}m ${rem}s` : `${mins}m`;
}

const ICON_COLOR_STYLES: Record<'emerald' | 'amber' | 'teal', string> = {
  emerald: 'bg-primary/10 text-primary',
  amber: 'bg-warning/10 text-warning',
  teal: 'bg-success/10 text-success',
};

// Compact single-row stat: icon | label + value | footer, all on one line.
// Deliberately bespoke (not the shared KpiHeroCard) — that component's
// vertical card shape leaves large empty space once these three cards are
// stretched across a full-width row.
function MetricCard({
  icon: Icon,
  label,
  value,
  unit,
  color,
  footer,
  detail,
  isLoading,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  unit?: string;
  color: 'emerald' | 'amber' | 'teal';
  footer: React.ReactNode;
  detail?: React.ReactNode;
  isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <div className="flex flex-col rounded-xl border border-border/40 bg-card p-3 shadow-soft">
        <div className="flex items-center gap-3">
          <Skeleton className="h-9 w-9 shrink-0 rounded-lg" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <Skeleton className="h-2.5 w-24" />
            <Skeleton className="h-5 w-20" />
          </div>
        </div>
        {detail && (
          <div className="mt-3 space-y-1">
            <Skeleton className="h-2 w-full" />
            <Skeleton className="h-2 w-full" />
            <Skeleton className="h-2 w-full" />
            <Skeleton className="h-2 w-full" />
            <Skeleton className="h-2 w-2/3" />
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col rounded-xl border border-border/60 bg-card p-3 shadow-soft">
      <div className="flex items-center gap-3">
        <div className={cn('flex h-9 w-9 shrink-0 items-center justify-center rounded-lg', ICON_COLOR_STYLES[color])}>
          <Icon className="h-[18px] w-[18px]" strokeWidth={2} />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-xs font-semibold leading-tight text-muted-foreground">{label}</p>
          <p className="mt-0.5 font-display font-extrabold tabular-nums leading-tight tracking-[-0.035em] text-foreground">
            <span className="text-lg">{value}</span>
            {unit ? <span className="ml-0.5 text-xs font-bold text-muted-foreground">{unit}</span> : null}
          </p>
        </div>
        <div className="flex shrink-0 flex-col items-end gap-0.5 text-[11px] leading-snug text-muted-foreground">
          {footer}
        </div>
      </div>
      {detail && <div className="mt-3">{detail}</div>}
    </div>
  );
}

export const AdvPartnerMetricsStrip = memo(function AdvPartnerMetricsStrip({
  totalPaid,
  totalRequests,
  totalCancelled,
  avgProcessingTimeSecs,
  completedUnder30s,
  completed30sTo2m,
  completed2mTo5m,
  completed5mTo15m,
  completedOver15m,
  feePercentage,
  avgFeePerRequest,
  successRate,
  isLoading = false,
  className,
}: AdvPartnerMetricsStripProps) {
  const effectiveTotal = totalRequests - totalCancelled;

  // Processing-time distribution over completed requests (paid_at − created_at).
  // Buckets: ≤30s / 30s–2m / 2–5m / 5–15m / >15m — mirrors the backend bucket SQL.
  const distribution = [
    { label: '<30 giây', count: completedUnder30s },
    { label: '30 giây – 2 phút', count: completed30sTo2m },
    { label: '2 – 5 phút', count: completed2mTo5m },
    { label: '5 – 15 phút', count: completed5mTo15m },
    { label: '>15 phút', count: completedOver15m },
  ];
  const maxCount = Math.max(...distribution.map((b) => b.count), 1);

  return (
    <section aria-label="Chỉ số hiệu suất" className={cn('grid grid-cols-1 gap-3 sm:grid-cols-3', className)}>
      <MetricCard
        icon={CheckCircle2}
        label="Tỉ lệ thành công"
        value={successRate.toFixed(1)}
        unit="%"
        color="emerald"
        footer={
          <>
            <span className="rounded bg-muted px-1.5 py-px font-financial text-[11px] font-medium text-foreground/70">
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
        color="amber"
        detail={
          <div className="space-y-1">
            {distribution.map((bucket) => {
              const width = bucket.count === 0 ? 0 : Math.max((bucket.count / maxCount) * 100, 2);
              return (
                <div key={bucket.label} className="flex items-center gap-2">
                  <span className="w-24 shrink-0 text-right text-[10px] font-medium tabular-nums text-muted-foreground">
                    {bucket.label}
                  </span>
                  <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted/60">
                    <div className="h-full rounded-full bg-primary/70" style={{ width: `${width}%` }} />
                  </div>
                  <span className="w-9 shrink-0 text-right text-[10px] font-semibold tabular-nums text-foreground">
                    {bucket.count}
                  </span>
                </div>
              );
            })}
          </div>
        }
        isLoading={isLoading}
      />

      <MetricCard
        icon={DollarSign}
        label="Phí thu trung bình"
        value={formatCurrency(avgFeePerRequest)}
        color="teal"
        footer={
          <>
            <span className="whitespace-nowrap rounded bg-muted px-1.5 py-px font-financial text-[11px] font-medium text-foreground/70">
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
