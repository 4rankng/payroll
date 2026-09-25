import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';
import { LucideIcon } from 'lucide-react';

interface DashboardSectionHeaderProps {
  title: string;
  eyebrow?: string;
  subtitle?: string;
  icon?: LucideIcon;
  className?: string;
  actions?: ReactNode;
}

export const DashboardSectionHeader = ({
  title,
  eyebrow,
  subtitle,
  icon: Icon,
  className,
  actions,
}: DashboardSectionHeaderProps) => (
  <div className={cn('admin-dashboard-section-header flex items-start justify-between gap-3', className)}>
    <div className="flex min-w-0 items-start gap-2.5">
      {Icon && (
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-primary/10 bg-primary/5">
          <Icon className="h-4 w-4 text-primary" />
        </div>
      )}
      <div className="min-w-0 space-y-0.5">
        {eyebrow && (
          <p className="text-xs font-semibold uppercase tracking-[0.22em] text-muted-foreground">
            {eyebrow}
          </p>
        )}
        <h2 className="text-sm font-semibold tracking-wide text-foreground">{title}</h2>
        {subtitle && <p className="text-xs leading-relaxed text-muted-foreground">{subtitle}</p>}
      </div>
    </div>
    {actions ? <div className="shrink-0">{actions}</div> : null}
  </div>
);
