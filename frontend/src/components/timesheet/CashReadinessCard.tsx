import { formatCurrency } from '@/utils/formatters';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type { CashReadinessResponse } from '@/types/api/cash-readiness.types';

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

const confidenceMeta: Record<string, { label: string; dot: string; text: string }> = {
  high: { label: 'Cao', dot: 'bg-emerald-500', text: 'text-emerald-600' },
  medium: { label: 'Trung bình', dot: 'bg-amber-500', text: 'text-amber-600' },
  low: { label: 'Thấp', dot: 'bg-rose-500', text: 'text-rose-600' },
};

/**
 * Advisory payment forecast for the next timesheet bulk transfer.
 *
 * The timesheet page is intentionally limited to the amount expected on the
 * next pay date. Wallet and advance-payment information belongs to the wallet
 * and advance-payment pages, not this forecast.
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
        <div className="mt-2 grid grid-cols-2 gap-3 border-t border-border/60 pt-3">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
        </div>
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

  const forecastTotal = Math.max(data.cash_to_prepare, 0);
  const conf = confidenceMeta[data.confidence] ?? confidenceMeta.low;
  const lowConfidence = data.confidence === 'low' || data.method === 'no-history';

  return (
    <div className="flex h-full flex-col">
      <span className="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
        <span className="h-1.5 w-1.5 rounded-full bg-primary/60" />
        Dự báo tiền trả
      </span>

      {/* Headline figure */}
      <p className="mt-2 font-financial text-[clamp(1.75rem,5vw,2.125rem)] font-bold leading-none tabular-nums tracking-tight text-foreground">
        {formatCurrency(forecastTotal)}
      </p>
      <p className="mt-1.5 text-[12px] text-muted-foreground">
        Kỳ {data.ky} · Thanh toán ngày {fmtDate(data.next_pay_date)}
      </p>

      {/* Forecast metadata — concise rows, not four cards */}
      <dl className="mt-4 grid grid-cols-2 gap-x-4 gap-y-2 border-t border-border/60 pt-3 text-[11px]">
        <div className="flex flex-col gap-0.5">
          <dt className="text-muted-foreground">Đã chốt</dt>
          <dd className="font-financial font-semibold tabular-nums text-foreground">
            {formatCurrency(data.confirmed_payable)}
          </dd>
        </div>
        <div className="flex flex-col gap-0.5">
          <dt className="text-muted-foreground">Dự báo thêm (p50)</dt>
          <dd className="font-financial font-semibold tabular-nums text-foreground">
            {formatCurrency(data.projected_p50)}
          </dd>
        </div>
        <div className="flex flex-col gap-0.5">
          <dt className="text-muted-foreground">Khoảng dự báo</dt>
          <dd className="break-words font-financial font-medium tabular-nums text-foreground">
            {formatCurrency(data.band_lower)}–{formatCurrency(data.band_upper)}
          </dd>
        </div>
        <div className="flex flex-col gap-0.5">
          <dt className="text-muted-foreground">Độ tin cậy</dt>
          <dd className={cn('inline-flex items-center gap-1 font-semibold', conf.text)}>
            <span className={cn('h-1.5 w-1.5 shrink-0 rounded-full', conf.dot)} />
            {conf.label}
            {lowConfidence && (
              <span className="font-normal text-muted-foreground">· {data.basis_cycles} kỳ</span>
            )}
          </dd>
        </div>
      </dl>
    </div>
  );
}
