import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';

export type InsightSeverity = 'info' | 'warning' | 'critical';

export interface DailyFinding {
  id: number;
  date: string;
  type: string;
  severity: InsightSeverity;
  facts: Record<string, unknown>;
  narrative: string;
  dismissed_at?: string | null;
  created_at: string;
}

interface InsightsEnvelope {
  status: string;
  data: DailyFinding[];
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

class InsightsService {
  async getDigest(date?: string): Promise<DailyFinding[]> {
    const query = date ? buildQueryString({ date }) : '';
    const response = await apiClient.get<InsightsEnvelope>(
      `${API_ENDPOINTS.insights.digest}${query}`
    );
    const raw = unwrap<DailyFinding[]>(response.data);
    return Array.isArray(raw) ? raw : [];
  }

  async dismiss(id: number): Promise<void> {
    await apiClient.post(API_ENDPOINTS.insights.dismiss(id), {});
  }
}

export const insightsService = new InsightsService();
