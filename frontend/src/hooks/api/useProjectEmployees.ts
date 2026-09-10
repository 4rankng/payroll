import { useInfiniteQuery, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { projectEmployeeService } from '@/services/api/project-employee.service';
import { showSuccessNotification } from '@/utils/error-handler';
import {
  projectEmployeesKey,
  projectDetailKey,
  employeeProjectsKey,
  employeeAssignmentCheckKey,
  projectAssignmentStatsKey,
  employeeAssignmentStatsKey,
  pendingScheduleChangesKey,
  shouldInvalidateForProjectEmployeeChange,
  checkInConfigurationKey,
  checkInConfigurationInfiniteKey,
  checkInConfigurableProjectsKey,
} from '@/lib/queryKeys';
import type {
  ProjectEmployeeListParams,
  EmployeeProjectListParams,
  AssignEmployeeRequest,
  RemoveEmployeeRequest,
  ProjectEmployeeListResponse,
  AssignEmployeeResponse,
  RemoveEmployeeResponse,
  ChangePaymentScheduleRequest,
  ChangePaymentScheduleResponse,
  CancelScheduleChangeResponse,
  PendingScheduleChangesResponse,
  CheckInConfigurationParams,
} from '@/types/api/project-employee.types';

// ========== PROJECT EMPLOYEE QUERIES ==========

export function useCheckInConfigurableProjects() {
  return useQuery({
    queryKey: checkInConfigurableProjectsKey(),
    queryFn: () => projectEmployeeService.getCheckInConfigurableProjects(),
    retry: false,
  });
}

export function useProjectEmployees(
  projectId: number,
  params: ProjectEmployeeListParams = {},
  enabled = true
) {
  return useQuery({
    queryKey: projectEmployeesKey(projectId, params),
    queryFn: () => projectEmployeeService.getProjectEmployees(projectId, params),
    enabled,
    // Removed explicit staleTime and keepPreviousData - using global defaults
  });
}

export function useActiveProjectEmployees(projectId: number, enabled = true) {
  return useProjectEmployees(
    projectId,
    { status: 'current', sortBy: 'start_date', sortOrder: 'desc' },
    enabled
  );
}

export function useCheckInConfiguration(
  projectId: number,
  params: CheckInConfigurationParams,
  enabled = true,
) {
  return useQuery({
    queryKey: checkInConfigurationKey(projectId, params),
    queryFn: () => projectEmployeeService.getCheckInConfiguration(projectId, params),
    enabled,
    placeholderData: undefined,
  });
}

export function useInfiniteCheckInConfiguration(
  projectId: number,
  params: Omit<CheckInConfigurationParams, 'page'>,
  enabled = true,
) {
  return useInfiniteQuery({
    queryKey: checkInConfigurationInfiniteKey(projectId, params),
    queryFn: ({ pageParam }) => projectEmployeeService.getCheckInConfiguration(projectId, {
      ...params,
      page: pageParam,
    }),
    initialPageParam: 1,
    getNextPageParam: (lastPage) => (
      lastPage.pagination.page < lastPage.pagination.totalPages
        ? lastPage.pagination.page + 1
        : undefined
    ),
    enabled,
  });
}

export function useEmployeeProjects(
  employeeId: number,
  params: EmployeeProjectListParams = {},
  enabled = true
) {
  return useQuery({
    queryKey: employeeProjectsKey(employeeId, params),
    queryFn: () => projectEmployeeService.getEmployeeProjects(employeeId, params),
    enabled,
    // Removed explicit staleTime and keepPreviousData - using global defaults
  });
}

export function useActiveEmployeeProjects(employeeId: number, enabled = true) {
  return useEmployeeProjects(
    employeeId,
    { status: 'active', sortBy: 'start_date', sortOrder: 'desc' },
    enabled
  );
}


// ========== ASSIGNMENT MUTATIONS ==========

export function useAssignEmployeeToProject() {
  const queryClient = useQueryClient();

  return useMutation<
    AssignEmployeeResponse,
    Error,
    { projectId: number; data: AssignEmployeeRequest }
  >({
    mutationFn: ({ projectId, data }) =>
      projectEmployeeService.assignEmployeeToProject(projectId, data),

    onSuccess: (serverResponse, { projectId }) => {
      // Update cache with authoritative server response only
      if (serverResponse.data && Array.isArray(serverResponse.data)) {
        // Update the main project employees cache
        queryClient.setQueryData(
          projectEmployeesKey(projectId, { pageSize: 100 }),
          (old: ProjectEmployeeListResponse | undefined) => {
            if (!old) {
              return {
                data: serverResponse.data,
                pagination: {
                  page: 1,
                  pageSize: 100,
                  totalRecords: serverResponse.data.length,
                  totalPages: 1
                },
                message: serverResponse.message
              };
            }

            // Add new assignments to existing data, avoiding duplicates
            const existingEmployees = old.data;
            const updatedEmployees = [...existingEmployees];

            serverResponse.data.forEach((newAssignment) => {
              const exists = existingEmployees.some(
                emp => emp.employee_id === newAssignment.employee_id
              );
              if (!exists) {
                updatedEmployees.push(newAssignment);
              }
            });

            return {
              ...old,
              data: updatedEmployees,
              pagination: {
                ...old.pagination,
                totalRecords: updatedEmployees.length
              }
            };
          }
        );
      }
    },

    onSettled: (_, __, { projectId }) => {
      // Ensure all related queries are fresh
      queryClient.invalidateQueries({
        predicate: (query) => shouldInvalidateForProjectEmployeeChange(
          query.queryKey,
          projectId
        )
      });

      // Also invalidate employee list queries to update employee tables
      queryClient.invalidateQueries({
        queryKey: ['employees', 'list']
      });
      // Invalidate search queries that might include assigned employees
      queryClient.invalidateQueries({
        queryKey: ['employees', 'search']
      });
    },
  });
}



export function useUpdateEmployeeProjectAssignment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      employeeId,
      data
    }: {
      employeeId: number;
      data: {
        project_id: number;
        assignment_id?: number;
        position?: string;
        start_date?: string;
        // Empty string clears the end date; null is a silent no-op server-side.
        end_date?: string;
      }
    }) => projectEmployeeService.updateEmployeeProjectAssignment(employeeId, data),
    onSuccess: (updatedAssignment, { employeeId, data }) => {
      // Invalidate employee projects cache
      queryClient.invalidateQueries({
        queryKey: employeeProjectsKey(employeeId)
      });

      // Invalidate project employees cache if we have the project ID
      if (data.project_id) {
        queryClient.invalidateQueries({
          predicate: (query) => shouldInvalidateForProjectEmployeeChange(
            query.queryKey,
            data.project_id,
            [employeeId]
          )
        });
      }

      // Invalidate individual employee data to refresh current_projects
      queryClient.invalidateQueries({
        queryKey: ['employees', employeeId]
      });

      // Also invalidate employee list queries to update employee tables
      queryClient.invalidateQueries({
        queryKey: ['employees', 'list']
      });
      // Invalidate search queries that might include this employee
      queryClient.invalidateQueries({
        queryKey: ['employees', 'search']
      });
    },
  });
}

