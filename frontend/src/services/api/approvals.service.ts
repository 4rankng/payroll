import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  ApprovalListFilters,
  ApprovalPagination,
  ApprovalListResult,
  ApprovalRequest,
  FileApprovalData,
} from '@/types/api/approval.types';

interface ApprovalEnvelope {
  status: string;
  data: ApprovalRequest[] | ApprovalRequest;
  message?: string;
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}


// The repo's apiClient resolves to the response body, and some endpoints wrap
// payloads in a {status, data} envelope while others return them bare —
// normalize once here so callers never guess the shape.
function unwrap<T>(body: unknown): T {
  if (body && typeof body === 'object' && 'data' in (body as Record<string, unknown>)) {
    return ((body as Record<string, unknown>).data as T) ?? (body as T);
  }
  return body as T;
}

class ApprovalsService {
  /** List approval requests. Admin sees all; the backend scopes partners to their own. */
  async getApprovals(filters?: ApprovalListFilters): Promise<ApprovalListResult> {
    const query: Record<string, string | number> = { page: filters?.page ?? 1, pageSize: filters?.pageSize ?? 20 };
    if (filters?.statuses?.length) query.status = filters.statuses.join(',');
    const url = `${API_ENDPOINTS.approvals.base}${buildQueryString(query)}`;
    const response = await apiClient.get<ApprovalEnvelope>(url);
    const raw = unwrap<ApprovalRequest[]>(response.data);
    const items = Array.isArray(raw) ? raw : raw ? [raw] : [];
    // apiClient resolves to the payload; pagination rides on the client
    // result itself, NOT inside data — reading it from data silently nulls
    // every count (the Tong Quan row needs totalRecords).
    const paginated = response as unknown as { pagination?: ApprovalPagination };
    return { items, pagination: paginated.pagination ?? null };
  }

  async getApproval(id: number): Promise<ApprovalRequest> {
    const response = await apiClient.get<ApprovalEnvelope>(API_ENDPOINTS.approvals.byId(id));
    const raw = unwrap<ApprovalRequest>(response.data);
    if (!raw || Array.isArray(raw)) {
      throw new Error('API response missing expected data');
    }
    return raw;
  }

  /** File a request with identifying params only — recipients/content are server-derived. */
  async fileApproval(data: FileApprovalData): Promise<ApprovalRequest> {
    const response = await apiClient.post<ApprovalEnvelope>(API_ENDPOINTS.approvals.base, data);
    const raw = unwrap<ApprovalRequest>(response.data);
    if (!raw || Array.isArray(raw)) {
      throw new Error('API response missing expected data');
    }
    return raw;
  }

  async approve(id: number): Promise<ApprovalRequest> {
    const response = await apiClient.post<ApprovalEnvelope>(API_ENDPOINTS.approvals.approve(id), {});
    const raw = unwrap<ApprovalRequest>(response.data);
    if (!raw || Array.isArray(raw)) {
      throw new Error('API response missing expected data');
    }
    return raw;
  }

  /** Reject = "Thử lại ngày mai": the note becomes the next run's guidance. */
  async reject(id: number, note: string): Promise<ApprovalRequest> {
    const response = await apiClient.post<ApprovalEnvelope>(API_ENDPOINTS.approvals.reject(id), { note });
    const raw = unwrap<ApprovalRequest>(response.data);
    if (!raw || Array.isArray(raw)) {
      throw new Error('API response missing expected data');
    }
    return raw;
  }

  /** Cancel = terminal "Huỷ / không gửi lại" — no retry, no escalation. */
  async cancel(id: number, note: string): Promise<ApprovalRequest> {
    const response = await apiClient.post<ApprovalEnvelope>(API_ENDPOINTS.approvals.cancel(id), { note });
    const raw = unwrap<ApprovalRequest>(response.data);
    if (!raw || Array.isArray(raw)) {
      throw new Error('API response missing expected data');
    }
    return raw;
  }
}

export const approvalsService = new ApprovalsService();
