import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  transactionService,
  type Transaction,
  type UpdateTransactionEvidenceRequest,
  type UpdateTransactionEvidenceResponse,
} from '@/services/api/transaction.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

interface UpdateEvidenceVariables extends UpdateTransactionEvidenceRequest {
  id: number;
}

interface MutationContext {
  previousTransaction?: Transaction;
}

export function useUpdateTransactionEvidence() {
  const queryClient = useQueryClient();

  return useMutation<UpdateTransactionEvidenceResponse, unknown, UpdateEvidenceVariables, MutationContext>({
    mutationFn: ({ id, ...payload }) => transactionService.updateTransactionEvidence(id, payload),
    onMutate: async ({ id, url, asset_id }) => {
      await queryClient.cancelQueries({ queryKey: ['transaction', id] });

      const trimmedUrl = typeof url === 'string' ? url.trim() : undefined;

      const previousTransaction = queryClient.getQueryData<Transaction>(['transaction', id]);

      if (previousTransaction) {
        const optimisticTransaction: Transaction = {
          ...previousTransaction,
          ...(trimmedUrl ? { url: trimmedUrl } : {}),
          ...(typeof asset_id === 'number' ? { asset_id } : {}),
        };

        queryClient.setQueryData(['transaction', id], optimisticTransaction);
      }

      return { previousTransaction };
    },
    onSuccess: (response, variables) => {
      if (response.data) {
        queryClient.setQueryData(['transaction', variables.id], response.data);
      }

      queryClient.invalidateQueries({ queryKey: ['transactions'] });

      showSuccessNotification(response);
    },
    onError: (error, variables, context) => {
      if (context?.previousTransaction) {
        queryClient.setQueryData(['transaction', variables.id], context.previousTransaction);
      }

      showErrorNotification(error, 'Không thể cập nhật chứng từ giao dịch');
    },
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: ['transaction', variables.id] });
    },
  });
}
