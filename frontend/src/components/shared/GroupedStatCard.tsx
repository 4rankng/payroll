import { memo } from 'react';
import { cn } from '@/lib/utils';
import { LucideIcon } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { useCountUp } from '@/hooks/useCountUp';

export interface StatItem {
  label: string | React.ReactNode;
  value: string | number;
  unit?: string;
  variant?: 'default' | 'muted' | 'accent';
  onClick?: () => void;
}

export interface GroupedStatCardProps {
  title: string;
  icon?: LucideIcon;
  stats: StatItem[];
  onClick?: () => void;
  isLoading?: boolean;
  className?: string;
}

export const GroupedStatCard = ({ title, icon: Icon, stats, onClick, isLoading, className }: GroupedStatCardProps) => {
  const stripCls = cn(
    'overflow-hidden rounded-2xl border border-border/40 bg-card shadow-none',
    'grid grid-cols-1 gap-px min-[380px]:grid-cols-2 sm:flex',
  );

  if (isLoading) {
    return (
      <div className={cn('space-y-1.5', className)}>
        <div className="flex items-center gap-1.5 px-0.5">
          <Skeleton className="h-3 w-3 rounded" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className={stripCls}>
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex flex-col items-center gap-1.5 px-3 py-3 flex-1 min-w-0">
              <Skeleton className="h-4 w-12" />
              <Skeleton className="h-3 w-14" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!stats.length) return null;

  return (
    <div
      className={cn('space-y-1.5', onClick && 'cursor-pointer', className)}
      onClick={onClick}
    >
      <div className="flex items-center gap-1.5 px-0.5">
        {Icon && <Icon className="h-3 w-3 text-primary shrink-0" />}
        <span className="font-display text-[13px] font-bold tracking-tight text-muted-foreground">
          {title}
        </span>
      </div>

      <div className={stripCls}>
        {stats.map((stat, i) => (
          <AnimatedStatCell key={i} stat={stat} />
        ))}
      </div>
    </div>
  );
};

const AnimatedStatCell = memo(function AnimatedStatCell({ stat }: { stat: StatItem }) {
  const isNumeric = typeof stat.value === 'number';
  const animatedValue = useCountUp(isNumeric ? (stat.value as number) : 0, 600);

  const formatted = isNumeric
    ? animatedValue.toLocaleString('vi-VN')
    : (stat.value as string);

  const isAccent = stat.variant === 'accent';

  return (
    <div
      onClick={(e) => { if (stat.onClick) { e.stopPropagation(); stat.onClick(); } }}
      className={cn(
        'flex flex-col items-center justify-center gap-1 px-3 py-3 flex-1 min-w-0 transition-colors',
        isAccent ? 'bg-primary/5' : 'bg-card',
        stat.onClick && 'cursor-pointer hover:bg-muted/50 active:bg-muted/80',
      )}
    >
      <span className={cn(
        'w-full break-words text-center font-display text-sm font-extrabold tabular-nums leading-tight tracking-tight sm:text-base',
        isAccent ? 'text-primary' : 'text-foreground',
      )}>
        {formatted}{stat.unit}
      </span>
      <span className="mt-0.5 w-full break-words px-1 text-center text-[11px] font-medium leading-tight text-muted-foreground">
        {stat.label}
      </span>
    </div>
  );
});
