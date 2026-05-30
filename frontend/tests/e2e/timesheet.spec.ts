import { test, expect } from '@playwright/test';
import { LoginPage } from '../page-objects/LoginPage';
import { ApiHelpers } from '../utils/api-helpers';
import { CustomAssertions } from '../utils/assertions';
import { TestDataFactory } from '../fixtures/test-data';

test.describe('Timesheet Workflow', () => {
  let loginPage: LoginPage;
  let apiHelpers: ApiHelpers;
  let assertions: CustomAssertions;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    apiHelpers = new ApiHelpers(page);
    assertions = new CustomAssertions(page);

    // Mock authentication
    await apiHelpers.mockLoginSuccess({ email: 'admin@example.com', password: 'admin123' });
    await loginPage.goto();
    await loginPage.login('admin@example.com', 'admin123');
  });

  test('should display timesheet page correctly', async ({ page }) => {
    await apiHelpers.mockTimesheetList([]);
    await page.goto('/timesheet');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('[data-testid="timesheet-page"], .timesheet-page')).toBeVisible();
  });

  test('should submit timesheet successfully', async ({ page }) => {
    const timesheetData = TestDataFactory.createTimesheet();

    // Mock timesheet submission
    await apiHelpers.mockApiResponse('**/api/timesheet', {
      success: true,
      data: { ...timesheetData, id: 'mock-id' }
    }, 201);

    await page.goto('/timesheet');

    // Fill timesheet form
    await page.fill('input[name="date"]', timesheetData.date);
    await page.fill('input[name="hours"]', timesheetData.hours.toString());
    await page.fill('textarea[name="description"]', timesheetData.description);

    // Submit timesheet
    await page.click('button:has-text("Gửi"), button:has-text("Submit")');

    await assertions.expectSuccessToast('Timesheet submitted successfully');
  });

  test('should validate required fields when submitting timesheet', async ({ page }) => {
    await page.goto('/timesheet');

    // Try to submit without filling required fields
    await page.click('button:has-text("Gửi"), button:has-text("Submit")');

    await assertions.expectFormValidationError('date');
    await assertions.expectFormValidationError('hours');
    await assertions.expectFormValidationError('description');
  });

  test('should approve timesheet', async ({ page }) => {
    const timesheetData = TestDataFactory.createTimesheet();

    // Mock timesheet list with pending timesheet
    await apiHelpers.mockTimesheetList([{ ...timesheetData, status: 'pending' }]);
    await page.goto('/timesheet');

    // Mock approval API
    await apiHelpers.mockApproveTimesheet('1');

    // Click approve button
    await page.click('button:has-text("Phê duyệt"), button:has-text("Approve")');

    await assertions.expectSuccessToast('Timesheet approved successfully');
  });

  test('should reject timesheet', async ({ page }) => {
    const timesheetData = TestDataFactory.createTimesheet();

    // Mock timesheet list with pending timesheet
    await apiHelpers.mockTimesheetList([{ ...timesheetData, status: 'pending' }]);
    await page.goto('/timesheet');

    // Mock rejection API
    await apiHelpers.mockRejectTimesheet('1');

    // Click reject button
    await page.click('button:has-text("Loại"), button:has-text("Reject")');

    await assertions.expectSuccessToast('Timesheet rejected');
  });

  test('should handle bulk approval', async ({ page }) => {
    const timesheets = [
      TestDataFactory.createTimesheet(),
      TestDataFactory.createTimesheet(),
      TestDataFactory.createTimesheet()
    ];

    // Mock timesheet list
    await apiHelpers.mockTimesheetList(timesheets);
    await page.goto('/timesheet');

    // Select all timesheets
    await page.check('[data-testid="select-all"], .select-all input[type="checkbox"]');

    // Mock bulk approval API
    await apiHelpers.mockApiResponse('**/api/timesheet/bulk-approve', {
      success: true,
      message: '3 timesheets approved successfully'
    });

    // Click bulk approve
    await page.click('button:has-text("Phê duyệt đã chọn"), button:has-text("Approve Selected")');

    await assertions.expectSuccessToast('3 timesheets approved successfully');
  });

  test('should filter timesheets by status', async ({ page }) => {
    const timesheets = [
      { ...TestDataFactory.createTimesheet(), status: 'pending' },
      { ...TestDataFactory.createTimesheet(), status: 'approved' },
      { ...TestDataFactory.createTimesheet(), status: 'rejected' }
    ];

    // Mock initial timesheet list
    await apiHelpers.mockTimesheetList(timesheets);
    await page.goto('/timesheet');

    // Mock filtered results
    await apiHelpers.mockApiResponse('**/api/timesheet?status=pending', {
      success: true,
      data: [timesheets[0]],
      total: 1
    });

    // Filter by pending status
    await page.selectOption('select[name="status"], .status-filter', 'pending');

    // Verify only pending timesheets are shown
    const rows = page.locator('[data-testid="timesheet-row"], .timesheet-row');
    await expect(rows).toHaveCount(1);
  });

  test('should export timesheet data', async ({ page }) => {
    const timesheets = [TestDataFactory.createTimesheet()];

    // Mock timesheet list
    await apiHelpers.mockTimesheetList(timesheets);
    await page.goto('/timesheet');

    // Mock export API
    await apiHelpers.mockApiResponse('**/api/timesheet/export', {
      success: true,
      data: 'csv-data',
      filename: 'timesheets.csv'
    });

    // Click export button
    await page.click('button:has-text("Xuất"), button:has-text("Export")');

    // Should trigger download
    const downloadPromise = page.waitForEvent('download');
    await downloadPromise;
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock API error
    await apiHelpers.mockApiResponse('**/api/timesheet', {
      success: false,
      error: 'Internal server error'
    }, 500);

    await page.goto('/timesheet');

    // Try to submit timesheet
    const timesheetData = TestDataFactory.createTimesheet();
    await page.fill('input[name="date"]', timesheetData.date);
    await page.fill('input[name="hours"]', timesheetData.hours.toString());
    await page.fill('textarea[name="description"]', timesheetData.description);

    await page.click('button:has-text("Gửi"), button:has-text("Submit")');

    await assertions.expectErrorToast('Failed to submit timesheet');
  });

  test('should be responsive on mobile devices', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await apiHelpers.mockTimesheetList([]);
    await page.goto('/timesheet');

    await assertions.expectMobileLayout();
    await expect(page.locator('[data-testid="timesheet-page"], .timesheet-page')).toBeVisible();
  });
});

// Admin-specific timesheet tests
test.describe('Admin Timesheet Management', () => {
  test('should have access to all timesheet management features', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const apiHelpers = new ApiHelpers(page);

    const adminUser = {
      email: 'admin@example.com',
      password: 'admin123'
    };

    await apiHelpers.mockLoginSuccess(adminUser);
    await loginPage.login(adminUser.email, adminUser.password);

    await apiHelpers.mockTimesheetList([]);
    await page.goto('/timesheet');

    // Admin should see all timesheet management features
    await expect(page.locator('button:has-text("Phê duyệt")')).toBeVisible();
    await expect(page.locator('button:has-text("Loại")')).toBeVisible();
    await expect(page.locator('button:has-text("Xuất")')).toBeVisible();
  });
});

// Partner-specific timesheet tests
test.describe('Partner Timesheet Management', () => {
  test('should have limited access to timesheet features', async ({ page }) => {
    const loginPage = new LoginPage(page);
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

    await apiHelpers.mockTimesheetList([]);
    await page.goto('/timesheet');

    // Partner should see limited features
    await expect(page.locator('button:has-text("Gửi")')).toBeVisible();
    // Partner might not see approval features
  });
});
