import { memo, useMemo } from 'react';
import { Wallet, DollarSign } from 'lucide-react';
import type { StatItem } from '@/components/shared/GroupedStatCard';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/skeleton';
import { useCountUp } from '@/hooks/useCountUp';

export interface AdvancePaymentStatsStripProps {
  overviewStats: StatItem[];
  financeStats: StatItem[];
  isLoading?: boolean;
  className?: string;
  thirdSlot?: React.ReactNode;
}

interface LightStatItem {
  label: string;
  value: string | number;
  highlight?: boolean;
}

const LightStatRow = memo(function LightStatRow({ item }: { item: LightStatItem }) {
  const isNumeric = typeof item.value === 'number';
  const animated = useCountUp(isNumeric ? (item.value as number) : 0, 600);
  const formatted = isNumeric ? animated.toLocaleString('vi-VN') : (item.value as string);

  return (
    <div className="flex flex-col gap-1.5 p-4 bg-card/90 hover:bg-muted/80 transition-all duration-200">
      <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{item.label}</span>
      <span className={cn(
        'font-display font-bold tabular-nums text-xl sm:text-2xl',
        item.highlight ? 'text-primary' : 'text-foreground',
      )}>
        {formatted}
      </span>
    </div>
  );
});

interface LightStatCardProps {
  icon: React.ReactNode;
  title: string;
  items: LightStatItem[];
  isLoading?: boolean;
  skeletonCount?: number;
}

const LightStatCard = memo(function LightStatCard({
  icon,
  title,
  items,
  isLoading,
  skeletonCount = 3,
}: LightStatCardProps) {
  return (
    <div className="w-full space-y-3">
      <div className="flex items-center gap-2 px-1">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/10">
          {icon}
        </div>
        <span className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
          {title}
        </span>
      </div>

      <div className="rounded-2xl border border-border/40 shadow-sm hover:shadow-md transition-shadow overflow-hidden bg-card/90 backdrop-blur-sm">
        <div className="grid grid-cols-2 gap-[1px] bg-border/40">
          {isLoading
            ? Array.from({ length: skeletonCount }).map((_, i) => (
                <div key={i} className="flex flex-col gap-2 p-4 bg-card">
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className="h-6 w-24" />
                </div>
              ))
            : items.map((item, i) => <LightStatRow key={i} item={item} />)
          }
        </div>
      </div>
    </div>
  );
});

export const AdvancePaymentStatsStrip = memo(function AdvancePaymentStatsStrip({
  overviewStats,
  financeStats,
  isLoading = false,
  className,
  thirdSlot,
}: AdvancePaymentStatsStripProps) {
  const overviewItems = useMemo<LightStatItem[]>(
    () => overviewStats.map((s) => ({
      label: typeof s.label === 'string' ? s.label : '',
      value: s.value,
      highlight: s.variant === 'accent',
    })),
    [overviewStats],
  );

  const financeItems = useMemo<LightStatItem[]>(
    () => financeStats.map((s) => ({
      label: typeof s.label === 'string' ? s.label : '',
      value: s.value,
      highlight: s.variant === 'accent',
    })),
    [financeStats],
  );

  if (!isLoading && overviewItems.length === 0 && financeItems.length === 0) return null;

  const hasThird = Boolean(thirdSlot);

  return (
    <div className={cn(
      'grid grid-cols-1 md:grid-cols-2 gap-6',
      hasThird && 'lg:grid-cols-3',
      className,
    )}>
      <LightStatCard
        icon={<Wallet className="h-3.5 w-3.5 text-primary" />}
        title="Tổng quan"
        items={overviewItems}
        isLoading={isLoading}
        skeletonCount={4}
      />
      {(isLoading || financeItems.length > 0) && (
        <LightStatCard
          icon={<DollarSign className="h-3.5 w-3.5 text-primary" />}
          title="Tài chính"
          items={financeItems}
          isLoading={isLoading}
          skeletonCount={3}
        />
      )}
      {hasThird && thirdSlot}
    </div>
  );
});
