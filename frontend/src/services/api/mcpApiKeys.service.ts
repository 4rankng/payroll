import { apiClient, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';

/** Scope of an MCP API key — the breadth gate for the agent surface. */
export type MCPAPIKeyScope = 'read_only' | 'full';

/** Admin view of an MCP API key (plaintext key is never included here). */
export interface MCPAPIKey {
  id: number;
  name: string;
  scope: MCPAPIKeyScope;
  key_prefix: string;
  created_by: number;
  last_used_at: string | null;
  revoked_at: string | null;
  /** Last admin scope toggle (read_only <-> full); null = never toggled. */
  scope_updated_at: string | null;
  created_at: string;
}

export interface CreateMCPAPIKeyPayload {
  name: string;
  scope: MCPAPIKeyScope;
}

/**
 * Body for PUT /admin/mcp-api-keys/:id/scope. Only the closed MCP scope
 * vocabulary is accepted — revocation stays on its own endpoint.
 */
export interface SetMCPAPIKeyScopePayload {
  scope: MCPAPIKeyScope;
}

/** Create response includes the one-time plaintext `key`. */
export interface CreateMCPAPIKeyResult extends MCPAPIKey {
  key: string;
}

class McpApiKeysService {
  async list(): Promise<ApiResponse<MCPAPIKey[]>> {
    return apiClient.get<MCPAPIKey[]>(API_ENDPOINTS.mcpApiKeys.base);
  }

  async create(payload: CreateMCPAPIKeyPayload): Promise<ApiResponse<CreateMCPAPIKeyResult>> {
    return apiClient.post<CreateMCPAPIKeyResult>(API_ENDPOINTS.mcpApiKeys.base, payload);
  }

  async revoke(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(API_ENDPOINTS.mcpApiKeys.byId(id));
  }

  /**
   * Toggle read_only <-> full on the SAME key (no revoke/recreate). The
   * backend requires an OTP-verified admin session and audit-logs every
   * change; the new scope applies to the agent's very next call.
   */
  async setScope(id: number, payload: SetMCPAPIKeyScopePayload): Promise<ApiResponse<MCPAPIKey>> {
    return apiClient.put<MCPAPIKey>(API_ENDPOINTS.mcpApiKeys.scope(id), payload);
  }
}

export const mcpApiKeysService = new McpApiKeysService();
