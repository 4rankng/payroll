import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/services/api/client';
import type { DisbursementSettings } from '@/types/api/disbursement-settings.types';
import { DISBURSEMENT_SETTINGS_REFETCH_INTERVAL_MS } from '@/lib/cache/queryCacheTimes';

/**
 * Fetches the consolidated disbursement settings endpoint.
 * Returns active provider name, capabilities, fee info, and balances.
 * Cached for 60 seconds client-side to avoid redundant round-trips.
 * Polls on an interval so the wallet card's balances and NCC divergence
 * badge stay current without a manual sync click.
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
    refetchInterval: DISBURSEMENT_SETTINGS_REFETCH_INTERVAL_MS,
  });
}
