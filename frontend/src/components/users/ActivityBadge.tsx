import { Badge } from '@/components/ui/badge';
import { Activity } from 'lucide-react';

interface ActivityBadgeProps {
  count: number;
  onClick?: () => void;
  isActive?: boolean;
}

export const ActivityBadge = ({ count, onClick, isActive = false }: ActivityBadgeProps) => {
  return (
    <Badge
      variant={isActive ? 'default' : 'info'}
      className={`flex items-center gap-1.5 text-sm min-h-[44px] px-4 transition-all ${
        onClick ? 'cursor-pointer' : ''
      } ${isActive ? 'ring-2 ring-primary/20' : ''}`}
      onClick={onClick}
    >
      <Activity className="h-4 w-4" />
      <span>Hoạt động hôm nay:</span>
      <span className="font-semibold">{count.toLocaleString()}</span>
    </Badge>
  );
};
