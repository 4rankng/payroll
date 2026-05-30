import { useMutation, useQueryClient } from '@tanstack/react-query';
import { transactionService } from '@/services/api/transaction.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

interface ReverseTransactionParams {
  id: number;
  reason?: string;
}

export function useReverseTransaction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, reason }: ReverseTransactionParams) =>
      transactionService.reverseTransaction(id, reason),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['transaction', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['ledger'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });

      showSuccessNotification('Đảo ngược giao dịch thành công');
    },
    onError: (error: unknown) => {
      showErrorNotification(error, 'Không thể đảo ngược giao dịch');
    },
  });
}
