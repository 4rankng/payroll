/**
 * React Query Hooks for Advance Payment (Ứng lương tháng) feature
 */

import { useEffect, useRef } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "@/components/ui/sonner";
import { advancePaymentService } from "@/services/api/advance-payment.service";
import { QueryKeys } from "@/lib/queryKeys";
import {
  showSuccessNotification,
} from "@/utils/error-handler";
import { formatCurrency } from "@/utils/formatters";
import type {
  AdvancePaymentInfo,
  AdvancePaymentHistoryItem,
  AdvancePaymentListItem,
  AdvancePaymentFilters,
  AdvancePaymentSummary,
  AdvancePaymentFileHistoryItem,
  CreateAdvancePaymentRequest,
  CalculateFeeRequest,
  ImportPollingOptions,
  FlexPayEmployeeListItem,
  FlexPayEmployeeFilters,
  FlexPayEmployeeListMeta,
  AvailableMonth,
} from "@/types/api/advance-payment.types";
import type { EmployeeImportStatus } from "@/types/api/employee.types";

// ============================================================================
// Constants
// ============================================================================

const PENDING_POLL_INTERVAL = 30_000;

// ============================================================================
// Employee Hooks
// ============================================================================

/**
 * Get current month's advance payment info for the logged-in employee.
 * Polls every 30s while a pending request exists.
 *
 * 30s staleTime + window-focus refetch keeps the limit/pending widgets in
 * sync with the disbursement worker, which can flip a request from
 * PENDING → COMPLETED in well under a minute.
 */
export function useAdvancePaymentInfo(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: QueryKeys.advancePayments.employee.info,
    queryFn: () => advancePaymentService.getAdvancePaymentInfo(),
    enabled: options?.enabled !== undefined ? options.enabled : true,
    refetchOnWindowFocus: true,
    refetchInterval: (query) => {
      const data = query.state.data as
        | { data?: AdvancePaymentInfo }
        | undefined;
      return data?.data?.pendingAmount ? PENDING_POLL_INTERVAL : false;
    },
  });
}

/**
 * Submit a new advance payment request
 */
export function useRequestAdvancePayment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateAdvancePaymentRequest) =>
      advancePaymentService.requestAdvancePayment(data),
    onSuccess: async (response) => {
      // Invalidate employee info to refresh remaining amount
      queryClient.invalidateQueries({ queryKey: QueryKeys.advancePayments.employee.info });
      // Invalidate history
      queryClient.invalidateQueries({
        queryKey: ["employee", "advance-payment", "history"],
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
}

/**
 * Get self-check-in advance info for a check-in-enabled employee (dedicated
 * /me/check-in-advance path). Polls every 30s while a pending request exists.
 */
export function useCheckInAdvanceInfo(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: QueryKeys.advancePayments.employee.checkInAdvanceInfo,
    queryFn: () => advancePaymentService.getCheckInAdvanceInfo(),
    enabled: options?.enabled !== undefined ? options.enabled : true,
    refetchOnWindowFocus: true,
    refetchInterval: (query) => {
      const data = query.state.data as { data?: AdvancePaymentInfo } | undefined;
      return data?.data?.pendingAmount ? PENDING_POLL_INTERVAL : false;
    },
  });
}

/**
 * Submit a new advance request under the self-check-in flow.
 */
export function useRequestCheckInAdvance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateAdvancePaymentRequest) =>
      advancePaymentService.requestCheckInAdvance(data),
    onSuccess: async (response) => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.advancePayments.employee.checkInAdvanceInfo });
      queryClient.invalidateQueries({ queryKey: ["employee", "advance-payment", "history"] });
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
}

/**
 * Get advance payment history for the logged-in employee.
 * Polls every 30s while any request is PENDING and notifies on status change.
 */
export function useAdvancePaymentHistory(
  filters?: { page?: number; pageSize?: number },
  options?: { enabled?: boolean },
) {
  const previousStatusesRef = useRef<Map<number, string>>(new Map());

  const query = useQuery({
    queryKey: QueryKeys.advancePayments.employee.history(filters),
    queryFn: () => advancePaymentService.getAdvancePaymentHistory(filters),
    enabled: options?.enabled !== undefined ? options.enabled : true,
    // Match useAdvancePaymentInfo: 30s staleTime + window-focus refetch so
    // the history list catches the PENDING → COMPLETED transition the
    // disbursement worker fires shortly after submission.
    refetchOnWindowFocus: true,
    refetchInterval: (q) => {
      const data = q.state.data as
        | { data?: AdvancePaymentHistoryItem[] }
        | undefined;
      return data?.data?.some((item) => item.status === "PENDING")
        ? PENDING_POLL_INTERVAL
        : false;
    },
  });

  const history = query.data?.data;

  useEffect(() => {
    if (!history || history.length === 0) return;

    const current = new Map<number, string>();

    for (const item of history) {
      current.set(item.id, item.status);

      const prev = previousStatusesRef.current.get(item.id);
      if (prev !== "PENDING" || item.status === "PENDING") continue;

      const amount = formatCurrency(item.netAmount);
      const messages: Record<string, { title: string; desc: string }> = {
        COMPLETED: {
          title: "Ứng lương thành công",
          desc: `${amount} đã được chuyển vào tài khoản`,
        },
        FAILED: {
          title: "Ứng lương thất bại",
          desc: `Yêu cầu ${amount} thất bại, vui lòng liên hệ quản lý`,
        },
        CANCELLED: {
          title: "Yêu cầu đã bị hủy",
          desc: `Yêu cầu ứng lương ${amount} đã bị hủy`,
        },
      };

      const msg = messages[item.status];
      if (msg) {
        toast(msg.title, { description: msg.desc, duration: 6000 });
        navigator.vibrate?.([200, 100, 200]);
      }
    }

    previousStatusesRef.current = current;
  }, [history]);

  return query;
}

