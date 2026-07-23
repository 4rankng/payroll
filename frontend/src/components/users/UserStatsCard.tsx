import { cn } from '@/lib/utils';
import { LucideIcon, Users } from 'lucide-react';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';

export interface StatItemProps {
  label: string;
  value: number;
  icon?: LucideIcon;
  color?: 'blue' | 'emerald' | 'amber' | 'teal';
  onClick?: () => void;
  isActive?: boolean;
}

interface UserStatsCardProps {
  stats: StatItemProps[];
}

const DEFAULT_COLORS: Array<'blue' | 'emerald' | 'amber' | 'teal'> = ['blue', 'teal', 'emerald', 'amber'];

export const UserStatsCard = ({ stats }: UserStatsCardProps) => {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
      {stats.map((stat, index) => (
        <KpiHeroCard
          key={index}
          label={stat.label}
          value={stat.value}
          icon={stat.icon ?? Users}
          color={stat.color ?? DEFAULT_COLORS[index % DEFAULT_COLORS.length]}
          isActive={stat.isActive}
          onClick={stat.onClick}
        />
      ))}
    </div>
  );
};
