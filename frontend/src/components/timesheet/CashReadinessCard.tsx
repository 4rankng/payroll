import { Skeleton } from '@/components/ui/skeleton';
import type { CashReadinessResponse } from '@/types/api/cash-readiness.types';
import { formatCurrency } from '@/utils/formatters';
import { getCashReadinessDisplay, formatCompactVnd } from './cashReadinessDisplay';

interface CashReadinessCardProps {
  data?: CashReadinessResponse;
  isLoading?: boolean;
  isError?: boolean;
}

/** Label LED tones — calibration semantics, matching the signal dot at full
 *  strength so the pulsing signal reads at panel scale. */
const SIGNAL_TONE = {
  uncalibrated: 'bg-muted-foreground/40',
  learning: 'bg-amber-500',
  measured: 'bg-emerald-500',
} as const;

const fmtDate = (rfc3339: string): string => {
  const date = new Date(rfc3339);
  if (Number.isNaN(date.getTime())) return rfc3339;
  return date.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' });
};

/** Shared desktop/mobile advisory forecast. Display only; never triggers funding. */
export function CashReadinessCard({ data, isLoading, isError }: CashReadinessCardProps) {
  if (isLoading) {
    return (
      <div className="treasury-panel flex h-full flex-col gap-3" aria-busy="true" aria-label="Đang tải dự báo tiền trả">
        <div className="flex items-center justify-between gap-2">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-3 w-36" />
        </div>
        <Skeleton className="h-8 w-52" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="treasury-panel flex h-full flex-col justify-center" role="status">
        <span className="inline-flex items-center gap-1.5 text-xs font-bold uppercase tracking-[0.12em] text-muted-foreground">
          <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/50" aria-hidden="true" />
          Dự báo tiền trả
        </span>
        <p className="mt-2 text-[12px] text-muted-foreground">
          Tạm thời không có dữ liệu dự báo thanh toán.
        </p>
      </div>
    );
  }

  const display = getCashReadinessDisplay(data);
  const reserveSpan = display.intervalUpper - display.intervalLower;
  const reservePositionPct = reserveSpan > 0
    ? Math.min(100, Math.max(0, ((display.recommendedReserve - display.intervalLower) / reserveSpan) * 100))
    : 50;

  return (
    <section className="treasury-panel flex h-full flex-col" aria-label={`Dự báo tiền trả kỳ ${data.ky}`}>
      {/* Measurement mesh — faint dot grid under the panel label. */}
      <div className="treasury-mesh" aria-hidden />

      {/* Compact header — label and period share one line to keep the panel short. */}
      <div className="relative z-[1] flex flex-wrap items-center justify-between gap-x-2 gap-y-0.5">
        <span className="inline-flex items-center gap-1.5 text-xs font-bold uppercase tracking-[0.12em] text-muted-foreground">
          <span
            className={`treasury-signal ${SIGNAL_TONE[display.reliabilityState]}`}
            aria-hidden="true"
            title={display.reliabilityLabel}
          />
          Dự báo tiền trả
        </span>
        <span className="text-xs text-muted-foreground">
          Kỳ {data.ky} · Thanh toán ngày {fmtDate(data.next_pay_date)}
        </span>
      </div>

      <div className="relative z-[1] mt-1.5 min-w-0">
        <p className="text-xs font-semibold uppercase tracking-[0.12em] text-primary">
          Nên chuẩn bị
        </p>
        {/* Unit-suffixed so the figure never wraps mid-number in the 3-col
            panel; the exact amount stays in the tooltip. */}
        <p
          className="treasury-value mt-0.5 font-financial text-[clamp(1.375rem,2.2vw,1.75rem)] font-bold leading-tight tabular-nums tracking-tight text-primary"
          title={formatCurrency(display.recommendedReserve)}
        >
          {formatCompactVnd(display.recommendedReserve)}
        </p>
        {display.showExpectedPayout && (
          <p className="mt-1 text-xs text-muted-foreground">
            Dự kiến chi trả{' '}
            <span className="font-financial font-semibold tabular-nums text-foreground">
              {formatCurrency(display.expectedPayout)}
            </span>
          </p>
        )}
      </div>

      {display.dataWarning && (
        <p className="relative z-[1] mt-1.5 rounded-md border border-amber-500/25 bg-amber-500/5 px-2 py-1.5 text-xs leading-snug text-muted-foreground">
          {display.dataWarning}
        </p>
      )}

      <div className="treasury-rail relative z-[1] mt-2.5 pt-2.5">
        {/* One baseline line + a position marker instead of two truncating
            chips; exact bounds stay in the tooltip. The calibration caveat
            lives in the panel footer, out of the forecast column. */}
        <div className="flex items-baseline justify-between gap-2">
          <p className="text-[11px] font-medium text-muted-foreground">Dải dự báo trung tâm</p>
          <p
            className="font-financial text-xs font-semibold tabular-nums text-foreground"
            title={`Từ ${formatCurrency(display.intervalLower)} đến ${formatCurrency(display.intervalUpper)}`}
            aria-label={`Từ ${formatCurrency(display.intervalLower)} đến ${formatCurrency(display.intervalUpper)}`}
          >
            {formatCompactVnd(display.intervalLower)} – {formatCompactVnd(display.intervalUpper)}
          </p>
        </div>
        <div className="relative mt-1.5 h-1 rounded-full bg-muted/60">
          <div
            className="absolute top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary"
            style={{ left: `${reservePositionPct}%` }}
            role="presentation"
          />
        </div>
      </div>
    </section>
  );
}
