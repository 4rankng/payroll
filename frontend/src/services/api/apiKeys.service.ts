import { apiClient, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';

/** Admin view of a machine API key (plaintext key is never included here). */
export interface APIKey {
  id: number;
  name: string;
  key_prefix: string;
  created_by: number;
  last_used_at: string | null;
  revoked_at: string | null;
  created_at: string;
}

export interface CreateAPIKeyPayload {
  name: string;
}

/** Create response includes the one-time plaintext `key`. */
export interface CreateAPIKeyResult extends APIKey {
  key: string;
}

class ApiKeysService {
  async list(): Promise<ApiResponse<APIKey[]>> {
    return apiClient.get<APIKey[]>(API_ENDPOINTS.apiKeys.base);
  }

  async create(payload: CreateAPIKeyPayload): Promise<ApiResponse<CreateAPIKeyResult>> {
    return apiClient.post<CreateAPIKeyResult>(API_ENDPOINTS.apiKeys.base, payload);
  }

  async revoke(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(API_ENDPOINTS.apiKeys.byId(id));
  }
}

export const apiKeysService = new ApiKeysService();
