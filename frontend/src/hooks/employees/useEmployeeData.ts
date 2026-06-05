import { useState, useCallback, useMemo } from 'react';
import {
  useEmployees,
  useEmployeeSearch,
  useCreateEmployee,
  useUpdateEmployee,
  useDeleteEmployee
} from '@/hooks/api/useEmployees';
import { Employee, CreateEmployeeData, UpdateEmployeeData } from '@/types/api/employee.types';
import type { EmployeeFilters } from '@/types/api/employee.types';
import type { Bank } from '@/types/api/bank.types';
import { useDebounce } from '@/hooks/useDebounce';

export interface PaginationState {
  page: number;
  pageSize: number;
  totalPages: number;
  totalRecords: number;
}

export interface UseEmployeeDataReturn {
  employees: Employee[];
  loading: boolean;
  error: string | null;
  searchEmployees: (term: string) => void;
  searchTerm: string;
  updateFilters: (filters: Partial<EmployeeFilters>) => void;
  clearFilters: () => void;
  currentFilters: EmployeeFilters;
  pagination: PaginationState;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  onSortChange: (sortBy: string, sortOrder: 'asc' | 'desc') => void;
  bulkUpdateEmployeeStatus: (employeeIds: number[], status: string) => Promise<void>;
  deleteEmployee: (employeeId: number) => Promise<void>;
  createEmployee: (employee: CreateEmployeeData) => Promise<void>;
  updateEmployee: (employeeId: number, employee: UpdateEmployeeData, bankObject?: Bank | null) => Promise<void>;
  refreshData: () => void;
}

export const useEmployeeData = (): UseEmployeeDataReturn => {
  const [filters, setFilters] = useState<EmployeeFilters>({ page: 1, pageSize: 20, sortBy: 'created_at', sortOrder: 'desc' });
  const [searchTerm, setSearchTerm] = useState('');

  // Debounce search term to avoid excessive API calls
  const debouncedSearchTerm = useDebounce(searchTerm, 300);

  // Use search API when there's a search term, otherwise use regular employees API
  const shouldUseSearch = debouncedSearchTerm.trim().length > 0;

  // Combine filters with search term when searching
  const searchFilters = shouldUseSearch ? {
    ...filters,
    search: debouncedSearchTerm,
    pageSize: filters.pageSize || 20
  } : filters;

  const { data: employeesData, isLoading: isLoadingEmployees, error: employeesError, refetch: refetchEmployees } = useEmployees(
    shouldUseSearch ? undefined : searchFilters
  );

  const { data: searchData, isLoading: isSearching, error: searchError, refetch: refetchSearch } = useEmployeeSearch(
    debouncedSearchTerm,
    filters
  );

  // Extract pagination from API response
  const pagination = useMemo((): PaginationState => {
    const apiPagination = shouldUseSearch
      ? (searchData as { pagination?: PaginationState })?.pagination
      : (employeesData as { pagination?: PaginationState })?.pagination;

    return {
      page: apiPagination?.page || filters.page || 1,
      pageSize: apiPagination?.pageSize || filters.pageSize || 20,
      totalPages: apiPagination?.totalPages || 1,
      totalRecords: apiPagination?.totalRecords || 0
    };
  }, [shouldUseSearch, searchData, employeesData, filters]);
  
  const createMutation = useCreateEmployee();
  const updateMutation = useUpdateEmployee();
  const deleteMutation = useDeleteEmployee();

  // Use search data if searching, otherwise use regular data
  const employees = useMemo(() => {
    if (shouldUseSearch) {
      return searchData?.data || [];
    }
    return employeesData?.data || [];
  }, [shouldUseSearch, searchData?.data, employeesData?.data]);
  
  // Only use isLoadingEmployees (initial page load) for the full-page skeleton.
  // isSearching must NOT be included here — it is true on every new search query key,
  // which causes the entire page to unmount and remount the skeleton on each keystroke.
  const loading = (!shouldUseSearch && isLoadingEmployees) ||
                  createMutation.isPending ||
                  updateMutation.isPending ||
                  deleteMutation.isPending;
                  
  const error = (shouldUseSearch ? searchError : employeesError) ? 'Failed to load employees' : null;

  const searchEmployees = useCallback((term: string) => {
    setSearchTerm(term);
    // Reset to page 1 when searching
    setFilters(prev => ({ ...prev, page: 1 }));
    // Empty deps is correct: setSearchTerm and setFilters are stable useState setters
  }, []);

  const updateFilters = useCallback((newFilters: Partial<EmployeeFilters>) => {
    setFilters(prev => {
      // Reset to page 1 when filters change (except for page/pageSize changes)
      const { page, pageSize, ...otherNewFilters } = newFilters;
      const hasNonPaginationFilters = Object.keys(otherNewFilters).length > 0;
      return {
        ...prev,
        ...newFilters,
        ...(hasNonPaginationFilters ? { page: 1 } : {})
      };
    });
    // Clear search when applying filters
    if (Object.keys(newFilters).length > 0 && !newFilters.page && !newFilters.pageSize) {
      setSearchTerm('');
    }
  }, []);

  const clearFilters = useCallback(() => {
    setFilters({ page: 1, pageSize: 20 });
    setSearchTerm('');
  }, []);

  const onPageChange = useCallback((page: number) => {
    setFilters(prev => ({ ...prev, page }));
  }, []);

  const onPageSizeChange = useCallback((pageSize: number) => {
    setFilters(prev => ({ ...prev, pageSize, page: 1 }));
  }, []);

  const onSortChange = useCallback((sortBy: string, sortOrder: 'asc' | 'desc') => {
    setFilters(prev => ({ ...prev, sortBy, sortOrder, page: 1 }));
  }, []);

  const bulkUpdateEmployeeStatus = useCallback(async (employeeIds: number[], status: string) => {
    // This would need to be implemented in the API
    // For now, this is a placeholder - actual implementation would depend on backend API
    console.warn('Bulk update not implemented yet. IDs:', employeeIds, 'Status:', status);
  }, []);

  const deleteEmployee = useCallback(async (employeeId: number) => {
    await deleteMutation.mutateAsync(employeeId);
  }, [deleteMutation]);

  const createEmployee = useCallback(async (employeeData: CreateEmployeeData) => {
    await createMutation.mutateAsync(employeeData);
  }, [createMutation]);

  const updateEmployee = useCallback(async (employeeId: number, employeeData: UpdateEmployeeData, bankObject?: Bank | null): Promise<void> => {
    await updateMutation.mutateAsync({ id: employeeId, data: employeeData, bank: bankObject });
  }, [updateMutation]);

  const refreshData = useCallback(() => {
    if (shouldUseSearch) {
      refetchSearch();
    } else {
      refetchEmployees();
    }
  }, [shouldUseSearch, refetchSearch, refetchEmployees]);

  return {
    employees,
    loading,
    error,
    searchEmployees,
    searchTerm,
    updateFilters,
    clearFilters,
    currentFilters: filters,
    pagination,
    onPageChange,
    onPageSizeChange,
    onSortChange,
    bulkUpdateEmployeeStatus,
    deleteEmployee,
    createEmployee,
    updateEmployee,
    refreshData,
  };
};