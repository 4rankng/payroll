import { Badge } from '@/components/ui/badge';
import { Calendar, CalendarDays } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ProjectPaymentTypeBadgeProps {
  isMonthly: boolean;
  className?: string;
}

export function ProjectPaymentTypeBadge({ isMonthly, className }: ProjectPaymentTypeBadgeProps) {
  const config = isMonthly
    ? {
        label: 'Lương tháng',
        icon: Calendar,
        className: 'bg-purple-100 text-purple-700 border-purple-200 hover:bg-purple-100'
      }
    : {
        label: 'Lương tuần',
        icon: CalendarDays,
        className: 'bg-blue-100 text-blue-700 border-blue-200 hover:bg-blue-100'
      };

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
      {config.label}
    </Badge>
  );
}
