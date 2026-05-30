import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '@/services/api/dashboard.service';
import type { FinancialChartParams } from '@/types/api/dashboard.types';

export const useFinancialChart = (params: FinancialChartParams, enabled = true) => {
  return useQuery({
    queryKey: ['dashboard', 'financial-chart', params.period],
    queryFn: async () => {
      const response = await dashboardService.getFinancialChart(params);
      return response.data;
    },
    enabled, // Only fetch when enabled
  });
};
