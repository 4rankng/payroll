import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '@/services/api/dashboard.service';
import type { FinancialOverviewParams } from '@/types/api/dashboard.types';
import { handleApiError } from '@/services/api/client';

export const useDashboardFinancialOverview = (params?: FinancialOverviewParams) => {
  return useQuery({
    queryKey: ['dashboard', 'financial-overview', params],
    queryFn: async () => {
      try {
        const response = await dashboardService.getFinancialOverview(params);
        return response.data;
      } catch (error) {
        throw new Error(handleApiError(error));
      }
    },
    refetchInterval: 30 * 1000,
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};