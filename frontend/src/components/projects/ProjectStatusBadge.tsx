import { Badge } from '@/components/ui/badge';
import { CheckCircle, Clock, XCircle, AlertCircle, Edit, Pause } from 'lucide-react';
import { cn } from '@/lib/utils';
import { getVietnameseProjectStatus } from '@/utils/vietnamese';

export type ProjectStatus = 'draft' | 'active' | 'paused' | 'completed' | 'cancelled';

interface ProjectStatusBadgeProps {
  status: ProjectStatus | string;
  className?: string;
}

const STATUS_CONFIG = {
  draft: {
    label: () => getVietnameseProjectStatus('draft'),
    icon: Edit,
    className: 'bg-muted/50 text-foreground border-border hover:bg-muted'
  },
  active: {
    label: () => getVietnameseProjectStatus('active'),
    icon: CheckCircle,
    className: 'bg-utility-success-50 text-fg-success-primary border-utility-success-200 hover:bg-utility-success-100'
  },
  paused: {
    label: () => getVietnameseProjectStatus('paused'),
    icon: Pause,
    className: 'bg-utility-warning-50 text-fg-warning-primary border-utility-warning-200 hover:bg-utility-warning-100'
  },
  completed: {
    label: () => getVietnameseProjectStatus('completed'),
    icon: CheckCircle,
    className: 'bg-utility-blue-50 text-utility-blue-700 border-utility-blue-200 hover:bg-utility-blue-100'
  },
  cancelled: {
    label: () => getVietnameseProjectStatus('cancelled'),
    icon: XCircle,
    className: 'bg-utility-error-50 text-fg-error-primary border-utility-error-200 hover:bg-utility-error-100'
  }
};

export function ProjectStatusBadge({ status, className }: ProjectStatusBadgeProps) {
  // Normalize status to lowercase to handle both uppercase and lowercase values from API
  const normalizedStatus = status?.toLowerCase() as ProjectStatus;

  // Handle undefined or invalid status values
  if (!status || !STATUS_CONFIG[normalizedStatus]) {
    return (
      <Badge
        variant="outline"
        className={cn(
          'flex items-center gap-1 typography-body-small border',
          'bg-muted/50 text-foreground border-border hover:bg-muted',
          className
        )}
      >
        <AlertCircle className="h-3 w-3" />
        {status || 'Không xác định'}
      </Badge>
    );
  }

  const config = STATUS_CONFIG[normalizedStatus];
  const Icon = config.icon;

  return (
    <Badge 
      variant="outline"
      className={cn(
        'flex items-center gap-1 typography-body-small border',
        config.className,
        className
      )}
    >
      <Icon className="h-3 w-3" />
      {config.label()}
    </Badge>
  );
}