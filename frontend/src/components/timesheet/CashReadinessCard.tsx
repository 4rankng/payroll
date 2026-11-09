import { Skeleton } from '@/components/ui/skeleton';
import type { CashReadinessResponse } from '@/types/api/cash-readiness.types';
import { formatCurrency } from '@/utils/formatters';
import { getCashReadinessDisplay } from './cashReadinessDisplay';

interface CashReadinessCardProps {
  data?: CashReadinessResponse;
  isLoading?: boolean;
  isError?: boolean;
}

const RELIABILITY_DOT = {
  uncalibrated: 'bg-muted-foreground/50',
  learning: 'bg-amber-500/80',
  measured: 'bg-emerald-500/80',
} as const;

/** Label LED tones — calibration semantics, matching RELIABILITY_DOT at full
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
        <p className="treasury-value mt-0.5 break-words font-financial text-[clamp(1.25rem,2.5vw,1.5rem)] font-bold leading-none tabular-nums tracking-tight text-primary">
          {formatCurrency(display.recommendedReserve)}
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

      <div className="treasury-rail relative z-[1] mt-2 pt-2 lg:mt-auto">
        <p className="mb-1 text-[11px] font-medium text-muted-foreground">Khoảng dự báo trung tâm</p>
        {/* TỪ/ĐẾN share one baseline: identical cell anatomy, both left-aligned —
            only the tint differs, never the alignment or type size. */}
        <dl
          className="grid grid-cols-2 gap-2"
          aria-label={`Từ ${formatCurrency(display.intervalLower)} đến ${formatCurrency(display.intervalUpper)}`}
        >
          <div className="min-w-0 rounded-md bg-muted/45 px-2.5 py-1.5">
            <dt className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Từ</dt>
            <dd
              className="mt-0.5 truncate font-financial text-sm font-semibold tabular-nums text-foreground"
              title={formatCurrency(display.intervalLower)}
            >
              {formatCurrency(display.intervalLower)}
            </dd>
          </div>
          <div className="min-w-0 rounded-md bg-primary/5 px-2.5 py-1.5">
            <dt className="text-[11px] font-semibold uppercase tracking-[0.12em] text-primary">Đến</dt>
            <dd
              className="mt-0.5 truncate font-financial text-sm font-semibold tabular-nums text-primary"
              title={formatCurrency(display.intervalUpper)}
            >
              {formatCurrency(display.intervalUpper)}
            </dd>
          </div>
        </dl>

        {display.reliabilityState !== 'uncalibrated' && (
          <div className="mt-2 flex items-start gap-2 rounded-md bg-muted/45 px-2 py-1.5">
            <span
              className={`mt-1 h-1.5 w-1.5 shrink-0 rounded-full ${RELIABILITY_DOT[display.reliabilityState]}`}
              aria-hidden="true"
            />
            <div className="min-w-0">
              <p className="text-xs font-semibold text-foreground">{display.reliabilityLabel}</p>
              <p className="text-xs leading-snug text-muted-foreground">{display.reliabilityDescription}</p>
              {display.accuracyMetrics.length > 0 && (
                <dl className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5">
                  {display.accuracyMetrics.map((metric) => (
                    <div key={metric.label} className="inline-flex gap-1 text-xs">
                      <dt className="text-muted-foreground">{metric.label}</dt>
                      <dd className="font-semibold tabular-nums text-foreground">{metric.value}</dd>
                    </div>
                  ))}
                </dl>
              )}
            </div>
          </div>
        )}
      </div>
    </section>
  );
}
