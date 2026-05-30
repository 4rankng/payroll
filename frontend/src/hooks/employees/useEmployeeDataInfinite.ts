import { useMemo, useState, useCallback } from "react";
import {
  useEmployeesInfinite,
  useEmployeeSearch,
  useCreateEmployee,
  useUpdateEmployee,
  useDeleteEmployee,
} from "@/hooks/api/useEmployees";
import { useDebounce } from "@/hooks/useDebounce";
import type {
  Employee,
  EmployeeFilters,
  CreateEmployeeData,
  UpdateEmployeeData,
} from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";

export const useEmployeeDataInfinite = () => {
  const [filters, setFilters] = useState<Omit<EmployeeFilters, "page">>({
    pageSize: 20,
    sortBy: "created_at",
    sortOrder: "desc",
  });
  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearch = useDebounce(searchTerm, 300);

  const shouldUseSearch = debouncedSearch.trim().length > 0;

  const infiniteQuery = useEmployeesInfinite(
    shouldUseSearch ? undefined : filters,
  );

  const searchFilters = {
    ...filters,
    search: debouncedSearch,
    pageSize: filters.pageSize || 20,
  };
  const { data: searchData, isLoading: isSearching } = useEmployeeSearch(
    debouncedSearch,
    searchFilters,
  );

  const employees = useMemo(() => {
    if (shouldUseSearch) return searchData?.data || [];
    if (!infiniteQuery.data?.pages) return [];
    return infiniteQuery.data.pages.flatMap((page) => page.data);
  }, [shouldUseSearch, searchData?.data, infiniteQuery.data?.pages]);

  const pagination = useMemo(() => {
    if (shouldUseSearch) return searchData?.pagination;
    const lastPage =
      infiniteQuery.data?.pages?.[infiniteQuery.data.pages.length - 1];
    return lastPage?.pagination;
  }, [shouldUseSearch, searchData?.pagination, infiniteQuery.data?.pages]);

  const updateFilters = useCallback((newFilters: Partial<EmployeeFilters>) => {
    setFilters((prev) => {
      const { page, ...rest } = newFilters;
      const hasNonPagination = Object.keys(rest).length > 0;
      return {
        ...prev,
        ...rest,
      };
    });
    if (Object.keys(newFilters).some((k) => k !== "page" && k !== "pageSize")) {
      setSearchTerm("");
    }
  }, []);

  const clearFilters = useCallback(() => {
    setFilters({ pageSize: 20, sortBy: "created_at", sortOrder: "desc" });
    setSearchTerm("");
  }, []);

  const searchEmployees = useCallback((term: string) => {
    setSearchTerm(term);
  }, []);

  return {
    employees,
    loading: (shouldUseSearch ? isSearching : infiniteQuery.isLoading) ?? false,
    searchEmployees,
    searchTerm,
    updateFilters,
    clearFilters,
    currentFilters: filters as EmployeeFilters,
    pagination,
    hasMore: shouldUseSearch ? false : (infiniteQuery.hasNextPage ?? false),
    isFetchingNextPage: infiniteQuery.isFetchingNextPage,
    fetchNextPage: infiniteQuery.fetchNextPage,
  };
};
