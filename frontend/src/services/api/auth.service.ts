import { apiClient, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import { authManager, type AppRole } from '@/lib/auth';
import type {
  User,
  BasicUserProfile,
  LoginCredentials,
  LoginResponse,
  ChangePasswordData,
  ChangePasswordRequest,
  CompleteUserProfile,
  PasswordStrengthResponse,
  VerifyOTPRequest,
  ResendOTPRequest,
} from '@/types/api/auth.types';

// RT-C3: distinct storage key for the pending OTP session id. MUST NOT be
// 'auth_token' — that key is read by AuthManager.isTokenValid() and would
// cause a pending OTP id to be mistaken for a real session. sessionStorage is
// used (not localStorage) so it survives a page refresh but clears on tab
// close — the desired security behavior for an in-flight login.
const OTP_PENDING_KEY = 'otp_pending';

/** Read the pending OTP session id (if any) from sessionStorage. */
export function getPendingOtpSessionId(): string | null {
  try {
    const raw = sessionStorage.getItem(OTP_PENDING_KEY);
    if (!raw) return null;
    return JSON.parse(raw)?.session_id ?? null;
  } catch {
    return null;
  }
}

/** Clear the pending OTP session id (call on successful verify or abandon). */
export function clearPendingOtp(): void {
  sessionStorage.removeItem(OTP_PENDING_KEY);
}

/**
 * Persist the user profile returned by the login endpoints. Display data
 * only — the credential itself is the server-managed HttpOnly cookie
 * (PAYR-37B item 4).
 */
function storeUserProfile(user: User): void {
  localStorage.setItem('userRole', user.role);
  localStorage.setItem('userName', user.fullname);
  localStorage.setItem('userEmail', user.email ?? '');
  localStorage.setItem('userUsername', user.username);
  localStorage.setItem('userId', String(user.id));
  localStorage.setItem('userCreatedAt', user.created_at);
  localStorage.setItem('userUpdatedAt', user.updated_at);
  if (user.last_login) {
    localStorage.setItem('userLastLogin', user.last_login);
  }
}

class AuthService {
  /**
   * Login user with credentials.
   *
   * RT-C3: when the response requires the email-OTP second factor
   * (otp_required=true), this method stores the pending session id under a
   * DISTINCT key (otp_pending in sessionStorage) and returns WITHOUT creating
   * a session (no profile write, no cookie — the cookie only arrives from
   * /auth/login/verify). The caller (useAuth) must route to the OTP entry
   * screen instead of navigating to the dashboard.
   */
  async login(credentials: LoginCredentials): Promise<ApiResponse<LoginResponse>> {
    const response = await apiClient.post<LoginResponse>(
      API_ENDPOINTS.auth.login,
      credentials
    );

    if (response.data?.otp_required && response.data.otp_session_id) {
      // Pending OTP step — do NOT set the auth token. Store only the session id.
      sessionStorage.setItem(
        OTP_PENDING_KEY,
        JSON.stringify({ session_id: response.data.otp_session_id })
      );
      return response;
    }

    if (response.data && response.data.user) {
      // Session = HttpOnly cookie set by the server (PAYR-37B item 4). The
      // response body's access_token is ignored — client JS must never hold
      // the token. Store only the user profile for display + role-based UI.
      const user = response.data.user;
      storeUserProfile(user);
      authManager.setSessionMeta(response.data.expires_in ?? null);
    }

    return response;
  }

  /**
   * Login user with Google OAuth ID token. Same OTP gate as password login
   * (RT-H5): admin/partner Google logins may require the second factor too.
   */
  async loginWithGoogle(idToken: string): Promise<ApiResponse<LoginResponse>> {
    const response = await apiClient.post<LoginResponse>(
      API_ENDPOINTS.auth.google,
      { id_token: idToken }
    );

    if (response.data?.otp_required && response.data.otp_session_id) {
      sessionStorage.setItem(
        OTP_PENDING_KEY,
        JSON.stringify({ session_id: response.data.otp_session_id })
      );
      return response;
    }

    if (response.data && response.data.user) {
      const user = response.data.user;
      storeUserProfile(user);
      authManager.setSessionMeta(response.data.expires_in ?? null);
    }

    return response;
  }

  /**
   * Complete the email-OTP login by submitting the 6-digit code. On success
   * the server sets the session cookie + returns the user; this method stores
   * the profile exactly as login() would have, and clears the pending id.
   */
  async verifyLoginOtp(req: VerifyOTPRequest): Promise<ApiResponse<LoginResponse>> {
    const response = await apiClient.post<LoginResponse>(
      API_ENDPOINTS.auth.loginVerifyOtp,
      req
    );

    if (response.data && response.data.user) {
      const user = response.data.user;
      storeUserProfile(user);
      authManager.setSessionMeta(response.data.expires_in ?? null);
      clearPendingOtp();
    }

    return response;
  }

  /**
   * Request a new OTP code for a pending session (email didn't arrive).
   * Returns the (unchanged) session id and remaining TTL in seconds.
   */
  async resendOtp(req: ResendOTPRequest): Promise<ApiResponse<LoginResponse>> {
    return apiClient.post<LoginResponse>(API_ENDPOINTS.auth.loginResendOtp, req);
  }

  /**
   * Clear all locally-stored session state (cached user fields + expiry
   * metadata). Shared by logout() and by callers that skip the logout API
   * call because the backend already invalidated the session (e.g. after a
   * password change). Key removal lives in AuthManager.clearStoredUser.
   */
  clearLocalSession(): void {
    authManager.clearSession();
  }

  /**
   * Logout user and blacklist token
   */
  async logout(): Promise<ApiResponse<void>> {
    try {
      const response = await apiClient.post(API_ENDPOINTS.auth.logout);
      return response as ApiResponse<void>;
    } finally {
      // Clear local storage even if API call fails
      this.clearLocalSession();
    }
  }


  /**
   * Get current user profile
   */
  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<User>(API_ENDPOINTS.auth.me);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Change user password
   */
  async changePassword(data: ChangePasswordData | ChangePasswordRequest): Promise<ApiResponse<void>> {
    const response = await apiClient.post(API_ENDPOINTS.auth.changePassword, data);
    return response as ApiResponse<void>;
  }

  /**
   * Get complete user profile (extended user info)
   */
  async getCompleteUserProfile(): Promise<CompleteUserProfile> {
    const response = await apiClient.get<CompleteUserProfile>(API_ENDPOINTS.auth.me);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Update current user profile
   */
  async updateProfile(data: { fullname?: string; email?: string; cccd?: string; mobile?: string }): Promise<ApiResponse<CompleteUserProfile>> {
    // Ensure all fields are properly set in the request body
    const requestData: Record<string, string | undefined> = {
      fullname: data.fullname,
      email: data.email, // This can be empty string
    };

    if (data.cccd !== undefined) {
      requestData.cccd = data.cccd;
    }
    if (data.mobile !== undefined) {
      requestData.mobile = data.mobile;
    }

    const response = await apiClient.put<CompleteUserProfile>(API_ENDPOINTS.auth.updateProfile, requestData);

    // Update localStorage with new values
    if (data.fullname !== undefined) {
      localStorage.setItem('userName', data.fullname);
    }
    if (data.email !== undefined) {
      localStorage.setItem('userEmail', data.email);
    }

    return response;
  }

  /**
   * Check password strength
   */
  async checkPasswordStrength(password: string): Promise<PasswordStrengthResponse> {
    // Mock implementation - replace with actual API call if available
    const score = this.calculatePasswordScore(password);
    const feedback = this.getPasswordFeedback(password);
    return {
      score,
      feedback,
      isStrong: score >= 3
    };
  }

  /**
   * Calculate password strength score (0-4)
   */
  private calculatePasswordScore(password: string): number {
    let score = 0;
    if (password.length >= 8) score++;
    if (/[a-z]/.test(password)) score++;
    if (/[A-Z]/.test(password)) score++;
    if (/[0-9]/.test(password)) score++;
    if (/[^A-Za-z0-9]/.test(password)) score++;
    return Math.min(score, 4);
  }

  /**
   * Get password feedback messages
   */
  private getPasswordFeedback(password: string): string[] {
    const feedback: string[] = [];
    if (password.length < 8) feedback.push('Mật khẩu phải có ít nhất 8 ký tự');
    if (!/[a-z]/.test(password)) feedback.push('Mật khẩu phải có ít nhất 1 chữ thường');
    if (!/[A-Z]/.test(password)) feedback.push('Mật khẩu phải có ít nhất 1 chữ hoa');
    if (!/[0-9]/.test(password)) feedback.push('Mật khẩu phải có ít nhất 1 số');
    if (!/[^A-Za-z0-9]/.test(password)) feedback.push('Mật khẩu phải có ít nhất 1 ký tự đặc biệt');
    return feedback;
  }


  /**
   * Check if a session is considered active (UI heuristic — the server
   * authenticates every request via the HttpOnly cookie).
   */
  isAuthenticated(): boolean {
    return authManager.hasSession();
  }

  /**
   * Get user role from the stored profile
   */
  getUserRole(): AppRole | null {
    return authManager.getUserRole();
  }

  /**
   * Check if user has required role
   */
  hasRole(role: AppRole): boolean {
    return authManager.hasRole(role);
  }

  /**
   * Check if user has any of the required roles
   */
  hasAnyRole(roles: AppRole[]): boolean {
    return authManager.hasAnyRole(roles);
  }

  /**
   * Validate session by checking local session state
   */
  validateSession(): boolean {
    return this.isAuthenticated();
  }

  /**
   * One-time migration bootstrap (PAYR-37B item 4): a session created by the
   * pre-cookie bundle still holds the JWT in localStorage. Send it once as a
   * Bearer header — the server validates it AND auto-sets the session cookie
   * on the response — then persist the profile and delete the legacy token.
   * Returns true when a legacy session was converted (or a stale one was
   * cleaned up), so the caller can refresh auth state.
   */
  async bootstrapLegacySession(): Promise<boolean> {
    const legacyToken = authManager.getLegacyToken();
    if (!legacyToken || authManager.hasSession()) {
      return false;
    }

    try {
      const response = await apiClient.get<User>(API_ENDPOINTS.auth.me, {
        headers: { Authorization: `Bearer ${legacyToken}` },
      });
      if (response.data?.id) {
        storeUserProfile(response.data);
        // Remaining cookie lifetime is server-managed; no expiry metadata.
        authManager.setSessionMeta(null);
      }
    } finally {
      // Whether the exchange succeeded or the token was rejected/expired,
      // the legacy localStorage token has no future — remove it.
      authManager.clearLegacyToken();
    }
    return true;
  }

  /**
   * Request a password-reset magic link. The backend ALWAYS returns the same
   * success message whether or not the email exists (anti-enumeration).
   */
  async requestPasswordReset(email: string): Promise<ApiResponse<void>> {
    return apiClient.post<void>(API_ENDPOINTS.auth.passwordResetRequest, { email });
  }

  /**
   * Confirm a password reset with a single-use magic-link token + new password.
   * On success the backend invalidates all existing sessions for the user.
   */
  async confirmPasswordReset(payload: { token: string; new_password: string }): Promise<ApiResponse<void>> {
    return apiClient.post<void>(API_ENDPOINTS.auth.passwordResetConfirm, payload);
  }

  /**
   * Request a Zalo-OTP password reset. Always returns a session id — real or
   * dummy (anti-enumeration). The OTP is delivered via ZNS to the employee's
   * mobile if it exists in the system.
   */
  async requestZaloReset(mobile: string): Promise<ApiResponse<{ otp_session_id: string }>> {
    return apiClient.post<{ otp_session_id: string }>(API_ENDPOINTS.auth.zaloResetRequest, { mobile });
  }

  /**
   * Confirm a Zalo-OTP password reset with the session id + 6-digit code + new
   * password. On success the backend invalidates all existing sessions.
   */
  async confirmZaloReset(payload: {
    otp_session_id: string;
    code: string;
    new_password: string;
  }): Promise<ApiResponse<void>> {
    return apiClient.post<void>(API_ENDPOINTS.auth.zaloResetConfirm, payload);
  }

}

export const authService = new AuthService();
