import { formatCurrency } from '@/utils/formatters';
import { Skeleton } from '@/components/ui/skeleton';
import type { CashReadinessResponse } from '@/types/api/cash-readiness.types';

const CONFIDENCE_LABELS: Record<string, string> = {
  high: 'Độ tin cậy cao',
  medium: 'Độ tin cậy vừa',
  low: 'Độ tin cậy thấp',
};

const CONFIDENCE_DOT: Record<string, string> = {
  high: 'bg-emerald-500/70',
  medium: 'bg-amber-500/70',
  low: 'bg-rose-500/70',
};

interface CashReadinessCardProps {
  data?: CashReadinessResponse;
  isLoading?: boolean;
  isError?: boolean;
}

const fmtDate = (rfc3339: string): string => {
  const d = new Date(rfc3339);
  if (Number.isNaN(d.getTime())) return rfc3339;
  return d.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' });
};

/**
 * Advisory payment forecast for the next timesheet bulk transfer.
 *
 * The headline is the expected final payout for the target Ky only: approvals
 * already observed inside that Ky plus projected approvals through its pay
 * date. The full P50–P95 range remains visible below it. Outstanding payments,
 * wallet, and advance-payment information are separate operational concerns.
 *
 * DISPLAY ONLY — the backend never feeds this into balance/disbursement.
 */
export function CashReadinessCard({ data, isLoading, isError }: CashReadinessCardProps) {
  if (isLoading) {
    return (
      <div className="flex h-full flex-col gap-3" aria-busy="true">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-9 w-52" />
        <Skeleton className="h-3 w-40" />
        <Skeleton className="mt-2 h-8 w-44 border-t border-border/60 pt-3" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="flex h-full flex-col justify-center">
        <span className="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
          <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/50" />
          Dự báo tiền trả
        </span>
        <p className="mt-2 text-[13px] text-muted-foreground">
          Tạm thời không có dữ liệu dự báo thanh toán.
        </p>
      </div>
    );
  }

  const forecastTotal = Math.max(data.expected_total, 0);

  return (
    <div className="flex h-full flex-col">
      <span className="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
        <span className="h-1.5 w-1.5 rounded-full bg-primary/60" />
        Dự báo tiền trả
      </span>

      {/* Target-Ky point estimate: observed approved + projected remaining mean */}
      <p className="mt-2 font-financial text-[clamp(1.75rem,5vw,2.125rem)] font-bold leading-none tabular-nums tracking-tight text-foreground">
        {formatCurrency(forecastTotal)}
      </p>
      <p className="mt-1.5 text-[12px] text-muted-foreground">
        Kỳ {data.ky} · Thanh toán ngày {fmtDate(data.next_pay_date)}
      </p>

      {data.confidence && data.basis_cycles > 0 && data.method !== 'no-history' && (
        <p className="mt-1 text-[11px] text-muted-foreground">
          <span className={`mr-1 inline-block h-1.5 w-1.5 rounded-full align-middle ${CONFIDENCE_DOT[data.confidence] ?? 'bg-muted-foreground/50'}`} />
          {CONFIDENCE_LABELS[data.confidence] ?? data.confidence} · {data.basis_cycles} kỳ dữ liệu
        </p>
      )}

      <dl className="mt-5 border-t border-border/60 pt-3 lg:mt-auto">
        <dt className="text-[11px] font-medium text-muted-foreground">Khoảng dự báo</dt>
        <dd
          className="mt-2 grid grid-cols-1 gap-2.5 sm:grid-cols-2 sm:gap-0"
          aria-label={`Từ ${formatCurrency(data.band_lower)} đến ${formatCurrency(data.band_upper)}`}
        >
          <div className="min-w-0 sm:pr-5">
            <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
              Từ
            </span>
            <span className="mt-1 block whitespace-nowrap font-financial text-[clamp(0.95rem,1.35vw,1.125rem)] font-semibold leading-none tabular-nums text-foreground">
              {formatCurrency(data.band_lower)}
            </span>
          </div>

          <div className="min-w-0 border-t border-border/50 pt-2.5 sm:border-l sm:border-t-0 sm:py-0 sm:pl-5 sm:text-right">
            <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
              Đến
            </span>
            <span className="mt-1 block whitespace-nowrap font-financial text-[clamp(0.95rem,1.35vw,1.125rem)] font-semibold leading-none tabular-nums text-foreground">
              {formatCurrency(data.band_upper)}
            </span>
          </div>
        </dd>
      </dl>
    </div>
  );
}
