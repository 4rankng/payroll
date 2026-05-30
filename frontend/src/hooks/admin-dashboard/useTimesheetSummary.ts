import { useQuery } from '@tanstack/react-query';
import { timesheetService } from '@/services/api/timesheet.service';
import { handleApiError } from '@/services/api/client';

export const useTimesheetSummary = () => {
  return useQuery({
    queryKey: ['dashboard', 'timesheet-summary'],
    queryFn: async () => {
      try {
        const response = await timesheetService.getSummary();
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