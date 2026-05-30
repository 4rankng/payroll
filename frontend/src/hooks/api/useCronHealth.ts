import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { cronHealthService } from '@/services/api/cron-health.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification, showErrorNotification } from '@/utils/error-handler';

export const useCronJobs = () =>
  useQuery({
    queryKey: QueryKeys.cronHealth.jobs(),
    queryFn: () => cronHealthService.getJobs(),
  });

export const useToggleCronJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      cronHealthService.toggleJob(name, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.cronHealth.all });
      showSuccessNotification('Cập nhật trạng thái cron job thành công');
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || 'Không thể cập nhật trạng thái cron job');
    },
  });
};
