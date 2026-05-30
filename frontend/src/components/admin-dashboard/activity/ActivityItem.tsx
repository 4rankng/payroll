import React from 'react';
import type { RecentActivity } from '@/types/api/dashboard.types';
import { formatVietnameseRelativeTime } from '@/utils/vietnamese';
import { Activity } from 'lucide-react';

interface ActivityItemProps {
  activity: RecentActivity;
  isLast?: boolean;
}

export const ActivityItem: React.FC<ActivityItemProps> = ({ activity, isLast = false }) => {
  return (
    <div role="listitem" className="flex gap-3 group">
      {/* Timeline dot + line */}
      <div className="flex flex-col items-center flex-shrink-0">
        <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center ring-2 ring-background group-hover:bg-primary/20 transition-colors">
          <Activity className="h-3.5 w-3.5 text-primary" />
        </div>
        {!isLast && <div className="w-px flex-1 bg-border/50 mt-1 min-h-[12px]" />}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0 pb-3">
        <p className="text-sm font-medium text-foreground leading-snug break-words">
          {activity.message}
        </p>
        <span className="text-xs text-muted-foreground mt-0.5 block">
          {formatVietnameseRelativeTime(activity.created_at)}
        </span>
      </div>
    </div>
  );
};