export function useRemoveEmployeesFromProject() {
  const queryClient = useQueryClient();

  return useMutation<
    RemoveEmployeeResponse,
    Error,
    { projectId: number; data: RemoveEmployeeRequest[] }
  >({
    mutationFn: ({ projectId, data }) =>
      projectEmployeeService.removeEmployeesFromProject(projectId, data),

    onMutate: async ({ projectId }) => {
      // Cancel outgoing refetches for project employees
      await queryClient.cancelQueries({
        queryKey: projectEmployeesKey(projectId, { pageSize: 100 })
      });

      // No optimistic updates - wait for API response
      return {};
    },

    onSuccess: (_, { projectId }) => {
      // Invalidate cache to refresh with server data
      queryClient.invalidateQueries({
        queryKey: projectEmployeesKey(projectId, { pageSize: 100 })
      });
    },

    onSettled: (_, __, { projectId, data: removeRequests }) => {
      const removedEmployeeIds = removeRequests.map(req => req.employee_id);

      // Ensure all related queries are fresh
      queryClient.invalidateQueries({
        predicate: (query) => shouldInvalidateForProjectEmployeeChange(
          query.queryKey,
          projectId,
          removedEmployeeIds
        )
      });

      // Invalidate individual employee detail caches to update current_projects
      removedEmployeeIds.forEach(employeeId => {
        queryClient.invalidateQueries({
          queryKey: ['employees', 'detail', employeeId]
        });
        // Also invalidate the older pattern for backward compatibility
        queryClient.invalidateQueries({
          queryKey: ['employees', employeeId]
        });
      });

      // Also invalidate employee list queries to update employee tables
      queryClient.invalidateQueries({
        queryKey: ['employees', 'list']
      });
      // Invalidate search queries that might include removed employees
      queryClient.invalidateQueries({
        queryKey: ['employees', 'search']
      });
    },
  });
}


