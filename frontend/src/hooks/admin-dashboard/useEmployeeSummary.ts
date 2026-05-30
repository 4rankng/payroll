import { useQuery } from '@tanstack/react-query';
import { employeeService } from '@/services/api/employee.service';
import { handleApiError } from '@/services/api/client';

export const useEmployeeSummary = () => {
  return useQuery({
    queryKey: ['dashboard', 'employee-summary'],
    queryFn: async () => {
      try {
        const response = await employeeService.getSummary();
        return response;
      } catch (error) {
        throw new Error(handleApiError(error));
      }
    },
    refetchInterval: 30 * 1000,
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};