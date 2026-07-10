import { cn } from '@/lib/utils';

interface FundingProgressProps {
  /** Wallet balance currently available (VND). */
  available: number;
  /** Cash required for the next payment (VND). */
  required: number;
  className?: string;
}

/**
 * Funding meter for the cash-readiness panel.
 *
 * Fill width = wallet balance ÷ required cash. Navy while funding is
 * incomplete, emerald once the wallet covers the requirement. Red is never
 * used on the bar itself — the shortfall is called out separately in the
 * equation above, so the bar stays a calm progress indicator.
 */
export function FundingProgress({ available, required, className }: FundingProgressProps) {
  const safeRequired = Math.max(required, 0);
  const ratio = safeRequired > 0 ? Math.min(available / safeRequired, 1) : 1;
  const sufficient = available >= safeRequired && safeRequired > 0;

  // One decimal, vi-VN decimal comma — e.g. 11,4%.
  const pct = safeRequired > 0
    ? (Math.max(available, 0) / safeRequired) * 100
    : 100;
  const pctLabel = sufficient
    ? 'Đã đủ tiền'
    : `Đã có ${pct.toLocaleString('vi-VN', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}% số tiền cần thiết`;

  // Keep a sliver visible even when funded <2% so the bar is never invisible.
  const fillWidth = `${Math.max(ratio * 100, sufficient ? 100 : 3)}%`;

  return (
    <div className={cn('space-y-1.5', className)}>
      <div className="flex items-center justify-between gap-2">
        <span className="text-[11px] font-medium text-muted-foreground">
          Mức vốn sẵn có
        </span>
        <span
          className={cn(
            'text-[11px] font-semibold tabular-nums',
            sufficient ? 'text-emerald-600' : 'text-foreground',
          )}
        >
          {pctLabel}
        </span>
      </div>
      <div
        className="h-2 w-full overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(ratio * 100)}
        aria-label="Tỷ lệ vốn đã sẵn có"
      >
        <div
          className={cn(
            'h-full rounded-full transition-[width] duration-300 ease-out',
            sufficient ? 'bg-emerald-500' : 'bg-primary',
          )}
          style={{ width: fillWidth }}
        />
      </div>
    </div>
  );
}
