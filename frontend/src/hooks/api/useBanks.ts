import { useQuery, useMutation, useQueryClient, type UseQueryOptions } from '@tanstack/react-query';
import { bankService } from '@/services/api/bank.service';
import { QueryKeys } from '@/lib/queryKeys';
import type { Bank, BankFilters, CreateBankRequest } from '@/types/api/bank.types';
import type { ApiError } from '@/types/api.types';
import { REFERENCE_DATA_STALE_TIME_MS } from '@/lib/cache/queryCacheTimes';

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
    staleTime: REFERENCE_DATA_STALE_TIME_MS, // Banks are near-static reference data
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

  return useMutation<Bank, ApiError, CreateBankRequest>({
    mutationFn: (data: CreateBankRequest) => bankService.createBank(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.banks.all });
    },
    meta: {
      successMessage: 'Tạo ngân hàng thành công',
    },
  });
};
