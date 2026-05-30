import { Page } from '@playwright/test';
import type { Request } from '@playwright/test';

export class ApiHelpers {
  constructor(private page: Page) {}

  // Mock API responses for testing
  async mockApiResponse(url: string, response: unknown, status: number = 200) {
    await this.page.route(url, async (route) => {
      await route.fulfill({
        status,
        contentType: 'application/json',
        body: JSON.stringify(response),
      });
    });
  }

  // Mock authentication endpoints
  async mockLoginSuccess(user: { email: string; password: string }) {
    await this.mockApiResponse('**/api/auth/login', {
      success: true,
      token: 'mock-jwt-token',
      user: {
        id: '1',
        email: user.email,
        role: 'admin',
        name: 'Test User'
      }
    });
  }

  async mockLoginFailure() {
    await this.mockApiResponse('**/api/auth/login', {
      success: false,
      error: 'Invalid credentials'
    }, 401);
  }

  async mockRateLimit() {
    await this.mockApiResponse('**/api/auth/login', {
      success: false,
      error: 'Too many login attempts. Please try again later.'
    }, 429);
  }

  // Mock employee endpoints
  async mockEmployeesList(employees: Record<string, unknown>[] = []) {
    await this.mockApiResponse('**/api/employees', {
      success: true,
      data: employees,
      total: employees.length
    });
  }

  async mockCreateEmployee(employee: Record<string, unknown>) {
    await this.mockApiResponse('**/api/employees', {
      success: true,
      data: { ...employee, id: 'mock-id' }
    }, 201);
  }

  async mockUpdateEmployee(id: string, employee: Record<string, unknown>) {
    await this.mockApiResponse(`**/api/employees/${id}`, {
      success: true,
      data: { ...employee, id }
    });
  }

  async mockDeleteEmployee(id: string) {
    await this.mockApiResponse(`**/api/employees/${id}`, {
      success: true,
      message: 'Employee deleted successfully'
    }, 204);
  }

  // Mock project endpoints
  async mockProjectsList(projects: Record<string, unknown>[] = []) {
    await this.mockApiResponse('**/api/projects', {
      success: true,
      data: projects,
      total: projects.length
    });
  }

  // Mock timesheet endpoints
  async mockTimesheetList(timesheets: Record<string, unknown>[] = []) {
    await this.mockApiResponse('**/api/timesheet', {
      success: true,
      data: timesheets,
      total: timesheets.length
    });
  }

  async mockApproveTimesheet(id: string) {
    await this.mockApiResponse(`**/api/timesheet/${id}/approve`, {
      success: true,
      message: 'Timesheet approved successfully'
    });
  }

  async mockRejectTimesheet(id: string) {
    await this.mockApiResponse(`**/api/timesheet/${id}/reject`, {
      success: true,
      message: 'Timesheet rejected'
    });
  }

  // Mock dashboard data
  async mockDashboardStats() {
    await this.mockApiResponse('**/api/dashboard/stats', {
      success: true,
      data: {
        totalEmployees: 25,
        activeProjects: 8,
        pendingApprovals: 12,
        totalPayroll: 125000
      }
    });
  }

  // Network error simulation
  async mockNetworkError(url: string) {
    await this.page.route(url, async (route) => {
      await route.abort('failed');
    });
  }

  // Slow network simulation
  async mockSlowResponse(url: string, delay: number = 2000) {
    await this.page.route(url, async (route) => {
      await new Promise(resolve => setTimeout(resolve, delay));
      await route.continue();
    });
  }

  // Clear all mocks
  async clearMocks() {
    await this.page.unrouteAll();
  }

  // Wait for API call
  async waitForApiCall(url: string, timeout: number = 5000) {
    await this.page.waitForResponse(response =>
      response.url().includes(url) && response.status() < 400,
      { timeout }
    );
  }

  // Intercept and validate API calls
  async interceptApiCall(url: string, validateCallback: (request: Request) => void) {
    await this.page.route(url, async (route) => {
      const request = route.request();
      validateCallback(request);
      await route.continue();
    });
  }
}
