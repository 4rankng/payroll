import { useInfiniteQuery } from '@tanstack/react-query';
import { dashboardService } from '@/services/api/dashboard.service';
import type { RecentActivitiesParams } from '@/types/api/dashboard.types';
import { handleApiError } from '@/services/api/client';

export const useActivitiesPagination = (params?: Omit<RecentActivitiesParams, 'page'>) => {
  return useInfiniteQuery({
    queryKey: ['dashboard', 'activities', params],
    queryFn: async ({ pageParam = 1 }) => {
      try {
        const response = await dashboardService.getRecentActivities({
          ...params,
          page: pageParam
        });
        return {
          activities: response.data,
          pagination: response.pagination
        };
      } catch (error) {
        throw new Error(handleApiError(error));
      }
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      const { page, totalPages } = lastPage.pagination;
      return page < totalPages ? page + 1 : undefined;
    },
    refetchInterval: 30 * 1000,
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};
