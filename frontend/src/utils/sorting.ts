import type { SortingState } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';

/**
 * Convert TanStack Table SortingState to API sortBy/sortOrder params
 */
export function sortingToApiParams(
  sorting: SortingState,
  fieldMap?: Record<string, string>,
): { sortBy: string; sortOrder: 'asc' | 'desc' } {
  if (sorting.length === 0) {
    return { sortBy: 'created_at', sortOrder: 'desc' };
  }
  const { id, desc } = sorting[0];
  return { sortBy: fieldMap?.[id] ?? id, sortOrder: desc ? 'desc' : 'asc' };
}

/**
 * Convert API sortBy/sortOrder to TanStack Table SortingState
 */
export function apiParamsToSorting(sortBy: string, sortOrder: 'asc' | 'desc'): SortingState {
  return [{ id: sortBy, desc: sortOrder === 'desc' }];
}

/**
 * Hook to sync TanStack Table sorting with API sort params
 */
export function useTableSorting(
  sortBy: string,
  sortOrder: 'asc' | 'desc',
  onSortChange: (sortBy: string, sortOrder: 'asc' | 'desc') => void,
  fieldMap?: Record<string, string>,
) {
  const sorting = useMemo(() => apiParamsToSorting(sortBy, sortOrder), [sortBy, sortOrder]);

  const handleSortingChange = useCallback(
    (updaterOrValue: SortingState | ((old: SortingState) => SortingState)) => {
      const newSorting = typeof updaterOrValue === 'function'
        ? updaterOrValue(sorting)
        : updaterOrValue;
      const params = sortingToApiParams(newSorting, fieldMap);
      onSortChange(params.sortBy, params.sortOrder);
    },
    [onSortChange, sorting, fieldMap],
  );

  return { sorting, onSortingChange: handleSortingChange };
}
