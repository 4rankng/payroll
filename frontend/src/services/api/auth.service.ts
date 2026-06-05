import { apiClient, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import { authManager } from '@/lib/auth';
import type {
  User,
  BasicUserProfile,
  LoginCredentials,
  LoginResponse,
  ChangePasswordData,
  ChangePasswordRequest,
  CompleteUserProfile,
  PasswordStrengthResponse,
} from '@/types/api/auth.types';


class AuthService {
  /**
   * Login user with credentials
   */
  async login(credentials: LoginCredentials): Promise<ApiResponse<LoginResponse>> {
    const response = await apiClient.post<LoginResponse>(
      API_ENDPOINTS.auth.login,
      credentials
    );

    if (response.data && response.data.access_token) {
      // Store token in auth manager
      authManager.setToken(response.data.access_token);

      // Store user info in localStorage for quick access
      localStorage.setItem('userRole', response.data.user.role);
      localStorage.setItem('userName', response.data.user.fullname);
      localStorage.setItem('userEmail', response.data.user.email);
      localStorage.setItem('userUsername', response.data.user.username);
      localStorage.setItem('userCreatedAt', response.data.user.created_at);
      localStorage.setItem('userUpdatedAt', response.data.user.updated_at);
      if (response.data.user.last_login) {
        localStorage.setItem('userLastLogin', response.data.user.last_login);
      }
    }

    return response;
  }

  /**
   * Login user with Google OAuth ID token
   */
  async loginWithGoogle(idToken: string): Promise<ApiResponse<LoginResponse>> {
    const response = await apiClient.post<LoginResponse>(
      API_ENDPOINTS.auth.google,
      { id_token: idToken }
    );

    if (response.data && response.data.access_token) {
      // Store token in auth manager
      authManager.setToken(response.data.access_token);

      // Store user info in localStorage for quick access
      localStorage.setItem('userRole', response.data.user.role);
      localStorage.setItem('userName', response.data.user.fullname);
      localStorage.setItem('userEmail', response.data.user.email ?? '');
      localStorage.setItem('userUsername', response.data.user.username);
      localStorage.setItem('userCreatedAt', response.data.user.created_at);
      localStorage.setItem('userUpdatedAt', response.data.user.updated_at);
      if (response.data.user.last_login) {
        localStorage.setItem('userLastLogin', response.data.user.last_login);
      }
    }

    return response;
  }

  /**
   * Logout user and blacklist token
   */
  async logout(): Promise<ApiResponse<void>> {
    try {
      const response = await apiClient.post(API_ENDPOINTS.auth.logout);
      return response;
    } finally {
      // Clear local storage even if API call fails
      authManager.removeToken();
      localStorage.removeItem('userRole');
      localStorage.removeItem('userName');
      localStorage.removeItem('userEmail');
      localStorage.removeItem('userUsername');
      localStorage.removeItem('userCreatedAt');
      localStorage.removeItem('userUpdatedAt');
      localStorage.removeItem('userLastLogin');
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
    return response;
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
   * Check if user is authenticated
   */
  isAuthenticated(): boolean {
    return authManager.isTokenValid();
  }

  /**
   * Get user role from stored token
   */
  getUserRole(): 'admin' | 'partner' | 'employee' | 'adv_partner' | null {
    return authManager.getUserRole();
  }

  /**
   * Check if user has required role
   */
  hasRole(role: 'admin' | 'partner' | 'employee' | 'adv_partner'): boolean {
    return authManager.hasRole(role);
  }

  /**
   * Check if user has any of the required roles
   */
  hasAnyRole(roles: ('admin' | 'partner' | 'employee' | 'adv_partner')[]): boolean {
    return authManager.hasAnyRole(roles);
  }

  /**
   * Validate session by checking if token is valid
   */
  validateSession(): boolean {
    return this.isAuthenticated();
  }

}

export const authService = new AuthService();
