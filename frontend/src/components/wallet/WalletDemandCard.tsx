import { AlertTriangle, CheckCircle2, Wallet as WalletIcon } from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';
import type { WalletDemandForecastResponse } from '@/types/api/wallet.types';

interface WalletDemandCardProps {
  data?: WalletDemandForecastResponse;
}

const CONFIDENCE_VN: Record<string, string> = {
  high: 'cao',
  medium: 'trung bình',
  low: 'thấp',
};

const METHOD_VN: Record<string, string> = {
  'monte-carlo': 'mô phỏng Monte Carlo',
  'gamma-fit': 'phân phối gamma',
  'cohort-median': 'trung vị các kỳ trước',
  'avg-final': 'trung bình tổng các kỳ trước',
  'no-history': 'nhu cầu hiện tại',
};

/**
 * Advisory balance-prediction card for the Wallet page. Mirrors the hero card's
 * visual language (rounded-2xl, watermark icon, amber accent, font-financial).
 *
 * DISPLAY ONLY — the backend never feeds this back into SyncBalance / topups.
 */
export function WalletDemandCard({ data }: WalletDemandCardProps) {
  const pred = data?.prediction;
  const noHistory = pred?.method === 'no-history';
  const covPct = Math.round((pred?.coverage_probability ?? 0.95) * 100);
  const needsTopUp = (pred?.shortfall ?? 0) > 0;

  return (
    <div className="relative h-full flex flex-col justify-center overflow-hidden rounded-2xl border border-border/60 bg-card p-5 shadow-sm">
      {/* Watermark icon — bleeds bottom-right */}
      <WalletIcon
        className="absolute -right-4 -bottom-4 h-32 w-32 text-amber-500/[0.06] pointer-events-none"
        strokeWidth={1.2}
      />

      <div className="relative">
        <div className="flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-amber-500 shadow-[0_0_0_3px_rgba(245,158,11,0.18)]" />
          <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-amber-700">
            Dự báo nhu cầu ứng lương
          </span>
        </div>

        {!pred ? (
          <p className="mt-3 text-xs text-muted-foreground">Chưa đủ dữ liệu</p>
        ) : (
          <>
            <p className="mt-2 font-financial text-[26px] font-bold leading-none tracking-[-0.02em] tabular-nums text-foreground">
              {formatCurrency(pred.recommended_balance)}
            </p>
            <p className="text-[11px] text-muted-foreground mt-2">
              {noHistory
                ? 'Chưa đủ dữ liệu các kỳ trước. Hiển thị nhu cầu hiện tại chờ chi trả.'
                : 'Mức nên giữ lại để chi trả phần còn lại của kỳ này'}
            </p>

            <div
              className={`mt-3 rounded-xl border p-3 ${
                needsTopUp ? 'border-rose-200 bg-rose-50' : 'border-emerald-200 bg-emerald-50'
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p
                    className={`text-[10px] font-semibold uppercase tracking-wider ${
                      needsTopUp ? 'text-rose-700' : 'text-emerald-700'
                    }`}
                  >
                    {needsTopUp ? 'Cần nạp thêm' : 'Số dư ví hiện có'}
                  </p>
                  <p
                    className={`mt-1 font-financial text-base font-bold leading-none tabular-nums ${
                      needsTopUp ? 'text-rose-700' : 'text-emerald-800'
                    }`}
                  >
                    {formatCurrency(needsTopUp ? pred.shortfall : pred.current_available)}
                  </p>
                </div>
                <span
                  className={`inline-flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-[10.5px] font-semibold ${
                    needsTopUp ? 'bg-rose-100 text-rose-700' : 'bg-emerald-100 text-emerald-700'
                  }`}
                >
                  {needsTopUp ? (
                    <AlertTriangle className="h-3.5 w-3.5" />
                  ) : (
                    <CheckCircle2 className="h-3.5 w-3.5" />
                  )}
                  {needsTopUp ? 'Chưa đủ' : 'Đủ chi trả'}
                </span>
              </div>
              <p className={`mt-2 text-[11px] ${needsTopUp ? 'text-rose-700' : 'text-emerald-700'}`}>
                {needsTopUp
                  ? 'Nạp thêm để đạt mức nên giữ lại trong ví.'
                  : noHistory
                    ? 'Đủ chi trả theo nhu cầu hiện tại.'
                    : 'Đủ chi trả theo mức dự báo.'}
              </p>
            </div>

            <p className="mt-3 text-[10.5px] text-muted-foreground">
              Phủ {covPct}% các kỳ tương tự · {METHOD_VN[pred.method] ?? pred.method} · Tin cậy:{' '}
              {CONFIDENCE_VN[pred.confidence] ?? pred.confidence}
            </p>
            {!noHistory && pred.p50_reference != null && (
              <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px] text-muted-foreground">
                {(
                  [
                    ['p50', pred.p50_reference],
                    ['p90', pred.p90_reference],
                    ['p99', pred.p99_reference],
                  ] as const
                ).map(([label, val]) => (
                  <span key={label} className="inline-flex items-center gap-1">
                    <span className="font-semibold uppercase">{label}</span>
                    <span className="tabular-nums">{formatCurrency(val ?? 0)}</span>
                  </span>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
