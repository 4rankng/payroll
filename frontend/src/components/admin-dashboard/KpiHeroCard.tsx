import { memo } from 'react';
// Staff day console: the `icon` prop stays in the interface for caller
// compatibility (both lucide and @untitledui/icons fit the ComponentType
// slot) but is no longer rendered — console strip cells carry a kicker
// label, the figure, and a hint, with no icon-box decoration.
import type { ComponentType, SVGProps } from 'react';
import { cn } from '@/lib/utils';
import { useCountUp } from '@/hooks/useCountUp';
import { splitCurrencyDisplay } from '@/utils/formatters';

export interface KpiHeroCardProps {
  label: string;
  value: string | number;
  unit?: string;
  formattedValue?: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  color: 'blue' | 'emerald' | 'amber' | 'teal' | 'rose';
  sublabel?: string;
  trend?: { value: string; positive: boolean };
  badge?: { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral' };
  /** Optional rich footer (secondary stats) rendered below the value block */
  footer?: React.ReactNode;
  isActive?: boolean;
  onClick?: () => void;
  className?: string;
  variant?: 'compact' | 'stack';
}

const BADGE_STYLES: Record<NonNullable<KpiHeroCardProps['badge']>['variant'], string> = {
  success: 'bg-success/10 text-success',
  warning: 'bg-warning/10 text-warning',
  danger: 'bg-destructive/10 text-destructive',
  neutral: 'bg-muted text-muted-foreground',
};

const COLOR_STYLES: Record<
  KpiHeroCardProps['color'],
  { accent: string }
> = {
  blue: { accent: 'bg-info' },
  emerald: { accent: 'bg-primary' },
  amber: { accent: 'bg-warning' },
  teal: { accent: 'bg-success' },
  rose: { accent: 'bg-destructive' },
};

export const KpiHeroCard = memo(function KpiHeroCard({
  label,
  value,
  unit,
  formattedValue,
  color,
  sublabel,
  trend,
  badge,
  footer,
  isActive,
  onClick,
  className,
  variant = 'compact',
}: KpiHeroCardProps) {
  const numericValue = typeof value === 'number' ? value : 0;
  const animated = useCountUp(numericValue, 800);
  const colorStyle = COLOR_STYLES[color];

  const displayValue = formattedValue
    ? formattedValue
    : typeof value === 'number'
      ? animated.toLocaleString('vi-VN')
      : value;
  const { amount: displayAmount, unit: displayUnit } = splitCurrencyDisplay(displayValue, unit);
  const CardRoot = onClick ? 'button' : 'div';

  return (
    <CardRoot
      className={cn(
        'console-panel group relative h-full w-full overflow-hidden text-left',
        isActive && 'border-primary/50',
        onClick && 'cursor-pointer hover:border-primary/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        className,
      )}
      onClick={onClick}
      {...(onClick ? { type: 'button' as const } : {})}
    >
      <div className={cn('relative flex h-full flex-col p-4', variant === 'stack' && 'sm:p-5')}>
        <div className="flex items-start justify-between gap-2">
          <p className="console-kicker min-w-0 pt-0.5">{label}</p>
          {(trend || badge) && (
            <div className="flex flex-wrap items-center justify-end gap-1.5">
              {trend && (
                <span className={cn(
                  'inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs font-semibold tabular-nums',
                  trend.positive ? 'bg-success/10 text-success' : 'bg-destructive/10 text-destructive',
                )}>
                  <span aria-hidden="true">{trend.positive ? '↑' : '↓'}</span>
                  {trend.value}
                </span>
              )}
              {badge && (
                <span className={cn(
                  'inline-flex items-center rounded-full px-2 py-1 text-xs font-semibold whitespace-nowrap',
                  BADGE_STYLES[badge.variant],
                )}>
                  {badge.label}
                </span>
              )}
            </div>
          )}
        </div>

        <p className={cn(
          'mt-1.5 console-value font-display font-semibold leading-tight tracking-[-0.01em] text-foreground text-xl sm:text-2xl',
          variant === 'stack' ? 'whitespace-nowrap' : 'break-words',
        )}>
          <span>{displayAmount}</span>
          {displayUnit ? (
            <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">
              {displayUnit}
            </span>
          ) : null}
        </p>

        {(sublabel || footer) && (
          <div className="mt-auto pt-2 text-xs leading-snug text-muted-foreground">
            {sublabel && <p>{sublabel}</p>}
            {footer && (
              <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5">
                {footer}
              </div>
            )}
          </div>
        )}
      </div>

      {isActive && (
        <span className={cn('absolute inset-x-0 top-0 h-0.5', colorStyle.accent)} />
      )}

      {onClick && (
        <span className={cn(
          'pointer-events-none absolute inset-x-0 bottom-0 h-0.5 origin-left scale-x-0 transition-transform duration-200 group-hover:scale-x-100 group-focus-visible:scale-x-100',
          colorStyle.accent,
        )} />
      )}
    </CardRoot>
  );
});
