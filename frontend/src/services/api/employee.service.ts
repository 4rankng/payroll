import { apiClient, buildQueryString, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  Employee,
  EmployeeSummary,
  CreateEmployeeData,
  UpdateEmployeeData,
  EmployeeFilters,
  EmployeesResponse,
  EmployeePayrollResponse,
  EmployeePayrollFilters,
  EmployeeProjectsResponse,
  EmployeeIndividualSummary,
  EmployeeTimesheetResponse,
  EmployeeTimesheetFilters,
  EmployeeImportStatus,
  EmployeeImportResponse,
  EmployeeCurrentProjectsResponse,
  EmployeeCurrentProjectsFilters,
  EmployeeUsersResponse,
  GrantEmployeeAccessData
} from '@/types/api/employee.types';

class EmployeeService {
  /**
   * Get employees summary statistics
   */
  async getSummary(): Promise<EmployeeSummary> {
    const response = await apiClient.get<EmployeeSummary>(
      API_ENDPOINTS.employees.summary
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get paginated list of employees
   */
  async getEmployees(filters?: EmployeeFilters): Promise<EmployeesResponse> {
    // Convert month to date range if provided and no explicit fromDate/toDate
    let apiFilters = filters;
    if (filters?.month && !filters.fromDate && !filters.toDate) {
      const { fromDate, toDate } = this.convertMonthToDateRange(filters.month);
      apiFilters = {
        ...filters,
        fromDate,
        toDate,
        month: undefined
      };
    }

    const queryString = apiFilters ? buildQueryString(apiFilters as Record<string, unknown>) : '';
    const response = await apiClient.get(
      `${API_ENDPOINTS.employees.base}${queryString}`
    );

    // The API client returns the full response with status, data, message, pagination
    // We need to extract just the data and pagination for our interface
    return {
      data: (response.data as Employee[]) || [],
      pagination: response.pagination || {
        page: 1,
        pageSize: 100,
        totalPages: 1,
        totalRecords: 0
      }
    };
  }

  /**
   * Get employees with missing bank details
   */
  async getEmployeesWithMissingBankDetails(filters?: EmployeeFilters): Promise<EmployeesResponse> {
    // Convert month to date range if provided and no explicit fromDate/toDate
    let apiFilters = filters;
    if (filters?.month && !filters.fromDate && !filters.toDate) {
      const { fromDate, toDate } = this.convertMonthToDateRange(filters.month);
      apiFilters = {
        ...filters,
        fromDate,
        toDate,
        month: undefined
      };
    }

    const queryString = apiFilters ? buildQueryString(apiFilters as Record<string, unknown>) : '';
    const response = await apiClient.get(
      `${API_ENDPOINTS.employees.missingBankDetails}${queryString}`
    );

    return {
      data: (response.data as Employee[]) || [],
      pagination: response.pagination || {
        page: 1,
        pageSize: 100,
        totalPages: 1,
        totalRecords: 0
      }
    };
  }

  /**
   * Convert month filter (YYYY-MM) to date range
   */
  private convertMonthToDateRange(month: string): { fromDate: string; toDate: string } {
    const [year, monthNum] = month.split('-').map(Number);
    const firstDay = new Date(year, monthNum - 1, 1);
    const lastDay = new Date(year, monthNum, 0);

    return {
      fromDate: firstDay.toISOString().split('T')[0],
      toDate: lastDay.toISOString().split('T')[0]
    };
  }

  /**
   * Search employees using the base employees endpoint
   */
  async searchEmployees(params: {
    search: string;
    pageSize?: number;
    status?: 'working' | 'unassigned';
    month?: string;
    project_id?: number;
    fromDate?: string;
    toDate?: string;
  }): Promise<EmployeesResponse> {
    // Convert month to date range if provided and no explicit fromDate/toDate
    const apiParams: Record<string, unknown> = { ...params };
    if (params.month && !params.fromDate && !params.toDate) {
      const { fromDate, toDate } = this.convertMonthToDateRange(params.month);
      apiParams.fromDate = fromDate;
      apiParams.toDate = toDate;
      delete apiParams.month;
    }

    const queryString = buildQueryString(apiParams);
    const response = await apiClient.get(
      `${API_ENDPOINTS.employees.base}${queryString}`
    );

    return {
      data: response.data || [],
      pagination: response.pagination || {
        page: 1,
        pageSize: params.pageSize || 50,
        totalPages: 1,
        totalRecords: response.data?.length || 0
      }
    };
  }

  /**
   * Get single employee by ID
   */
  async getEmployeeById(id: number): Promise<Employee> {
    const response = await apiClient.get<Employee>(
      API_ENDPOINTS.employees.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get employee summary by ID
   */
  async getEmployeeSummaryById(id: number): Promise<EmployeeIndividualSummary> {
    const response = await apiClient.get<EmployeeIndividualSummary>(
      API_ENDPOINTS.employees.summaryById(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get single employee by CCCD
   */
  async getEmployeeByCCCD(cccd: string): Promise<Employee> {
    const response = await apiClient.get<Employee>(
      API_ENDPOINTS.employees.byCCCD(cccd)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create new employee
   */
  async createEmployee(data: CreateEmployeeData): Promise<ApiResponse<Employee>> {
    const response = await apiClient.post<Employee>(
      API_ENDPOINTS.employees.base,
      data
    );
    return response;
  }

  /**
   * Update existing employee
   */
  async updateEmployee(id: number, data: UpdateEmployeeData): Promise<ApiResponse<Employee>> {
    const response = await apiClient.put<Employee>(
      API_ENDPOINTS.employees.byId(id),
      data
    );
    return response;
  }

  /**
   * Mark employee as inactive (soft delete)
   */
  async deleteEmployee(id: number): Promise<ApiResponse<void>> {
    const response = await apiClient.delete(API_ENDPOINTS.employees.byId(id));
    return response;
  }

  /**
   * Get employee project assignments
   */
  async getEmployeeProjects(id: number, params?: {
    status?: 'active' | 'ended' | 'inactive';
    page?: number;
    pageSize?: number;
  }): Promise<EmployeeProjectsResponse> {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get<EmployeeProjectsResponse>(
      `${API_ENDPOINTS.employees.projects(id)}${queryString}`
    );

    return {
      data: response.data || [],
      pagination: response.pagination || {
        page: 1,
        pageSize: 20,
        totalPages: 1,
        totalRecords: 0
      }
    };
  }

  /**
   * Get employee current projects with timesheets
   */
  async getEmployeeCurrentProjects(
    id: number,
    filters?: EmployeeCurrentProjectsFilters
  ): Promise<EmployeeCurrentProjectsResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<EmployeeCurrentProjectsResponse>(
      `${API_ENDPOINTS.employees.currentProjects(id)}${queryString}`
    );

    return {
      data: response.data || []
    };
  }

  /**
   * Get employee timesheet summary
   */
  async getEmployeeTimesheetSummary(
    id: number,
    params?: { project_id?: number; fromDate?: string; toDate?: string }
  ) {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get(
      `${API_ENDPOINTS.employees.timesheetSummary(id)}${queryString}`
    );
    return response.data;
  }

  /**
   * Import employees from Excel file (async)
   * Returns an import ID that can be used to track progress
   */
  async importEmployees(file: File): Promise<EmployeeImportResponse> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await apiClient.post<EmployeeImportResponse>(
      API_ENDPOINTS.employees.import,
      formData
    );
    if (!response.data) {
      throw new Error('API response missing import_id');
    }
    return response.data;
  }

  /**
   * Get employee import status by import ID
   */
  async getImportStatus(importId: string): Promise<EmployeeImportStatus> {
    const response = await apiClient.get<EmployeeImportStatus>(
      API_ENDPOINTS.employees.importStatus(importId)
    );
    if (!response.data) {
      throw new Error('API response missing import status');
    }
    return response.data;
  }

  /**
   * Export employees to Excel
   */
  async exportEmployees(filters?: EmployeeFilters): Promise<ApiResponse<void>> {
    // Convert month to date range if provided and no explicit fromDate/toDate
    let apiFilters = filters;
    if (filters?.month && !filters.fromDate && !filters.toDate) {
      const { fromDate, toDate } = this.convertMonthToDateRange(filters.month);
      apiFilters = {
        ...filters,
        fromDate,
        toDate,
        month: undefined
      };
    }

    const queryString = apiFilters ? buildQueryString(apiFilters) : '';
    const response = await apiClient.download(
      `${API_ENDPOINTS.employees.export}${queryString}`,
      `employees_export_${new Date().toISOString().split('T')[0]}.xlsx`
    );
    return response;
  }

  /**
   * Export single employee detail to Excel using HoSoNhanSu template
   */
  async exportEmployeeDetail(id: number, fullname: string): Promise<void> {
    const filename = `HoSoNhanSu_${fullname}_${new Date().toISOString().split('T')[0]}.xlsx`;
    await apiClient.download(
      API_ENDPOINTS.employees.exportDetail(id),
      filename
    );
  }

  /**
   * Validate CCCD uniqueness
   */
  async validateCCCD(cccd: string, excludeId?: number): Promise<boolean> {
    try {
      const response = await apiClient.get(
        `${API_ENDPOINTS.employees.base}/validate-cccd?cccd=${cccd}${excludeId ? `&exclude=${excludeId}` : ''}`
      );
      return response.data || false;
    } catch (error) {
      return false;
    }
  }

  /**
   * Validate email uniqueness
   */
  async validateEmail(email: string, excludeId?: number): Promise<boolean> {
    try {
      const response = await apiClient.get(
        `${API_ENDPOINTS.employees.base}/validate-email?email=${email}${excludeId ? `&exclude=${excludeId}` : ''}`
      );
      return response.data || false;
    } catch (error) {
      return false;
    }
  }

  /**
   * Get employee payroll history
   */
  async getEmployeePayroll(id: number, filters?: EmployeePayrollFilters): Promise<EmployeePayrollResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<EmployeePayrollResponse>(
      `${API_ENDPOINTS.employees.payroll(id)}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get employee timesheet history
   */
  async getEmployeeTimesheet(id: number, filters?: EmployeeTimesheetFilters): Promise<EmployeeTimesheetResponse> {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<EmployeeTimesheetResponse>(
      `${API_ENDPOINTS.employees.timesheet(id)}${queryString}`
    );
    return {
      data: response.data || [],
      pagination: response.pagination || {
        page: 1,
        pageSize: 100,
        totalPages: 1,
        totalRecords: 0
      }
    };
  }

  /**
   * Bulk import employees with enhanced validation
   */
  async bulkImportEmployees(
    file: File,
    options?: {
      validate_only?: boolean;
      skip_duplicates?: boolean;
      update_existing?: boolean;
    },
    onProgress?: (progress: number) => void
  ) {
    const formData = new FormData();
    formData.append('file', file);
    if (options) {
      Object.entries(options).forEach(([key, value]) => {
        formData.append(key, value.toString());
      });
    }

    const response = await apiClient.upload(
      `${API_ENDPOINTS.employees.base}/bulk-import`,
      formData,
      onProgress
    );
    return response.data;
  }

  /**
   * Get employee working statistics
   */
  async getEmployeeWorkingStatistics(
    id: number,
    params?: {
      period?: number; // days
      fromDate?: string;
      toDate?: string;
    }
  ) {
    const queryString = params ? buildQueryString(params) : '';
    const response = await apiClient.get(
      `${API_ENDPOINTS.employees.byId(id)}/working-statistics${queryString}`
    );
    return response.data;
  }

  /**
   * Change employee password (Admin/Partner only)
   */
  async changeEmployeePassword(
    id: number,
    data: { new_password: string }
  ): Promise<ApiResponse<void>> {
    const response = await apiClient.put(
      API_ENDPOINTS.employees.changePassword(id),
      data
    );
    return response;
  }

  /**
   * Combined update: employee info + username + password in one call.
   * Uses the dedicated adv-partner endpoint.
   */
  async updateAdvPartnerUser(
    id: number,
    data: {
      fullname?: string;
      email?: string;
      cccd?: string;
      mobile?: string;
      bank_id?: number;
      bank_account_number?: string;
      bank_account_name?: string;
      username?: string;
      password?: string;
    }
  ): Promise<ApiResponse<Employee>> {
    const response = await apiClient.put<Employee>(
      API_ENDPOINTS.advPartner.updateUser(id),
      data
    );
    return response;
  }

  /**
   * Get users who have access to an employee
   */
  async getEmployeeUsers(employeeId: number): Promise<EmployeeUsersResponse> {
    const response = await apiClient.get<EmployeeUsersResponse>(
      API_ENDPOINTS.employees.users(employeeId)
    );
    return {
      data: response.data || []
    };
  }

  /**
   * Grant access to an employee for a specific user
   */
  async grantEmployeeAccess(
    employeeId: number,
    data: GrantEmployeeAccessData
  ): Promise<ApiResponse<void>> {
    const response = await apiClient.post(
      API_ENDPOINTS.employees.users(employeeId),
      data
    );
    return response;
  }

  /**
   * Revoke employee access from a user
   */
  async revokeEmployeeAccess(
    employeeId: number,
    userId: number
  ): Promise<ApiResponse<void>> {
    const response = await apiClient.delete(
      API_ENDPOINTS.employees.userAccess(employeeId, userId)
    );
    return response;
  }

}

export const employeeService = new EmployeeService();
