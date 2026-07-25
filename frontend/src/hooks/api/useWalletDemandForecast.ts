import { useQuery } from '@tanstack/react-query';
import { QueryKeys } from '@/lib/queryKeys';
import { walletService } from '@/services/api/wallet.service';

export function useWalletDemandForecast(enabled = true) {
  return useQuery({
    queryKey: QueryKeys.wallet.demandForecast(),
    queryFn: () => walletService.getDemandForecast(),
    enabled,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnWindowFocus: false,
  });
}
