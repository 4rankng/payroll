import { memo } from 'react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';

export interface AdvPartnerMetricsStripProps {
  totalPaid: number;
  totalRequests: number;
  totalCancelled: number;
  completedUnder30s: number;
  /** From backend — completed with processing time 30s–2m */
  completed30sTo2m: number;
  /** From backend — completed with processing time 2m–5m */
  completed2mTo5m: number;
  /** From backend — completed with processing time 5m–15m */
  completed5mTo15m: number;
  /** From backend — completed with processing time >15m */
  completedOver15m: number;
  /** From backend — totalPaid / (totalRequests - totalCancelled) * 100 */
  successRate: number;
  isLoading?: boolean;
  className?: string;
  /** Strip the card chrome so the rail can be embedded in a shared surface,
   *  e.g. the admin "pipeline + health" band. */
  bare?: boolean;
}

export const AdvPartnerMetricsStrip = memo(function AdvPartnerMetricsStrip({
  totalPaid,
  totalRequests,
  totalCancelled,
  completedUnder30s,
  completed30sTo2m,
  completed2mTo5m,
  completed5mTo15m,
  completedOver15m,
  successRate,
  isLoading = false,
  className,
  bare = false,
}: AdvPartnerMetricsStripProps) {
  const effectiveTotal = totalRequests - totalCancelled;

  // Success rail segments (share of ALL requests) + status dot tone.
  const railTotal = Math.max(totalRequests, 1);
  const paidShare = (totalPaid / railTotal) * 100;
  const cancelledShare = (totalCancelled / railTotal) * 100;
  const successDotTone =
    successRate >= 99 ? 'bg-emerald-500' : successRate >= 90 ? 'bg-amber-500' : 'bg-red-500';

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
      {/* Zone header */}
      <div className="px-4 pt-3 pb-1 text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
        Hiệu suất
      </div>

      {isLoading ? (
        <div className="space-y-2 px-4 pb-3">
          <Skeleton className="h-6 w-full rounded-md" />
          <Skeleton className="h-14 w-full rounded-lg" />
        </div>
      ) : (
        <div className="divide-y divide-border/60">
          {/* Success hero — status dot + label left, big tabular % right,
              hairline rail below shows paid vs cancelled share of all requests. */}
          <div className="px-4 py-2.5">
            <div className="flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-center gap-1.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                <span
                  aria-hidden="true"
                  className={cn('h-1.5 w-1.5 shrink-0 rounded-full', successDotTone)}
                />
                Tỉ lệ thành công
              </div>
              <div className="font-financial text-[20px] font-bold leading-none tabular-nums text-foreground">
                {successRate.toFixed(1)}
                <span className="text-[12px] font-semibold text-muted-foreground">%</span>
              </div>
            </div>
            <div
              aria-hidden="true"
              className="mt-2 flex h-1 overflow-hidden rounded-full bg-muted ring-1 ring-inset ring-black/[0.06]"
            >
              <div className="bg-emerald-500" style={{ width: `${paidShare}%` }} />
              <div className="bg-zinc-400/60" style={{ width: `${cancelledShare}%` }} />
            </div>
            <div className="mt-1 text-right font-financial text-[10.5px] text-muted-foreground tabular-nums">
              {totalPaid}/{effectiveTotal} HT · {totalCancelled} hủy
            </div>
          </div>

          {/* Processing-time HUD (completed requests) — number-first cells,
              tone encodes fast/mid/slow, hairline separators, zero-muted. */}
          <div className="px-4 py-2.5">
            <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
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
                  <div className="mt-1 text-[9.5px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                    {cell.label}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </section>
  );
});
