import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { insightsService } from '@/services/api/insights.service';
import { QueryKeys } from '@/lib/queryKeys';

export const useInsightDigest = (date?: string) =>
  useQuery({
    queryKey: QueryKeys.insights.digest(date),
    queryFn: () => insightsService.getDigest(date),
    // The digest is rebuilt once a day at 07:30; cache for the session.
    staleTime: 10 * 60_000,
  });

export const useDismissInsight = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => insightsService.dismiss(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.insights.all });
    },
  });
}