/**
 * Calculate fee for a given advance payment amount
 * Not cached - always fetches fresh calculation
 */
export function useCalculateFee() {
  return useMutation({
    mutationFn: (data: CalculateFeeRequest) =>
      advancePaymentService.calculateFee(data),
    onError: (error) => {
      // Only log, don't show notification for fee calculation errors
      console.error("Fee calculation error:", error);
    },
  });
}

/**
 * Cancel a pending advance payment request
 * Only PENDING requests can be cancelled by employees
 */
export function useCancelAdvancePaymentRequest() {
  const queryClient = useQueryClient();

  const invalidate = () => {
    // Invalidate employee info to refresh remaining amount
    queryClient.invalidateQueries({ queryKey: QueryKeys.advancePayments.employee.info });
    // Invalidate history to show updated status
    queryClient.invalidateQueries({
      queryKey: ["employee", "advance-payment", "history"],
    });
  };

  return useMutation({
    mutationFn: (id: number) =>
      advancePaymentService.cancelAdvancePaymentRequest(id),
    onSuccess: (response) => {
      invalidate();
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Refetch on error too — a common failure mode is the request having
    // already advanced PENDING → COMPLETED in the background. Refreshing
    // the cache shows the user the real current state instead of the stale
    // "Chờ xử lý" pill they were trying to cancel.
    onError: invalidate,
  });
}

// ============================================================================
// Admin Hooks
// ============================================================================

/**
 * Get paginated list of all advance payment requests
 */
export function useAdvancePayments(
  filters?: AdvancePaymentFilters,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.list(filters),
    queryFn: () => advancePaymentService.getAdvancePayments(filters),
    enabled: options?.enabled !== undefined ? options.enabled : true,
    // Adaptive polling: fast (15s) when there are PENDING/APPROVED rows that
    // the disbursement cron is actively working through, slow (60s) when
    // everything is settled. Stops entirely when the tab is in the background
    // (refetchIntervalInBackground defaults to false). Window-focus refetch
    // covers the "came back to tab" case at no extra cost.
    refetchOnWindowFocus: true,
    refetchInterval: (query) => {
      const rows = (query.state.data as { data?: { status: string }[] } | undefined)?.data;
      const hasActive = rows?.some(
        (r) => r.status === "PENDING" || r.status === "APPROVED",
      );
      return hasActive ? 15_000 : 60_000;
    },
  });
}

/**
 * Get advance payment summary statistics
 */
export function useAdvancePaymentSummary(
  params?: { fromDate?: string; toDate?: string; forMonth?: string },
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.summary(params),
    queryFn: () => advancePaymentService.getAdvancePaymentSummary(params),
    enabled: options?.enabled !== undefined ? options.enabled : true,
    // Mirror the adaptive interval from the list query: fast when there are
    // unsettled requests (summary totals are changing), slow when quiet.
    refetchOnWindowFocus: true,
    refetchInterval: (query) => {
      const summary = (query.state.data as { data?: { totalPending?: number; totalApproved?: number } } | undefined)?.data;
      const hasActive = (summary?.totalPending ?? 0) + (summary?.totalApproved ?? 0) > 0;
      return hasActive ? 15_000 : 60_000;
    },
  });
}

/**
 * Export pending requests to Excel file
 */
export function useExportAdvancePayments() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params?: { fromDate?: string; toDate?: string }) =>
      advancePaymentService.exportAdvancePayments(params),
    onSuccess: () => {
      showSuccessNotification("Đã xuất file Excel thành công");
      queryClient.invalidateQueries({ queryKey: QueryKeys.advancePayments.admin.files });
    },
  });
}

/**
 * Upload bank transfer result file
 */
export function useUploadBankResult() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (formData: FormData) =>
      advancePaymentService.uploadBankResult(formData),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: ["admin", "advance-payments"],
      });
      queryClient.invalidateQueries({
        queryKey: ["employee", "advance-payment"],
      });

      showSuccessNotification(response.message || "Tải lên thành công");
    },
  });
}

/**
 * Import Flexible Payroll Template (Async with polling)
 */
