import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

interface NotificationBadgeProps {
  count: number;
  className?: string;
}

export const NotificationBadge = ({ count, className }: NotificationBadgeProps) => {
  if (count <= 0) return null;

  const displayCount = count > 99 ? '99+' : count.toString();

  return (
    <Badge
      variant="destructive"
      className={cn(
        "absolute top-1 right-1 h-4 min-w-4 flex items-center justify-center px-1 text-xs font-semibold rounded-full",
        "animate-badge-pulse",
        className
      )}
    >
      {displayCount}
    </Badge>
  );
};