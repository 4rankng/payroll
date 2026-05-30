import { memo } from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { useCountUp } from '@/hooks/useCountUp';

export interface InlineStatItem {
  label: string;
  value: string | number;
  unit?: string;
  highlight?: boolean;
  valueClassName?: string;
  onClick?: () => void;
}

export interface InlineStatStripProps {
  items: InlineStatItem[];
  isLoading?: boolean;
  className?: string;
  /**
   * Force single-row (no wrap). Default: auto — wraps to 2-col grid on
   * mobile when items.length > 4.
   */
  wrap?: boolean;
  /**
   * Enhanced premium styling with gradients and hover effects
   */
  variant?: 'default' | 'premium' | 'subtle';
  /**
   * Layout direction. 'vertical' stacks items as rows (label left, value right).
   * Default: 'horizontal' (items side by side).
   */
  direction?: 'horizontal' | 'vertical';
}

export const InlineStatStrip = memo(function InlineStatStrip({
  items,
  isLoading = false,
  className,
  wrap,
  variant = 'premium',
  direction = 'horizontal',
}: InlineStatStripProps) {
  const isVertical = direction === 'vertical';

  // Variant styling
  const getContainerStyles = () => {
    const base = 'rounded-lg border overflow-hidden transition-all duration-300';

    if (variant === 'premium') {
      return cn(
        base,
        'bg-card border-border/40 shadow-soft hover:shadow-card'
      );
    }

    if (variant === 'subtle') {
      return cn(
        base,
        'bg-card border-border/30'
      );
    }

    return cn(
      base,
      'bg-border/40 border-border/60'
    );
  };

  // Horizontal layout (original)
  if (!isVertical) {
    const shouldWrap = wrap ?? items.length > 4;
    const isFourCol = !shouldWrap && items.length === 4;

    const containerCls = cn(
      getContainerStyles(),
      isFourCol ? 'grid grid-cols-2 sm:flex gap-px' :
      shouldWrap ? 'grid grid-cols-2 sm:flex gap-px' : 'flex items-stretch gap-px',
      className,
    );

    const cellCls = cn(
      'flex flex-col items-center justify-center flex-1 min-w-0 transition-colors',
      'px-3 py-3 bg-card',
      variant === 'premium' && 'hover:bg-muted/30'
    );

    if (isLoading) {
      return (
        <div className={containerCls}>
          {items.map((_, i) => (
            <div key={i} className={cn(cellCls, 'gap-1.5')}>
              <Skeleton className="h-3.5 w-12 rounded" />
              <Skeleton className="h-3 w-10 rounded" />
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
              className={cn(cellCls, item.onClick && 'cursor-pointer', 'group')}
              onClick={item.onClick}
              role={item.onClick ? 'button' : undefined}
              tabIndex={item.onClick ? 0 : undefined}
              onKeyDown={item.onClick ? (e) => {
                if (e.key === 'Enter' || e.key === ' ') item.onClick!();
              } : undefined}
            >
              <span className={cn(
                'font-display font-semibold tabular-nums tracking-tight leading-none w-full text-center transition-all',
                'text-base',
                item.highlight ? 'text-primary' : 'text-foreground',
                variant === 'premium' && 'group-hover:scale-105',
                item.valueClassName,
              )}>
                {formatted}{item.unit}
              </span>

              <span className="text-xs text-muted-foreground leading-none text-center truncate w-full px-1 mt-0.5">
                {item.label}
              </span>
            </div>
          );
        })}
      </div>
    );
  }

  // Vertical layout — items stacked as rows
  const containerCls = cn(
    getContainerStyles(),
    'divide-y divide-border/30',
    className,
  );

  const rowCls = cn(
    'flex items-center justify-between px-3 py-2 bg-card transition-colors',
    variant === 'premium' && 'hover:bg-muted/30'
  );

  if (isLoading) {
    return (
      <div className={containerCls}>
        {items.map((_, i) => (
          <div key={i} className={rowCls}>
            <Skeleton className="h-3 w-16 rounded" />
            <Skeleton className="h-3.5 w-10 rounded" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className={containerCls}>
      {items.map((item, i) => (
        <AnimatedStatRow key={i} item={item} rowCls={rowCls} />
      ))}
    </div>
  );
});

/** Vertical stat row with optional count-up animation for numeric values */
const AnimatedStatRow = memo(function AnimatedStatRow({
  item,
  rowCls,
}: {
  item: InlineStatItem;
  rowCls: string;
}) {
  const isNumeric = typeof item.value === 'number';
  const animatedValue = useCountUp(isNumeric ? (item.value as number) : 0, 600);

  const formatted = isNumeric
    ? animatedValue.toLocaleString('vi-VN')
    : (item.value as string);

  return (
    <div
      className={cn(rowCls, item.onClick && 'cursor-pointer', 'group')}
      onClick={item.onClick}
      role={item.onClick ? 'button' : undefined}
      tabIndex={item.onClick ? 0 : undefined}
      onKeyDown={item.onClick ? (e) => {
        if (e.key === 'Enter' || e.key === ' ') item.onClick!();
      } : undefined}
    >
      <span className="text-xs text-muted-foreground truncate">
        {item.label}
      </span>
      <span className={cn(
        'font-display font-semibold tabular-nums tracking-tight text-base whitespace-nowrap ml-3',
        item.highlight ? 'text-primary' : 'text-foreground',
        item.valueClassName,
      )}>
        {formatted}{item.unit}
      </span>
    </div>
  );
});