// ========== PAYMENT SCHEDULE MUTATIONS ==========

/**
 * Hook to change payment schedule for an assignment
 */
export function useChangePaymentSchedule() {
  const queryClient = useQueryClient();

  return useMutation<
    ChangePaymentScheduleResponse,
    Error,
    { assignmentId: number; data: ChangePaymentScheduleRequest }
  >({
    mutationFn: ({ assignmentId, data }) =>
      projectEmployeeService.changePaymentSchedule(assignmentId, data),

    onSuccess: () => {
      // Backend returns data:null for this in-place mutation; the mutationFn resolves
      // to null, so we must not dereference it. Show a fixed success message (mirrors
      // useCancelScheduleChange) and let the invalidations below refresh the UI.
      showSuccessNotification('Cập nhật chu kỳ thanh toán thành công');

      // Invalidate all queries that might be affected by this change
      queryClient.invalidateQueries({
        queryKey: ['project-employees']
      });
      queryClient.invalidateQueries({
        queryKey: ['projects']
      });
      queryClient.invalidateQueries({
        queryKey: ['employees']
      });
      queryClient.invalidateQueries({
        queryKey: pendingScheduleChangesKey()
      });
    },
  });
}

/**
 * Hook to cancel pending payment schedule change
 */
export function useCancelScheduleChange() {
  const queryClient = useQueryClient();

  return useMutation<
    CancelScheduleChangeResponse,
    Error,
    { assignmentId: number }
  >({
    mutationFn: ({ assignmentId }) =>
      projectEmployeeService.cancelScheduleChange(assignmentId),

    onSuccess: (response) => {
      showSuccessNotification('Đã hủy thay đổi chu kỳ thanh toán');

      // Invalidate all queries that might be affected by this change
      queryClient.invalidateQueries({
        queryKey: ['project-employees']
      });
      queryClient.invalidateQueries({
        queryKey: ['projects']
      });
      queryClient.invalidateQueries({
        queryKey: ['employees']
      });
      queryClient.invalidateQueries({
        queryKey: pendingScheduleChangesKey()
      });
    },
  });
}

/**
 * Hook to get pending payment schedule changes (ADMIN only)
 */
