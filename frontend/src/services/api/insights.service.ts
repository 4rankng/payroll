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

class InsightsService {
  async getDigest(date?: string): Promise<DailyFinding[]> {
    const query = date ? buildQueryString({ date }) : '';
    const response = await apiClient.get<InsightsEnvelope>(
      `${API_ENDPOINTS.insights.digest}${query}`
    );
    return Array.isArray(response.data?.data) ? response.data.data : [];
  }

  async dismiss(id: number): Promise<void> {
    await apiClient.post(API_ENDPOINTS.insights.dismiss(id), {});
  }
}

export const insightsService = new InsightsService();
