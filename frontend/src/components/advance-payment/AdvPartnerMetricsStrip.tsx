import { memo } from 'react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';
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
  /** Strip the card chrome so the rail can be embedded in a shared surface,
   *  e.g. the admin "pipeline + health" band. */
  bare?: boolean;
}

function formatSeconds(secs: number): string {
  if (secs === 0) return '—';
  if (secs < 60) return `${Math.round(secs)}s`;
  const mins = Math.floor(secs / 60);
  const rem = Math.round(secs % 60);
  return rem > 0 ? `${mins}m ${rem}s` : `${mins}m`;
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
  bare = false,
}: AdvPartnerMetricsStripProps) {
  const effectiveTotal = totalRequests - totalCancelled;

  // Processing-time cells over completed requests (paid_at − created_at):
  // <30s / <5m / >5m — the last two cells each sum two backend buckets.
  const processingCells = [
    { label: '<30 giây', short: '<30s', count: completedUnder30s },
    { label: '<5 phút', short: '<5p', count: completed30sTo2m + completed2mTo5m },
    { label: '>5 phút', short: '>5p', count: completed5mTo15m + completedOver15m },
  ];

  const chrome = bare
    ? 'rounded-none border-0 bg-transparent p-0 shadow-none'
    : 'rounded-xl border border-border/70 bg-card shadow-soft';

  return (
    <section
      aria-label="Hiệu suất xử lý"
      className={cn('overflow-hidden', chrome, className)}
    >
      {/* Zone header */}
      <div className="px-4 pt-3 pb-1 text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
        Hiệu suất
      </div>

      {isLoading ? (
        <div className="space-y-0 divide-y divide-border/60">
          <div className="px-4 py-2"><Skeleton className="h-6 w-full rounded-md" /></div>
          <div className="px-4 py-2"><Skeleton className="h-6 w-full rounded-md" /></div>
          <div className="px-4 py-2"><Skeleton className="h-6 w-full rounded-md" /></div>
        </div>
      ) : (
        <div className="divide-y divide-border/60">
          {/* Success rate */}
          <div className="flex items-center justify-between gap-3 px-4 py-2">
            <div className="min-w-0">
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                Tỉ lệ thành công
              </div>
              <div className="mt-0.5 flex items-baseline gap-1.5">
                <span className="font-financial text-[15px] font-semibold leading-tight tabular-nums text-foreground">
                  {successRate.toFixed(1)}%
                </span>
                <span className="truncate font-financial text-[10.5px] text-muted-foreground tabular-nums">
                  {totalPaid}/{effectiveTotal} HT · {totalCancelled} hủy
                </span>
              </div>
            </div>
          </div>

          {/* Avg processing time + distribution chips */}
          <div className="flex items-center justify-between gap-3 px-4 py-2">
            <div className="min-w-0">
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                Thời gian xử lý TB
              </div>
              <div className="mt-0.5 flex items-baseline gap-1.5">
                <span className="font-financial text-[15px] font-semibold leading-tight tabular-nums text-foreground">
                  {formatSeconds(avgProcessingTimeSecs)}
                </span>
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-1.5">
              {processingCells.map((cell) => (
                <span
                  key={cell.label}
                  title={`${cell.label}: ${cell.count} yêu cầu`}
                  className="inline-flex items-baseline gap-1 rounded-md bg-muted/70 px-1.5 py-1"
                >
                  <span className="text-[9.5px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                    {cell.short}
                  </span>
                  <span className="font-financial text-[12px] font-bold leading-none tabular-nums text-foreground">
                    {cell.count}
                  </span>
                </span>
              ))}
            </div>
          </div>

          {/* Avg fee per request */}
          <div className="flex items-center justify-between gap-3 px-4 py-1.5">
            <div className="min-w-0">
              <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                Phí thu trung bình
              </div>
              <div className="mt-0.5 flex items-baseline gap-1.5">
                <span className="font-financial text-[15px] font-semibold leading-tight tabular-nums text-foreground">
                  {formatCurrency(avgFeePerRequest)}
                </span>
                <span className="truncate font-financial text-[10.5px] text-muted-foreground tabular-nums">
                  / yêu cầu · {feePercentage.toFixed(1)}% GN
                </span>
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  );
});
