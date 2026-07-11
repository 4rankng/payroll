import { memo } from 'react';
import { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useCountUp } from '@/hooks/useCountUp';

export interface KpiHeroCardProps {
  label: string;
  value: string | number;
  unit?: string;
  formattedValue?: string;
  icon: LucideIcon;
  color: 'blue' | 'emerald' | 'amber' | 'violet';
  sublabel?: string;
  trend?: { value: string; positive: boolean };
  badge?: { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral' };
  isActive?: boolean;
  onClick?: () => void;
  className?: string;
  variant?: 'compact' | 'stack';
}

const BADGE_STYLES: Record<string, string> = {
  success: 'border-emerald-300 text-emerald-700 bg-emerald-50',
  warning: 'border-amber-300 text-amber-700 bg-amber-50',
  danger: 'border-rose-300 text-rose-600 bg-rose-50',
  neutral: 'border-slate-300 text-slate-600 bg-slate-50',
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
  isActive,
  onClick,
  className,
  variant = 'compact',
}: KpiHeroCardProps) {
  const numericValue = typeof value === 'number' ? value : 0;
  const animated = useCountUp(numericValue, 800);
  // iconText: the small inline icon next to the label.
  // watermark: large faint icon decoration on the right side of the card
  // (mirrors the user-supplied reference design — Stripe/Linear "watermark" feel).
  const colorMap: Record<string, { iconText: string; watermark: string }> = {
    blue:    { iconText: 'text-blue-600',    watermark: 'text-blue-500/15' },
    emerald: { iconText: 'text-emerald-600', watermark: 'text-emerald-500/15' },
    amber:   { iconText: 'text-amber-600',   watermark: 'text-amber-500/15' },
    violet:  { iconText: 'text-violet-600',  watermark: 'text-violet-500/15' },
  };
  const c = colorMap[color] ?? colorMap.blue;

  const displayValue = formattedValue
    ? formattedValue
    : typeof value === 'number'
      ? `${animated.toLocaleString('vi-VN')}${unit ? ` ${unit}` : ''}`
      : `${value}${unit ? ` ${unit}` : ''}`;

  const stackValueBlock = (
    <div className="flex flex-col gap-0.5 items-start">
      <p
        className="break-words font-display text-xl font-extrabold tabular-nums leading-tight tracking-normal text-foreground sm:text-2xl"
      >
        {displayValue}
      </p>
      {sublabel && (
        <p className="text-[11px] text-muted-foreground">{sublabel}</p>
      )}
      {(trend || badge) && (
        <div className="flex items-center gap-2 flex-wrap justify-start">
          {trend && (
            <div className={cn(
              'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold',
              trend.positive ? 'bg-success/10 text-success' : 'bg-destructive/10 text-destructive',
            )}>
              <span>{trend.positive ? '↑' : '↓'}</span>
              <span>{trend.value}</span>
            </div>
          )}
          {badge && (
            <span className={cn(
              'inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-semibold whitespace-nowrap',
              BADGE_STYLES[badge.variant],
            )}>
              {badge.label}
            </span>
          )}
        </div>
      )}
    </div>
  );

  return (
    <div
      className={cn(
        'group relative h-full overflow-hidden rounded-xl border bg-card',
        'shadow-soft transition-all duration-300',
        isActive ? 'border-primary/30 bg-primary/5' : 'border-border/40',
        onClick && 'cursor-pointer hover:-translate-y-0.5 hover:shadow-card active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        className,
      )}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => { if (e.key === 'Enter' || e.key === ' ') onClick(); } : undefined}
    >
      {/* Watermark decoration — large faint icon on the right side, fully
          contained inside the card via overflow-hidden + right-3. Subtle
          scale on hover for liveliness. Behind content via pointer-events-none. */}
      <Icon
        className={cn(
          'absolute right-3 top-1/2 -translate-y-1/2 h-14 w-14 sm:h-16 sm:w-16 pointer-events-none',
          'transition-transform duration-300 group-hover:scale-105',
          c.watermark,
        )}
        strokeWidth={1.5}
      />

      {variant === 'compact' ? (
        /* Mobile (<sm): stacked layout for 2-col grid. sm+: horizontal compact layout */
        <div className="relative px-3 py-3 sm:px-4">
          {/* Mobile stacked */}
          <div className="flex flex-col gap-1 sm:hidden">
            <div className="flex items-center gap-1.5">
              <Icon className={cn('h-3 w-3 shrink-0', c.iconText)} strokeWidth={2.2} />
              <span className="text-[11px] font-bold uppercase tracking-normal text-muted-foreground leading-tight line-clamp-2">
                {label}
              </span>
            </div>
            <p className="break-words font-display text-[clamp(1.125rem,6vw,1.375rem)] font-extrabold tabular-nums leading-tight tracking-normal text-foreground">
              {displayValue}
            </p>
            {sublabel && (
              <p className="text-[11px] font-medium text-muted-foreground leading-snug">{sublabel}</p>
            )}
            {(trend || badge) && (
              <div className="flex items-center gap-2 flex-wrap">
                {trend && (
                  <div className={cn(
                    'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold',
                    trend.positive ? 'bg-success/10 text-success' : 'bg-destructive/10 text-destructive',
                  )}>
                    <span>{trend.positive ? '↑' : '↓'}</span>
                    <span>{trend.value}</span>
                  </div>
                )}
                {badge && (
                  <span className={cn(
                    'inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold whitespace-nowrap',
                    BADGE_STYLES[badge.variant],
                  )}>
                    {badge.label}
                  </span>
                )}
              </div>
            )}
          </div>
          {/* sm+ horizontal layout */}
          <div className="hidden sm:block pr-14">
            <div className="flex items-center gap-1.5">
              <Icon className={cn('h-3.5 w-3.5 shrink-0', c.iconText)} strokeWidth={2.2} />
              <span className="text-xs font-bold uppercase tracking-normal text-muted-foreground leading-tight">
                {label}
              </span>
            </div>
            <p
              className="mt-1 break-words font-display text-xl font-extrabold tabular-nums leading-tight tracking-normal text-foreground sm:text-2xl"
            >
              {displayValue}
            </p>
            {(sublabel || trend || badge) && (
              <div className="flex items-center gap-2 flex-wrap mt-1">
                {sublabel && (
                  <p className="text-[11px] text-muted-foreground leading-snug">{sublabel}</p>
                )}
                {trend && (
                  <div className={cn(
                    'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold',
                    trend.positive ? 'bg-success/10 text-success' : 'bg-destructive/10 text-destructive',
                  )}>
                    <span>{trend.positive ? '↑' : '↓'}</span>
                    <span>{trend.value}</span>
                  </div>
                )}
                {badge && (
                  <span className={cn(
                    'inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-semibold whitespace-nowrap',
                    BADGE_STYLES[badge.variant],
                  )}>
                    {badge.label}
                  </span>
                )}
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="relative px-4 py-3 space-y-1.5 pr-14">
          <div className="flex items-center gap-1.5">
            <Icon className={cn('h-3.5 w-3.5 shrink-0', c.iconText)} strokeWidth={2.2} />
            <span className="text-xs font-bold uppercase tracking-normal text-muted-foreground leading-tight">
              {label}
            </span>
          </div>
          {stackValueBlock}
        </div>
      )}

      {isActive && (
        <div className="absolute top-2 right-2 h-1.5 w-1.5 rounded-full bg-primary z-10" />
      )}

      {onClick && (
        <div className="pointer-events-none absolute inset-x-0 bottom-0 h-0.5 scale-x-0 bg-primary/70 transition-transform duration-300 group-hover:scale-x-100 group-focus-visible:scale-x-100" />
      )}
    </div>
  );
});
