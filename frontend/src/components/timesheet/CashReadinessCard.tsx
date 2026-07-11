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
 * The headline uses the arithmetic mean of the forecast distribution, while
 * the full P50–P95 range remains visible below it. Wallet and advance-payment
 * information belongs to the wallet and advance-payment pages, not this
 * forecast.
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

      {/* Mathematical point estimate: confirmed payable + projected mean */}
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

      <dl className="mt-4 border-t border-border/60 pt-3 text-[11px]">
        <div className="flex flex-col gap-0.5">
          <dt className="text-muted-foreground">Khoảng dự báo</dt>
          <dd className="break-words font-financial font-medium tabular-nums text-foreground">
            {formatCurrency(data.band_lower)}–{formatCurrency(data.band_upper)}
          </dd>
        </div>
      </dl>
    </div>
  );
}