export function usePendingScheduleChanges(enabled = true) {
  return useQuery<PendingScheduleChangesResponse>({
    queryKey: pendingScheduleChangesKey(),
    queryFn: () => projectEmployeeService.getPendingScheduleChanges(),
    enabled,
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

// ========== CONVENIENCE HOOKS ==========

/**
 * Hook to check if an employee is assigned to a project
 */
export function useIsEmployeeAssigned(employeeId: number, projectId: number) {
  return useQuery({
    queryKey: employeeAssignmentCheckKey(employeeId, projectId),
    queryFn: () => projectEmployeeService.isEmployeeAssignedToProject(employeeId, projectId),
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook for assignment statistics
 */
export function useProjectAssignmentStats(projectId: number, enabled = true) {
  return useQuery({
    queryKey: projectAssignmentStatsKey(projectId),
    queryFn: () => projectEmployeeService.getProjectAssignmentStats(projectId),
    enabled,
    gcTime: 15 * 60 * 1000, // 15 minutes
  });
}

export function useEmployeeAssignmentStats(employeeId: number, enabled = true) {
  return useQuery({
    queryKey: employeeAssignmentStatsKey(employeeId),
    queryFn: () => projectEmployeeService.getEmployeeAssignmentStats(employeeId),
    enabled,
    gcTime: 15 * 60 * 1000, // 15 minutes
  });
}



/**
 * Hook for convenient employee assignment
 */
export function useAssignEmployee() {
  const assignMutation = useAssignEmployeeToProject();

  const assignEmployee = async (
    projectId: number,
    employeeId: number,
    options: {
      employee_code?: string;
      start_date: string;
      end_date?: string;
    }
  ) => {
    // Proceed with assignment (validation removed since endpoint doesn't exist)
    return assignMutation.mutateAsync({
      projectId,
      data: {
        employee_id: employeeId,
        employee_code: options.employee_code,
        start_date: options.start_date,
        // Note: end_date is not supported in AssignEmployeeRequest
      },
    });
  };

  return {
    assignEmployee,
    isAssigning: assignMutation.isPending,
    isLoading: assignMutation.isPending,
    error: assignMutation.error,
  };
}

export function useToggleCheckInEnabled() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectId,
      employeeId,
      enabled,
    }: {
      projectId: number;
      employeeId: number;
      enabled: boolean;
    }) =>
      projectEmployeeService.toggleCheckInEnabled(
        projectId,
        employeeId,
        enabled
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        predicate: (query) =>
          shouldInvalidateForProjectEmployeeChange(
            query.queryKey,
            variables.projectId,
            [variables.employeeId]
          ),
      });
      queryClient.invalidateQueries({
        queryKey: projectAssignmentStatsKey(variables.projectId),
      });
      showSuccessNotification(
        variables.enabled
          ? "Đã bật điểm danh cho nhân viên"
          : "Đã tắt điểm danh cho nhân viên"
      );
    },
  });
}

export function useCancelPendingCheckInEnable() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectId,
      employeeId,
    }: {
      projectId: number;
      employeeId: number;
    }) =>
      projectEmployeeService.cancelPendingCheckInEnable(projectId, employeeId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        predicate: (query) =>
          shouldInvalidateForProjectEmployeeChange(
            query.queryKey,
            variables.projectId,
            [variables.employeeId]
          ),
      });
      queryClient.invalidateQueries({
        queryKey: projectAssignmentStatsKey(variables.projectId),
      });
      showSuccessNotification("Đã hủy yêu cầu bật điểm danh đang chờ kích hoạt");
    },
  });
}

export function useBulkToggleCheckInEnabled() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectId,
      employeeIds,
      enabled,
    }: {
      projectId: number;
      employeeIds: number[];
      enabled: boolean;
    }) =>
      projectEmployeeService.bulkToggleCheckInEnabled(
        projectId,
        employeeIds,
        enabled
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        predicate: (query) =>
          shouldInvalidateForProjectEmployeeChange(
            query.queryKey,
            variables.projectId,
            variables.employeeIds
          ),
      });
      queryClient.invalidateQueries({
        queryKey: projectAssignmentStatsKey(variables.projectId),
      });
      showSuccessNotification(
        variables.enabled
          ? `Đã bật điểm danh cho ${variables.employeeIds.length} nhân viên`
          : `Đã tắt điểm danh cho ${variables.employeeIds.length} nhân viên`
      );
    },
  });
}

export function useDisableInactiveCheckInEmployees() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId }: { projectId: number }) =>
      projectEmployeeService.disableInactiveCheckInEmployees(projectId),
    onSuccess: (result, variables) => {
      queryClient.invalidateQueries({
        predicate: (query) =>
          shouldInvalidateForProjectEmployeeChange(
            query.queryKey,
            variables.projectId,
          ),
      });
      showSuccessNotification(
        result.disabled_count > 0
          ? `Đã tắt điểm danh cho ${result.disabled_count} nhân viên chưa điểm danh`
          : "Không còn nhân viên chưa điểm danh cần tắt",
      );
    },
  });
}

export function useDisablePendingCheckInEmployees() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId }: { projectId: number }) =>
      projectEmployeeService.disablePendingCheckInEmployees(projectId),
    onSuccess: (result, variables) => {
      queryClient.invalidateQueries({
        predicate: (query) =>
          shouldInvalidateForProjectEmployeeChange(
            query.queryKey,
            variables.projectId,
          ),
      });
      showSuccessNotification(
        result.disabled_count > 0
          ? `Đã hủy chờ kích hoạt cho ${result.disabled_count} nhân viên`
          : "Không còn nhân viên chờ kích hoạt",
      );
    },
  });
}
