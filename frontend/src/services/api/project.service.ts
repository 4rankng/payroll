import { apiClient, buildQueryString, ApiResponse } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  Project,
  ProjectSummary,
  PartnerProjectSummary,
  CreateProjectData,
  UpdateProjectData,
  UpdateProjectStatusData,
  ProjectFilters,
  ProjectEmployee,
  AssignEmployeeData,
  ProjectAssignmentResponse,
  PayRateConfig,
  CreatePayRateData,
  ProjectUser,
  ProjectUsersResponse,
  GrantProjectAccessData,
  ProjectTimesheetFilters,
} from '@/types/api/project.types';
import type { Timesheet } from '@/types/api/timesheet.types';

class ProjectService {
  /**
   * Get projects summary statistics
   */
  async getSummary(): Promise<ProjectSummary> {
    const response = await apiClient.get<ProjectSummary>(
      API_ENDPOINTS.projects.summary
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get partner project summary statistics (partner role only)
   */
  async getPartnerSummary(): Promise<PartnerProjectSummary> {
    const response = await apiClient.get<PartnerProjectSummary>(
      API_ENDPOINTS.projects.partnerSummary
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get paginated list of projects
   */
  async getProjects(filters?: ProjectFilters) {
    const queryString = filters
      ? (() => {
          const { sortBy, sortOrder, ...rest } = filters;
          return buildQueryString({
            ...rest,
            sort_by: sortBy,
            sort_order: sortOrder,
          });
        })()
      : '';
    const response = await apiClient.get<Project[]>(
      `${API_ENDPOINTS.projects.base}${queryString}`
    );
    return response;
  }

  /**
   * Search projects using dedicated search functionality
   */
  async searchProjects(params: {
    search: string;
    pageSize?: number;
    status?: string | string[];
  }) {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<Project[]>(
      `${API_ENDPOINTS.projects.base}${queryString}`
    );
    return response;
  }

  /**
   * Get single project by ID
   */
  async getProjectById(id: number): Promise<Project> {
    const response = await apiClient.get<Project>(
      API_ENDPOINTS.projects.byId(id)
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create new project
   */
  async createProject(data: CreateProjectData): Promise<ApiResponse<Project>> {
    const response = await apiClient.post<Project>(
      API_ENDPOINTS.projects.base,
      data
    );
    return response;
  }

  /**
   * Update existing project
   */
  async updateProject(id: number, data: UpdateProjectData): Promise<ApiResponse<Project>> {
    const response = await apiClient.put<Project>(
      API_ENDPOINTS.projects.byId(id),
      data
    );
    return response;
  }

  /**
   * Update project status
   */
  async updateProjectStatus(id: number, data: UpdateProjectStatusData): Promise<ApiResponse<Project>> {
    const response = await apiClient.put<Project>(
      API_ENDPOINTS.projects.status(id),
      data
    );
    return response;
  }

  /**
   * Start project (change status to active)
   */
  async startProject(id: number): Promise<ApiResponse<Project>> {
    return this.updateProjectStatus(id, { status: 'active' });
  }

  /**
   * Complete project (change status to completed)
   */
  async completeProject(id: number): Promise<ApiResponse<Project>> {
    return this.updateProjectStatus(id, { status: 'completed' });
  }

  /**
   * Pause project (change status to paused)
   */
  async pauseProject(id: number): Promise<ApiResponse<Project>> {
    return this.updateProjectStatus(id, { status: 'paused' });
  }

  /**
   * Resume project (change status to active)
   */
  async resumeProject(id: number): Promise<ApiResponse<Project>> {
    return this.updateProjectStatus(id, { status: 'active' });
  }

  /**
   * Cancel project (change status to cancelled)
   */
  async cancelProject(id: number): Promise<ApiResponse<Project>> {
    return this.updateProjectStatus(id, { status: 'cancelled' });
  }

  /**
   * Mark project as inactive (soft delete)
   */
  async deleteProject(id: number): Promise<ApiResponse<void>> {
    const response = await apiClient.delete<void>(API_ENDPOINTS.projects.byId(id));
    return response;
  }

  /**
   * Get project employees
   */
  async getProjectEmployees(
    projectId: number,
    filters?: { page?: number; pageSize?: number; status?: string }
  ) {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<ProjectEmployee[]>(
      `${API_ENDPOINTS.projects.employees(projectId)}${queryString}`
    );
    return response;
  }

  /**
   * Assign employee to project (batch assignment)
   */
  async assignEmployee(projectId: number, data: AssignEmployeeData[]): Promise<ApiResponse<ProjectAssignmentResponse>> {
    const response = await apiClient.post<ProjectAssignmentResponse>(
      API_ENDPOINTS.projects.assignEmployee(projectId),
      data // Send as array for batch assignment
    );
    return response;
  }

  /**
   * Remove employee from project
   */
  async removeEmployee(
    projectId: number,
    employeeId: number,
    lastDate?: string
  ): Promise<ApiResponse<void>> {
    const payload = [{
      employee_id: employeeId,
      ...(lastDate && { last_date: lastDate })
    }];

    const response = await apiClient.post<void>(
      API_ENDPOINTS.projects.employeesRemove(projectId),
      payload
    );
    return response;
  }

  /**
   * Remove multiple employees from project
   */
  async removeEmployees(projectId: number, data: Array<{
    employee_id: number;
    last_date: string;
  }>) {
    const response = await apiClient.post(
      API_ENDPOINTS.projects.employeesRemove(projectId),
      data
    );
    return response.data;
  }

  /**
   * Bulk assign employees to project
   */
  async bulkAssignEmployees(projectId: number, assignments: AssignEmployeeData[]) {
    const response = await apiClient.post(
      API_ENDPOINTS.projects.assignEmployee(projectId),
      assignments // Direct array assignment, no wrapper object
    );
    return response.data;
  }

  /**
   * Get project timesheets
   */
  async getProjectTimesheets(
    projectId: number,
    filters?: ProjectTimesheetFilters
  ) {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<Timesheet[]>(
      `${API_ENDPOINTS.projects.timesheets(projectId)}${queryString}`
    );
    return response;
  }

  /**
   * Import timesheets for project
   */
  async importTimesheets(projectId: number, file: File, onProgress?: (progress: number) => void) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('project_id', projectId.toString());

    const response = await apiClient.upload(
      API_ENDPOINTS.projects.timesheetsImport(projectId),
      formData,
      onProgress
    );
    return response.data;
  }

  /**
   * Get project pay rates
   */
  async getProjectPayRates(
    projectId: number,
    filters?: { status?: string; date?: string }
  ) {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get<PayRateConfig[]>(
      `${API_ENDPOINTS.projects.payrate(projectId)}${queryString}`
    );
    return response.data;
  }

  /**
   * Get active pay rate for project
   */
  async getActivePayRate(projectId: number, date?: string): Promise<PayRateConfig> {
    const queryString = date ? buildQueryString({ date }) : '';
    const response = await apiClient.get<PayRateConfig>(
      `${API_ENDPOINTS.projects.payrateActive(projectId)}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Create pay rate configuration
   */
  async createPayRate(projectId: number, data: CreatePayRateData): Promise<ApiResponse<PayRateConfig>> {
    const response = await apiClient.post<PayRateConfig>(
      API_ENDPOINTS.projects.payrate(projectId),
      data
    );
    return response;
  }

  /**
   * Update pay rate configuration
   */
  async updatePayRate(
    projectId: number,
    payrateId: number,
    data: Partial<CreatePayRateData>
  ): Promise<PayRateConfig> {
    const response = await apiClient.put<PayRateConfig>(
      API_ENDPOINTS.projects.payrateById(projectId, payrateId),
      data
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Delete pay rate configuration
   */
  async deletePayRate(projectId: number, payrateId: number): Promise<void> {
    await apiClient.delete(
      API_ENDPOINTS.projects.payrateById(projectId, payrateId)
    );
  }

  /**
   * Get project financial summary
   */
  async getFinancialSummary(projectId: number) {
    const response = await apiClient.get(
      API_ENDPOINTS.projects.financial(projectId)
    );
    return response.data;
  }

  /**
   * Update project financials
   */
  async updateFinancials(projectId: number, data: unknown) {
    const response = await apiClient.put(
      API_ENDPOINTS.projects.financial(projectId),
      data
    );
    return response.data;
  }

  /**
   * Get project revenue history
   */
  async getRevenueHistory(
    projectId: number,
    filters?: { page?: number; pageSize?: number; fromDate?: string; toDate?: string }
  ) {
    const queryString = filters ? buildQueryString(filters) : '';
    const response = await apiClient.get(
      `${API_ENDPOINTS.projects.revenue(projectId)}${queryString}`
    );
    return response.data;
  }

  /**
   * Get users who have access to a project
   */
  async getProjectUsers(projectId: number): Promise<ProjectUser[]> {
    try {
      const response = await apiClient.get<unknown>(API_ENDPOINTS.projects.users(projectId));

      // Handle the response - backend may return either array directly or wrapped in {data: [...]}
      const data = response.data as unknown;
      if (Array.isArray(data)) {
        return data as ProjectUser[];
      }
      if (data && typeof data === 'object' && Array.isArray((data as { data?: unknown }).data)) {
        return (data as { data: ProjectUser[] }).data;
      }
      return [];
    } catch {
      return [];
    }
  }

  /**
   * Grant access to a project for a specific user
   */
  async grantProjectAccess(projectId: number, data: GrantProjectAccessData): Promise<ApiResponse<void>> {
    const response = await apiClient.post<void>(
      API_ENDPOINTS.projects.users(projectId),
      data
    );
    return response;
  }

  /**
   * Revoke access to a project from a specific user
   */
  async revokeProjectAccess(projectId: number, userId: number): Promise<ApiResponse<void>> {
    const response = await apiClient.delete<void>(
      API_ENDPOINTS.projects.userAccess(projectId, userId)
    );
    return response;
  }

}

export const projectService = new ProjectService();
