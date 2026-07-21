import { AlertTriangle, CheckCircle2, Wallet as WalletIcon } from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';
import type { WalletDemandForecastResponse } from '@/types/api/wallet.types';

interface WalletDemandCardProps {
  data?: WalletDemandForecastResponse;
}

/**
 * Advisory balance-prediction card for the Wallet page. Mirrors the hero card's
 * visual language (rounded-2xl, watermark icon, emerald accent, font-financial).
 *
 * DISPLAY ONLY — the backend never feeds this back into SyncBalance / topups.
 */
export function WalletDemandCard({ data }: WalletDemandCardProps) {
  const pred = data?.prediction;
  const noHistory = pred?.method === 'no-history';
  const needsTopUp = (pred?.shortfall ?? 0) > 0;

  return (
    <div className="relative h-full flex flex-col justify-center overflow-hidden rounded-2xl border border-border/60 bg-card p-5 shadow-sm">
      {/* Watermark icon — bleeds bottom-right */}
      <WalletIcon
        className="absolute -right-4 -bottom-4 h-32 w-32 text-primary/[0.06] pointer-events-none"
        strokeWidth={1.2}
      />

      <div className="relative">
        <div className="flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-primary shadow-[0_0_0_3px_rgba(8,120,62,0.16)]" />
          <span className="text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
            Mức cần giữ trong ví
          </span>
        </div>

        {!pred ? (
          <p className="mt-3 text-xs text-muted-foreground">Chưa đủ dữ liệu</p>
        ) : (
          <>
            <p className="mt-2 break-words font-financial text-[clamp(1.375rem,6vw,1.625rem)] font-bold leading-tight tracking-normal tabular-nums text-foreground">
              {formatCurrency(pred.recommended_balance)}
            </p>
            <p className="text-[11px] text-muted-foreground mt-2">
              {noHistory
                ? 'Chưa đủ dữ liệu các kỳ trước. Hiển thị nhu cầu hiện tại chờ chi trả.'
                : 'Cần cho phần còn lại của kỳ'}
            </p>

            <div
              className={`mt-3 rounded-xl border p-3 ${
                needsTopUp ? 'border-rose-200 bg-rose-50' : 'border-emerald-200 bg-emerald-50'
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p
                    className={`text-[11px] font-semibold uppercase tracking-wider ${
                      needsTopUp ? 'text-rose-700' : 'text-emerald-700'
                    }`}
                  >
                    {needsTopUp ? 'Cần nạp thêm' : 'Số dư ví hiện có'}
                  </p>
                  <p
                    className={`mt-1 break-words font-financial text-base font-bold leading-tight tabular-nums ${
                      needsTopUp ? 'text-rose-700' : 'text-emerald-800'
                    }`}
                  >
                    {formatCurrency(needsTopUp ? pred.shortfall : pred.current_available)}
                  </p>
                </div>
                <span
                  className={`inline-flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-[11px] font-semibold ${
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
                  ? 'Nạp thêm để đạt mức cần giữ cho phần còn lại của kỳ.'
                  : noHistory
                    ? 'Đủ chi trả theo nhu cầu hiện tại.'
                    : 'Đủ chi trả cho phần còn lại của kỳ.'}
              </p>
            </div>

          </>
        )}
      </div>
    </div>
  );
}
