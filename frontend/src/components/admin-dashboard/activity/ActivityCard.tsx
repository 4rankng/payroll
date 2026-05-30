import React, { useMemo } from 'react';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Loader2 } from 'lucide-react';
import type { RecentActivity } from '@/types/api/dashboard.types';
import { ActivityHeader } from './ActivityHeader';
import { ActivityItem } from './ActivityItem';
import { ActivityEmptyState } from './ActivityEmptyState';

interface ActivityCardProps {
  activities?: RecentActivity[];
  isLoading?: boolean;
  isLoadingMore?: boolean;
  error?: string | null;
  onRefresh?: () => void;
  onLoadMore?: () => void;
  hasMoreActivities?: boolean;
  totalActivities?: number;
  maxItems?: number;
}

export const ActivityCard: React.FC<ActivityCardProps> = ({
  activities,
  isLoading = false,
  isLoadingMore = false,
  error = null,
  onRefresh,
  onLoadMore,
  hasMoreActivities = false,
  totalActivities = 0,
  maxItems
}) => {
  const activitiesArray = useMemo(() =>
    Array.isArray(activities) ? activities : [],
    [activities]
  );

  const displayedActivities = useMemo(() =>
    maxItems ? activitiesArray.slice(0, maxItems) : activitiesArray,
    [activitiesArray, maxItems]
  );

  const shouldShowLoadMore = useMemo(() => {
    return hasMoreActivities && onLoadMore && activitiesArray.length > 0;
  }, [hasMoreActivities, onLoadMore, activitiesArray.length]);

  const renderContent = () => {
    if (isLoading) {
      return <ActivityEmptyState type="loading" />;
    }

    if (error) {
      return (
        <ActivityEmptyState
          type="error"
          error={error}
          onRefresh={onRefresh}
        />
      );
    }

    if (activitiesArray.length === 0) {
      return (
        <ActivityEmptyState
          type="empty"
          onRefresh={onRefresh}
        />
      );
    }

    return (
      <div
        className="mt-2 space-y-0"
        role="list"
        aria-label="Hoạt động gần đây"
      >
        {displayedActivities.map((activity, index) => (
          <ActivityItem
            key={activity.id}
            activity={activity}
            isLast={index === displayedActivities.length - 1 && !shouldShowLoadMore}
          />
        ))}

        {shouldShowLoadMore && (
          <div className="pt-4 mt-4 border-t border-border/50">
            <div className="flex items-center justify-between gap-3">
              <p className="typography-body-small text-muted-foreground">
                Hiển thị {activitiesArray.length} / {totalActivities} hoạt động
              </p>
              <Button
                onClick={onLoadMore}
                disabled={isLoadingMore}
                className="btn-admin-primary hover:scale-105 transition-transform"
                size="sm"
              >
                {isLoadingMore ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Đang tải...
                  </>
                ) : (
                  'Tải thêm'
                )}
              </Button>
            </div>
          </div>
        )}
      </div>
    );
  };

  return (
    <Card className="h-full">
      <CardHeader className="pb-4">
        <ActivityHeader
          activitiesCount={totalActivities || activitiesArray.length}
          isLoading={isLoading}
          onRefresh={onRefresh}
        />
      </CardHeader>
      <CardContent className="pt-0">
        {renderContent()}
      </CardContent>
    </Card>
  );
};
