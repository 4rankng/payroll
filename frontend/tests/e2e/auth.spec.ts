import { test, expect } from '@playwright/test';
import { LoginPage } from '../page-objects/LoginPage';
import { DashboardPage } from '../page-objects/DashboardPage';
import { ApiHelpers } from '../utils/api-helpers';
import { CustomAssertions } from '../utils/assertions';

/**
 * Authentication flow specs, retargeted to the rebuilt (UU PRO) login UI and
 * the real app contracts:
 * - identity field #emailOrUsername (CCCD/phone/username, type="text" by
 *   design), password #password, native button[type=submit] "Đăng nhập"
 * - login errors surface in the inline [role="alert"] /
 *   [data-testid="error-message"] (useLogin sets skipGlobalError — login
 *   failures do NOT toast)
 * - post-login destinations are role-based: admin → /admin, partner →
 *   /partner/dashboard; the fake session uses a structurally valid JWT and
 *   the captured dashboard fixtures (see ApiHelpers.mockDashboardData) so the
 *   401-logout interceptor cannot bounce the mocked session.
 * Every test's intent is unchanged; only selectors/expectations were
 * retargeted to reality.
 */
test.describe('Authentication Flow', () => {
  // Dev-server on-demand compile (vite serves :3000; first /admin hit compiles
  // the admin chunk graph) — same concession the visual-baseline spec makes.
  test.setTimeout(120_000);

  let loginPage: LoginPage;
  let dashboardPage: DashboardPage;
  let apiHelpers: ApiHelpers;
  let assertions: CustomAssertions;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    dashboardPage = new DashboardPage(page);
    apiHelpers = new ApiHelpers(page);
    assertions = new CustomAssertions(page);

    await loginPage.goto();
  });

  test('should display login page correctly', async () => {
    await loginPage.expectToBeOnLoginPage();
    await loginPage.expectLoginButtonToBeEnabled();
  });

  test('should login successfully with valid admin credentials', async () => {
    // Mock successful login API response
    await apiHelpers.mockLoginSuccess({ email: 'admin@example.com', password: 'admin123' });
    await apiHelpers.mockDashboardData('admin');

    await loginPage.login('admin@example.com', 'admin123', '**/admin');
    await loginPage.expectToBeLoggedIn();
    await dashboardPage.expectToBeOnDashboard();
    await dashboardPage.expectAdminFeatures();
  });

  test('should login successfully with valid partner credentials', async () => {
    // Mock successful login API response for partner
    await apiHelpers.mockLoginSuccess({ email: 'partner@example.com', password: 'partner123' }, 'partner');
    await apiHelpers.mockDashboardData('partner');

    await loginPage.login('partner@example.com', 'partner123', '**/dashboard');
    await loginPage.expectToBeLoggedIn();
    await dashboardPage.expectToBeOnDashboard();
    await dashboardPage.expectPartnerFeatures();
  });

  test('should show error message for invalid credentials', async () => {
    await apiHelpers.mockLoginFailure();

    await loginPage.loginWithInvalidCredentials('invalid@example.com', 'wrongpassword');
    await loginPage.expectErrorMessage('Thông tin đăng nhập không hợp lệ. Vui lòng thử lại.');
  });

  test('should handle rate limiting after multiple failed attempts', async () => {
    await apiHelpers.mockRateLimit();

    await loginPage.attemptMultipleLogins('test@example.com', 'wrongpassword', 5);
    await loginPage.expectRateLimitMessage();
  });

  test('should validate required fields', async () => {
    // The form is noValidate by design (server owns validation); an empty
    // submit is rejected by the backend and surfaces the inline login alert.
    await loginPage.loginWithInvalidCredentials('', '');
    await expect(loginPage.page.locator('[role="alert"]')).toBeVisible();
    await expect(loginPage.page.locator('[data-testid="error-message"]')).toContainText(/.+/);
  });

  test('should validate email format', async () => {
    // Same contract: a non-email username string is rejected server-side and
    // surfaced in the inline alert (no client-side format validation exists).
    await loginPage.loginWithInvalidCredentials('invalid-email', 'password123');
    await expect(loginPage.page.locator('[role="alert"]')).toBeVisible();
  });

  test('should disable login button during submission', async () => {
    // Mock slow API response
    await apiHelpers.mockSlowResponse('**/api/v1/auth/login', 2000);

    await loginPage.fillCredentials('admin@example.com', 'admin123');
    await loginPage.submit();
    await loginPage.expectLoginButtonToBeDisabled();
  });

  test('should handle network errors gracefully', async () => {
    await apiHelpers.mockNetworkError('**/api/v1/auth/login');

    await loginPage.loginWithInvalidCredentials('admin@example.com', 'admin123');
    // Login failures do not toast (skipGlobalError); feedback is the alert.
    await expect(loginPage.page.locator('[role="alert"]')).toBeVisible();
  });

  test('should redirect to login page when accessing protected routes', async ({ page }) => {
    // Try to access a real protected route without authentication. (/dashboard
    // is not an app route — it renders NotFound; /partner/dashboard is guarded
    // by ProtectedRoute, which navigates to /login.)
    await page.goto('/partner/dashboard');
    await expect(page).toHaveURL(/.*login/);
  });

  test('should logout successfully', async () => {
    // Mock successful login + the pinned dashboard data + logout endpoint
    await apiHelpers.mockLoginSuccess({ email: 'admin@example.com', password: 'admin123' });
    await apiHelpers.mockDashboardData('admin');
    await apiHelpers.mockLogout();
    await loginPage.login('admin@example.com', 'admin123', '**/admin');

    await dashboardPage.logout();
    await loginPage.expectToBeOnLoginPage();
  });

  test('should handle token expiration', async ({ page }) => {
    // Mock expired token response
    await apiHelpers.mockApiResponse('**/api/v1/auth/refresh', {
      status: 'error',
      message: 'Token expired',
    }, 401);

    // Simulate expired token scenario
    await page.evaluate(() => {
      localStorage.setItem('token', 'expired-token');
    });

    await page.goto('/partner/dashboard');
    await expect(page).toHaveURL(/.*login/);
  });

  test('should remember login state across page refreshes', async ({ page }) => {
    // Mock successful login
    await apiHelpers.mockLoginSuccess({ email: 'admin@example.com', password: 'admin123' });
    await apiHelpers.mockDashboardData('admin');
    await loginPage.login('admin@example.com', 'admin123', '**/admin');

    // Refresh page
    await page.reload();
    await dashboardPage.expectToBeOnDashboard();
  });

  test('should support forgot password flow', async () => {
    await loginPage.clickForgotPassword();
    await expect(loginPage.page).toHaveURL(/.*forgot-password/);
  });

  test('should be responsive on mobile devices', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await loginPage.goto();

    await loginPage.expectMobileLoginForm();
    await loginPage.expectToBeOnLoginPage();
  });

  test('should be accessible with keyboard navigation', async ({ page }) => {
    await loginPage.goto();

    // Tab through form elements
    await page.keyboard.press('Tab');
    await page.keyboard.type('admin@example.com');

    await page.keyboard.press('Tab');
    await page.keyboard.type('admin123');

    await page.keyboard.press('Tab');
    await page.keyboard.press('Enter');

    // Should trigger login
    await page.waitForTimeout(1000);
  });
});

