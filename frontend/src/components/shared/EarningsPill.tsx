import { memo } from 'react';
import { Banknote } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

export interface EarningsPillProps {
  amount: number;
  /** Per-unit rate label, e.g. "150.000đ/h" */
  subtitle?: string;
  /** Amount above which the pill turns red (default: 3_500_000) */
  warnThreshold?: number;
  size?: 'sm' | 'default';
  isLoading?: boolean;
  className?: string;
}

/**
 * Compact earnings display pill — amount + optional rate subtitle.
 * Implements the earnings pill pattern from TimesheetEntryModal and GroupedEntryCard.
 */
export const EarningsPill = memo(function EarningsPill({
  amount,
  subtitle,
  warnThreshold = 3_500_000,
  size = 'default',
  isLoading = false,
  className,
}: EarningsPillProps) {
  if (isLoading) {
    return (
      <div
        className={cn(
          'flex items-center gap-1.5 px-3 rounded-xl bg-muted/50',
          size === 'default' ? 'h-11' : 'h-8',
          className,
        )}
      >
        <div className="w-3 h-3 border-2 border-muted-foreground border-t-transparent rounded-full animate-spin" />
        <span className="text-xs text-muted-foreground">Đang tải...</span>
      </div>
    );
  }

  const isWarn = amount > warnThreshold;

  return (
    <div className={cn('flex flex-col items-end gap-0.5', className)}>
      <span
        className={cn(
          'font-bold tabular-nums leading-none',
          size === 'default' ? 'text-xl' : 'text-base',
          isWarn ? 'text-red-600' : 'text-slate-800',
        )}
      >
        {amount.toLocaleString('vi-VN')}đ
      </span>
      {subtitle && (
        <span
          className={cn(
            'tabular-nums text-muted-foreground',
            size === 'default' ? 'text-xs' : 'text-[11px]',
          )}
        >
          {subtitle}
        </span>
      )}
    </div>
  );
});
