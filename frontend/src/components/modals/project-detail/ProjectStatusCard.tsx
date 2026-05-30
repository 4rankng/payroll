import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { LucideIcon } from 'lucide-react';

interface ProjectStatusCardProps {
  icon: LucideIcon;
  value: string | number;
  label: string;
  color?: string;
}

export function ProjectStatusCard({ 
  icon: Icon, 
  value, 
  label,
}: ProjectStatusCardProps) {
  const formattedValue = typeof value === 'number'
    ? value.toLocaleString('vi-VN')
    : value;

  return (
    <KpiHeroCard
      label={label}
      value={value}
      formattedValue={formattedValue}
      icon={Icon}
      color="blue"
    />
  );
}
