import { apiClient, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';

/**
 * Scope of a payroll-ops-mcp key (pk_ops_*) — the breadth gate for the
 * admin-equivalent agent channel. The key executes as its creating admin;
 * read_write additionally permits mutation verbs and carries a write window
 * (default 8h) after which the effective scope reverts to read_only.
 */
export type OpsAPIKeyScope = 'read_only' | 'read_write';

/** Admin view of an ops key (plaintext key is never included here). */
export interface OpsAPIKey {
  id: number;
  name: string;
  scope: OpsAPIKeyScope;
  /** Scope at read time — a lapsed write window reads as read_only. */
  effective_scope: OpsAPIKeyScope;
  write_window_expires_at: string | null;
  key_prefix: string;
  created_by: number;
  last_used_at: string | null;
  revoked_at: string | null;
  created_at: string;
}

export interface CreateOpsAPIKeyPayload {
  name: string;
  /** Defaults to read_only on the backend when omitted. */
  scope?: OpsAPIKeyScope;
  /** Only meaningful with read_write; defaults to 8h on the backend. */
  write_window_hours?: number;
}

export interface SetOpsAPIKeyScopePayload {
  scope: OpsAPIKeyScope;
  write_window_hours?: number;
}

/** Create response includes the one-time plaintext `key`. */
export interface CreateOpsAPIKeyResult extends OpsAPIKey {
  key: string;
}

class OpsApiKeysService {
  async list(): Promise<ApiResponse<OpsAPIKey[]>> {
    return apiClient.get<OpsAPIKey[]>(API_ENDPOINTS.opsApiKeys.base);
  }

  async create(payload: CreateOpsAPIKeyPayload): Promise<ApiResponse<CreateOpsAPIKeyResult>> {
    return apiClient.post<CreateOpsAPIKeyResult>(API_ENDPOINTS.opsApiKeys.base, payload);
  }

  async revoke(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(API_ENDPOINTS.opsApiKeys.byId(id));
  }

  /**
   * Toggle read_only ↔ read_write. The backend requires an OTP-verified
   * admin session and audit-logs every change; the new scope applies to the
   * agent's next call.
   */
  async setScope(id: number, payload: SetOpsAPIKeyScopePayload): Promise<ApiResponse<OpsAPIKey>> {
    return apiClient.put<OpsAPIKey>(API_ENDPOINTS.opsApiKeys.scope(id), payload);
  }
}

export const opsApiKeysService = new OpsApiKeysService();
