import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  Setting,
  CreateSettingData,
  UpdateSettingData,
  SettingsFilters,
  SettingsResponse,
  SettingResponse,
} from '@/types/api/settings.types';

class SettingsService {
  /**
   * Create a new setting (Admin only)
   */
  async createSetting(data: CreateSettingData): Promise<Setting> {
    const response = await apiClient.post<SettingResponse>(
      API_ENDPOINTS.settings.base,
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /**
   * Get setting by ID (Admin only)
   */
  async getSettingById(id: number): Promise<Setting> {
    const response = await apiClient.get<SettingResponse>(
      API_ENDPOINTS.settings.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /**
   * Get setting by key
   */
  async getSettingByKey(key: string): Promise<Setting> {
    const response = await apiClient.get<Setting>(
      API_ENDPOINTS.settings.byKey(key)
    );

    if (!response?.data) {
      throw new Error(`Setting with key "${key}" returned no data`);
    }

    return response.data;
  }

  /**
   * Update setting (Admin only)
   */
  async updateSetting(id: number, data: UpdateSettingData): Promise<SettingResponse> {
    const response = await apiClient.put<SettingResponse>(
      API_ENDPOINTS.settings.byId(id),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Delete setting (Admin only)
   */
  async deleteSetting(id: number): Promise<void> {
    await apiClient.delete(API_ENDPOINTS.settings.byId(id));
  }

  /**
   * List settings with filtering (Admin only)
   */
  async getSettings(filters?: SettingsFilters): Promise<Setting[]> {
    const queryString = filters ? buildQueryString(filters) : '';
    const url = `${API_ENDPOINTS.settings.base}${queryString}`;
    const response = await apiClient.get<{ status: string; data: Setting[]; message: string; pagination?: unknown }>(url);
    const settings = response.data.data || response.data;
    return Array.isArray(settings) ? settings : [];
  }
}

export const settingsService = new SettingsService();
