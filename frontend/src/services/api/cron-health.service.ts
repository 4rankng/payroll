import { apiClient } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type { CronJob } from '@/types/api/cron-health.types';

class CronHealthService {
  async getJobs(): Promise<CronJob[]> {
    const response = await apiClient.get<CronJob[]>(
      API_ENDPOINTS.cronHealth.jobs
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async toggleJob(name: string, enabled: boolean): Promise<{ name: string; is_enabled: boolean }> {
    const response = await apiClient.put<{ name: string; is_enabled: boolean }>(
      API_ENDPOINTS.cronHealth.toggle(name),
      { enabled }
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }
}

export const cronHealthService = new CronHealthService();
