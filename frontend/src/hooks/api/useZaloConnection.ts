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

// Result of POST /admin/zalo/test — mirrors the backend zalo.SendResult.
// error_code 0 = success; non-zero is a Zalo business error (e.g. -124 bad
// token, -118 phone not linked to Zalo) surfaced with a Vietnamese message.
export interface ZaloTestSendResult {
  msg_id?: string;
  error_code: number;
  error_msg: string;
  http_status?: number;
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
    access_token?: string;
    refresh_token?: string;
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

  /**
   * Fire one test ZNS to verify the stored tokens work. Defaults to the OTP
   * template (617976) with sample data when template_id/template_data are
   * omitted. Does NOT touch the password-reset flow — no OTP stored in Redis.
   */
  async testSend(payload: {
    phone: string;
    template_id?: string;
    template_data?: Record<string, string>;
  }): Promise<ApiResponse<ZaloTestSendResult>> {
    return apiClient.post<ZaloTestSendResult>(API_ENDPOINTS.zalo.test, payload);
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

/** Fire a test ZNS to verify the connection. See ZaloTestSendResult for shape. */
export const useTestZaloSend = () =>
  useMutation({
    mutationFn: zaloAdminService.testSend.bind(zaloAdminService),
  });