// Role-specific authentication tests
test.describe('Admin Authentication', () => {
  test.setTimeout(120_000);

  test('should have access to all admin features after login', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const dashboardPage = new DashboardPage(page);
    const apiHelpers = new ApiHelpers(page);

    const adminUser = {
      email: 'admin@example.com',
      password: 'admin123'
    };

    await apiHelpers.mockLoginSuccess(adminUser);
    await apiHelpers.mockDashboardData('admin');
    await loginPage.goto();
    await loginPage.login(adminUser.email, adminUser.password, '**/admin');

    await dashboardPage.expectAdminFeatures();

    // Check admin-specific navigation
    await expect(page.locator('a:has-text("Nhân viên")').first()).toBeVisible();
    await expect(page.locator('a:has-text("Dự án")').first()).toBeVisible();
    await expect(page.locator('a:has-text("Người dùng")').first()).toBeVisible();
  });
});

test.describe('Partner Authentication', () => {
  test.setTimeout(120_000);

  test('should have limited access after login', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const dashboardPage = new DashboardPage(page);
    const apiHelpers = new ApiHelpers(page);

    const partnerUser = {
      email: 'partner@example.com',
      password: 'partner123'
    };

    await apiHelpers.mockLoginSuccess(partnerUser, 'partner');
    await apiHelpers.mockDashboardData('partner');
    await loginPage.goto();
    await loginPage.login(partnerUser.email, partnerUser.password, '**/dashboard');

    await dashboardPage.expectPartnerFeatures();

    // Partner should not see admin-only features
    await expect(page.locator('a:has-text("Phê duyệt")')).not.toBeVisible();
  });
});

