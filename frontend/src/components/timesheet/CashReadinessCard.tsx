import { AlertTriangle, CalendarClock } from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';
import { FundingProgress } from './FundingProgress';
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
 * Advisory cash-prep panel for the next timesheet bulk transfer.
 *
 * Treasury-ledger layout: eyebrow → headline figure → pay date / deadline →
 * a three-line funding equation (the shortfall is the only red surface) →
 * funding meter → compact forecast metadata.
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
        <div className="mt-1 space-y-2 border-t border-border/60 pt-3">
          <Skeleton className="h-5 w-full" />
          <Skeleton className="h-5 w-full" />
          <Skeleton className="h-5 w-full" />
        </div>
        <Skeleton className="h-2 w-full rounded-full" />
        <div className="grid grid-cols-2 gap-2 pt-1">
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
          Cần chuẩn bị
        </span>
        <p className="mt-2 text-[13px] text-muted-foreground">
          Tạm thời không có dữ liệu dự báo thanh toán.
        </p>
      </div>
    );
  }

  const required = Math.max(data.cash_to_prepare, 0);
  const surplus = data.wallet_available - required;
  const needsCash = surplus < 0;
  const conf = confidenceMeta[data.confidence] ?? confidenceMeta.low;
  const lowConfidence = data.confidence === 'low' || data.method === 'no-history';

  return (
    <div className="flex h-full flex-col">
      {/* Eyebrow + status badge */}
      <div className="flex items-center justify-between gap-2">
        <span className="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
          <span className="h-1.5 w-1.5 rounded-full bg-primary/60" />
          Cần chuẩn bị
        </span>
        {needsCash ? (
          <span className="inline-flex items-center gap-1 rounded-md bg-rose-50 px-1.5 py-0.5 text-[10.5px] font-semibold text-rose-700 ring-1 ring-rose-100">
            <AlertTriangle className="h-3 w-3" strokeWidth={2.25} />
            Thiếu tiền
          </span>
        ) : (
          <span className="inline-flex items-center gap-1 rounded-md bg-emerald-50 px-1.5 py-0.5 text-[10.5px] font-semibold text-emerald-700 ring-1 ring-emerald-100">
            Đủ tiền
          </span>
        )}
      </div>

      {/* Headline figure */}
      <p className="mt-2 font-financial text-[clamp(1.75rem,5vw,2.125rem)] font-bold leading-none tabular-nums tracking-tight text-foreground">
        {formatCurrency(required)}
      </p>
      <p className="mt-1.5 text-[12px] text-muted-foreground">
        Kỳ {data.ky} · Thanh toán ngày {fmtDate(data.next_pay_date)}
      </p>
      <p className="mt-1 inline-flex items-center gap-1 text-[11px] font-medium text-amber-600">
        <CalendarClock className="h-3.5 w-3.5" strokeWidth={2} />
        Chuẩn bị trước {fmtDate(data.prepare_by_date)}
      </p>

      {/* Funding equation */}
      <div className="mt-3 space-y-0.5 border-t border-border/60 pt-2">
        <EquationRow label="Số dư ví" amount={data.wallet_available} />
        <EquationRow label="Số tiền cần trả" amount={required} />
        <EquationRow
          label={needsCash ? 'Thiếu hụt' : 'Thặng dư'}
          amount={Math.abs(surplus)}
          tone={needsCash ? 'warn' : 'good'}
        />
      </div>

      {/* Funding meter */}
      <div className="mt-3">
        <FundingProgress available={data.wallet_available} required={required} />
      </div>

      {/* Forecast metadata — concise rows, not four cards */}
      <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 border-t border-border/60 pt-2 text-[11px]">
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

interface EquationRowProps {
  label: string;
  amount: number;
  tone?: 'warn' | 'good';
}

function EquationRow({ label, amount, tone }: EquationRowProps) {
  const isWarn = tone === 'warn';
  const isGood = tone === 'good';
  return (
    <div
      className={cn(
        'flex items-center justify-between gap-3 rounded-md px-2 py-1.5 -mx-2',
        isWarn && 'bg-rose-50',
        isGood && 'bg-emerald-50/60',
      )}
    >
      <span
        className={cn(
          'text-[12px]',
          isWarn ? 'font-semibold text-rose-700' : isGood ? 'font-semibold text-emerald-700' : 'text-muted-foreground',
        )}
      >
        {label}
      </span>
      <span
        className={cn(
          'font-financial text-[13px] font-semibold tabular-nums',
          isWarn ? 'text-rose-700' : isGood ? 'text-emerald-700' : 'text-foreground',
        )}
      >
        {formatCurrency(amount)}
      </span>
    </div>
  );
}
