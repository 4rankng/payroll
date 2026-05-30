import { useQuery, useMutation, useQueryClient, type UseQueryOptions } from '@tanstack/react-query';
import { bankService } from '@/services/api/bank.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification } from '@/utils/error-handler';
import type { Bank, BankFilters, CreateBankRequest } from '@/types/api/bank.types';
import type { ApiError } from '@/types/api.types';

// Fetch ALL banks — one request, cached for 30 minutes since banks rarely change
export const useAllBanks = () => {
  return useQuery({
    queryKey: QueryKeys.banks.list(),
    queryFn: async () => {
      const res = await bankService.getBanks({ pageSize: 500 });
      return res.data;
    },
    gcTime: 60 * 60 * 1000,
    refetchOnMount: 'always',
    refetchOnWindowFocus: false,
  });
};

// Legacy: Get paginated banks list
export const useBanks = (filters?: BankFilters) => {
  return useQuery({
    queryKey: [...QueryKeys.banks.list(), filters],
    queryFn: () => bankService.getBanks(filters),
  });
};

// Get single bank
type BankDetailsQueryKey = ReturnType<typeof QueryKeys.banks.detail>;

type UseBankOptions = Omit<
  UseQueryOptions<Bank, ApiError, Bank, BankDetailsQueryKey>,
  'queryKey' | 'queryFn'
>;

export const useBank = (id: number, options?: UseBankOptions) => {
  const { enabled: enabledOverride, ...reactQueryOptions } = options ?? {};
  const shouldFetchBank = Boolean(id) && (enabledOverride ?? true);

  return useQuery<Bank, ApiError, Bank, BankDetailsQueryKey>({
    queryKey: QueryKeys.banks.detail(id),
    queryFn: () => bankService.getBankById(id),
    enabled: shouldFetchBank,
    ...reactQueryOptions,
  });
};

// Create bank
export const useCreateBank = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateBankRequest) => bankService.createBank(data),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.banks.all });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};
