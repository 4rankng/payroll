import { useState, useCallback, useMemo } from 'react';
import { useEmployees, useEmployeeSearch } from '@/hooks/api/useEmployees';
import type { Employee } from '@/types/api/employee.types';
import { useDebounce } from '@/hooks/useDebounce';

export const usePartnerEmployeesData = (scope?: 'global') => {
  const [searchTerm, setSearchTerm] = useState('');
  const [projectId, setProjectId] = useState<number | null>(null);
  const [statusFilter, setStatusFilter] = useState<'working' | 'unassigned' | undefined>(undefined);
  const [month, setMonth] = useState<string | undefined>(undefined);
  const [fromDate, setFromDate] = useState<string | undefined>(undefined);
  const [toDate, setToDate] = useState<string | undefined>(undefined);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [sortBy, setSortBy] = useState('created_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  // Debounce search term to avoid excessive API calls
  const debouncedSearchTerm = useDebounce(searchTerm, 300);

  // Use search API when there's a search term, otherwise use regular employees API
  const shouldUseSearch = debouncedSearchTerm.trim().length > 0;

  // Use the existing employees hook which calls /api/v1/employees with pagination
  const { data: employeesData, isLoading: isLoadingEmployees, error: employeesError } = useEmployees({
    page,
    pageSize,
    sortBy,
    sortOrder,
    projectId: projectId || undefined,
    status: statusFilter,
    month,
    fromDate,
    toDate,
    scope,
  });

  const { data: searchData, isLoading: isSearching, error: searchError } = useEmployeeSearch(
    debouncedSearchTerm,
    {
      pageSize: 50,
      status: statusFilter,
      month,
      projectId: projectId || undefined,
      fromDate,
      toDate,
      sortBy,
      sortOrder,
      scope,
    }
  );

  // Use search data if searching, otherwise use regular data
  const employees = useMemo(() => {
    let rawData: Employee[] = [];
    if (shouldUseSearch) {
      rawData = searchData?.data || [];
    } else {
      rawData = employeesData?.data || [];
    }

    // Deduplicate by ID to prevent duplicate records
    const uniqueEmployees = rawData.reduce((acc: Employee[], current: Employee) => {
      const exists = acc.find(emp => emp.id === current.id);
      if (!exists) {
        acc.push(current);
      }
      return acc;
    }, []);

    return uniqueEmployees;
  }, [shouldUseSearch, searchData?.data, employeesData?.data]) as Employee[];

  // Get pagination info
  const pagination = useMemo(() => {
    if (shouldUseSearch) {
      return searchData?.pagination || { page: 1, pageSize: 50, totalPages: 1, totalRecords: 0 };
    }
    return employeesData?.pagination || { page: 1, pageSize: 20, totalPages: 1, totalRecords: 0 };
  }, [shouldUseSearch, searchData?.pagination, employeesData?.pagination]);
  
  const isLoading = shouldUseSearch ? isSearching : isLoadingEmployees;
  const error = shouldUseSearch ? searchError : employeesError;

  const searchEmployees = useCallback((term: string) => {
    setSearchTerm(term);
    setPage(1); // Reset to first page when searching
  }, []);

  const clearSearch = useCallback(() => {
    setSearchTerm('');
    setPage(1); // Reset to first page when clearing search
  }, []);

  const filterByProject = useCallback((projectId: number | null) => {
    setProjectId(projectId);
    setPage(1); // Reset to first page when filtering
  }, []);

  const clearProjectFilter = useCallback(() => {
    setProjectId(null);
    setPage(1); // Reset to first page when clearing filter
  }, []);

  const updateStatusFilter = useCallback((status: 'working' | 'unassigned' | undefined) => {
    setStatusFilter(status);
    setPage(1); // Reset to first page when filtering
  }, []);

  const updateMonth = useCallback((monthValue: string | undefined) => {
    setMonth(monthValue);
    setPage(1); // Reset to first page when filtering
  }, []);

  const clearAllFilters = useCallback(() => {
    setProjectId(null);
    setStatusFilter(undefined);
    setMonth(undefined);
    setSearchTerm('');
    setSortBy('created_at');
    setSortOrder('desc');
    setPage(1);
  }, []);

  const handlePageChange = useCallback((newPage: number) => {
    setPage(newPage);
  }, []);

  const handlePageSizeChange = useCallback((newPageSize: number) => {
    setPageSize(newPageSize);
    setPage(1); // Reset to first page when changing page size
  }, []);

  const updateFromDate = useCallback((date: string | undefined) => {
    setFromDate(date);
    setPage(1); // Reset to first page when filtering
  }, []);

  const updateToDate = useCallback((date: string | undefined) => {
    setToDate(date);
    setPage(1); // Reset to first page when filtering
  }, []);

  const clearDateFilters = useCallback(() => {
    setFromDate(undefined);
    setToDate(undefined);
    setPage(1); // Reset to first page when clearing filters
  }, []);

  const handleSortChange = useCallback((newSortBy: string, newSortOrder: 'asc' | 'desc') => {
    setSortBy(newSortBy);
    setSortOrder(newSortOrder);
    setPage(1); // Reset to first page when sorting
  }, []);

  return {
    employees,
    pagination,
    isLoading,
    error,
    searchEmployees,
    clearSearch,
    searchTerm,
    projectId,
    filterByProject,
    clearProjectFilter,
    // New filter controls
    statusFilter,
    month,
    fromDate,
    toDate,
    updateStatusFilter,
    updateMonth,
    updateFromDate,
    updateToDate,
    clearAllFilters,
    clearDateFilters,
    isSearching: shouldUseSearch && isSearching,
    handlePageChange,
    handlePageSizeChange,
    // Sort controls
    sortBy,
    sortOrder,
    handleSortChange,
  };
};
