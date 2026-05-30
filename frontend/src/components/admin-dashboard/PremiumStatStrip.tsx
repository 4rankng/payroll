import { memo } from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

export interface PremiumStatItem {
  label: string;
  value: string | number;
  unit?: string;
  highlight?: boolean;
  onClick?: () => void;
  trend?: {
    value: string;
    positive: boolean;
  };
}

export interface PremiumStatStripProps {
  items: PremiumStatItem[];
  isLoading?: boolean;
  className?: string;
  wrap?: boolean;
  icon?: React.ReactNode;
}

export const PremiumStatStrip = memo(function PremiumStatStrip({
  items,
  isLoading = false,
  className,
  wrap,
  icon,
}: PremiumStatStripProps) {
  const shouldWrap = wrap ?? items.length > 4;
  const isFourCol = !shouldWrap && items.length === 4;

  const containerCls = cn(
    'rounded-xl overflow-hidden border',
    'shadow-sm transition-all duration-300',
    isFourCol ? 'grid grid-cols-2 sm:flex gap-px' :
    shouldWrap ? 'grid grid-cols-2 sm:flex gap-px' : 'flex items-stretch gap-px',
    className,
  );

  const cellCls = cn(
    'flex flex-col items-center justify-center gap-2 px-4 py-4 bg-card flex-1 min-w-0',
    'transition-colors duration-200'
  );

  if (isLoading) {
    return (
      <div className={containerCls}>
        {items.map((_, i) => (
          <div key={i} className={cn(cellCls, 'gap-2')}>
            <Skeleton className="h-8 w-16 rounded" />
            <Skeleton className="h-4 w-20 rounded" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className={containerCls}>
      {items.map((item, i) => {
        const formatted =
          typeof item.value === 'number'
            ? item.value.toLocaleString('vi-VN')
            : item.value;

        return (
          <div
            key={i}
            className={cn(
              cellCls,
              item.onClick && 'cursor-pointer hover:bg-muted/40',
              'group'
            )}
            onClick={item.onClick}
            role={item.onClick ? 'button' : undefined}
            tabIndex={item.onClick ? 0 : undefined}
            onKeyDown={item.onClick ? (e) => { 
              if (e.key === 'Enter' || e.key === ' ') item.onClick!(); 
            } : undefined}
          >
            {/* Value - Premium Typography */}
            <span className={cn(
              'font-financial font-bold text-2xl tracking-tight leading-none w-full text-center',
              item.highlight ? 'text-financial-positive' : 'text-foreground',
              'group-hover:scale-105 transition-transform duration-200'
            )}>
              {formatted}{item.unit}
            </span>

            {/* Label */}
            <span className="text-xs font-semibold text-muted-foreground leading-none text-center uppercase tracking-wider w-full px-1 mt-1">
              {item.label}
            </span>

            {/* Trend Indicator */}
            {item.trend && (
              <span className={cn(
                'text-xs font-medium',
                item.trend.positive ? 'text-financial-positive' : 'text-financial-negative',
                'flex items-center gap-1 mt-1'
              )}>
                {item.trend.positive ? '↑' : '↓'} {item.trend.value}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
});
