import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query';
import { employeePortalService } from '@/services/api/employee-portal.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification } from '@/utils/error-handler';
import type {
  UpdateEmployeeProfileData,
  ChangePasswordData,
  EmployeeTimesheetFilters,
} from '@/types/api/auth.types';

// Re-export for backward compatibility
export const EMPLOYEE_QUERY_KEYS = {
  profile: ['employee', 'profile'] as const,
  timesheets: (filters?: EmployeeTimesheetFilters) =>
    ['employee', 'timesheets', filters] as const,
  summary: (weeks: number) =>
    ['employee', 'summary', weeks] as const,
};

/**
 * Hook to get employee's own profile
 */
export function useEmployeeProfile() {
  return useQuery({
    queryKey: EMPLOYEE_QUERY_KEYS.profile,
    queryFn: () => employeePortalService.getMyProfile(),
  });
}

/**
 * Hook to update employee's profile
 */
export function useUpdateEmployeeProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateEmployeeProfileData) =>
      employeePortalService.updateMyProfile(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: EMPLOYEE_QUERY_KEYS.profile });
      showSuccessNotification('Cập nhật thông tin thành công');
    },
  });
}

/**
 * Hook to update employee's password
 */
export function useUpdateEmployeePassword() {
  return useMutation({
    mutationFn: (data: ChangePasswordData) =>
      employeePortalService.updateMyPassword(data),
    onSuccess: () => {
      showSuccessNotification('Đổi mật khẩu thành công');
    },
  });
}

/**
 * Hook to get employee's timesheets with filters
 */
export function useEmployeeTimesheets(filters?: EmployeeTimesheetFilters) {
  return useQuery({
    queryKey: EMPLOYEE_QUERY_KEYS.timesheets(filters),
    queryFn: () => employeePortalService.getMyTimesheets(filters),
  });
}

/**
 * Infinite query variant for employee timesheets
 */
export function useEmployeeTimesheetsInfinite(
  filters?: Omit<EmployeeTimesheetFilters, 'page' | 'pageSize'>,
  pageSize: number = 50
) {
  return useInfiniteQuery({
    queryKey: EMPLOYEE_QUERY_KEYS.timesheets({ ...filters, pageSize }),
    initialPageParam: 1,
    queryFn: ({ pageParam }) =>
      employeePortalService.getMyTimesheets({ ...(filters || {}), page: pageParam as number, pageSize }),
    getNextPageParam: (lastPage) => {
      if (!lastPage?.pagination) return undefined;
      const { page, totalPages } = lastPage.pagination;
      return page < totalPages ? page + 1 : undefined;
    },
  });
}

/**
 * Hook to get employee's salary summary
 */
export function useEmployeeSummary(weeks: number = 4) {
  return useQuery({
    queryKey: EMPLOYEE_QUERY_KEYS.summary(weeks),
    queryFn: () => employeePortalService.getMySummary(weeks),
  });
}
