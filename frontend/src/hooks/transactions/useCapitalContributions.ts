import { useQuery } from '@tanstack/react-query';
import { ledgerService } from '@/services/api/ledger.service';
import type { OwnerContribution } from '@/types/api/financial.types';

export function useCapitalContributions() {
  return useQuery({
    queryKey: ['capital-contributions'],
    queryFn: async () => {
      // Fetch ledger summary without date filters to get all-time data
      const summary = await ledgerService.getLedgerSummary('', '');

      // Return owner contributions from the summary
      return summary.by_owners || [];
    },
    retry: 3,
  });
}
