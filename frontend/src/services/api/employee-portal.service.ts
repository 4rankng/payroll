import { apiClient, buildQueryString, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  EmployeeProfile,
  UpdateEmployeeProfileData,
  ChangePasswordData,
  EmployeeTimesheetResponse,
  EmployeeTimesheetFilters,
  EmployeeSummaryResponse,
} from '@/types/api/auth.types';

/**
 * Employee Self-Service Portal API Service
 * Handles employee-specific endpoints for viewing timesheets and managing profile
 */
class EmployeePortalService {
  /**
   * Get employee's own profile
   * GET /api/v1/me
   */
  async getMyProfile(): Promise<EmployeeProfile> {
    const response = await apiClient.get<EmployeeProfile>(
      API_ENDPOINTS.employee.profile
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Update employee's own profile
   * PUT /api/v1/me
   */
  async updateMyProfile(data: UpdateEmployeeProfileData): Promise<ApiResponse<EmployeeProfile>> {
    const response = await apiClient.put<EmployeeProfile>(
      API_ENDPOINTS.employee.updateProfile,
      data
    );

    // Update localStorage if username or fullname changed
    if (data.fullname !== undefined) {
      localStorage.setItem('userName', data.fullname);
    }
    if (data.username !== undefined) {
      localStorage.setItem('userUsername', data.username);
    }
    if (data.email !== undefined && data.email) {
      localStorage.setItem('userEmail', data.email);
    }

    return response;
  }

  /**
   * Update employee's password
   * PUT /api/v1/me/password
   */
  async updateMyPassword(data: ChangePasswordData): Promise<ApiResponse<void>> {
    const response = await apiClient.put(
      API_ENDPOINTS.employee.updatePassword,
      data
    );
    return response;
  }

  /**
   * Get employee's timesheet entries
   * GET /api/v1/me/timesheet
   */
  async getMyTimesheets(filters?: EmployeeTimesheetFilters): Promise<EmployeeTimesheetResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<EmployeeTimesheetResponse>(
      `${API_ENDPOINTS.employee.timesheets}${queryString}`
    );
    // The API returns the full response structure
    return response as EmployeeTimesheetResponse;
  }

  /**
   * Get employee's salary summary statistics
   * GET /api/v1/me/summary
   */
  async getMySummary(weeks: number = 4): Promise<EmployeeSummaryResponse> {
    const queryString = buildQueryString({ weeks });
    const response = await apiClient.get<EmployeeSummaryResponse>(
      `${API_ENDPOINTS.employee.summary}${queryString}`
    );
    return response as EmployeeSummaryResponse;
  }
}

export const employeePortalService = new EmployeePortalService();
