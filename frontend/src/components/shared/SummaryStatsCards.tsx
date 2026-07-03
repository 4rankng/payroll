import { Skeleton } from "@/components/ui/skeleton";
import { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface StatCardConfig {
  title: string;
  value: string | number;
  icon?: LucideIcon | string;
  description?: string;
  change?: string;
  changePercentage?: number;
  onClick?: () => void;
  isActive?: boolean;
}

export interface StatGroupConfig {
  title: string;
  stats: StatCardConfig[];
}

interface SummaryStatsCardsProps {
  stats?: StatCardConfig[];
  groups?: StatGroupConfig[];
  isLoading?: boolean;
  error?: boolean | null;
  columns?: 1 | 2 | 3 | 4 | 5 | 6;
  errorMessage?: string;
  className?: string;
}

export const SummaryStatsCards = ({
  stats,
  groups,
  isLoading = false,
  error = false,
  columns = 4,
  errorMessage = 'Không thể tải dữ liệu',
  className = '',
}: SummaryStatsCardsProps) => {
  const allStats = groups ? groups.flatMap(g => g.stats) : stats || [];
  const count = allStats.length || columns;

  if (isLoading) {
    return (
      <div className={cn('rounded-xl border border-border/40 bg-card shadow-soft overflow-hidden', className)}>
        <div className="grid grid-cols-1 gap-px min-[380px]:grid-cols-2 sm:flex">
          {Array.from({ length: count }).map((_, i) => (
            <div key={i} className="flex flex-col items-center justify-center gap-1 px-3 py-3 flex-1 min-w-0">
              <Skeleton className="h-4 w-14" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error || allStats.length === 0) {
    return (
      <div className="rounded-xl border border-border/40 bg-card px-4 py-3 text-center shadow-soft">
        <p className="text-xs text-muted-foreground">{errorMessage}</p>
      </div>
    );
  }

  const renderStrip = (items: StatCardConfig[]) => {
    return (
      <div className={cn(
        'rounded-xl border border-border/40 bg-card overflow-hidden shadow-soft',
        'grid grid-cols-1 gap-px min-[380px]:grid-cols-2 sm:flex',
      )}>
        {items.map((stat, i) => {
          const formatted = typeof stat.value === 'number'
            ? stat.value.toLocaleString('vi-VN')
            : stat.value;
          return (
            <div
              key={i}
              role={stat.onClick ? 'button' : undefined}
              tabIndex={stat.onClick ? 0 : undefined}
              onKeyDown={stat.onClick ? (e) => { if (e.key === 'Enter' || e.key === ' ') stat.onClick!(); } : undefined}
              onClick={stat.onClick}
              className={cn(
                'relative flex flex-col items-center justify-center gap-1 px-3 py-3 flex-1 min-w-0 transition-colors',
                stat.isActive ? 'bg-primary/5' : 'bg-card',
                stat.onClick && 'cursor-pointer hover:bg-muted/50 active:bg-muted/80',
              )}
            >
              <span className={cn(
                'break-words text-center font-display font-extrabold tabular-nums leading-tight tracking-tight',
                stat.isActive ? 'text-primary' : 'text-foreground',
              )}>
                {formatted}
              </span>
              <span className="mt-0.5 w-full break-words px-1 text-center text-xs leading-tight text-muted-foreground">
                {stat.title}
              </span>
              {stat.isActive && (
                <div className="absolute top-1.5 right-1.5 h-1.5 w-1.5 rounded-full bg-primary" />
              )}
            </div>
          );
        })}
      </div>
    );
  };

  if (groups && groups.length > 0) {
    return (
      <div className={cn('space-y-3', className)}>
        {groups.map((group, gi) => (
          <div key={gi} className="space-y-1.5">
            <span className="text-xs font-bold uppercase tracking-wider text-muted-foreground px-0.5">
              {group.title}
            </span>
            {renderStrip(group.stats)}
          </div>
        ))}
      </div>
    );
  }

  return <div className={className}>{renderStrip(allStats)}</div>;
};
