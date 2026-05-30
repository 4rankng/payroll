import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
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
    return apiClient.get<ProjectEmployeeListResponse>(
      `/projects/${projectId}/employees${queryString}`
    );
  }

  /**
   * Assign employee to a project
   */
  async assignEmployeeToProject(
    projectId: number,
    data: AssignEmployeeRequest
  ): Promise<AssignEmployeeResponse> {
    return apiClient.post<AssignEmployeeResponse>(
      `/projects/${projectId}/employees`,
      [data]
    );
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
    return apiClient.post<RemoveEmployeeResponse>(
      `/projects/${projectId}/employees/remove`,
      data
    );
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
    return apiClient.get<EmployeeProjectListResponse>(
      `/employees/${employeeId}/projects${queryString}`
    );
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
    return apiClient.put<UpdateAssignmentResponse>(
      `/employees/${employeeId}/projects`,
      data
    );
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
  ): Promise<ChangePaymentScheduleResponse> {
    return apiClient.post<ChangePaymentScheduleResponse>(
      `${this.baseUrl}/${assignmentId}/payment-schedule`,
      data
    );
  }

  /**
   * Cancel pending payment schedule change
   * DELETE /api/v1/project-employees/:id/payment-schedule
   */
  async cancelScheduleChange(assignmentId: number): Promise<CancelScheduleChangeResponse> {
    return apiClient.delete<CancelScheduleChangeResponse>(
      `${this.baseUrl}/${assignmentId}/payment-schedule`
    );
  }

  /**
   * Get all pending schedule changes (ADMIN only)
   * GET /api/v1/project-employees/pending-schedule-changes
   */
  async getPendingScheduleChanges(): Promise<PendingScheduleChangesResponse> {
    return apiClient.get<PendingScheduleChangesResponse>(
      `${this.baseUrl}/pending-schedule-changes`
    );
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
    });
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
    return apiClient.get(API_ENDPOINTS.projects.assignmentStats(projectId));
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
    return apiClient.get(API_ENDPOINTS.employees.assignmentStats(employeeId));
  }

  /**
   * Toggle check-in enabled status for a specific employee in a project
   */
  async toggleCheckInEnabled(
    projectId: number,
    employeeId: number,
    enabled: boolean
  ): Promise<{ status: "success"; message: string }> {
    return apiClient.patch(
      `/projects/${projectId}/employees/${employeeId}/checkin-enabled`,
      { check_in_enabled: enabled }
    );
  }

  /**
   * Bulk toggle check-in enabled status for multiple employees in a project
   */
  async bulkToggleCheckInEnabled(
    projectId: number,
    employeeIds: number[],
    enabled: boolean
  ): Promise<{ status: "success"; message: string }> {
    return apiClient.patch(
      `/projects/${projectId}/employees/checkin-enabled/bulk`,
      { employee_ids: employeeIds, check_in_enabled: enabled }
    );
  }
}

// Export singleton instance
export const projectEmployeeService = new ProjectEmployeeService();
