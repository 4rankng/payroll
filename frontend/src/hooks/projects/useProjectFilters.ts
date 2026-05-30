import { useState, useMemo, useCallback, useEffect } from "react";
import { Project, ProjectFilters } from "@/types/api/project.types";
import { useDebounce } from "@/hooks/useDebounce";

interface UseProjectFiltersProps {
  initialFilters?: Partial<ProjectFilters>;
}

export const useProjectFilters = ({
  initialFilters = {}
}: UseProjectFiltersProps) => {
  const [searchTerm, setSearchTerm] = useState(initialFilters.search || "");
  const [statusFilter, setStatusFilter] = useState<Project['status'][] | 'all'>(
    Array.isArray(initialFilters.status) ? initialFilters.status :
    initialFilters.status && initialFilters.status !== 'all' ? [initialFilters.status] : 'all'
  );
  const [monthFilter, setMonthFilter] = useState<string | undefined>(initialFilters.month);
  const [sortBy, setSortBy] = useState<string>(initialFilters.sortBy || 'created_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>(initialFilters.sortOrder || 'desc');
  const [page, setPageState] = useState(initialFilters.page || 1);
  const [pageSize, setPageSizeState] = useState(initialFilters.pageSize || 10);

  // Debounce search term to avoid too many API calls
  const debouncedSearchTerm = useDebounce(searchTerm, 300);

  // Reset to page 1 when filters change
  useEffect(() => {
    setPageState(1);
  }, [debouncedSearchTerm, statusFilter, monthFilter, sortBy, sortOrder]);

  // Build the filters object that will be sent to the API
  const apiFilters = useMemo<ProjectFilters>(() => {
    const filters: ProjectFilters = {
      page,
      pageSize,
      sortBy,
      sortOrder,
    };

    if (debouncedSearchTerm && debouncedSearchTerm.trim()) {
      filters.search = debouncedSearchTerm.trim();
    }

    // Handle status filter - API accepts array of statuses
    if (statusFilter && statusFilter !== 'all') {
      filters.status = statusFilter;
    }

    if (monthFilter) {
      filters.month = monthFilter;
    }

    return filters;
  }, [debouncedSearchTerm, statusFilter, monthFilter, sortBy, sortOrder, page, pageSize]);


  const hasFilters = searchTerm !== "" || statusFilter !== 'all' || monthFilter !== undefined;

  const clearFilters = useCallback(() => {
    setSearchTerm("");
    setStatusFilter('all');
    setMonthFilter(undefined);
    setSortBy('created_at');
    setSortOrder('desc');
    setPageState(1); // Reset to first page when clearing filters
  }, []);

  const setPage = useCallback((newPage: number) => {
    setPageState(newPage);
  }, []);

  const setPageSize = useCallback((newPageSize: number) => {
    setPageState(1); // Reset to first page when changing page size
    setPageSizeState(newPageSize);
  }, []);

  return {
    searchTerm,
    setSearchTerm,
    statusFilter,
    setStatusFilter,
    monthFilter,
    setMonthFilter,
    sortBy,
    setSortBy,
    sortOrder,
    setSortOrder,
    hasFilters,
    clearFilters,
    setPage,
    setPageSize,
    apiFilters,
    // Expose current pagination state
    page,
    pageSize,
  };
};