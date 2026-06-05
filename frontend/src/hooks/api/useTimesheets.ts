import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { timesheetService } from '@/services/api/timesheet.service';
import { SUCCESS_MESSAGES } from '@/config/constants';
import { showBulkOperationNotification, showErrorNotification, showSuccessNotification } from '@/utils/error-handler';
import type {
  CreateTimesheetData,
  UpdateTimesheetData,
  TimesheetFilters,
  BulkApproveData,
  BulkRejectData,
  BulkResetData,
  RejectTimesheetData,
  NewTimesheetEntry,
  BulkApproveByProjectData,
  BulkApproveByEmployeeData,
  Timesheet,
  TimesheetListResponse,
  TimesheetSummaryResponse,
  EmployeeTimesheetSummary,
  ListGroupedTimesheetsResponse,
} from '@/types/api/timesheet.types';
import { addItemToList, updateItemInList, removeItemFromList, updateSummaryCount, batchUpdateItemsInList } from '@/utils/cacheUpdates';
import { invalidateCache } from '@/lib/cache/invalidationService';
import { QueryKeys } from '@/lib/queryKeys';

// Helper function to normalize filters for query keys
// Removes undefined/null values to ensure consistent cache keys
const normalizeFilters = <T extends Record<string, unknown>>(filters?: T) => {
  if (!filters) return undefined;

  const normalized: Partial<T> = {};
  (Object.entries(filters) as [keyof T, T[keyof T]][]).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== ('' as unknown)) {
      normalized[key] = value;
    }
  });

  return Object.keys(normalized).length > 0 ? (normalized as T) : undefined;
};

// Get timesheet summary
export const useTimesheetSummary = (params?: {
  project_id?: number;
  employee_id?: number;
  fromDate?: string;
  toDate?: string;
}) => {
  return useQuery({
    queryKey: QueryKeys.timesheets.summary(normalizeFilters(params)),
    queryFn: () => timesheetService.getSummary(params),
  });
};

// Get paginated timesheets list
export const useTimesheets = (filters?: TimesheetFilters) => {
  return useQuery({
    queryKey: QueryKeys.timesheets.list(normalizeFilters(filters)),
    queryFn: () => timesheetService.getTimesheets(filters),
    staleTime: 0, // Always fetch fresh data when filters change
    placeholderData: undefined, // Don't show old data while fetching new filtered data
    refetchOnMount: true, // Refetch on component mount
  });
};

// Get paginated timesheets grouped by employee (for partner view)
// Uses server-side grouping to ensure consistent pagination by employee count
export const useGroupedTimesheets = (filters?: TimesheetFilters) => {
  return useQuery({
    queryKey: ['timesheets', 'grouped', normalizeFilters(filters)],
    queryFn: () => timesheetService.getGroupedTimesheets(filters),
    staleTime: 0,
    placeholderData: (prev) => prev, // keep previous data visible while new query loads
    refetchOnMount: true,
  });
};

// Get single timesheet
export const useTimesheet = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.timesheets.detail(id),
    queryFn: () => timesheetService.getTimesheetById(id),
    enabled,
  });
};

// Get timesheets by project and date
export const useTimesheetsByProjectAndDate = (projectId: number, date: string, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.timesheets.byProjectAndDate(projectId, date),
    queryFn: () => timesheetService.getTimesheetsByProjectAndDate(projectId, date),
    enabled: enabled && !!projectId && !!date,
  });
};

