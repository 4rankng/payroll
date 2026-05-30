import { useQuery } from '@tanstack/react-query';
import { transactionService, TransactionFilters } from '@/services/api/transaction.service';

export function useTransactions(filters?: TransactionFilters) {
  return useQuery({
    queryKey: ['transactions', filters],
    queryFn: () => transactionService.listTransactions(filters),
    staleTime: 0,
    placeholderData: (prev) => prev,
  });
}

export function useTransaction(id: number) {
  return useQuery({
    queryKey: ['transaction', id],
    queryFn: () => transactionService.getTransaction(id),
    enabled: !!id,
  });
}

export function usePendingTransactions(filters?: TransactionFilters) {
  return useQuery({
    queryKey: ['transactions', 'pending', filters],
    queryFn: () => transactionService.getPendingTransactions(filters),
  });
}