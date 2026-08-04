import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient, ApiResponse } from '@/services/api/client';
import { API_ENDPOINTS } from '@/config/api.config';

// --- types ------------------------------------------------------------------

export interface ZaloConnectionStatus {
  enabled: boolean;
  configured: boolean;
  connected: boolean;
  template_id: string;
  expires_at?: string;
  last_error?: string;
  callback_url: string;
}

// --- service ----------------------------------------------------------------

class ZaloAdminService {
  async getStatus(): Promise<ApiResponse<ZaloConnectionStatus>> {
    return apiClient.get<ZaloConnectionStatus>(API_ENDPOINTS.zalo.status);
  }

  async saveCredentials(payload: {
    app_id: string;
    secret_key: string;
    template_id: string;
  }): Promise<ApiResponse<void>> {
    return apiClient.put<void>(API_ENDPOINTS.zalo.credentials, payload);
  }

  async startOAuth(): Promise<ApiResponse<{ redirect_url: string }>> {
    return apiClient.post<{ redirect_url: string }>(API_ENDPOINTS.zalo.oauthStart, {});
  }

  /**
   * Complete the OAuth flow: exchange the code+state (received from Zalo's
   * redirect to the SPA) for tokens. Called by the SPA after detecting
   * ?zalo_oauth=1&code=...&state=... in the URL.
   */
  async completeOAuth(code: string, state: string): Promise<ApiResponse<void>> {
    return apiClient.post<void>(API_ENDPOINTS.zalo.oauthCallback, { code, state });
  }

  async setEnabled(enabled: boolean): Promise<ApiResponse<void>> {
    return apiClient.put<void>(API_ENDPOINTS.zalo.enabled, { enabled });
  }

  async refreshNow(): Promise<ApiResponse<void>> {
    return apiClient.post<void>(API_ENDPOINTS.zalo.refresh, {});
  }
}

export const zaloAdminService = new ZaloAdminService();

// --- hooks ------------------------------------------------------------------

const STATUS_KEY = ['zalo', 'status'] as const;

/**
 * useZaloStatus — live connection status. Refetches every 60s while the tab is
 * visible so the "expires in" countdown stays fresh after a server-side token
 * refresh (the Provider refreshes tokens internally during normal OTP sends).
 */
export const useZaloStatus = () =>
  useQuery({
    queryKey: STATUS_KEY,
    queryFn: () => zaloAdminService.getStatus(),
    refetchInterval: 60_000,
    refetchIntervalInBackground: false,
    retry: 1,
  });

export const useSaveZaloCredentials = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: zaloAdminService.saveCredentials.bind(zaloAdminService),
    onSuccess: () => qc.invalidateQueries({ queryKey: STATUS_KEY }),
  });
};

export const useStartZaloOAuth = () =>
  useMutation({
    mutationFn: zaloAdminService.startOAuth.bind(zaloAdminService),
  });

/** Complete the OAuth flow after Zalo redirects back to the SPA. */
export const useCompleteZaloOAuth = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ code, state }: { code: string; state: string }) =>
      zaloAdminService.completeOAuth(code, state),
    onSuccess: () => qc.invalidateQueries({ queryKey: STATUS_KEY }),
  });
};

export const useSetZaloEnabled = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (enabled: boolean) => zaloAdminService.setEnabled(enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: STATUS_KEY }),
  });
};

export const useRefreshZaloToken = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: zaloAdminService.refreshNow.bind(zaloAdminService),
    onSuccess: () => qc.invalidateQueries({ queryKey: STATUS_KEY }),
  });
};