// Create timesheet
export const useCreateTimesheet = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateTimesheetData) => timesheetService.createTimesheet(data as unknown as { projectId: number; employeeId: number; date: string; hoursWorked: number; paytype: string; description?: string }),
    onSuccess: (newTimesheet) => {
      // Set the new timesheet in cache for detail views first
      queryClient.setQueryData(QueryKeys.timesheets.detail(newTimesheet.id), newTimesheet);

      // Use centralized invalidation service with proper context
      // Extract employee_id and project_id from the timesheet response
      invalidateCache('timesheet:create', {
        timesheetId: newTimesheet.id,
        data: {
          employee_id: newTimesheet.employee_id,
          project_id: newTimesheet.project_id,
        },
      });

      // Only show success notification if response.message exists from API
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Create multiple timesheets (bulk creation)
export const useCreateTimesheets = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (entries: NewTimesheetEntry[]) =>
      timesheetService.createTimesheets(entries),
    onSuccess: async (result, variables) => {
      // Aggressive cache invalidation to ensure immediate UI updates
      // This ensures both admin and partner pages with different filter patterns get updated

      try {


        // First, invalidate ALL timesheet-related queries with a broad pattern
        await queryClient.invalidateQueries({
          predicate: (query) => {
            const key = query.queryKey;
            return Array.isArray(key) && key[0] === 'timesheets';
          },
          refetchType: 'active' // Only refetch active queries for immediate UI update
        });

        // Also invalidate any employee-related timesheet summaries and current-projects (for calendar view)
        await queryClient.invalidateQueries({
          predicate: (query) => {
            const key = query.queryKey;
            return Array.isArray(key) &&
                   key[0] === 'employees' &&
                   key.some(part => typeof part === 'string' &&
                     (part.includes('timesheet') || part.includes('current-projects'))
                   );
          },
          refetchType: 'active'
        });

        // If we have created timesheets with specific projects/dates, invalidate those too
        if (variables && variables.length > 0) {
          const uniqueProjects = [...new Set(variables.map(v => v.projectId))];
          const uniqueDates = [...new Set(variables.map(v => v.date))];

          const projectDateInvalidations = uniqueProjects.flatMap(projectId =>
            uniqueDates.map(date =>
              queryClient.invalidateQueries({
                queryKey: QueryKeys.timesheets.byProjectAndDate(projectId, date)
              })
            )
          );

          await Promise.all(projectDateInvalidations);
        }

        // Force an immediate refetch of any currently visible timesheet lists
        // This is an additional safety measure to ensure UI updates
        await new Promise<void>((resolve) => {
          setTimeout(async () => {
            await queryClient.refetchQueries({
              predicate: (query) => {
                const key = query.queryKey;
                return Array.isArray(key) &&
                       key[0] === 'timesheets' &&
                       key[1] === 'list' &&
                       query.getObserversCount() > 0; // Only if there are active observers (visible components)
              }
            });
            resolve();
          }, 50); // Small delay to ensure invalidation completes first
        });


        showSuccessNotification(result?.message || `Đã lưu ${result?.data?.total_created ?? 0} bản ghi`);
      } catch (error) {
        console.error('❌ Primary cache invalidation failed:', error);
        showSuccessNotification(result?.message || `Đã lưu ${result?.data?.total_created ?? 0} bản ghi`);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Update timesheet
export const useUpdateTimesheet = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateTimesheetData }) =>
      timesheetService.updateTimesheet(id, data),
    onSuccess: (updatedTimesheet, { id }) => {
      // Optimistically update caches with server response
      updateItemInList(queryClient, QueryKeys.timesheets.lists(), updatedTimesheet as unknown as Record<string, unknown>);
      queryClient.setQueryData(QueryKeys.timesheets.detail(id), updatedTimesheet);

      // Use centralized invalidation service
      invalidateCache('timesheet:update', {
        timesheetId: id,
        data: {
          employee_id: updatedTimesheet.employee_id,
          project_id: updatedTimesheet.project_id,
        },
      });

      // Only show success notification if response has a message
      // Otherwise rely on backend message or don't show notification
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Delete timesheet
export const useDeleteTimesheet = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => timesheetService.deleteTimesheet(id),
    onMutate: async (id: number) => {
      // Get the timesheet data before deletion for context
      const timesheet = queryClient.getQueryData(QueryKeys.timesheets.detail(id)) as Timesheet | undefined;
      return { timesheet };
    },
    onSuccess: (_, deletedId, context) => {
      // Optimistically update caches
      removeItemFromList(queryClient, QueryKeys.timesheets.lists(), deletedId);
      queryClient.removeQueries({ queryKey: QueryKeys.timesheets.detail(deletedId) });

      // Use centralized invalidation service
      // If we have the timesheet context, use it for targeted invalidation
      invalidateCache('timesheet:delete', {
        timesheetId: deletedId,
        data: context?.timesheet ? {
          employee_id: context.timesheet.employee_id,
          project_id: context.timesheet.project_id,
        } : {},
      });

      // Only show success notification if response has a message
      // Otherwise rely on backend message or don't show notification
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Approve timesheet
export const useApproveTimesheet = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => timesheetService.approveTimesheet(id),
    onSuccess: (approvedTimesheet) => {
      // Optimistically update caches with server response
      updateItemInList(queryClient, QueryKeys.timesheets.lists(), approvedTimesheet as unknown as Record<string, unknown>);
      queryClient.setQueryData(QueryKeys.timesheets.detail(approvedTimesheet.id), approvedTimesheet);

      // Use centralized invalidation service
      invalidateCache('timesheet:approve', {
        timesheetId: approvedTimesheet.id,
        data: {
          employee_id: approvedTimesheet.employee_id,
          project_id: approvedTimesheet.project_id,
        },
      });

      // Only show success notification if response has a message
      // Otherwise rely on backend message or don't show notification
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Add timesheet to next payroll run (admin only)
export const useAddTimesheetToPayroll = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => timesheetService.addToPayroll(id),
    onSuccess: (result) => {
      // Invalidate timesheet queries so any payment-related states refresh
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      // Show backend-provided success message (Vietnamese)
      if (result?.message) {
        showSuccessNotification(result.message);
      }
    },
    // Global error handling will show backend message
  });
};

// Reject timesheet
export const useRejectTimesheet = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: RejectTimesheetData }) =>
      timesheetService.rejectTimesheet(id, data),
    onSuccess: (rejectedTimesheet) => {
      // Optimistically update caches with server response
      updateItemInList(queryClient, QueryKeys.timesheets.lists(), rejectedTimesheet as unknown as Record<string, unknown>);
      queryClient.setQueryData(QueryKeys.timesheets.detail(rejectedTimesheet.id), rejectedTimesheet);

      // Use centralized invalidation service
      invalidateCache('timesheet:reject', {
        timesheetId: rejectedTimesheet.id,
        data: {
          employee_id: rejectedTimesheet.employee_id,
          project_id: rejectedTimesheet.project_id,
        },
      });

      // Only show success notification if response has a message
      // Otherwise rely on backend message or don't show notification
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Bulk approve timesheets
export const useBulkApproveTimesheets = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: BulkApproveData) => timesheetService.bulkApprove(data),
    onSuccess: (result) => {
      // Invalidate all affected queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      showBulkOperationNotification(result, 'approve');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Approve all pending timesheets
export const useApproveAllTimesheets = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => timesheetService.approveAll(),
    onSuccess: (result) => {
      // Invalidate all affected queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      showBulkOperationNotification(result, 'approve');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Bulk reject timesheets
export const useBulkRejectTimesheets = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: BulkRejectData) => timesheetService.bulkReject(data),
    onSuccess: (result) => {
      // Invalidate all affected queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      showBulkOperationNotification(result, 'reject');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Reset single timesheet (using bulk endpoint with proper context)
export const useResetTimesheet = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (timesheet: { id: number; employee_id?: number; project_id?: number }) =>
      timesheetService.bulkReset({
        timesheet_ids: [timesheet.id]
      }),
    onSuccess: (result, variables) => {
      // Use centralized invalidation service with proper context for single timesheet
      invalidateCache('timesheet:reset', {
        timesheetId: variables.id,
        data: {
          employee_id: variables.employee_id,
          project_id: variables.project_id,
        }
      });

      showSuccessNotification('Timesheet đã được đặt lại thành công');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Bulk reset timesheets
export const useBulkResetTimesheets = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: BulkResetData) => timesheetService.bulkReset(data),
    onSuccess: (result, variables) => {
      // Use centralized invalidation service for better cache management
      invalidateCache('timesheet:bulkReset');

      showBulkOperationNotification(result, 'reset');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Bulk approve timesheets by project
export const useBulkApproveByProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, data }: { projectId: number; data: BulkApproveByProjectData }) =>
      timesheetService.bulkApproveByProject(projectId, data),
    onSuccess: (result) => {
      // Invalidate all affected queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      showBulkOperationNotification(result, 'approve');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Bulk approve timesheets by employee
export const useBulkApproveByEmployee = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ employeeId, data }: { employeeId: number; data: BulkApproveByEmployeeData }) =>
      timesheetService.bulkApproveByEmployee(employeeId, data),
    onSuccess: (result) => {
      // Invalidate all affected queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      showBulkOperationNotification(result, 'approve');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Import timesheets from Excel
export const useImportTimesheets = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      file,
      projectId,
      onProgress,
    }: {
      file: File;
      projectId?: number;
      onProgress?: (progress: number) => void;
    }) => timesheetService.importTimesheets(file, projectId, onProgress),
    onSuccess: (result) => {
      // Invalidate all timesheet queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      // Map the result to the expected format for bulk notification
      const mappedResult = {
        total_created: result.successful_records || 0,
        total_failed: result.failed_records || 0,
        errors: (result.errors || []).map((e: { row: number; field: string; message: string }) => `${e.field}: ${e.message}`)
      };
      showBulkOperationNotification(mappedResult, 'import');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Upload timesheet entries from Excel file
export const useUploadEntriesExcel = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      file,
      onProgress,
    }: {
      file: File;
      onProgress?: (progress: number) => void;
    }) => timesheetService.uploadEntriesExcel(file, onProgress),
    onSuccess: (result) => {
      // Invalidate all timesheet queries to reflect the newly imported entries
      queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });

      // Map the result to the expected format for bulk notification
      const mappedResult = {
        total_created: result.created_count || 0,
        total_failed: result.error_count || 0,
        errors: (result.failed_entries || []).map((entry) => `${entry.Error}`)
      };
      showBulkOperationNotification(mappedResult, 'import');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Export timesheets to Excel
