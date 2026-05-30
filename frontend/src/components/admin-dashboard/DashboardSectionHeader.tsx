import { cn } from '@/lib/utils';
import { LucideIcon } from 'lucide-react';

interface DashboardSectionHeaderProps {
  title: string;
  subtitle?: string;
  icon?: LucideIcon;
  className?: string;
}

export const DashboardSectionHeader = ({ title, subtitle, icon: Icon, className }: DashboardSectionHeaderProps) => (
  <div className={cn('flex items-center gap-2', className)}>
    {Icon && (
      <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/5 border border-primary/10">
        <Icon className="h-3.5 w-3.5 text-primary/70" />
      </div>
    )}
    <div>
      <h2 className="text-sm font-semibold text-foreground tracking-wide">{title}</h2>
      {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
    </div>
  </div>
);
