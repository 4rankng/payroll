import { apiClient } from './client';
import { API_ENDPOINTS } from '@/config/api.config';

export interface HealthCheckResult {
  status: 'healthy' | 'unhealthy';
  duration: number;
  error?: string;
  timestamp: string;
}

export interface HealthStatus {
  status: 'healthy' | 'unhealthy';
  checks: Record<string, HealthCheckResult>;
  version: string;
}

class HealthService {
  async getHealthStatus(): Promise<HealthStatus> {
    const response = await apiClient.get<HealthStatus>(
      API_ENDPOINTS.health.base
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }
}

export const healthService = new HealthService();