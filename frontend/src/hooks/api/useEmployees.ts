import {
  useMutation,
  useQuery,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { employeeService } from "@/services/api/employee.service";
import { useDebounce } from "@/hooks/useDebounce";
import { QueryKeys } from "@/lib/queryKeys";
import type {
  CreateEmployeeData,
  UpdateEmployeeData,
  EmployeeFilters,
  EmployeePayrollFilters,
  EmployeeProjectsResponse,
  Employee,
  EmployeeTimesheetFilters,
  EmployeesResponse,
  EmployeeSummary,
  EmployeeCurrentProjectsResponse,
  EmployeeCurrentProjectsFilters,
  GrantEmployeeAccessData,
} from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";
import {
  addItemToList,
  updateItemInList,
  removeItemFromList,
  updateSummaryCount,
} from "@/utils/cacheUpdates";
import {
  showErrorNotification,
  showSuccessNotification,
} from "@/utils/error-handler";
import { invalidateCache } from "@/lib/cache/invalidationService";
import { authManager } from "@/lib/auth";

// Get employees summary
export const useEmployeesSummary = () => {
  const userRole = authManager.getUserRole();
  const hasPermission = (userRole === 'admin' || userRole === 'partner') && userRole !== null;

  return useQuery({
    queryKey: QueryKeys.employees.summary(),
    queryFn: () => employeeService.getSummary(),
    enabled: hasPermission,
    retry: false,
  });
};

// Get paginated employees list
export const useEmployees = (filters?: EmployeeFilters) => {
  const userRole = authManager.getUserRole();
  const hasPermission = (userRole === 'admin' || userRole === 'partner') && userRole !== null;

  return useQuery({
    queryKey: QueryKeys.employees.list(filters),
    queryFn: () => employeeService.getEmployees(filters),
    enabled: hasPermission,
    retry: false,
  });
};

// Hook that returns both data and pagination
export const useEmployeesWithPagination = (filters?: EmployeeFilters) => {
  return useQuery<EmployeesResponse>({
    queryKey: QueryKeys.employees.list(filters),
    queryFn: () => employeeService.getEmployees(filters),
  });
};

// Get infinite paginated employees list (for infinite scroll)
export const useEmployeesInfinite = (
  filters?: Omit<EmployeeFilters, "page">,
) => {
  const pageSize = filters?.pageSize || 20;

  return useInfiniteQuery({
    queryKey: QueryKeys.employees.infiniteList(filters),
    queryFn: ({ pageParam = 1 }) =>
      employeeService.getEmployees({
        ...filters,
        page: pageParam,
        pageSize,
      }),
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      const { page, totalPages } = lastPage.pagination;
      return page < totalPages ? page + 1 : undefined;
    },
  });
};

// Get employees with missing bank details
export const useEmployeesWithMissingBankDetails = (
  filters?: EmployeeFilters,
) => {
  return useQuery({
    queryKey: QueryKeys.employees.missingBankDetails(filters),
    queryFn: () => employeeService.getEmployeesWithMissingBankDetails(filters),
  });
};

// Search employees with debounced search term (500ms delay)
export const useEmployeeSearch = (
  searchTerm: string,
  filters?: Partial<EmployeeFilters>,
) => {
  const debouncedSearchTerm = useDebounce(searchTerm, 500);

  const searchParams = {
    search: debouncedSearchTerm,
    ...(filters || {}),
  };

  return useQuery({
    queryKey: QueryKeys.employees.search(searchParams),
    queryFn: () => employeeService.searchEmployees(searchParams),
    enabled: debouncedSearchTerm.trim().length > 0, // Only search when there's a debounced search term
  });
};

// Get single employee
export const useEmployee = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.employees.detail(id),
    queryFn: () => employeeService.getEmployeeById(id),
    enabled,
  });
};

// Get individual employee summary
export const useEmployeeSummary = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.employees.employeeSummary(id),
    queryFn: () => employeeService.getEmployeeSummaryById(id),
    enabled,
  });
};

// Get employee projects
export const useEmployeeProjects = (
  id: number,
  params?: {
    status?: "active" | "ended" | "inactive";
    page?: number;
    pageSize?: number;
  },
  enabled = true,
) => {
  return useQuery({
    queryKey: QueryKeys.employees.projects(id, params),
    queryFn: () => employeeService.getEmployeeProjects(id, params),
    enabled,
  });
};

// Get employee current projects with timesheets
export const useEmployeeCurrentProjects = (
  id: number,
  filters?: EmployeeCurrentProjectsFilters,
  enabled = true,
) => {
  return useQuery({
    queryKey: QueryKeys.employees.currentProjects(id, filters),
    queryFn: () => employeeService.getEmployeeCurrentProjects(id, filters),
    enabled,
  });
};

// Get employee payroll history
export const useEmployeePayroll = (
  id: number,
  filters?: EmployeePayrollFilters,
  enabled = true,
) => {
  return useQuery({
    queryKey: QueryKeys.employees.payroll(id, filters),
    queryFn: () => employeeService.getEmployeePayroll(id, filters),
    enabled,
  });
};

// Get employee timesheet history
export const useEmployeeTimesheet = (
  id: number,
  filters?: EmployeeTimesheetFilters,
  enabled = true,
) => {
  return useQuery({
    queryKey: QueryKeys.employees.timesheet(
      id,
      filters as unknown as Parameters<typeof QueryKeys.employees.timesheet>[1]
    ),
    queryFn: () => employeeService.getEmployeeTimesheet(id, filters),
    enabled,
  });
};

