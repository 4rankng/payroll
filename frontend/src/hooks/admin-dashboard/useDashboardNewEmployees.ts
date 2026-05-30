import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '@/services/api/dashboard.service';
import type { NewEmployeesParams } from '@/types/api/dashboard.types';
import { handleApiError } from '@/services/api/client';

const PAGE_SIZE = 20;

async function fetchAllNewEmployees(params?: Omit<NewEmployeesParams, 'page'>) {
  // Fetch first page to get total
  const first = await dashboardService.getNewEmployees({ ...params, page: 1, pageSize: PAGE_SIZE });
  const totalPages = first.pagination.totalPages;
  const allEmployees = [...first.data];

  // Fetch remaining pages in parallel
  if (totalPages > 1) {
    const rest = await Promise.all(
      Array.from({ length: totalPages - 1 }, (_, i) =>
        dashboardService.getNewEmployees({ ...params, page: i + 2, pageSize: PAGE_SIZE })
      )
    );
    rest.forEach(r => allEmployees.push(...r.data));
  }

  return {
    employees: allEmployees,
    pagination: { ...first.pagination, page: 1, totalPages: 1, pageSize: allEmployees.length }
  };
}

export const useDashboardNewEmployees = (params?: Omit<NewEmployeesParams, 'page'>) => {
  return useQuery({
    queryKey: ['dashboard', 'new-employees', params],
    queryFn: async () => {
      try {
        return await fetchAllNewEmployees(params);
      } catch (error) {
        throw new Error(handleApiError(error));
      }
    },
    refetchInterval: 30 * 1000,
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};
