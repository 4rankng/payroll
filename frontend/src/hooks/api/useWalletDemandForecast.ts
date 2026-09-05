import { useQuery } from '@tanstack/react-query';
import { QueryKeys } from '@/lib/queryKeys';
import { walletService } from '@/services/api/wallet.service';

export function useWalletDemandForecast(enabled = true) {
  return useQuery({
    queryKey: QueryKeys.wallet.demandForecast(),
    queryFn: () => walletService.getDemandForecast(),
    enabled,
    // 30s staleness + focus refetch (same policy as the advance-payment
    // widgets): the forecast reacts to cycle data within the session instead
    // of pinning the first snapshot — a day-old recommendation otherwise
    // lingers as a wrong "Cần nạp thêm" figure.
    staleTime: 30_000,
    refetchOnWindowFocus: true,
  });
}
