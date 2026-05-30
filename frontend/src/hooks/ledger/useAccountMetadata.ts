import { useQuery } from '@tanstack/react-query';
import { ledgerService } from '@/services/api/ledger.service';
import { authManager } from '@/lib/auth';

export const useAccountMetadata = () => {
  const role = authManager.getUserRole();
  const enabled = authManager.isTokenValid() && role === 'admin';
  return useQuery({
    queryKey: ['ledger-account-metadata'],
    queryFn: async () => {
      try {
        const result = await ledgerService.getAccountMetadata();
        return result || [];
      } catch (error) {
        console.error('Error fetching account metadata:', error);
        return [];
      }
    },
    enabled,
    retry: 3,
  });
};
