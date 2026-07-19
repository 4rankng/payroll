import { useMutation, useQueryClient } from '@tanstack/react-query';
import { transactionService } from '@/services/api/transaction.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

interface DeleteTransactionParams {
  id: number;
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: DeleteTransactionParams) =>
      transactionService.deleteTransaction(id),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['transaction', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['transactions', 'pending'] });
      queryClient.invalidateQueries({ queryKey: ['ledger'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });

      showSuccessNotification('Hủy giao dịch thành công');
    },
    onError: (error: unknown) => {
      showErrorNotification(error, 'Không thể hủy giao dịch');
    },
  });
}
