import { QueryClient, QueryKey } from '@tanstack/react-query';

// Generic interfaces for list responses
interface ListResponse<T> {
  data: T[];
  pagination?: {
    page: number;
    pageSize: number;
    total?: number;
    totalPages: number;
    totalRecords?: number;
  };
}

// API Response format (matches actual API structure)
interface ApiListResponse<T> {
  status: string;
  data: T[];
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

interface ApiResponse<T> {
  data: T;
  status: string;
  message: string;
}

/**
 * Add a new item to a list cache after CREATE operation
 */
export function addItemToList<T>(
  queryClient: QueryClient,
  queryKey: QueryKey,
  newItem: T
): void {
  queryClient.setQueryData(queryKey, (old: ListResponse<T> | ApiListResponse<T> | T[] | undefined) => {
    if (!old) return { data: [newItem] };

    // Handle array format
    if (Array.isArray(old)) {
      return [newItem, ...old];
    }

    // Handle API response format with status, data, pagination, message
    if ('status' in old && 'data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: [newItem, ...old.data],
        // Update pagination if present (increment total records)
        pagination: old.pagination ? {
          ...old.pagination,
          totalRecords: old.pagination.totalRecords + 1
        } : undefined
      };
    }

    // Handle simple ListResponse format
    if ('data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: [newItem, ...old.data]
      };
    }

    return old;
  });
}

/**
 * Update an existing item in a list cache after UPDATE operation
 */
export function updateItemInList<T extends Record<string, unknown>>(
  queryClient: QueryClient,
  queryKey: QueryKey,
  updatedItem: T,
  idField: keyof T = 'id'
): void {
  queryClient.setQueryData(queryKey, (old: ListResponse<T> | ApiListResponse<T> | T[] | undefined) => {
    if (!old) return old;

    // Handle array format
    if (Array.isArray(old)) {
      return old.map(item =>
        (item[idField] as unknown) === (updatedItem[idField] as unknown) ? updatedItem : item
      );
    }

    // Handle API response format
    if ('status' in old && 'data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: old.data.map(item =>
          item[idField] === updatedItem[idField] ? updatedItem : item
        )
      };
    }

    // Handle simple ListResponse format
    if ('data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: old.data.map(item =>
          item[idField] === updatedItem[idField] ? updatedItem : item
        )
      };
    }

    return old;
  });
}

/**
 * Remove an item from a list cache after DELETE operation
 */
export function removeItemFromList<T extends Record<string, unknown>>(
  queryClient: QueryClient,
  queryKey: QueryKey,
  itemId: number | string,
  idField: keyof T = 'id'
): void {
  queryClient.setQueryData(queryKey, (old: ListResponse<T> | ApiListResponse<T> | T[] | undefined) => {
    if (!old) return old;

    // Handle array format
    if (Array.isArray(old)) {
      return old.filter(item => item[idField] !== itemId);
    }

    // Handle API response format
    if ('status' in old && 'data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: old.data.filter(item => item[idField] !== itemId),
        // Update pagination if present (decrement total records)
        pagination: old.pagination ? {
          ...old.pagination,
          totalRecords: old.pagination.totalRecords - 1
        } : undefined
      };
    }

    // Handle simple ListResponse format
    if ('data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: old.data.filter(item => item[idField] !== itemId)
      };
    }

    return old;
  });
}

/**
 * Update a counter field in summary/stats data
 */
export function updateSummaryCount(
  queryClient: QueryClient,
  queryKey: QueryKey,
  field: string,
  delta: number
): void {
  queryClient.setQueryData(queryKey, (old: ApiResponse<unknown> | Record<string, unknown> | unknown) => {
    if (!old) return old;

    if (old && typeof old === 'object') {
      const record = old as Record<string, unknown>;
      if ('data' in record && record.data && typeof record.data === 'object') {
        return {
          ...record,
          data: {
            ...(record.data as Record<string, unknown>),
            [field]: Math.max(0, ((record.data as Record<string, number>)[field] || 0) + delta)
          }
        };
      }

      return {
        ...record,
        [field]: Math.max(0, ((record[field] as number) || 0) + delta)
      };
    }

    return old;
  });
}

/**
 * Batch update multiple items in a list cache
 */
export function batchUpdateItemsInList<T extends Record<string, unknown>>(
  queryClient: QueryClient,
  queryKey: QueryKey,
  updatedItems: T[],
  idField: keyof T = 'id'
): void {
  queryClient.setQueryData(queryKey, (old: ListResponse<T> | T[] | undefined) => {
    if (!old) return old;

    const updateMap = new Map(updatedItems.map(item => [item[idField], item]));

    // Handle array format
    if (Array.isArray(old)) {
      return old.map(item => updateMap.get(item[idField]) || item);
    }

    // Handle ListResponse format
    if ('data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: old.data.map(item => updateMap.get(item[idField]) || item)
      };
    }

    return old;
  });
}

/**
 * Batch remove multiple items from a list cache
 */
export function batchRemoveItemsFromList<T extends Record<string, unknown>>(
  queryClient: QueryClient,
  queryKey: QueryKey,
  itemIds: (number | string)[],
  idField: keyof T = 'id'
): void {
  queryClient.setQueryData(queryKey, (old: ListResponse<T> | T[] | undefined) => {
    if (!old) return old;

    const idsToRemove = new Set(itemIds);

    // Handle array format
    if (Array.isArray(old)) {
      return old.filter(item => !idsToRemove.has(item[idField] as unknown as string | number));
    }

    // Handle ListResponse format
    if ('data' in old && Array.isArray(old.data)) {
      return {
        ...old,
        data: old.data.filter(item => !idsToRemove.has(item[idField] as unknown as string | number))
      };
    }

    return old;
  });
}

/**
 * Smart invalidation - only invalidate queries with filters, sorting, or pagination
 */
export function invalidateFilteredQueries(
  queryClient: QueryClient,
  baseQueryKey: QueryKey
): void {
  queryClient.invalidateQueries({
    queryKey: baseQueryKey,
    predicate: (query) => {
      const queryKey = query.queryKey;

      // Check if query has additional parameters that might affect results
      return queryKey.some(key =>
        typeof key === 'object' && key !== null && (
          'sortBy' in key ||
          'filters' in key ||
          'search' in key ||
          'page' in key ||
          'status' in key ||
          'fromDate' in key ||
          'toDate' in key
        )
      );
    }
  });
}

/**
 * Update multiple summary fields at once
 */
export function updateSummaryFields(
  queryClient: QueryClient,
  queryKey: QueryKey,
  updates: Record<string, number>
): void {
  queryClient.setQueryData(queryKey, (old: ApiResponse<unknown> | Record<string, unknown> | unknown) => {
    if (!old) return old;

    const updatedFields: Record<string, number> = {};
    Object.entries(updates).forEach(([field, delta]) => {
      const oldObj = old as { data?: Record<string, number> } & Record<string, unknown>;
      const currentValue = (oldObj.data?.[field] as number) || (oldObj[field] as number) || 0;
      updatedFields[field] = Math.max(0, currentValue + delta);
    });

    if (old && typeof old === 'object') {
      const record = old as Record<string, unknown>;
      if ('data' in record && record.data && typeof record.data === 'object') {
        return {
          ...record,
          data: {
            ...record.data,
            ...updatedFields
          }
        };
      }

      return {
        ...record,
        ...updatedFields
      };
    }

    return old;
  });
}
