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
  isLoading,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  unit?: string;
  color: 'emerald' | 'amber' | 'teal';
  footer: React.ReactNode;
  isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-border/40 bg-card p-3 shadow-soft">
        <Skeleton className="h-9 w-9 shrink-0 rounded-lg" />
        <div className="min-w-0 flex-1 space-y-1.5">
          <Skeleton className="h-2.5 w-24" />
          <Skeleton className="h-5 w-20" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-3 rounded-xl border border-border/60 bg-card p-3 shadow-soft">
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
  );
}

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
        footer={
          <>
            <span className="rounded bg-muted px-1.5 py-px font-financial text-[11px] font-medium text-foreground/70">
              {completedUnder30s}/{totalPaid} &lt;30s
            </span>
            <span className="whitespace-nowrap">{completedUnder30s < totalPaid ? 'Cần tối ưu' : 'Tốt'}</span>
          </>
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
