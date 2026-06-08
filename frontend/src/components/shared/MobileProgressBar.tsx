import { cn } from '@/lib/utils';

interface MobileProgressBarProps {
  /** Percentage (0-100) */
  value: number;
  /** Color threshold config */
  thresholds?: {
    low: number;
    mid: number;
    colors: { low: string; mid: string; high: string };
  };
  /** Optional label above the bar */
  label?: string;
  /** Optional value text displayed at the right */
  displayValue?: string;
  /** Optional class override */
  className?: string;
}

const DEFAULT_THRESHOLDS = {
  low: 50,
  mid: 80,
  colors: {
    low: 'bg-emerald-400',
    mid: 'bg-amber-400',
    high: 'bg-orange-500',
  },
};

/**
 * Compact progress bar with color thresholds.
 * Used in advance payment usage, quota displays, etc.
 */
export const MobileProgressBar = ({
  value,
  thresholds = DEFAULT_THRESHOLDS,
  label,
  displayValue,
  className,
}: MobileProgressBarProps) => {
  const barColor =
    value > thresholds.mid
      ? thresholds.colors.high
      : value > thresholds.low
        ? thresholds.colors.mid
        : thresholds.colors.low;

  return (
    <div className={cn('space-y-1.5', className)}>
      {(label || displayValue) && (
        <div className="flex items-center justify-between text-xs">
          {label && <span className="text-muted-foreground">{label}</span>}
          {displayValue && (
            <span className="font-medium tabular-nums">{displayValue}</span>
          )}
        </div>
      )}
      <div className="h-2 bg-muted rounded-full overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all', barColor)}
          style={{ width: `${Math.min(100, Math.max(0, value))}%` }}
        />
      </div>
    </div>
  );
};
