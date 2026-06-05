import { useMutation, useQueryClient } from '@tanstack/react-query';
import { transactionService, type CreateTransactionRequest } from '@/services/api/transaction.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

export function useCreateTransaction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateTransactionRequest) => transactionService.createTransaction(data),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['ledger'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });

      showSuccessNotification('Tạo giao dịch thành công');
    },
    onError: (error: unknown) => {
      showErrorNotification(error, 'Không thể tạo giao dịch');
    },
  });
}
