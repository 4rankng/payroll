import React, { useMemo, useCallback } from 'react';
import type { RecentActivity } from '@/types/api/dashboard.types';
import { ActivityCard } from './activity/ActivityCard';

interface RecentActivitiesCardProps {
  activities?: RecentActivity[];
  isLoading?: boolean;
  isLoadingMore?: boolean;
  error?: string | null;
  onRefresh?: () => void;
  onLoadMore?: () => void;
  hasMoreActivities?: boolean;
  totalActivities?: number;
}

export const RecentActivitiesCard: React.FC<RecentActivitiesCardProps> = ({
  activities,
  isLoading = false,
  isLoadingMore = false,
  error = null,
  onRefresh,
  onLoadMore,
  hasMoreActivities = false,
  totalActivities = 0,
}) => {
  const safeActivities = useMemo<RecentActivity[]>(() => {
    if (!activities || !Array.isArray(activities)) {
      return [];
    }
    return activities;
  }, [activities]);

  const handleRefresh = useCallback(() => {
    onRefresh?.();
  }, [onRefresh]);

  const handleLoadMore = useCallback(() => {
    onLoadMore?.();
  }, [onLoadMore]);

  return (
    <ActivityCard
      activities={safeActivities}
      isLoading={isLoading}
      isLoadingMore={isLoadingMore}
      error={error}
      onRefresh={handleRefresh}
      onLoadMore={handleLoadMore}
      hasMoreActivities={hasMoreActivities}
      totalActivities={totalActivities}
      maxItems={undefined}
    />
  );
};
