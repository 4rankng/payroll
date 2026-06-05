import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { timesheetService } from '@/services/api/timesheet.service';
import { toast } from '@/components/ui/sonner';
import { getErrorMessage, showSuccessNotification } from '@/utils/error-handler';
import type { TimesheetEditRequestFilters, TimesheetFilters, TimesheetListResponse } from '@/types/api/timesheet.types';

const invalidateEditRequestCaches = async (
  queryClient: QueryClient,
  context?: { timesheetId?: number }
) => {
  // Invalidate all edit request related queries
  const invalidatePromises = [
    queryClient.invalidateQueries({ queryKey: ['timesheetEditRequests'] }),
    queryClient.invalidateQueries({ queryKey: ['partnerTimesheetEditRequests'] }),
    queryClient.invalidateQueries({ queryKey: ['timesheetEditRequest'] }),
    // Also invalidate all timesheet queries since edit requests affect timesheets
    queryClient.invalidateQueries({ queryKey: ['timesheets', 'list'] }),
  ];

  // If we have a specific timesheet ID, invalidate its detail cache
  if (typeof context?.timesheetId === 'number') {
    invalidatePromises.push(
      queryClient.invalidateQueries({ queryKey: ['timesheets', 'detail', context.timesheetId] })
    );
  }

  await Promise.all(invalidatePromises);
};

export function useCreateEditRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (timesheetId: number) => timesheetService.createEditRequest(timesheetId),
    onSuccess: async (_data, timesheetId) => {
      await invalidateEditRequestCaches(queryClient, { timesheetId });
      toast('Thành công', { description: 'Yêu cầu chỉnh sửa đã được gửi thành công' });
    },
    onError: (error: unknown) => {
      toast('Lỗi', { description: `Không thể tạo yêu cầu: ${getErrorMessage(error)}`, variant: 'destructive' });
    },
  });
}

export function useEditRequests(filters?: TimesheetEditRequestFilters, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['timesheetEditRequests', filters],
    queryFn: () => timesheetService.getEditRequests(filters),
    enabled: options?.enabled ?? true,
  });
}

export function useEditRequestById(id: number, enabled = true) {
  return useQuery({
    queryKey: ['timesheetEditRequest', id],
    queryFn: () => timesheetService.getEditRequestById(id),
    enabled,
  });
}

export function usePartnerTimesheetsWithEditRequests(filters?: TimesheetFilters, enabled = true) {
  return useQuery<TimesheetListResponse | undefined>({
    queryKey: ['partnerTimesheetEditRequests', filters],
    queryFn: () => {
      if (!filters) {
        return Promise.resolve(undefined);
      }
      return timesheetService.getTimesheets({
        ...filters,
        has_request_edit: filters.has_request_edit ?? true,
      });
    },
    enabled: enabled && !!filters,
  });
}

export function useApproveEditRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (requestId: number) => timesheetService.approveEditRequest(requestId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['timesheetEditRequests'] });
      queryClient.invalidateQueries({ queryKey: ['timesheets'] });
      toast('Thành công', { description: 'Yêu cầu đã được phê duyệt' });
    },
    onError: (error: unknown) => {
      toast('Lỗi', { description: `Không thể phê duyệt: ${getErrorMessage(error)}`, variant: 'destructive' });
    },
  });
}

export function useRejectEditRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (requestId: number) => timesheetService.rejectEditRequest(requestId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['timesheetEditRequests'] });
      queryClient.invalidateQueries({ queryKey: ['timesheets'] });
      toast('Thành công', { description: 'Yêu cầu đã bị từ chối' });
    },
    onError: (error: unknown) => {
      toast('Lỗi', { description: `Không thể từ chối: ${getErrorMessage(error)}`, variant: 'destructive' });
    },
  });
}

export function useCancelEditRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (timesheetId: number) => timesheetService.cancelEditRequest(timesheetId),
    onSuccess: async (response, timesheetId) => {
      await invalidateEditRequestCaches(queryClient, { timesheetId });

      // Use message from API response or fallback
      if (response.message) {
        showSuccessNotification(response.message);
      } else {
        showSuccessNotification('Yêu cầu chỉnh sửa đã được hủy thành công');
      }
    },
    // Error handling is done globally in React Query, but we can add custom error handling here if needed
    onError: (error: unknown) => {
      // Global error handler will show the error, but we can add custom logic here
      console.error('Cancel edit request failed:', error);
    },
  });
}