export const useExportTimesheets = () => {

  return useMutation({
    mutationFn: (filters?: TimesheetFilters) =>
      timesheetService.exportTimesheets(filters),
    onSuccess: (result) => {
      // Automatically download the file
      if (result.download_url && result.filename) {
        timesheetService.downloadExport(result.download_url, result.filename);
      }
      showSuccessNotification('Export thành công');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Export timesheets to Excel with date range (direct download)
export const useExportTimesheetsExcel = () => {
  return useMutation({
    mutationFn: (params: {
      fromDate: string;
      toDate: string;
      project_ids?: string;
      employeeId?: number; // Using camelCase to match API
      status?: string;
    }) => timesheetService.exportTimesheetsExcel(params),
    onSuccess: () => {
      showSuccessNotification('Export thành công');
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Get employee timesheets using employee-specific endpoint
export const useEmployeeTimesheets = (
  employeeId: number,
  filters?: {
    page?: number;
    pageSize?: number;
    fromDate?: string;
    toDate?: string;
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
  },
  enabled = true
) => {
  return useQuery({
    queryKey: ['employees', employeeId, 'timesheets', filters],
    queryFn: () => timesheetService.getEmployeeTimesheets(employeeId, filters),
    enabled: enabled && !!employeeId,
  });
};

// Preview timesheets before creation
export const usePreviewTimesheets = () => {
  return useMutation({
    mutationFn: (entries: NewTimesheetEntry[]) =>
      timesheetService.previewTimesheets(entries),
    // No cache invalidation needed for preview - it's read-only
    // No success notification needed for preview - handled by UI
    // Error handling will show validation errors from backend
  });
};

// Get employee timesheet summary
export const useEmployeeTimesheetSummary = (
  employeeId: number,
  params?: {
    project_id?: number;
    fromDate?: string;
    toDate?: string;
  },
  enabled = true
) => {
  return useQuery<EmployeeTimesheetSummary>({
    queryKey: QueryKeys.timesheets.employeeSummary(employeeId, normalizeFilters(params)),
    queryFn: () => timesheetService.getEmployeeTimesheetSummary(employeeId, params),
    enabled,
  });
};

// getUploadHistory is not yet implemented on TimesheetService
// export const useUploadHistory = (params?: { project_id?: number; limit?: number; offset?: number }) => {
//   return useQuery({
//     queryKey: ['timesheet-upload-history', params],
//     queryFn: () => timesheetService.getUploadHistory(params),
//     placeholderData: keepPreviousData,
//   });
// };
