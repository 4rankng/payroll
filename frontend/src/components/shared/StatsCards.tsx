import { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';

interface StatCardData {
  title: string;
  value: string | number;
  icon: LucideIcon;
  description?: string;
  change?: string;
  trend?: 'up' | 'down' | 'neutral';
  color?: 'blue' | 'green' | 'red' | 'yellow' | 'purple' | 'gray';
}

interface StatsCardsProps {
  stats: StatCardData[];
  columns?: 1 | 2 | 3 | 4;
}

const COLOR_MAP: Record<string, 'blue' | 'emerald' | 'amber' | 'teal' | 'rose'> = {
  blue: 'blue',
  green: 'emerald',
  red: 'rose',
  yellow: 'amber',
  purple: 'teal',
  gray: 'blue',
};

const gridColsMap: Record<number, string> = {
  1: 'grid-cols-1',
  2: 'grid-cols-1 min-[380px]:grid-cols-2',
  3: 'grid-cols-1 min-[380px]:grid-cols-2 sm:grid-cols-3',
  4: 'grid-cols-1 min-[380px]:grid-cols-2 sm:grid-cols-4',
};

export const StatsCards = ({ stats, columns = 4 }: StatsCardsProps) => (
  <div
    data-slot="stats-cards"
    className={cn('grid gap-2.5', gridColsMap[columns] ?? gridColsMap[4])}
  >
    {stats.map((stat, i) => (
      <KpiHeroCard
        key={i}
        label={stat.title}
        value={stat.value}
        icon={stat.icon}
        color={COLOR_MAP[stat.color ?? 'blue']}
        sublabel={stat.description}
        trend={
          stat.change
            ? { value: stat.change, positive: stat.trend === 'up' }
            : undefined
        }
        className="h-full admin-ledger-stat"
      />
    ))}
  </div>
);
