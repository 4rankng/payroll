import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '@/services/api/dashboard.service';
import type { MonthlyFinancialsPeriod } from '@/types/api/dashboard.types';

export const useMonthlyFinancials = (period: MonthlyFinancialsPeriod) => {
  return useQuery({
    queryKey: ['dashboard', 'monthly-financials', period],
    queryFn: () => dashboardService.getMonthlyFinancials(period),
  });
};
