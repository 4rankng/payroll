import { memo } from 'react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';

export interface AdvPartnerMetricsStripProps {
  completedUnder30s: number;
  /** From backend — completed with processing time 30s–2m */
  completed30sTo2m: number;
  /** From backend — completed with processing time 2m–5m */
  completed2mTo5m: number;
  /** From backend — completed with processing time 5m–15m */
  completed5mTo15m: number;
  /** From backend — completed with processing time >15m */
  completedOver15m: number;
  isLoading?: boolean;
  className?: string;
  /** Strip the card chrome so the rail can be embedded in a shared surface,
   *  e.g. the admin "pipeline + health" band. */
  bare?: boolean;
}

/**
 * Processing-time HUD for completed advance-payment requests.
 * The success rate and per-status counts are owned by the paired
 * AdvPartnerStatusOverview zone — never re-display them here.
 */
export const AdvPartnerMetricsStrip = memo(function AdvPartnerMetricsStrip({
  completedUnder30s,
  completed30sTo2m,
  completed2mTo5m,
  completed5mTo15m,
  completedOver15m,
  isLoading = false,
  className,
  bare = false,
}: AdvPartnerMetricsStripProps) {
  // Processing-time cells over completed requests (paid_at − created_at):
  // <30s / <5m / >5m — the last two cells each sum two backend buckets.
  // Tone encodes the speed: fast = emerald, mid = amber, slow = red.
  const processingCells = [
    { label: '<30 giây', count: completedUnder30s, tone: 'text-emerald-600' },
    { label: '<5 phút', count: completed30sTo2m + completed2mTo5m, tone: 'text-amber-600' },
    { label: '>5 phút', count: completed5mTo15m + completedOver15m, tone: 'text-red-600' },
  ];

  const chrome = bare
    ? 'rounded-none border-0 bg-transparent p-0 shadow-none'
    : 'rounded-xl border border-border/70 bg-card shadow-soft';

  return (
    <section
      aria-label="Hiệu suất xử lý"
      className={cn('overflow-hidden', chrome, className)}
    >
      {isLoading ? (
        <div className="space-y-2 px-4 pb-3 pt-3">
          <Skeleton className="h-4 w-24 rounded-md" />
          <Skeleton className="h-14 w-full rounded-lg" />
        </div>
      ) : (
        <div className="px-4 py-3">
          <div className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground">
            Thời gian xử lý
          </div>
          <div className="mt-1.5 grid grid-cols-3 divide-x divide-border/70 rounded-lg border border-border/70 bg-muted/30">
            {processingCells.map((cell) => (
              <div
                key={cell.label}
                title={`${cell.label}: ${cell.count} yêu cầu`}
                className="px-2 py-2 text-center"
              >
                <div
                  className={cn(
                    'font-financial text-[18px] font-bold leading-none tabular-nums',
                    cell.tone,
                    cell.count === 0 && 'opacity-40',
                  )}
                >
                  {cell.count}
                </div>
                <div className="mt-1 text-xs font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                  {cell.label}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
});
