import { useMutation, useQueryClient } from '@tanstack/react-query';
import { bankService } from '@/services/api/bank.service';
import { QueryKeys } from '@/lib/queryKeys';
import type { CreateBankRequest, UpdateBankRequest } from '@/types/api/bank.types';

export const useCreateBank = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateBankRequest) => bankService.createBank(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.banks.all });
    },
    meta: {
      invalidates: [QueryKeys.banks.all],
    },
  });
};

export const useUpdateBank = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateBankRequest }) =>
      bankService.updateBank(id, data),
    onSuccess: (updatedBank) => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.banks.all });

      queryClient.setQueryData(
        [...QueryKeys.banks.details(), updatedBank.id],
        updatedBank
      );
    },
    meta: {
      invalidates: [QueryKeys.banks.all],
    },
  });
};

export const useDeleteBank = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => bankService.deleteBank(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.banks.all });
    },
    meta: {
      invalidates: [QueryKeys.banks.all],
    },
  });
};