export function useImportFlexTemplate(options?: ImportPollingOptions) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (formData: FormData) =>
      advancePaymentService.importFlexTemplateWithPolling(formData, {
        maxWaitTime: 300000, // 5 minutes
        pollInterval: 1000, // 1 second
        onProgress: options?.onProgress,
        onStatusChange: options?.onStatusChange,
      }),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: ["admin", "advance-payments"],
      });
      queryClient.invalidateQueries({ queryKey: QueryKeys.advancePayments.admin.files });
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.availableMonths,
      });

      const result = response.result;
      if (result) {
        showSuccessNotification(
          `Nhập file thành công: ${result.totalRows} dòng, ${result.employeesCreated} nhân viên mới, ${result.assignmentsCreated} phân công mới`,
        );
      }
    },
  });
}

/**
 * Get uploaded file history
 */
export function useAdvancePaymentFileHistory(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.files,
    queryFn: () => advancePaymentService.getFileHistory(),
    enabled: options?.enabled !== undefined ? options.enabled : true,
  });
}

/**
 * Cancel an advance payment request (admin)
 * Only PENDING and APPROVED requests can be cancelled
 */
export function useAdminCancelAdvancePayment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => advancePaymentService.cancelAdvancePayment(id),
    onSuccess: (response) => {
      // Invalidate admin list and summary
      queryClient.invalidateQueries({
        queryKey: ["admin", "advance-payments"],
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
}

/**
 * Retry disbursement for an APPROVED or FAILED advance payment request (admin)
 * After retry, polls the status endpoint until the request reaches a terminal state.
 */
export function useRetryDisbursement(options?: {
  onPollingChange?: (id: number, isPolling: boolean) => void;
}) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const result = await advancePaymentService.retryDisbursement(id);
      // Start polling for status
      options?.onPollingChange?.(id, true);
      pollDisbursementStatus(id, queryClient, () => options?.onPollingChange?.(id, false));
      return result;
    },
    onSuccess: (response) => {
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
}

/**
 * Polls the disbursement status endpoint every 3 seconds until terminal state.
 * Invalidates the admin list on completion and calls the onDone callback.
 */
function pollDisbursementStatus(
  id: number,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  queryClient: any,
  onDone: () => void,
) {
  let attempts = 0;
  const maxAttempts = 60; // 3 minutes max

  const interval = setInterval(async () => {
    attempts++;
    try {
      const res = await advancePaymentService.getDisbursementStatus(id);
      if (res.data?.isTerminal || attempts >= maxAttempts) {
        clearInterval(interval);
        queryClient.invalidateQueries({ queryKey: ["admin", "advance-payments"] });
        onDone();
      }
    } catch {
      // On error, keep polling — network blips shouldn't stop us
    }
  }, 3000);
}

/**
 * Get flex pay employee list
 * Returns employees from the latest imported flex pay month
 */
export function useFlexPayEmployees(
  filters?: FlexPayEmployeeFilters,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.flexPayEmployees(filters),
    queryFn: () => advancePaymentService.getFlexPayEmployees(filters),
    enabled: options?.enabled !== undefined ? options.enabled : true,
  });
}

export function useFlexPayEmployeesInfinite(
  filters?: Omit<FlexPayEmployeeFilters, "page">,
) {
  const pageSize = (filters?.pageSize as number) || 50;
  return useInfiniteQuery({
    queryKey: QueryKeys.advancePayments.admin.flexPayEmployees({ ...filters, infinite: true } as Record<string, unknown>),
    queryFn: ({ pageParam = 1 }) =>
      advancePaymentService.getFlexPayEmployees({ ...filters, page: pageParam as number, pageSize }),
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      const p = lastPage.pagination;
      if (!p) return undefined;
      return p.page < p.totalPages ? p.page + 1 : undefined;
    },
  });
}

/**
 * Export flex pay employee list to Excel
 */
export function useExportFlexPayEmployees() {
  return useMutation({
    mutationFn: (params?: { forMonth?: string }) =>
      advancePaymentService.exportFlexPayEmployees(params),
    onSuccess: () => {
      showSuccessNotification("Đã xuất danh sách nhân viên thành công");
    },
  });
}

/**
 * Get available months with flex pay data
 * Returns list of months that have flex pay data imported
 */
export function useAvailableMonths(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.availableMonths,
    queryFn: () => advancePaymentService.getAvailableMonths(),
    enabled: options?.enabled !== undefined ? options.enabled : true,
  });
}

/**
 * Import flexible employee list from Excel file (async with polling)
 * Uploads an Excel file containing employees under flexible payment schedule
 * Polls for status and returns final EmployeeImportStatus when completed
 */
export function useImportFlexibleEmployeeList(options?: ImportPollingOptions) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (formData: FormData) =>
      advancePaymentService.importFlexibleEmployeeListWithPolling(formData, {
        maxWaitTime: 300000, // 5 minutes
        pollInterval: 1000, // 1 second
        onProgress: options?.onProgress,
        onStatusChange: options?.onStatusChange,
      }),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: ["admin", "flex-pay-employees"],
      });

      showSuccessNotification(
        `Nhập file thành công: ${response.total_rows} dòng, ${response.created_count} nhân viên mới, ${response.updated_count} đã cập nhật`,
      );
    },
  });
}

// ============================================================================
// Utility exports
// ============================================================================

export { QueryKeys as ADVANCE_PAYMENT_QUERY_KEYS };
