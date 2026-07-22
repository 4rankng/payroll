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

const fmtDate = (rfc3339: string): string => {
  const date = new Date(rfc3339);
  if (Number.isNaN(date.getTime())) return rfc3339;
  return date.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' });
};

/** Shared desktop/mobile advisory forecast. Display only; never triggers funding. */
export function CashReadinessCard({ data, isLoading, isError }: CashReadinessCardProps) {
  if (isLoading) {
    return (
      <div className="flex h-full flex-col gap-3" aria-busy="true" aria-label="Đang tải dự báo tiền trả">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-9 w-52" />
        <Skeleton className="h-14 w-full" />
        <Skeleton className="mt-1 h-16 w-full border-t border-border/60 pt-3" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="flex h-full flex-col justify-center" role="status">
        <span className="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
          <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/50" aria-hidden="true" />
          Dự báo tiền trả
        </span>
        <p className="mt-2 text-[13px] text-muted-foreground">
          Tạm thời không có dữ liệu dự báo thanh toán.
        </p>
      </div>
    );
  }

  const display = getCashReadinessDisplay(data);

  return (
    <section className="flex h-full flex-col" aria-label={`Dự báo tiền trả kỳ ${data.ky}`}>
      <span className="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
        <span className="h-1.5 w-1.5 rounded-full bg-primary/60" aria-hidden="true" />
        Dự báo tiền trả
      </span>

      <div className="mt-2 min-w-0">
        <p className="text-[10px] font-semibold uppercase tracking-[0.12em] text-primary">
          Nên chuẩn bị
        </p>
        <p className="mt-1 break-words font-financial text-[clamp(1.5rem,4vw,2rem)] font-bold leading-none tabular-nums tracking-tight text-primary">
          {formatCurrency(display.recommendedReserve)}
        </p>
        {display.showExpectedPayout && (
          <p className="mt-2 text-[11px] text-muted-foreground">
            Dự kiến chi trả{' '}
            <span className="font-financial font-semibold tabular-nums text-foreground">
              {formatCurrency(display.expectedPayout)}
            </span>
          </p>
        )}
      </div>

      <p className="mt-2 text-[11px] text-muted-foreground">
        Kỳ {data.ky} · Thanh toán ngày {fmtDate(data.next_pay_date)}
      </p>

      {display.dataWarning && (
        <p className="mt-2 rounded-md border border-amber-500/25 bg-amber-500/5 px-2 py-1.5 text-[11px] leading-snug text-muted-foreground">
          {display.dataWarning}
        </p>
      )}

      <div className="mt-3 border-t border-border/60 pt-3 lg:mt-auto">
        <p className="text-[11px] font-medium text-muted-foreground">Khoảng dự báo trung tâm</p>
        <dl
          className="mt-1.5 grid grid-cols-2 gap-2"
          aria-label={`Từ ${formatCurrency(display.intervalLower)} đến ${formatCurrency(display.intervalUpper)}`}
        >
          <div className="min-w-0 rounded-md bg-muted/45 px-2.5 py-2">
            <dt className="text-[9px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Từ</dt>
            <dd className="mt-0.5 whitespace-nowrap font-financial text-[clamp(0.7rem,2vw,0.8rem)] font-semibold tabular-nums text-foreground">
              {formatCurrency(display.intervalLower)}
            </dd>
          </div>
          <div className="min-w-0 rounded-md bg-primary/5 px-2.5 py-2 text-right">
            <dt className="text-[9px] font-semibold uppercase tracking-[0.12em] text-primary">Đến</dt>
            <dd className="mt-0.5 whitespace-nowrap font-financial text-[clamp(0.7rem,2vw,0.8rem)] font-semibold tabular-nums text-primary">
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
              <p className="text-[11px] font-semibold text-foreground">{display.reliabilityLabel}</p>
              <p className="text-[10px] leading-snug text-muted-foreground">{display.reliabilityDescription}</p>
              {display.accuracyMetrics.length > 0 && (
                <dl className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5">
                  {display.accuracyMetrics.map((metric) => (
                    <div key={metric.label} className="inline-flex gap-1 text-[10px]">
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
