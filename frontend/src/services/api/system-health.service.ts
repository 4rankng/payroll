import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  APISummaryItem,
  UserErrorBreakdown,
  LatencyTrendPoint,
  SlowestEndpoint,
  RecentError,
  CacheMetrics,
  EventBusMetrics,
  TopEndpoint,
  FailedLoginsResponse,
  FailedLoginAttempt,
  BrowserPlatformStat,
  BrowserPlatformUser,
  ErrorCountResponse,
  OSGroupStat,
  BrowserGroupStat,
} from '@/types/api/system-health.types';

class SystemHealthService {
  async getAPISummary(params?: { days?: number; sortBy?: string }): Promise<APISummaryItem[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<APISummaryItem[]>(
      `${API_ENDPOINTS.systemHealth.apiSummary}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getErrorsByUser(params?: { days?: number; minStatusCode?: number; limit?: number }): Promise<UserErrorBreakdown[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<UserErrorBreakdown[]>(
      `${API_ENDPOINTS.systemHealth.errorsByUser}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getLatencyTrend(params?: { days?: number; groupBy?: string }): Promise<LatencyTrendPoint[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<LatencyTrendPoint[]>(
      `${API_ENDPOINTS.systemHealth.latencyTrend}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getSlowestEndpoints(params?: { days?: number; limit?: number }): Promise<SlowestEndpoint[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<SlowestEndpoint[]>(
      `${API_ENDPOINTS.systemHealth.slowestEndpoints}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getRecentErrors(params?: { days?: number; limit?: number }): Promise<RecentError[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<RecentError[]>(
      `${API_ENDPOINTS.systemHealth.recentErrors}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getErrorCount(params?: { days?: number }): Promise<ErrorCountResponse> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<ErrorCountResponse>(
      `${API_ENDPOINTS.systemHealth.errorCount}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getEventBusMetrics(): Promise<EventBusMetrics> {
    const response = await apiClient.get<EventBusMetrics>(
      API_ENDPOINTS.systemHealth.eventBus
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getCacheMetrics(): Promise<CacheMetrics> {
    const response = await apiClient.get<CacheMetrics>(
      API_ENDPOINTS.systemHealth.cache
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getTopEndpoints(params?: { days?: number; limit?: number }): Promise<TopEndpoint[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<TopEndpoint[]>(
      `${API_ENDPOINTS.systemHealth.topEndpoints}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getEndpointLatencyTrend(params: { endpointId: number; days?: number; groupBy?: string }): Promise<LatencyTrendPoint[]> {
    const qs = buildQueryString(params as Record<string, unknown>);
    const response = await apiClient.get<LatencyTrendPoint[]>(
      `${API_ENDPOINTS.systemHealth.endpointTrend}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getFailedLogins(params?: { days?: number; limit?: number }): Promise<FailedLoginsResponse> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<FailedLoginsResponse>(
      `${API_ENDPOINTS.systemHealth.failedLogins}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getFailedLoginsByIdentifier(identifier: string, params?: { days?: number }): Promise<FailedLoginAttempt[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<FailedLoginAttempt[]>(
      `${API_ENDPOINTS.systemHealth.failedLoginsByIdentifier(identifier)}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getBrowserPlatformStats(params?: { days?: number }): Promise<BrowserPlatformStat[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<BrowserPlatformStat[]>(
      `${API_ENDPOINTS.systemHealth.browserPlatformStats}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getBrowserPlatformUsers(params: { browser: string; platform: string; days?: number }): Promise<BrowserPlatformUser[]> {
    const qs = buildQueryString(params as Record<string, unknown>);
    const response = await apiClient.get<BrowserPlatformUser[]>(
      `${API_ENDPOINTS.systemHealth.browserPlatformUsers}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getOSStats(params?: { days?: number }): Promise<OSGroupStat[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<OSGroupStat[]>(
      `${API_ENDPOINTS.systemHealth.osStats}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getOSUsers(params: { osFamily: string; days?: number }): Promise<BrowserPlatformUser[]> {
    const qs = buildQueryString(params as Record<string, unknown>);
    const response = await apiClient.get<BrowserPlatformUser[]>(
      `${API_ENDPOINTS.systemHealth.osUsers}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getBrowserStats(params?: { days?: number }): Promise<BrowserGroupStat[]> {
    const qs = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<BrowserGroupStat[]>(
      `${API_ENDPOINTS.systemHealth.browserStats}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  async getBrowserUsers(params: { browserFamily: string; days?: number }): Promise<BrowserPlatformUser[]> {
    const qs = buildQueryString(params as Record<string, unknown>);
    const response = await apiClient.get<BrowserPlatformUser[]>(
      `${API_ENDPOINTS.systemHealth.browserUsers}${qs}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }
}

export const systemHealthService = new SystemHealthService();
