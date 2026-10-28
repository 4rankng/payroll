import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ledgerService } from '@/services/api/ledger.service';

export const useLedgerEntry = (id: number, enabled = true) => {
  return useQuery({
    queryKey: ['ledger-entry', id],
    queryFn: () => ledgerService.getEntry(id),
    enabled: enabled && id > 0,
  });
};

export const useReverseLedgerEntry = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      ledgerService.reverseEntry(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ledger-entries'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-entry'] });
    },
  });
};