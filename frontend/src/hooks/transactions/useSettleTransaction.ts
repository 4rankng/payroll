import { useMutation, useQueryClient } from '@tanstack/react-query';
import { transactionService, SettleTransactionRequest } from '@/services/api/transaction.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

interface SettleTransactionParams {
  id: number;
  data: SettleTransactionRequest;
}

export function useSettleTransaction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: SettleTransactionParams) =>
      transactionService.settleTransaction(id, data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['transaction', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['ledger'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });

      showSuccessNotification('Thanh toán giao dịch thành công');
    },
    onError: (error: unknown) => {
      showErrorNotification(error, 'Không thể thanh toán giao dịch');
    },
  });
}