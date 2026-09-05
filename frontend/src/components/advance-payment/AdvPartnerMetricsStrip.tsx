import { memo } from 'react';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';

export interface AdvPartnerMetricsStripProps {
  totalPaid: number;
  /** Kept for the paired zone's props shape — per-status counts are
   *  displayed by AdvPartnerStatusOverview, never here. */
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

/** Gauge arc + halo per success band; dot classes stay Tailwind tokens. */
const GAUGE_TRACK = '#e4e7ec'; // --color-border
const gaugeTones = {
  ok: { from: '#34d399', to: '#059669', dot: 'bg-emerald-500', halo: 'drop-shadow-[0_0_5px_rgba(5,150,105,0.45)]' },
  warn: { from: '#fbbf24', to: '#d97706', dot: 'bg-amber-500', halo: 'drop-shadow-[0_0_5px_rgba(217,119,6,0.40)]' },
  bad: { from: '#f87171', to: '#dc2626', dot: 'bg-red-500', halo: 'drop-shadow-[0_0_5px_rgba(220,38,38,0.40)]' },
} as const;

export const AdvPartnerMetricsStrip = memo(function AdvPartnerMetricsStrip({
  totalPaid,
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
  // Processing-time cells over completed requests (paid_at − created_at):
  // <30s / <5m / >5m — the last two cells each sum two backend buckets.
  // Tone encodes the speed: fast = emerald, mid = amber, slow = red.
  const processingCells = [
    { label: '<30 giây', count: completedUnder30s, tone: 'text-emerald-600' },
    { label: '<5 phút', count: completed30sTo2m + completed2mTo5m, tone: 'text-amber-600' },
    { label: '>5 phút', count: completed5mTo15m + completedOver15m, tone: 'text-red-600' },
  ];

  const tone =
    successRate >= 99 ? gaugeTones.ok : successRate >= 90 ? gaugeTones.warn : gaugeTones.bad;
  const gaugeRate = Math.min(100, Math.max(0, successRate));

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
          {/* Success hero — HUD ring gauge carrying the success rate. The
              per-status counts intentionally live in the paired
              AdvPartnerStatusOverview zone; do NOT re-display them here. */}
          <div className="flex items-center gap-3 px-4 py-2.5">
            <div
              role="img"
              aria-label={`Tỷ lệ thành công ${successRate.toFixed(1)}%`}
              className={cn(
                'relative h-[52px] w-[52px] shrink-0 rounded-full',
                totalPaid > 0 && tone.halo,
              )}
              style={{
                background: `conic-gradient(from 0deg, ${tone.from} 0%, ${tone.to} ${gaugeRate}%, ${GAUGE_TRACK} ${gaugeRate}%, ${GAUGE_TRACK} 100%)`,
              }}
            >
              <div className="absolute inset-[5px] rounded-full bg-card shadow-[inset_0_1px_3px_rgba(16,24,40,0.10)]" />
              <div className="absolute inset-0 grid place-items-center">
                <span className="font-financial text-[13px] font-bold leading-none tabular-nums text-foreground">
                  {successRate.toFixed(1)}
                  <span className="text-[9px] font-semibold text-muted-foreground">%</span>
                </span>
              </div>
            </div>
            <div className="flex min-w-0 items-center gap-1.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
              <span
                aria-hidden="true"
                className={cn('h-1.5 w-1.5 shrink-0 rounded-full', tone.dot)}
              />
              Tỷ lệ thành công
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
