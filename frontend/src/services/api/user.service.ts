import { apiClient, buildQueryString, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type { User, UserSummary, CreateUserData, UpdateUserData, ResetPasswordData, UserActivitySummary } from '@/types/user';

export interface UserFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  role?: 'admin' | 'partner' | 'employee';
  search?: string;
  ids?: string; // Comma-separated user IDs for batch lookup (e.g., "1,2,3")
  last_login_today?: boolean;
}

export interface UserListResponse {
  data: User[];
  message: string;
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

class UserService {
  /**
   * Get users summary statistics
   */
  async getSummary(): Promise<UserSummary> {
    const response = await apiClient.get<UserSummary>(
      API_ENDPOINTS.users.summary
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get users by a comma-separated list of IDs — no pagination, returns all matches.
   */
  async getUsersByIds(ids: string): Promise<UserListResponse> {
    const response = await apiClient.get<User[]>(
      `${API_ENDPOINTS.users.base}?ids=${encodeURIComponent(ids)}`
    );
    return {
      data: response.data || [],
      message: response.message || '',
      pagination: response.pagination || { page: 1, pageSize: ids.split(',').length, totalPages: 1, totalRecords: 0 },
    };
  }

  /**
   * Get paginated list of users
   */
  async getUsers(filters?: UserFilters): Promise<UserListResponse> {
    const defaultFilters = {
      page: 1,
      pageSize: 100,
      sortBy: 'created_at',
      sortOrder: 'desc' as const,
      offset: 0,
      ...filters
    };

    const queryString = buildQueryString(defaultFilters);
    const fullUrl = `${API_ENDPOINTS.users.base}${queryString}`;



    const response = await apiClient.get<User[]>(fullUrl);



    // The API response structure is: { status, data: User[], message, pagination }
    // We need to return UserListResponse format
    return {
      data: response.data || [],
      message: response.message || '',
      pagination: response.pagination || {
        page: 1,
        pageSize: 100,
        totalPages: 1,
        totalRecords: 0
      }
    };
  }

  /**
   * Get single user by ID
   */
  async getUserById(id: number): Promise<User> {
    const response = await apiClient.get<User>(
      API_ENDPOINTS.users.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create new user
   */
  async createUser(data: CreateUserData): Promise<ApiResponse<User>> {
    const response = await apiClient.post<User>(
      API_ENDPOINTS.users.base,
      data
    );
    return response;
  }

  /**
   * Update existing user
   */
  async updateUser(id: number, data: UpdateUserData): Promise<ApiResponse<User>> {
    const response = await apiClient.put<User>(
      API_ENDPOINTS.users.byId(id),
      data
    );
    return response;
  }


  /**
   * Reset user password
   */
  async resetPassword(id: number, data: ResetPasswordData): Promise<ApiResponse<void>> {
    const response = await apiClient.post<void>(
      API_ENDPOINTS.users.resetPassword(id),
      data
    );
    return response;
  }


  /**
   * Delete user account (soft delete - Admin only)
   * Returns HTTP 204 No Content
   */
  async deleteUser(id: number): Promise<void> {
    await apiClient.delete(API_ENDPOINTS.users.byId(id));
  }

  /**
   * Get user activity summary
   */
  async getUserActivities(id: number, days?: number): Promise<UserActivitySummary> {
    const queryString = days ? `?days=${days}` : '';
    const response = await apiClient.get<UserActivitySummary>(
      `${API_ENDPOINTS.users.byId(id)}/activities${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }
}

export const userService = new UserService();
