import { Wallet as WalletIcon } from 'lucide-react';
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
                : 'Nên giữ trong ví để chi trả phần còn lại của kỳ này'}
            </p>

            <div className="mt-3 grid grid-cols-2 gap-2">
              <div className="rounded-lg border border-border/60 bg-muted/40 p-2.5">
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">
                  Số dư khả dụng
                </p>
                <p className="mt-0.5 font-financial text-sm font-bold tabular-nums text-foreground">
                  {formatCurrency(pred.current_available)}
                </p>
              </div>
              {pred.shortfall > 0 ? (
                <div className="rounded-lg border border-rose-200 bg-rose-50 p-2.5">
                  <p className="text-[10px] uppercase tracking-wider text-rose-700 font-semibold">
                    Cần nạp thêm
                  </p>
                  <p className="mt-0.5 font-financial text-sm font-bold tabular-nums text-rose-700">
                    {formatCurrency(pred.shortfall)}
                  </p>
                </div>
              ) : (
                <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-2.5">
                  <p className="text-[10px] uppercase tracking-wider text-emerald-700 font-semibold">
                    Dư khả dụng
                  </p>
                  <p className="mt-0.5 font-financial text-sm font-bold tabular-nums text-emerald-700">
                    {formatCurrency(pred.surplus)}
                  </p>
                </div>
              )}
            </div>

            <p className="mt-3 text-[10.5px] text-muted-foreground">
              Ước tính theo {METHOD_VN[pred.method] ?? pred.method} · Độ tin cậy:{' '}
              {CONFIDENCE_VN[pred.confidence] ?? pred.confidence}
            </p>
          </>
        )}
      </div>
    </div>
  );
}
