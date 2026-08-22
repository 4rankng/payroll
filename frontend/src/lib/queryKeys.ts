import type {
  ProjectEmployeeListParams,
  EmployeeProjectListParams,
  CheckInConfigurationParams,
} from '@/types/api/project-employee.types';

// Re-export QueryKeys from the centralized queryKeys/index.ts
export { QueryKeys, QueryKeyUtils } from '@/lib/queryKeys/index';

/**
 * Centralized query key builders for consistent cache operations
 * This ensures all related hooks use identical keys
 */

// ========== PROJECT EMPLOYEE KEYS ==========

/**
 * Consistent query key builder for project employees
 * Ensures cache operations use identical keys
 */
export function projectEmployeesKey(projectId: number, params: ProjectEmployeeListParams = {}) {
  // Normalize params to ensure consistent cache keys
  const normalizedParams = {
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 20,
    status: params.status ?? undefined,
    search: params.search?.trim() || undefined,
    check_in_enabled: params.check_in_enabled ?? undefined,
    sortBy: params.sortBy ?? undefined,
    sortOrder: params.sortOrder ?? undefined,
  };

  // Remove undefined values to keep keys clean
  const cleanParams = Object.fromEntries(
    Object.entries(normalizedParams).filter(([, value]) => value !== undefined)
  );

  return ['projects', projectId, 'employees', cleanParams] as const;
}

export function checkInConfigurationKey(
  projectId: number,
  params: CheckInConfigurationParams,
) {
  return [
    'projects',
    projectId,
    'employees',
    'check-in-configuration',
    {
      status: params.status,
      search: params.search?.trim() || undefined,
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 50,
    },
  ] as const;
}

export function checkInConfigurableProjectsKey() {
  return ['projects', 'check-in-configurable'] as const;
}

/**
 * Project detail query key
 */
export function projectDetailKey(projectId: number) {
  return ['projects', 'detail', projectId] as const;
}


/**
 * Employee projects query key
 */
export function employeeProjectsKey(employeeId: number, params: EmployeeProjectListParams = {}) {
  return ['employees', employeeId, 'projects', params] as const;
}


/**
 * Employee assignment check query key
 */
export function employeeAssignmentCheckKey(employeeId: number, projectId: number) {
  return ['employees', employeeId, 'assigned-to', projectId] as const;
}

/**
 * Project assignment statistics query key
 */
export function projectAssignmentStatsKey(projectId: number) {
  return ['projects', projectId, 'assignment-stats'] as const;
}

/**
 * Employee assignment statistics query key
 */
export function employeeAssignmentStatsKey(employeeId: number) {
  return ['employees', employeeId, 'assignment-stats'] as const;
}

/**
 * Pending payment schedule changes query key
 */
export function pendingScheduleChangesKey() {
  return ['project-employees', 'pending-schedule-changes'] as const;
}

export function bankTransferHistoriesKey(filters: Record<string, unknown>) {
  return ['payrolls', 'bank-transfer-histories', filters] as const;
}


// ========== QUERY KEY PREDICATES ==========

/**
 * Predicate function to match all project employee related queries
 */
export function isProjectEmployeeQuery(queryKey: readonly unknown[], projectId: number) {
  return (
    queryKey[0] === 'projects' &&
    queryKey[1] === projectId &&
    queryKey[2] === 'employees'
  );
}

/**
 * Predicate function to match project detail queries
 */
export function isProjectDetailQuery(queryKey: readonly unknown[], projectId: number) {
  return (
    queryKey[0] === 'projects' &&
    queryKey[1] === 'detail' &&
    queryKey[2] === projectId
  );
}


/**
 * Predicate function to match employee detail queries
 */
export function isEmployeeDetailQuery(queryKey: readonly unknown[], employeeIds: number[]) {
  return (
    queryKey[0] === 'employees' &&
    queryKey[1] === 'detail' &&
    employeeIds.includes(queryKey[2] as number)
  );
}

/**
 * Comprehensive predicate for project employee related invalidation
 */
export function shouldInvalidateForProjectEmployeeChange(
  queryKey: readonly unknown[],
  projectId: number,
  employeeIds: number[] = []
) {
  return (
    isProjectEmployeeQuery(queryKey, projectId) ||
    isProjectDetailQuery(queryKey, projectId) ||
    (employeeIds.length > 0 && isEmployeeDetailQuery(queryKey, employeeIds))
  );
}
