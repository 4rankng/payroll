import { readFileSync } from 'fs';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';
import { Page } from '@playwright/test';
import type { Request } from '@playwright/test';

const FIXTURE_DIR = join(dirname(fileURLToPath(import.meta.url)), '..', 'fixtures', 'visual-baseline');

/**
 * Build a structurally valid JWT the app can decode client-side
 * (authManager.isTokenValid only base64-decodes the payload and checks exp —
 * the signature is never verified in the browser).
 */
function makeMockJwt(role: string, username: string): string {
  const enc = (obj: unknown) => Buffer.from(JSON.stringify(obj)).toString('base64url');
  const now = Math.floor(Date.now() / 1000);
  const header = enc({ alg: 'HS256', typ: 'JWT' });
  const payload = enc({
    exp: now + 86_400,
    user_id: 1,
    username,
    role,
    sub: '1',
    nbf: now - 60,
    iat: now,
    jti: 'mock-jti',
  });
  return `${header}.${payload}.mock-signature-not-verified`;
}

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

  // Mock authentication endpoints. The app POSTs /api/v1/auth/login and reads
  // an ApiResponse<{ access_token, user }> body (see auth.service.login), and
  // the stored token must survive jwtDecode on the next mount — hence the
  // real JWT structure above.
  async mockLoginSuccess(user: { email: string; password: string }, role: 'admin' | 'partner' = 'admin') {
    await this.mockApiResponse('**/api/v1/auth/login', {
      status: 'success',
      data: {
        access_token: makeMockJwt(role, user.email),
        user: {
          id: 1,
          email: user.email,
          fullname: 'Test User',
          username: user.email.split('@')[0],
          role,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      },
    });
  }

  async mockLoginFailure() {
    await this.mockApiResponse('**/api/v1/auth/login', {
      status: 'error',
      message: 'Thông tin đăng nhập không hợp lệ. Vui lòng thử lại.',
      http_status: 401,
    }, 401);
  }

  async mockRateLimit() {
    // The axios interceptor owns the 429 message ("Quá nhiều yêu cầu…");
    // this body only supplies retry_after.
    await this.mockApiResponse('**/api/v1/auth/login', {
      status: 'error',
      message: 'Quá nhiều lần đăng nhập. Vui lòng thử lại sau ít phút.',
      http_status: 429,
      retry_after: 60,
    }, 429);
  }

  async mockLogout() {
    await this.mockApiResponse('**/api/v1/auth/logout', {
      status: 'success',
      message: 'Đăng xuất thành công',
    });
  }

  /**
   * Pin the post-login dashboard data with captured real responses (the same
   * certified fixtures the visual-baseline harness uses). Without these, the
   * fake token gets a 401 from the live backend and the global interceptor
   * force-logs the session out mid-test.
   */
  async mockDashboardData(role: 'admin' | 'partner') {
    const pins: Array<[pattern: string, fixture: string]> = role === 'admin'
      ? [
          ['**/api/v1/dashboard/summary*', 'admin--dashboard-summary.json'],
          ['**/api/v1/dashboard/bank-usage/projects*', 'admin--dashboard-bank-usage-projects.json'],
          ['**/api/v1/dashboard/monthly-financials*', 'admin--dashboard-monthly-financials.json'],
          ['**/api/v1/dashboard/employee-activity*', 'admin--dashboard-employee-activity.json'],
          ['**/api/v1/dashboard/new-employees*', 'admin--dashboard-new-employees.json'],
          ['**/api/v1/dashboard/recent-activities*', 'admin--dashboard-recent-activities.json'],
          ['**/api/v1/dashboard/top-paid-employees*', 'admin--dashboard-top-paid-employees.json'],
          ['**/api/v1/dashboard/check-in-health*', 'admin--dashboard-check-in-health.json'],
          ['**/api/v1/dashboard/salary-distribution*', 'admin--dashboard-salary-distribution.json'],
          ['**/api/v1/dashboard/project-weekly-profit*', 'admin--dashboard-project-weekly-profit.json'],
          ['**/api/v1/dashboard/project-profitability*', 'admin--dashboard-project-profitability.json'],
          ['**/api/v1/timesheets/grouped*', 'admin--timesheets-grouped.json'],
          ['**/api/v1/timesheets/summary*', 'admin--timesheets-summary.json'],
          ['**/api/v1/timesheets/edit-requests*', 'admin--timesheets-edit-requests.json'],
          ['**/api/v1/timesheets/cash-readiness*', 'admin--timesheets-cash-readiness.json'],
          ['**/api/v1/timesheets*', 'admin--timesheets-pending.json'],
          ['**/api/v1/users/summary', 'admin--users-summary.json'],
          ['**/api/v1/users*', 'admin--users.json'],
          ['**/api/v1/employees/summary', 'admin--employees-summary.json'],
          ['**/api/v1/employees/missing-bank-details', 'admin--employees-missing-bank-details.json'],
          ['**/api/v1/employees*', 'admin--employees.json'],
          ['**/api/v1/projects*', 'admin--projects.json'],
        ]
      : [
          ['**/api/v1/dashboard/partner*', 'partner--dashboard-partner.json'],
          ['**/api/v1/projects/partner-summary', 'partner--projects-partner-summary.json'],
          ['**/api/v1/projects*', 'partner--projects.json'],
          ['**/api/v1/timesheets/grouped*', 'partner--timesheets-grouped.json'],
          ['**/api/v1/timesheets/summary*', 'partner--timesheets-summary.json'],
          ['**/api/v1/timesheets/edit-requests*', 'partner--timesheets-edit-requests.json'],
          ['**/api/v1/timesheets*', 'partner--timesheets.json'],
          ['**/api/v1/employees?*', 'partner--employees.json'],
          ['**/api/v1/employees', 'partner--employees.json'],
          ['**/api/v1/employees/missing-bank-details', 'partner--employees-missing-bank-details.json'],
        ];

    // Layout chrome: empty notification feed (shape mirrors the harness).
    await this.mockApiResponse('**/api/v1/notifications/unread*', {
      status: 'success',
      data: { notifications: [], count: 0 },
    });
    await this.mockApiResponse('**/api/v1/notifications?*', {
      status: 'success',
      data: { notifications: [], count: 0 },
    });

    // AuthContext validates the session server-side on mount via
    // GET /auth/me (getCurrentUser → response.data = the User object); with
    // the fake JWT the live backend 401s and the interceptor force-logs out.
    // Answer with the same user the login mock issued.
    await this.mockApiResponse('**/api/v1/auth/me', {
      status: 'success',
      data: {
        id: 1,
        email: role === 'admin' ? 'admin@example.com' : 'partner@example.com',
        fullname: 'Test User',
        username: role,
        role,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    });

    // Global metadata lookups fired on authenticated layout mount (unpinned
    // requests 401 against the live backend with the fake token, and the
    // interceptor force-logs the session out). Shapes mirror the consumers
    // (AddTransactionSheet expects { transaction_types, statuses }; ledger
    // accounts render from an array).
    await this.mockApiResponse('**/api/v1/transactions/metadata', {
      status: 'success',
      data: { transaction_types: [], statuses: [] },
    });
    await this.mockApiResponse('**/api/v1/ledger/accounts/metadata', {
      status: 'success',
      data: [],
    });

    for (const [pattern, fixture] of pins) {
      await this.page.route(pattern, (route) => route.fulfill({ path: join(FIXTURE_DIR, fixture) }));
    }
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
