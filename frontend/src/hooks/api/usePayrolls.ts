import { useMutation, useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { bulkTransferService, type BulkTransferExportParams, type BulkTransferResultResponse, type BulkTransferUploadHistoriesParams, type BulkTransferUploadHistoriesResult, type AutoBulkTransferResponse, type AutoBulkTransferStatusResponse } from '@/services/api/bulk-transfer.service';
import { timesheetService } from '@/services/api/timesheet.service';
import { payrollService, type PaymentHistoryExportParams } from '@/services/api/payroll.service';
import { showSuccessNotification } from '@/utils/error-handler';
import type { PaymentHistoryFilters } from '@/types/api/payroll.types';

/**
 * Hook to export bulk transfer Excel file for bank transfers
 */
export const useExportBulkTransfer = () => {

  return useMutation({
    mutationFn: (params: BulkTransferExportParams) =>
      bulkTransferService.exportBulkTransfer(params),
    onSuccess: (response) => {
      // exportBulkTransfer returns void and downloads file directly
      // No toast notification needed as the file download is the success indicator
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to import bulk transfer result from Excel file
 */
export const useImportBulkTransferResult = () => {

  return useMutation({
    mutationFn: (file: File) =>
      bulkTransferService.importBulkTransferResult(file),
    onSuccess: (response) => {
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to import settlement result from payroll report
 */
export const useUploadSettlementResult = () => {
  return useMutation({
    mutationFn: (file: File) =>
      timesheetService.uploadSettlementResult(file),
    onSuccess: (response) => {
      const message = response.message ?? 'Nhập kết quả sao kê thành công';
      showSuccessNotification(message);
    },
  });
};

/**
 * Hook to export payroll report for paid timesheets
 */
export const useExportPayrollReport = () => {
  return useMutation({
    mutationFn: (params: { atDate: string }) =>
      timesheetService.exportPayrollReport(params),
    onSuccess: () => {
      showSuccessNotification('Xuất sao kê thành công');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to send payroll report via email
 */
export const useSendPayrollReportEmail = () => {
  return useMutation({
    mutationFn: (params: {
      reportAtDate: string;
      recipients: string[];
      cc?: string[];
      bcc?: string[];
    }) => timesheetService.sendPayrollReportEmail(params),
    onSuccess: () => {
      showSuccessNotification('Gửi email thành công');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to export timesheets to Excel
 * Supports filtering by status, projects, and date range
 */
export const useExportApprovedTimesheets = () => {
  return useMutation({
    mutationFn: (params: {
      fromDate: string;
      toDate: string;
      project_ids?: string;
      employee_id?: number;
      status?: string;
    }) =>
      timesheetService.exportTimesheetsExcel(params),
    onSuccess: () => {
      showSuccessNotification('Xuất bảng công thành công');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to export timesheet entries template
 * Generates an Excel template for entering timesheet entries for a specific project and date range
 */
export const useExportTimesheetEntriesTemplate = () => {
  return useMutation({
    mutationFn: (params: {
      fromDate: string;
      toDate: string;
      projectId: number;
    }) =>
      timesheetService.exportEntriesTemplate(params),
    onSuccess: () => {
      showSuccessNotification('Tải mẫu nhập công thành công');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to fetch bulk transfer upload histories
 */
export const useBulkTransferUploadHistories = (params?: BulkTransferUploadHistoriesParams, enabled: boolean = true) => {
  return useQuery<BulkTransferUploadHistoriesResult>({
    queryKey: ['bulk-transfer-upload-histories', params],
    queryFn: () => bulkTransferService.getBulkTransferUploadHistories(params),
    enabled,
    refetchOnWindowFocus: true,
  });
};

/**
 * Hook to fetch a specific bulk transfer upload history detail
 */
export const useBulkTransferUploadHistoryById = (id: number | null) => {
  return useQuery({
    queryKey: ['bulk-transfer-upload-history', id],
    queryFn: () => bulkTransferService.getBulkTransferUploadHistoryById(id!),
    enabled: id !== null,
  });
};

/**
 * Hook to fetch payment histories with standard pagination (desktop)
 */
export const usePaymentHistories = (filters?: PaymentHistoryFilters) => {
  return useQuery({
    queryKey: ['payment-histories', filters],
    queryFn: () => payrollService.getPaymentHistories(filters),
    enabled: filters !== undefined,
    refetchOnWindowFocus: true,
  });
};

/**
 * Hook to fetch payment histories with infinite scroll (mobile)
 */
export const useInfinitePaymentHistories = (filters?: Omit<PaymentHistoryFilters, 'page'>) => {
  return useInfiniteQuery({
    queryKey: ['payment-histories', 'infinite', filters],
    queryFn: async ({ pageParam = 1 }) => {
      const result = await payrollService.getPaymentHistories({
        ...filters,
        page: pageParam,
        pageSize: 20,
      });
      return result;
    },
    enabled: filters !== undefined,
    refetchOnWindowFocus: true,
    getNextPageParam: (lastPage) => {
      if (!lastPage?.pagination) {
        return undefined;
      }
      const { page, totalPages } = lastPage.pagination;
      const nextPage = page < totalPages ? page + 1 : undefined;
      return nextPage;
    },
    initialPageParam: 1,
  });
};

/**
 * Hook to export payment histories to Excel
 */
export const useExportPaymentHistories = () => {
  return useMutation({
    mutationFn: (params: PaymentHistoryExportParams) =>
      payrollService.exportPaymentHistories(params),
    onSuccess: () => {
      showSuccessNotification('Xuất lịch sử thanh toán thành công');
    },
    // Error handling is done globally in React Query - will display response.message from backend
  });
};

/**
 * Hook to check if auto bulk transfer is enabled
 */
export const useAutoBulkTransferConfig = () => {
  return useQuery({
    queryKey: ['auto-bulk-transfer-config'],
    queryFn: () => bulkTransferService.getAutoBulkTransferConfig(),
  });
};

/**
 * Hook to initiate an auto bulk transfer
 */
export const useInitiateAutoBulkTransfer = () => {
  return useMutation({
    mutationFn: (params: BulkTransferExportParams) =>
      bulkTransferService.initiateAutoBulkTransfer(params),
  });
};

/**
 * Hook to poll auto bulk transfer status
 */
export const useAutoBulkTransferStatus = (batchId: string | null) => {
  return useQuery<AutoBulkTransferStatusResponse>({
    queryKey: ['auto-bulk-transfer-status', batchId],
    queryFn: () => bulkTransferService.getAutoBulkTransferStatus(batchId!),
    enabled: batchId !== null,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data?.status === 'completed') return false;
      return 3000;
    },
  });
};
