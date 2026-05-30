import { test, expect } from '@playwright/test';
import { LoginPage } from '../page-objects/LoginPage';
import { DashboardPage } from '../page-objects/DashboardPage';
import { ApiHelpers } from '../utils/api-helpers';
import { CustomAssertions } from '../utils/assertions';

test.describe('Authentication Flow', () => {
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

    await loginPage.login('admin@example.com', 'admin123');
    await loginPage.expectToBeLoggedIn();
    await dashboardPage.expectToBeOnDashboard();
    await dashboardPage.expectAdminFeatures();
  });

  test('should login successfully with valid partner credentials', async () => {
    // Mock successful login API response for partner
    await apiHelpers.mockApiResponse('**/api/auth/login', {
      success: true,
      token: 'mock-jwt-token',
      user: {
        id: '2',
        email: 'partner@example.com',
        role: 'partner',
        name: 'Partner User'
      }
    });

    await loginPage.login('partner@example.com', 'partner123');
    await loginPage.expectToBeLoggedIn();
    await dashboardPage.expectToBeOnDashboard();
    await dashboardPage.expectPartnerFeatures();
  });

  test('should show error message for invalid credentials', async () => {
    await apiHelpers.mockLoginFailure();

    await loginPage.loginWithInvalidCredentials('invalid@example.com', 'wrongpassword');
    await loginPage.expectErrorMessage('Invalid credentials');
  });

  test('should handle rate limiting after multiple failed attempts', async () => {
    await apiHelpers.mockRateLimit();

    await loginPage.attemptMultipleLogins('test@example.com', 'wrongpassword', 5);
    await loginPage.expectRateLimitMessage();
  });

  test('should validate required fields', async () => {
    await loginPage.login('', '');
    await assertions.expectFormValidationError('email');
    await assertions.expectFormValidationError('password');
  });

  test('should validate email format', async () => {
    await loginPage.login('invalid-email', 'password123');
    await assertions.expectFormValidationError('email');
  });

  test('should disable login button during submission', async () => {
    // Mock slow API response
    await apiHelpers.mockSlowResponse('**/api/auth/login', 2000);

    await loginPage.login('admin@example.com', 'admin123');
    await loginPage.expectLoginButtonToBeDisabled();
  });

  test('should handle network errors gracefully', async () => {
    await apiHelpers.mockNetworkError('**/api/auth/login');

    await loginPage.login('admin@example.com', 'admin123');
    await assertions.expectErrorToast('Network error occurred');
  });

  test('should redirect to login page when accessing protected routes', async ({ page }) => {
    // Try to access dashboard without authentication
    await page.goto('/dashboard');
    await expect(page).toHaveURL(/.*login/);
  });

  test('should logout successfully', async () => {
    // Mock successful login
    await apiHelpers.mockLoginSuccess({ email: 'admin@example.com', password: 'admin123' });
    await loginPage.login('admin@example.com', 'admin123');

    // Mock logout API
    await apiHelpers.mockApiResponse('**/api/auth/logout', {
      success: true,
      message: 'Logged out successfully'
    });

    await dashboardPage.logout();
    await loginPage.expectToBeOnLoginPage();
  });

  test('should handle token expiration', async ({ page }) => {
    // Mock expired token response
    await apiHelpers.mockApiResponse('**/api/auth/refresh', {
      success: false,
      error: 'Token expired'
    }, 401);

    // Simulate expired token scenario
    await page.evaluate(() => {
      localStorage.setItem('token', 'expired-token');
    });

    await page.goto('/dashboard');
    await expect(page).toHaveURL(/.*login/);
  });

  test('should remember login state across page refreshes', async ({ page }) => {
    // Mock successful login
    await apiHelpers.mockLoginSuccess({ email: 'admin@example.com', password: 'admin123' });
    await loginPage.login('admin@example.com', 'admin123');

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

    await assertions.expectMobileLayout();
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
  test('should have access to all admin features after login', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const dashboardPage = new DashboardPage(page);
    const apiHelpers = new ApiHelpers(page);

    const adminUser = {
      email: 'admin@example.com',
      password: 'admin123'
    };

    await apiHelpers.mockLoginSuccess(adminUser);
    await loginPage.login(adminUser.email, adminUser.password);

    await dashboardPage.expectAdminFeatures();

    // Check admin-specific navigation
    await expect(page.locator('a:has-text("Nhân viên")')).toBeVisible();
    await expect(page.locator('a:has-text("Dự án")')).toBeVisible();
    await expect(page.locator('a:has-text("Phê duyệt")')).toBeVisible();
  });
});

test.describe('Partner Authentication', () => {
  test('should have limited access after login', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const dashboardPage = new DashboardPage(page);
    const apiHelpers = new ApiHelpers(page);

    const partnerUser = {
      email: 'partner@example.com',
      password: 'partner123'
    };

    await apiHelpers.mockApiResponse('**/api/auth/login', {
      success: true,
      token: 'mock-jwt-token',
      user: {
        id: '2',
        email: partnerUser.email,
        role: 'partner',
        name: 'Partner User'
      }
    });

    await loginPage.login(partnerUser.email, partnerUser.password);

    await dashboardPage.expectPartnerFeatures();

    // Partner should not see admin-only features
    await expect(page.locator('a:has-text("Phê duyệt")')).not.toBeVisible();
  });
});
