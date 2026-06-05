import { useQuery, UseQueryResult } from '@tanstack/react-query';
import { ledgerService } from '@/services/api/ledger.service';
import type { AccountMetadata } from '@/types/api/financial.types';
import { authManager } from '@/lib/auth';

/**
 * Hook to fetch and cache account metadata from the API
 * Returns dynamic account types, labels, categories, and normal sides
 */
export const useLedgerMetadata = () => {
  const role = authManager.getUserRole();
  const enabled = authManager.isTokenValid() && role === 'admin';
  return useQuery({
    queryKey: ['ledger-accounts-metadata'],
    queryFn: () => ledgerService.getAccountMetadata(),
    gcTime: 1000 * 60 * 60 * 24, // 24 hours
    retry: 3,
    refetchOnWindowFocus: false,
    enabled,
  });
};

/**
 * Hook to get account options formatted for form components
 */
export const useLedgerAccountOptions = () => {
  const { data: metadata, isLoading, error } = useLedgerMetadata();

  const hasMetadata = Array.isArray(metadata) && metadata.length > 0;
  const accountOptions = hasMetadata
    ? ledgerService.getAccountOptionsFromMetadata(metadata)
    : ledgerService.getAccountOptions(); // Fallback to static options

  return {
    accountOptions,
    metadata,
    isLoading,
    error,
  };
};

/**
 * Hook to get display name for an account
 */
export const useAccountDisplayName = (account: string) => {
  const { data: metadata } = useLedgerMetadata();

  const hasMetadata = Array.isArray(metadata) && metadata.length > 0;
  if (!hasMetadata) {
    return ledgerService.getAccountDisplayName(account);
  }

  return ledgerService.getAccountDisplayNameFromMetadata(account, metadata!);
};
