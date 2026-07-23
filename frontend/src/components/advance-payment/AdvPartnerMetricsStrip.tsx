import { memo } from 'react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { CheckCircle2, Clock, DollarSign } from 'lucide-react';
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

// Each metric composes the shared KpiHeroCard; the secondary stats ride in the
// additive `footer` slot so we no longer maintain a parallel MetricCard clone.
// The loading skeleton matches KpiHeroCard's compact shape.
function MetricCard({
  icon,
  label,
  value,
  unit,
  color,
  footer,
  isLoading,
}: {
  icon: typeof CheckCircle2;
  label: string;
  value: string;
  unit?: string;
  color: 'emerald' | 'amber' | 'teal';
  footer: React.ReactNode;
  isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <div className="rounded-xl border border-border/40 bg-card p-3 shadow-soft">
        <div className="space-y-1.5">
          <Skeleton className="h-2.5 w-24" />
          <Skeleton className="h-6 w-20" />
          <Skeleton className="h-3 w-36" />
        </div>
      </div>
    );
  }

  return (
    <KpiHeroCard
      label={label}
      value={value}
      unit={unit}
      icon={icon}
      color={color}
      footer={footer}
      className="h-full"
    />
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
