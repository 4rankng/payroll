import { memo } from 'react';
import { cn } from '@/lib/utils';

export interface PremiumStatItem {
  label: string;
  value: string | number;
  unit?: string;
  highlight?: boolean;
  onClick?: () => void;
  icon?: React.ReactNode;
}

export interface PremiumStatStripProps {
  items: PremiumStatItem[];
  isLoading?: boolean;
  className?: string;
  variant?: 'default' | 'elevated' | 'bordered';
  size?: 'compact' | 'default';
}

export const PremiumStatStrip = memo(function PremiumStatStrip({
  items,
  isLoading = false,
  className,
  variant = 'default',
  size = 'default',
}: PremiumStatStripProps) {
  const variantStyles = {
    default: 'bg-card border-border/60',
    elevated: 'bg-gradient-to-br from-white to-muted/20 border-border/40 shadow-soft',
    bordered: 'bg-card border-primary/20',
  };

  const sizeStyles = {
    compact: 'px-3 py-2.5',
    default: 'px-4 py-3',
  };

  const cellSizeStyles = {
    compact: 'gap-0.5',
    default: 'gap-1',
  };

  if (isLoading) {
    return (
      <div className={cn(
        'rounded-lg border overflow-hidden',
        variantStyles[variant],
        'animate-pulse'
      )}>
        <div className="flex items-stretch gap-px">
          {items.map((_, i) => (
            <div key={i} className="flex-1 min-w-0">
              <div className={cn('h-12 bg-muted/30', sizeStyles[size])} />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className={cn(
      'rounded-lg border overflow-hidden transition-all duration-200',
      variantStyles[variant],
      className
    )}>
      <div className="flex items-stretch gap-px">
        {items.map((item, i) => {
          const formatted =
            typeof item.value === 'number'
              ? item.value.toLocaleString('vi-VN')
              : item.value;

          return (
            <div
              key={i}
              className={cn(
                'flex flex-col items-center justify-center flex-1 min-w-0 transition-colors',
                item.onClick && 'cursor-pointer hover:bg-muted/40',
                sizeStyles[size]
              )}
              onClick={item.onClick}
              role={item.onClick ? 'button' : undefined}
              tabIndex={item.onClick ? 0 : undefined}
              onKeyDown={item.onClick ? (e) => {
                if (e.key === 'Enter' || e.key === ' ') item.onClick!();
              } : undefined}
            >
              {/* Icon */}
              {item.icon && (
                <div className="mb-1">{item.icon}</div>
              )}

              {/* Value */}
              <span className={cn(
                'font-display font-bold tabular-nums tracking-tight leading-none',
                size === 'compact' ? 'text-base' : 'text-lg',
                item.highlight ? 'text-financial-pending' : 'text-foreground'
              )}>
                {formatted}{item.unit}
              </span>

              {/* Label */}
              <span className="text-xs text-muted-foreground leading-none text-center truncate w-full px-1 mt-0.5">
                {item.label}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
});
