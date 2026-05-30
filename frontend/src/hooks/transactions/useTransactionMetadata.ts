import { useQuery } from '@tanstack/react-query';
import { transactionService } from '@/services/api/transaction.service';
import { authManager } from '@/lib/auth';
import type { TransactionMetadata } from '@/services/api/transaction.service';

/**
 * Hook to fetch and cache transaction metadata (types and statuses)
 * Cached for 30 minutes to reduce API calls
 */
export const useTransactionMetadata = () => {
  const role = authManager.getUserRole();
  const enabled = authManager.isTokenValid() && role === 'admin';
  return useQuery<TransactionMetadata>({
    queryKey: ['transactionMetadata'],
    queryFn: () => transactionService.getTransactionMetadata(),
    enabled,
    gcTime: 60 * 60 * 1000, // 1 hour (formerly cacheTime)
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};
