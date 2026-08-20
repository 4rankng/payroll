import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  ProjectEmployeeAssignment,
  ProjectEmployeeListResponse,
  ProjectEmployeeListParams,
  EmployeeProjectListResponse,
  EmployeeProjectListParams,
  AssignEmployeeResponse,
  AssignEmployeeRequest,
  RemoveEmployeeResponse,
  RemoveEmployeeRequest,
  UpdateAssignmentResponse,
  ChangePaymentScheduleRequest,
  ChangePaymentScheduleResponse,
  CancelScheduleChangeResponse,
  PendingScheduleChangesResponse
} from '@/types/api/project-employee.types';

/**
 * Project-Employee Assignment API Service
 * Manages employee assignments to projects, including assignment lifecycle and timesheet summaries
 */
export class ProjectEmployeeService {
  private readonly baseUrl = '/project-employees';

  // ========== PROJECT-BASED OPERATIONS ==========

  /**
   * Get all employee assignments for a specific project
   */
  async getProjectEmployees(
    projectId: number,
    params: ProjectEmployeeListParams = {}
  ): Promise<ProjectEmployeeListResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<ProjectEmployeeAssignment[]>(
      `/projects/${projectId}/employees${queryString}`
    );
    return {
      status: "success",
      data: response.data ?? [],
      pagination: response.pagination ?? { page: 1, pageSize: 50, totalPages: 0, totalRecords: 0 },
      message: response.message ?? "",
    };
  }

  /**
   * Assign employee to a project
   */
  async assignEmployeeToProject(
    projectId: number,
    data: AssignEmployeeRequest
  ): Promise<AssignEmployeeResponse> {
    const response = await apiClient.post<AssignEmployeeResponse>(
      `/projects/${projectId}/employees`,
      [data]
    );
    return response.data!;
  }

  /**
   * Check if project allows employee assignments (not completed or cancelled)
   */
  private canAssignToProject(project: { status: string }): boolean {
    return project.status !== 'completed' && project.status !== 'cancelled';
  }


  /**
   * Remove employees from project
   */
  async removeEmployeesFromProject(
    projectId: number,
    data: RemoveEmployeeRequest[]
  ): Promise<RemoveEmployeeResponse> {
    const response = await apiClient.post<RemoveEmployeeResponse>(
      `/projects/${projectId}/employees/remove`,
      data
    );
    return response.data!;
  }

  // ========== EMPLOYEE-BASED OPERATIONS ==========

  /**
   * Get all project assignments for a specific employee
   */
  async getEmployeeProjects(
    employeeId: number,
    params: EmployeeProjectListParams = {}
  ): Promise<EmployeeProjectListResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<ProjectEmployeeAssignment[]>(
      `/employees/${employeeId}/projects${queryString}`
    );
    return {
      status: "success",
      data: response.data ?? [],
      message: response.message ?? "",
    };
  }

  /**
   * Update employee project assignment via employee endpoint
   */
  async updateEmployeeProjectAssignment(
    employeeId: number,
    data: {
      project_id: number;
      position?: string;
      start_date?: string;
      end_date?: string | null;
    }
  ): Promise<UpdateAssignmentResponse> {
    const response = await apiClient.put<UpdateAssignmentResponse>(
      `/employees/${employeeId}/projects`,
      data
    );
    return response.data!;
  }

  // ========== ASSIGNMENT-BASED OPERATIONS ==========

  // ========== PAYMENT SCHEDULE MANAGEMENT ==========

  /**
   * Request payment schedule change for an assignment
   * POST /api/v1/project-employees/:id/payment-schedule
   */
  async changePaymentSchedule(
    assignmentId: number,
    data: ChangePaymentScheduleRequest
  ): Promise<ChangePaymentScheduleResponse | null> {
    const response = await apiClient.post<ChangePaymentScheduleResponse>(
      `${this.baseUrl}/${assignmentId}/payment-schedule`,
      data
    );
    // Backend returns data:null for this in-place mutation (handler calls
    // response.Success(c, nil, ...)), so do not assert non-null — return the
    // honest nullable value. Callers (useChangePaymentSchedule.onSuccess)
    // intentionally ignore the payload and refresh via invalidateQueries.
    return response.data ?? null;
  }

  /**
   * Cancel pending payment schedule change
   * DELETE /api/v1/project-employees/:id/payment-schedule
   */
  async cancelScheduleChange(assignmentId: number): Promise<CancelScheduleChangeResponse> {
    const response = await apiClient.delete<CancelScheduleChangeResponse>(
      `${this.baseUrl}/${assignmentId}/payment-schedule`
    );
    return response.data!;
  }

  /**
   * Get all pending schedule changes (ADMIN only)
   * GET /api/v1/project-employees/pending-schedule-changes
   */
  async getPendingScheduleChanges(): Promise<PendingScheduleChangesResponse> {
    const response = await apiClient.get<PendingScheduleChangesResponse>(
      `${this.baseUrl}/pending-schedule-changes`
    );
    return response.data!;
  }

  // ========== CONVENIENCE METHODS ==========

  /**
   * Get active employee assignments for a project
   */
  async getActiveProjectEmployees(projectId: number): Promise<ProjectEmployeeListResponse> {
    return this.getProjectEmployees(projectId, {
      status: 'active',
      sortBy: 'start_date',
      sortOrder: 'desc'
    });
  }

  /**
   * Get active project assignments for an employee
   */
  async getActiveEmployeeProjects(employeeId: number): Promise<EmployeeProjectListResponse> {
    return this.getEmployeeProjects(employeeId, {
      status: 'active',
      sortBy: 'start_date',
      sortOrder: 'desc'
    });
  }

  /**
   * Check if employee is assigned to project
   */
  async isEmployeeAssignedToProject(
    employeeId: number,
    projectId: number
  ): Promise<boolean> {
    try {
      const response = await this.getEmployeeProjects(employeeId, { status: 'active' });
      return response.data.some(assignment =>
        assignment.project_id === projectId && assignment.status === 'active'
      );
    } catch (error) {
      return false;
    }
  }

  /**
   * Assign single employee to project with validation
   */
  async assignEmployee(
    projectId: number,
    employeeId: number,
    options: {
      employee_code?: string;
      start_date: string;
      end_date?: string;
      validate_overlap?: boolean;
      project?: { status: string };
    }
  ): Promise<AssignEmployeeResponse> {
    // Check if project allows new assignments
    if (options.project && !this.canAssignToProject(options.project)) {
      throw new Error('Cannot assign employees to completed or cancelled projects');
    }

    // Optional: Check for existing assignment if validation is requested
    if (options.validate_overlap) {
      const isAssigned = await this.isEmployeeAssignedToProject(employeeId, projectId);
      if (isAssigned) {
        throw new Error('Employee is already assigned to this project');
      }
    }

    return this.assignEmployeeToProject(projectId, {
      employee_id: employeeId,
      employee_code: options.employee_code,
      start_date: options.start_date,
      end_date: options.end_date
    } as unknown as AssignEmployeeRequest);
  }


  /**
   * Get assignment statistics for a project
   */
  async getProjectAssignmentStats(projectId: number): Promise<{
    data: {
      total_assignments: number;
      active_assignments: number;
      ended_assignments: number;
      unique_employees: number;
      average_assignment_duration: number;
    };
  }> {
    const response = await apiClient.get(API_ENDPOINTS.projects.assignmentStats(projectId));
    return response.data! as {
      data: {
        total_assignments: number;
        active_assignments: number;
        ended_assignments: number;
        unique_employees: number;
        average_assignment_duration: number;
      };
    };
  }

  /**
   * Get assignment statistics for an employee
   */
  async getEmployeeAssignmentStats(employeeId: number): Promise<{
    data: {
      total_assignments: number;
      active_assignments: number;
      ended_assignments: number;
      unique_projects: number;
      total_working_days: number;
      average_assignment_duration: number;
    };
  }> {
    const response = await apiClient.get(API_ENDPOINTS.employees.assignmentStats(employeeId));
    return response.data! as {
      data: {
        total_assignments: number;
        active_assignments: number;
        ended_assignments: number;
        unique_projects: number;
        total_working_days: number;
        average_assignment_duration: number;
      };
    };
  }

  /**
   * Toggle check-in enabled status for a specific employee in a project
   */
  async toggleCheckInEnabled(
    projectId: number,
    employeeId: number,
    enabled: boolean
  ): Promise<{ status: "success"; message: string }> {
    const response = await apiClient.patch<{ status: "success"; message: string }>(
      `/projects/${projectId}/employees/${employeeId}/checkin-enabled`,
      { check_in_enabled: enabled }
    );
    return response.data!;
  }

  /**
   * Bulk toggle check-in enabled status for multiple employees in a project
   */
  async bulkToggleCheckInEnabled(
    projectId: number,
    employeeIds: number[],
    enabled: boolean
  ): Promise<{ status: "success"; message: string }> {
    const response = await apiClient.patch<{ status: "success"; message: string }>(
      `/projects/${projectId}/employees/checkin-enabled/bulk`,
      { employee_ids: employeeIds, check_in_enabled: enabled }
    );
    return response.data!;
  }

  /**
   * Cancel a pending (not yet activated) check-in enable.
   * The enable activates on day 1 of the next month; cancel removes it.
   */
  async cancelPendingCheckInEnable(
    projectId: number,
    employeeId: number
  ): Promise<{ status: "success"; message: string }> {
    const response = await apiClient.delete<{ status: "success"; message: string }>(
      `/projects/${projectId}/employees/${employeeId}/checkin-enabled`
    );
    return response.data!;
  }
}

// Export singleton instance
export const projectEmployeeService = new ProjectEmployeeService();
