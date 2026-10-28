import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/services/api/client';
import { QueryKeys } from '@/lib/queryKeys';
import type { BackendZnsMessagesResponse, ZnsMessage, ZnsMessagesQueryParams } from '@/types/api/zns.types';

const BASE = '/admin/zns-notifications';

function buildZnsQueryString(params: ZnsMessagesQueryParams): string {
  const parts: string[] = [];
  if (params.page) parts.push(`page=${params.page}`);
  if (params.pageSize) parts.push(`pageSize=${params.pageSize}`);
  if (params.feature) parts.push(`feature=${encodeURIComponent(params.feature)}`);
  if (params.templateId) parts.push(`templateId=${encodeURIComponent(params.templateId)}`);
  if (params.fromDate) parts.push(`fromDate=${encodeURIComponent(params.fromDate)}`);
  if (params.toDate) parts.push(`toDate=${encodeURIComponent(params.toDate)}`);
  if (params.q) parts.push(`q=${encodeURIComponent(params.q.trim())}`);
  params.status?.forEach((s) => parts.push(`status=${encodeURIComponent(s)}`));
  return parts.length ? `?${parts.join('&')}` : '';
}

/** Admin ZNS tracking list over zns_messages (all features, newest first). */
export function useZnsMessages(params: ZnsMessagesQueryParams) {
  return useQuery({
    queryKey: QueryKeys.zns.messages(params),
    queryFn: () =>
      apiClient.get<ZnsMessage[]>(`${BASE}${buildZnsQueryString(params)}`) as Promise<BackendZnsMessagesResponse>,
  });
}
