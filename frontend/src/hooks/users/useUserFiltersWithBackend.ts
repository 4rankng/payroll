import { useState, useMemo, useCallback } from 'react';
import type { UserFilters } from '@/services/api/user.service';

export const useUserFiltersWithBackend = () => {
  const [search, setSearch] = useState('');
  const [role, setRole] = useState<'admin' | 'partner' | 'employee' | undefined>(undefined);
  const [lastLoginToday, setLastLoginToday] = useState<boolean>(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [sortBy, setSortBy] = useState('created_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  const filters: UserFilters = useMemo(() => {
    const result: UserFilters = {
      page,
      pageSize,
      sortBy,
      sortOrder,
    };

    if (search.trim()) {
      result.search = search.trim();
    }

    if (role) {
      result.role = role;
    }

    if (lastLoginToday) {
      result.last_login_today = true;
    }

    return result;
  }, [search, role, lastLoginToday, page, pageSize, sortBy, sortOrder]);

  const clearFilters = () => {
    setSearch('');
    setRole(undefined);
    setLastLoginToday(false);
    setSortBy('created_at');
    setSortOrder('desc');
    setPage(1);
    setPageSize(20);
  };

  const onPageChange = useCallback((newPage: number) => {
    setPage(newPage);
  }, []);

  const onPageSizeChange = useCallback((newPageSize: number) => {
    setPageSize(newPageSize);
    setPage(1);
  }, []);

  const onSortChange = useCallback((newSortBy: string, newSortOrder: 'asc' | 'desc') => {
    setSortBy(newSortBy);
    setSortOrder(newSortOrder);
    setPage(1);
  }, []);

  // Wrap setters to reset page when filters change
  const handleSearchChange = useCallback((value: string) => {
    setSearch(value);
    setPage(1);
  }, []);

  const handleRoleChange = useCallback((value: 'admin' | 'partner' | 'employee' | undefined) => {
    setRole(value);
    setPage(1);
  }, []);

  const hasActiveFilters = search.trim() !== '' || role !== undefined || lastLoginToday;

  return {
    search,
    setSearch: handleSearchChange,
    role,
    setRole: handleRoleChange,
    lastLoginToday,
    setLastLoginToday,
    filters,
    clearFilters,
    hasActiveFilters,
    page,
    pageSize,
    onPageChange,
    onPageSizeChange,
    sortBy,
    sortOrder,
    onSortChange,
  };
};
