import { memo } from 'react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useCountUp } from '@/hooks/useCountUp';

export interface KpiHeroCardProps {
  label: string;
  value: string | number;
  unit?: string;
  formattedValue?: string;
  icon: LucideIcon;
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
  { icon: string; accent: string }
> = {
  blue: { icon: 'bg-info/10 text-info', accent: 'bg-info' },
  emerald: { icon: 'bg-primary/10 text-primary', accent: 'bg-primary' },
  amber: { icon: 'bg-warning/10 text-warning', accent: 'bg-warning' },
  teal: { icon: 'bg-success/10 text-success', accent: 'bg-success' },
  rose: { icon: 'bg-destructive/10 text-destructive', accent: 'bg-destructive' },
};

export const KpiHeroCard = memo(function KpiHeroCard({
  label,
  value,
  unit,
  formattedValue,
  icon: Icon,
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
      ? `${animated.toLocaleString('vi-VN')}${unit ? ` ${unit}` : ''}`
      : `${value}${unit ? ` ${unit}` : ''}`;
  const CardRoot = onClick ? 'button' : 'div';

  return (
    <CardRoot
      className={cn(
        'group relative h-full w-full overflow-hidden rounded-2xl border bg-card text-left',
        'shadow-soft transition-[border-color,box-shadow,transform] duration-200',
        isActive ? 'border-primary/30 bg-primary/[0.03]' : 'border-border/60',
        onClick && 'cursor-pointer hover:border-primary/25 hover:shadow-card active:scale-[0.995] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        className,
      )}
      onClick={onClick}
      {...(onClick ? { type: 'button' as const } : {})}
    >
      <div className={cn('relative flex h-full flex-col p-4', variant === 'stack' && 'sm:p-5')}>
        <div className="flex items-start justify-between gap-3">
          <div className={cn('flex h-9 w-9 shrink-0 items-center justify-center rounded-xl', colorStyle.icon)}>
            <Icon className="h-[18px] w-[18px]" strokeWidth={2} />
          </div>
          {(trend || badge) && (
            <div className="flex min-h-7 flex-wrap items-center justify-end gap-1.5">
              {trend && (
                <span className={cn(
                  'inline-flex items-center gap-1 rounded-full px-2 py-1 text-[11px] font-semibold tabular-nums',
                  trend.positive ? 'bg-success/10 text-success' : 'bg-destructive/10 text-destructive',
                )}>
                  <span aria-hidden="true">{trend.positive ? '↑' : '↓'}</span>
                  {trend.value}
                </span>
              )}
              {badge && (
                <span className={cn(
                  'inline-flex items-center rounded-full px-2 py-1 text-[11px] font-semibold whitespace-nowrap',
                  BADGE_STYLES[badge.variant],
                )}>
                  {badge.label}
                </span>
              )}
            </div>
          )}
        </div>

        <div className="mt-3 min-w-0">
          <p className="text-xs font-semibold leading-tight text-muted-foreground">
            {label}
          </p>
          <p className={cn(
            'mt-1 break-words font-display font-extrabold tabular-nums leading-tight tracking-tight text-foreground',
            variant === 'stack' ? 'text-2xl sm:text-[1.75rem]' : 'text-xl sm:text-2xl',
          )}>
            {displayValue}
          </p>
        </div>

        {(sublabel || footer) && (
          <div className="mt-auto pt-2 text-[11px] leading-snug text-muted-foreground">
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
