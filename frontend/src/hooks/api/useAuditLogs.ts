import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { apiClient } from '@/services/api/client';
import { QueryKeys } from '@/lib/queryKeys';
import type {
  BackendAuditLog,
  AuditLogsQueryParams,
} from '@/types/api/audit.types';

const BASE = '/audit';
const PAGE_SIZE = 30;

function buildAuditQueryString(params: AuditLogsQueryParams & { page?: number }): string {
  const parts: string[] = [];
  if (params.page)      parts.push(`page=${params.page}`);
  if (params.pageSize)  parts.push(`pageSize=${params.pageSize}`);
  if (params.userId)    parts.push(`userId=${params.userId}`);
  if (params.ipAddress) parts.push(`ipAddress=${encodeURIComponent(params.ipAddress)}`);
  if (params.fromDate)  parts.push(`fromDate=${encodeURIComponent(params.fromDate)}`);
  if (params.toDate)    parts.push(`toDate=${encodeURIComponent(params.toDate)}`);
  if (params.sortBy)    parts.push(`sortBy=${params.sortBy}`);
  if (params.sortOrder) parts.push(`sortOrder=${params.sortOrder}`);
  params.action?.forEach((a) => parts.push(`action=${encodeURIComponent(a)}`));
  params.entityType?.forEach((et) => parts.push(`entityType=${encodeURIComponent(et)}`));
  return parts.length ? `?${parts.join('&')}` : '';
}

export function useInfiniteAuditLogs(filters: Omit<AuditLogsQueryParams, 'page' | 'pageSize'> = {}) {
  return useInfiniteQuery({
    queryKey: QueryKeys.audit.logs({ ...filters, infinite: true } as Record<string, unknown>),
    queryFn: async ({ pageParam = 1 }) => {
      const response = await apiClient.get<BackendAuditLog[]>(
        `${BASE}/logs${buildAuditQueryString({ ...filters, page: pageParam as number, pageSize: PAGE_SIZE })}`
      );
      return response;
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      if (!lastPage?.pagination) return undefined;
      const { page, totalPages } = lastPage.pagination;
      return page < totalPages ? page + 1 : undefined;
    },
  });
}

export function useAuditLogDetail(id: number | null) {
  return useQuery({
    queryKey: QueryKeys.audit.logDetail(id ?? 0),
    queryFn: async () => {
      const response = await apiClient.get<BackendAuditLog>(`${BASE}/logs/${id}`);
      return response.data!;
    },
    enabled: id !== null && id > 0,
  });
}
