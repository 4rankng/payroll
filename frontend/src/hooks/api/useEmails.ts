import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { emailService } from '@/services/api/email.service';
import type { EmailHistoryFilters, EmailHistoryResult } from '@/types/api/email.types';
import { showSuccessNotification, showErrorNotification } from '@/utils/error-handler';

export const emailQueryKeys = {
  all: ['email'] as const,
  historyLists: () => [...emailQueryKeys.all, 'history'] as const,
  historyList: (pageSize: number) => [...emailQueryKeys.historyLists(), { pageSize }] as const,
};

export interface UseEmailHistoryOptions {
  pageSize?: number;
  enabled?: boolean;
}

export const useEmailHistory = (options: UseEmailHistoryOptions = {}) => {
  const { pageSize = 20, enabled = true } = options;

  return useInfiniteQuery<EmailHistoryResult>({
    queryKey: emailQueryKeys.historyList(pageSize),
    queryFn: async ({ pageParam = 1 }) => {
      const filters: EmailHistoryFilters = {
        page: pageParam as number,
        pageSize,
      };
      return emailService.getEmailHistory(filters);
    },
    enabled,
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      const pagination = lastPage.pagination;
      const nextPage = pagination.page < pagination.totalPages ? pagination.page + 1 : undefined;
      return nextPage;
    },
    refetchOnWindowFocus: enabled,
  });
};

export const useSettleFromEmailHistory = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => emailService.settleFromEmailHistory(id),
    onSuccess: () => {
      showSuccessNotification('Đã xác nhận nhận tiền và xử lý đối soát thành công');
      queryClient.invalidateQueries({ queryKey: emailQueryKeys.historyLists() });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['timesheets'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
    },
    onError: (error) => {
      showErrorNotification(error, 'Xác nhận đối soát thất bại');
    },
  });
};

export const useUploadSettlement = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) => emailService.uploadSettlement(id, file),
    onSuccess: (response) => {
      const { processedTimesheets, skippedTimesheets } = response;
      if (processedTimesheets > 0 && skippedTimesheets > 0) {
        showSuccessNotification(`Đã xử lý ${processedTimesheets} timesheets mới, bỏ qua ${skippedTimesheets} đã thanh toán`);
      } else if (skippedTimesheets > 0) {
        showSuccessNotification(`Tất cả ${skippedTimesheets} timesheets đã được thanh toán trước đó`);
      } else {
        showSuccessNotification(`Đã xử lý file đối soát thành công`);
      }
      queryClient.invalidateQueries({ queryKey: emailQueryKeys.historyLists() });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['timesheets'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
    },
    onError: (error) => {
      showErrorNotification(error, 'Tải file đối soát thất bại');
    },
  });
};

export const useStandaloneSettlementUpload = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (file: File) => emailService.uploadStandaloneSettlement(file),
    onSuccess: () => {
      showSuccessNotification('Đã xử lý file đối soát thành công');
      queryClient.invalidateQueries({ queryKey: emailQueryKeys.historyLists() });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['timesheets'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
    },
    onError: (error) => {
      showErrorNotification(error, 'Xử lý file đối soát thất bại');
    },
  });
};