// Create employee
export const useCreateEmployee = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateEmployeeData) =>
      employeeService.createEmployee(data),
    onSuccess: async (response) => {
      try {
        // Extract employee data from backend response
        const newEmployee = response.data!;

        if (!newEmployee) {
          console.error("No employee data in response:", response);
          throw new Error("No employee data received from server");
        }

        // Set the new employee in cache for detail views (optimistic UI)
        if (newEmployee.id) {
          queryClient.setQueryData(
            QueryKeys.employees.detail(newEmployee.id),
            newEmployee,
          );
        }

        // Use centralized cache invalidation system
        await invalidateCache("employee:create", {
          employeeId: newEmployee.id,
        });

        if (response.message) {
          showSuccessNotification(response.message);
        }
      } catch (error) {
        console.error("Error in onSuccess handler:", error);

        // Still invalidate queries to get fresh data even if cache update fails
        await invalidateCache("employee:create", {});

        // No fallback message - only show notification if response.message exists
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Update employee
export const useUpdateEmployee = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      data,
      bank,
    }: {
      id: number;
      data: UpdateEmployeeData;
      bank?: Bank | null;
    }) => employeeService.updateEmployee(id, data),
    onSuccess: (response, { id, bank: selectedBank }) => {
      const employeeData = response.data!;

      // If bank object is missing in response but we have bank data, add it
      if (!employeeData.bank && selectedBank) {
        employeeData.bank = selectedBank;
      }

      // Use server response to update caches directly
      updateItemInList(queryClient, QueryKeys.employees.list(), employeeData);

      // Update detail cache with server response
      queryClient.setQueryData(QueryKeys.employees.detail(id), employeeData);

      // Invalidate list queries to force refetch
      queryClient.invalidateQueries({ queryKey: QueryKeys.employees.lists() });

      // Invalidate the detail query to trigger a re-render
      queryClient.invalidateQueries({ queryKey: QueryKeys.employees.detail(id) });

      // Invalidate user queries — employee email is synced to the linked user account,
      // so the Users page must reflect the change (email shown in Người dùng detail panel)
      queryClient.invalidateQueries({ queryKey: QueryKeys.users.all });

      if (response.message) {
        showSuccessNotification(response.message);
      }

      return employeeData;
    },
    // Error handling is now done globally in React Query
  });
};

// Delete employee
export const useDeleteEmployee = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => employeeService.deleteEmployee(id),
    onSuccess: (response, deletedId) => {
      // Use server response to update caches directly
      removeItemFromList(queryClient, QueryKeys.employees.list(), deletedId);

      // Remove from detail cache completely
      queryClient.removeQueries({ queryKey: QueryKeys.employees.detail(deletedId) });

      // Update summary count
      updateSummaryCount(
        queryClient,
        QueryKeys.employees.summary(),
        "total_employees",
        -1,
      );

      // Invalidate list queries to force refetch
      queryClient.invalidateQueries({ queryKey: QueryKeys.employees.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Import employees from Excel
export const useImportEmployees = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      file,
      onProgress,
    }: {
      file: File;
      onProgress?: (progress: number) => void;
    }) => employeeService.importEmployees(file, onProgress),
    onSuccess: (result) => {
      // Invalidate all employee queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.employees.all });

      if (result.message) {
        showSuccessNotification(result.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Export employees to Excel
export const useExportEmployees = () => {
  return useMutation({
    mutationFn: (filters?: EmployeeFilters) =>
      employeeService.exportEmployees(filters),
    onSuccess: (response) => {
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Validate CCCD uniqueness
export const useValidateCCCD = () => {
  return useMutation({
    mutationFn: ({ cccd, excludeId }: { cccd: string; excludeId?: number }) =>
      employeeService.validateCCCD(cccd, excludeId),
  });
};

// Validate email uniqueness
export const useValidateEmail = () => {
  return useMutation({
    mutationFn: ({ email, excludeId }: { email: string; excludeId?: number }) =>
      employeeService.validateEmail(email, excludeId),
  });
};

// Get employee by CCCD
export const useEmployeeByCCCD = (cccd: string, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.employees.byCCCD(cccd),
    queryFn: () => employeeService.getEmployeeByCCCD(cccd),
    enabled: enabled && cccd.length > 0,
  });
};

// Change employee password (Admin/Partner only)
export const useChangeEmployeePassword = () => {
  return useMutation({
    mutationFn: (vars: { id: number; password: string }) =>
      employeeService.changeEmployeePassword(vars.id, { new_password: vars.password }),
    onSuccess: (response) => {
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Get users who have access to an employee
export const useEmployeeUsers = (employeeId: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.employees.users(employeeId),
    queryFn: () => employeeService.getEmployeeUsers(employeeId),
    enabled,
  });
};

// Grant access to an employee
export const useGrantEmployeeAccess = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      employeeId,
      data,
    }: {
      employeeId: number;
      data: GrantEmployeeAccessData;
    }) => employeeService.grantEmployeeAccess(employeeId, data),
    onSuccess: (response, { employeeId }) => {
      // Invalidate the employee users list
      queryClient.invalidateQueries({ queryKey: QueryKeys.employees.users(employeeId) });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Revoke employee access from a user
export const useRevokeEmployeeAccess = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      employeeId,
      userId,
    }: {
      employeeId: number;
      userId: number;
    }) => employeeService.revokeEmployeeAccess(employeeId, userId),
    onSuccess: (response, { employeeId }) => {
      // Invalidate the employee users list
      queryClient.invalidateQueries({ queryKey: QueryKeys.employees.users(employeeId) });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};
