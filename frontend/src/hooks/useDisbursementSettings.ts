import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/services/api/client';
import type { DisbursementSettings } from '@/types/api/disbursement-settings.types';

/**
 * Fetches the consolidated disbursement settings endpoint.
 * Returns active provider name, capabilities, fee info, and balances.
 * Cached for 60 seconds client-side to avoid redundant round-trips.
 */
export function useDisbursementSettings() {
  return useQuery<DisbursementSettings>({
    queryKey: ['disbursement-settings'],
    queryFn: async () => {
      const res = await apiClient.get<DisbursementSettings>(
        '/admin/settings/disbursement',
      );
      return res.data!;
    },
  });
}
