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
    className: 'bg-green-50 text-green-700 border-green-200 hover:bg-green-100'
  },
  paused: {
    label: () => getVietnameseProjectStatus('paused'),
    icon: Pause,
    className: 'bg-amber-50 text-amber-700 border-amber-200 hover:bg-amber-100'
  },
  completed: {
    label: () => getVietnameseProjectStatus('completed'),
    icon: CheckCircle,
    className: 'bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100'
  },
  cancelled: {
    label: () => getVietnameseProjectStatus('cancelled'),
    icon: XCircle,
    className: 'bg-red-50 text-red-700 border-red-200 hover:bg-red-100'
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