import { AlertTriangle, Banknote, CalendarClock, CheckCircle2 } from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';
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

const confidenceStyles: Record<string, { badge: string; dot: string; label: string }> = {
  high: { badge: 'bg-emerald-100 text-emerald-700', dot: 'bg-emerald-500', label: 'Độ tin cậy cao' },
  medium: { badge: 'bg-amber-100 text-amber-700', dot: 'bg-amber-500', label: 'Độ tin cậy trung bình' },
  low: { badge: 'bg-rose-100 text-rose-700', dot: 'bg-rose-500', label: 'Độ tin cậy thấp' },
};

/**
 * Advisory cash-prep card for the next timesheet bulk transfer. Mirrors the
 * wallet demand card's visual language (rounded-2xl, watermark, font-financial,
 * tabular-nums) but uses a neutral + emerald/amber/rose palette (no gold).
 *
 * DISPLAY ONLY — the backend never feeds this into balance/disbursement.
 */
export function CashReadinessCard({ data, isLoading, isError }: CashReadinessCardProps) {
  if (isLoading) {
    return (
      <div className="h-[208px] animate-pulse rounded-2xl border border-border/60 bg-card p-5 shadow-sm">
        <div className="h-3 w-32 rounded bg-muted" />
        <div className="mt-4 h-8 w-48 rounded bg-muted" />
        <div className="mt-3 h-3 w-40 rounded bg-muted/70" />
        <div className="mt-4 h-16 rounded-xl bg-muted/50" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="flex h-[208px] flex-col justify-center rounded-2xl border border-border/60 bg-card p-5 shadow-sm">
        <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
          Chuẩn bị tiền trả
        </span>
        <p className="mt-3 text-xs text-muted-foreground">
          Tạm thời không có dữ liệu dự báo.
        </p>
      </div>
    );
  }

  const needsCash = data.gap > 0;
  const conf = confidenceStyles[data.confidence] ?? confidenceStyles.low;
  const lowConfidence = data.confidence === 'low' || data.method === 'no-history';

  return (
    <div className="relative flex h-full flex-col justify-center overflow-hidden rounded-2xl border border-border/60 bg-card p-5 shadow-sm">
      <Banknote
        className="pointer-events-none absolute -bottom-4 -right-4 h-32 w-32 text-slate-500/[0.06]"
        strokeWidth={1.2}
      />

      <div className="relative">
        <div className="flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-slate-400 shadow-[0_0_0_3px_rgba(100,116,139,0.16)]" />
          <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-slate-600">
            Chuẩn bị tiền trả
          </span>
        </div>

        <p className="mt-2 break-words font-financial text-[clamp(1.375rem,6vw,1.625rem)] font-bold leading-tight tracking-normal tabular-nums text-foreground">
          {formatCurrency(data.cash_to_prepare)}
        </p>
        <p className="mt-1 text-[11px] text-muted-foreground">
          Cần chuẩn bị cho kỳ {data.ky} — trả ngày {fmtDate(data.next_pay_date)}
        </p>

        <div
          className={`mt-3 rounded-xl border p-3 ${
            needsCash ? 'border-rose-200 bg-rose-50' : 'border-emerald-200 bg-emerald-50'
          }`}
        >
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <p
                className={`text-[10px] font-semibold uppercase tracking-wider ${
                  needsCash ? 'text-rose-700' : 'text-emerald-700'
                }`}
              >
                {needsCash ? 'Cần thêm tiền' : 'Đã đủ tiền'}
              </p>
              <p
                className={`mt-1 break-words font-financial text-base font-bold leading-tight tabular-nums ${
                  needsCash ? 'text-rose-700' : 'text-emerald-800'
                }`}
              >
                {formatCurrency(needsCash ? data.gap : data.wallet_available)}
              </p>
              <p className={`mt-1 text-[11px] ${needsCash ? 'text-rose-700' : 'text-emerald-700'}`}>
                {needsCash
                  ? `Số dư ví ${formatCurrency(data.wallet_available)} · thiếu cho lần trả tiếp`
                  : 'Số dư ví đủ chi trả lần tới'}
              </p>
            </div>
            <span
              className={`inline-flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-[10.5px] font-semibold ${
                needsCash ? 'bg-rose-100 text-rose-700' : 'bg-emerald-100 text-emerald-700'
              }`}
            >
              {needsCash ? <AlertTriangle className="h-3.5 w-3.5" /> : <CheckCircle2 className="h-3.5 w-3.5" />}
              {needsCash ? 'Thiếu' : 'Đủ'}
            </span>
          </div>
        </div>

        {/* Breakdown: confirmed now + projected p50, band, prepare-by */}
        <div className="mt-3 grid grid-cols-2 gap-2 text-[11px]">
          <div className="rounded-lg bg-muted/40 px-2.5 py-2">
            <p className="text-muted-foreground">Đã chốt (chờ TT)</p>
            <p className="mt-0.5 font-semibold tabular-nums text-foreground">
              {formatCurrency(data.confirmed_payable)}
            </p>
          </div>
          <div className="rounded-lg bg-muted/40 px-2.5 py-2">
            <p className="text-muted-foreground">Dự báo thêm (p50)</p>
            <p className="mt-0.5 font-semibold tabular-nums text-foreground">
              {formatCurrency(data.projected_p50)}
            </p>
          </div>
        </div>

        <div className="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
          <span>
            Khoảng:{' '}
            <span className="font-medium tabular-nums text-foreground">
              {formatCurrency(data.band_lower)}–{formatCurrency(data.band_upper)}
            </span>
          </span>
          <span className="inline-flex items-center gap-1">
            <CalendarClock className="h-3.5 w-3.5" />
            Chuẩn bị trước {fmtDate(data.prepare_by_date)}
          </span>
        </div>

        {/* Confidence badge — never show a bare p50 without it */}
        <div className="mt-2 flex items-center gap-1.5">
          <span className={`h-1.5 w-1.5 rounded-full ${conf.dot}`} />
          <span className={`inline-flex items-center rounded-md px-1.5 py-0.5 text-[10px] font-semibold ${conf.badge}`}>
            {conf.label}
          </span>
          {lowConfidence && (
            <span className="text-[10px] text-muted-foreground">
              · ít dữ liệu ({data.basis_cycles} kỳ)
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
