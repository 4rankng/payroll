import { test, expect } from '@playwright/test';
import { LoginPage } from '../page-objects/LoginPage';
import { ApiHelpers } from '../utils/api-helpers';
import { CustomAssertions } from '../utils/assertions';
import { TestDataFactory } from '../fixtures/test-data';

test.describe('Project Management', () => {
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

  test('should display projects page correctly', async ({ page }) => {
    await apiHelpers.mockProjectsList([]);
    await page.goto('/projects');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('[data-testid="projects-page"], .projects-page')).toBeVisible();
  });

  test('should add new project successfully', async ({ page }) => {
    const projectData = TestDataFactory.createProject();

    // Mock project creation
    await apiHelpers.mockApiResponse('**/api/projects', {
      success: true,
      data: { ...projectData, id: 'mock-id' }
    }, 201);

    await page.goto('/projects');

    // Open add project modal
    await page.click('button:has-text("Thêm dự án"), button:has-text("Add Project")');
    await page.waitForSelector('[data-testid="project-modal"], .project-modal');

    // Fill project form
    await page.fill('input[name="name"]', projectData.name);
    await page.fill('textarea[name="description"]', projectData.description);
    await page.fill('input[name="startDate"]', projectData.startDate);
    await page.fill('input[name="endDate"]', projectData.endDate);
    await page.fill('input[name="budget"]', projectData.budget.toString());

    // Save project
    await page.click('button:has-text("Lưu"), button:has-text("Save")');

    await assertions.expectSuccessToast('Project created successfully');
  });

  test('should validate required fields when adding project', async ({ page }) => {
    await page.goto('/projects');

    // Open add project modal
    await page.click('button:has-text("Thêm dự án"), button:has-text("Add Project")');
    await page.waitForSelector('[data-testid="project-modal"], .project-modal');

    // Try to save without filling required fields
    await page.click('button:has-text("Lưu"), button:has-text("Save")');

    await assertions.expectFormValidationError('name');
    await assertions.expectFormValidationError('startDate');
    await assertions.expectFormValidationError('endDate');
  });

  test('should edit existing project', async ({ page }) => {
    const originalProject = TestDataFactory.createProject();
    const updatedProject = { ...originalProject, name: 'Updated Project Name' };

    // Mock initial project list
    await apiHelpers.mockProjectsList([originalProject]);
    await page.goto('/projects');

    // Mock update API
    await apiHelpers.mockApiResponse('**/api/projects/1', {
      success: true,
      data: updatedProject
    });

    // Click edit button
    await page.click('button:has-text("Chỉnh sửa"), button:has-text("Edit")');
    await page.waitForSelector('[data-testid="project-modal"], .project-modal');

    // Update project name
    await page.fill('input[name="name"]', updatedProject.name);

    // Save changes
    await page.click('button:has-text("Lưu"), button:has-text("Save")');

    await assertions.expectSuccessToast('Project updated successfully');
  });

  test('should delete project with confirmation', async ({ page }) => {
    const projectData = TestDataFactory.createProject();

    // Mock initial project list
    await apiHelpers.mockProjectsList([projectData]);
    await page.goto('/projects');

    // Mock delete API
    await apiHelpers.mockApiResponse('**/api/projects/1', {
      success: true,
      message: 'Project deleted successfully'
    }, 204);

    // Click delete button
    await page.click('button:has-text("Xóa"), button:has-text("Delete")');

    // Confirm deletion
    await page.click('button:has-text("Xác nhận"), button:has-text("Confirm")');

    await assertions.expectSuccessToast('Project deleted successfully');
  });

  test('should assign employees to project', async ({ page }) => {
    const projectData = TestDataFactory.createProject();
    const employees = [
      TestDataFactory.createEmployee(),
      TestDataFactory.createEmployee()
    ];

    // Mock project and employees data
    await apiHelpers.mockProjectsList([projectData]);
    await apiHelpers.mockApiResponse('**/api/employees', {
      success: true,
      data: employees
    });

    await page.goto('/projects');

    // Click assign employees button
    await page.click('button:has-text("Gán nhân viên"), button:has-text("Assign Employees")');
    await page.waitForSelector('[data-testid="assign-modal"], .assign-modal');

    // Select employees
    await page.check(`input[value="${employees[0].id}"]`);
    await page.check(`input[value="${employees[1].id}"]`);

    // Mock assignment API
    await apiHelpers.mockApiResponse('**/api/projects/1/assign', {
      success: true,
      message: 'Employees assigned successfully'
    });

    // Confirm assignment
    await page.click('button:has-text("Xác nhận"), button:has-text("Confirm")');

    await assertions.expectSuccessToast('Employees assigned successfully');
  });

  test('should view project statistics', async ({ page }) => {
    const projectData = TestDataFactory.createProject();

    // Mock project data
    await apiHelpers.mockProjectsList([projectData]);
    await page.goto('/projects');

    // Mock project statistics API
    await apiHelpers.mockApiResponse('**/api/projects/1/stats', {
      success: true,
      data: {
        totalHours: 120,
        totalCost: 15000,
        employeeCount: 5,
        completionPercentage: 75
      }
    });

    // Click view statistics button
    await page.click('button:has-text("Thống kê"), button:has-text("Statistics")');

    // Verify statistics are displayed
    await expect(page.locator('[data-testid="project-stats"], .project-stats')).toBeVisible();
    await expect(page.locator('text=120')).toBeVisible(); // Total hours
    await expect(page.locator('text=75%')).toBeVisible(); // Completion percentage
  });

  test('should filter projects by status', async ({ page }) => {
    const projects = [
      { ...TestDataFactory.createProject(), status: 'active' },
      { ...TestDataFactory.createProject(), status: 'completed' },
      { ...TestDataFactory.createProject(), status: 'on-hold' }
    ];

    // Mock initial project list
    await apiHelpers.mockProjectsList(projects);
    await page.goto('/projects');

    // Mock filtered results
    await apiHelpers.mockApiResponse('**/api/projects?status=active', {
      success: true,
      data: [projects[0]],
      total: 1
    });

    // Filter by active status
    await page.selectOption('select[name="status"], .status-filter', 'active');

    // Verify only active projects are shown
    const rows = page.locator('[data-testid="project-row"], .project-row');
    await expect(rows).toHaveCount(1);
  });

  test('should search projects by name', async ({ page }) => {
    const projects = [
      TestDataFactory.createProject(),
      TestDataFactory.createProject(),
      TestDataFactory.createProject()
    ];
    projects[0].name = 'Website Development';
    projects[1].name = 'Mobile App';
    projects[2].name = 'Database Migration';

    // Mock initial project list
    await apiHelpers.mockProjectsList(projects);
    await page.goto('/projects');

    // Mock search results
    await apiHelpers.mockApiResponse('**/api/projects?search=Website', {
      success: true,
      data: [projects[0]],
      total: 1
    });

    // Search for projects
    await page.fill('input[placeholder*="Tìm kiếm"], input[placeholder*="Search"]', 'Website');
    await page.waitForTimeout(500); // Wait for debounced search

    // Verify search results
    const rows = page.locator('[data-testid="project-row"], .project-row');
    await expect(rows).toHaveCount(1);
    await expect(rows.first()).toContainText('Website Development');
  });

  test('should export project data', async ({ page }) => {
    const projects = [TestDataFactory.createProject()];

    // Mock project list
    await apiHelpers.mockProjectsList(projects);
    await page.goto('/projects');

    // Mock export API
    await apiHelpers.mockApiResponse('**/api/projects/export', {
      success: true,
      data: 'csv-data',
      filename: 'projects.csv'
    });

    // Click export button
    await page.click('button:has-text("Xuất"), button:has-text("Export")');

    // Should trigger download
    const downloadPromise = page.waitForEvent('download');
    await downloadPromise;
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock API error
    await apiHelpers.mockApiResponse('**/api/projects', {
      success: false,
      error: 'Internal server error'
    }, 500);

    await page.goto('/projects');

    // Try to add project
    await page.click('button:has-text("Thêm dự án"), button:has-text("Add Project")');
    await page.waitForSelector('[data-testid="project-modal"], .project-modal');

    const projectData = TestDataFactory.createProject();
    await page.fill('input[name="name"]', projectData.name);
    await page.fill('textarea[name="description"]', projectData.description);

    await page.click('button:has-text("Lưu"), button:has-text("Save")');

    await assertions.expectErrorToast('Failed to create project');
  });

  test('should be responsive on mobile devices', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await apiHelpers.mockProjectsList([]);
    await page.goto('/projects');

    await assertions.expectMobileLayout();
    await expect(page.locator('[data-testid="projects-page"], .projects-page')).toBeVisible();
  });
});

// Admin-specific project tests
test.describe('Admin Project Management', () => {
  test('should have access to all project management features', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const apiHelpers = new ApiHelpers(page);

    const adminUser = {
      email: 'admin@example.com',
      password: 'admin123'
    };

    await apiHelpers.mockLoginSuccess(adminUser);
    await loginPage.login(adminUser.email, adminUser.password);

    await apiHelpers.mockProjectsList([]);
    await page.goto('/projects');

    // Admin should see all project management features
    await expect(page.locator('button:has-text("Thêm dự án")')).toBeVisible();
    await expect(page.locator('button:has-text("Xuất")')).toBeVisible();
    await expect(page.locator('button:has-text("Thống kê")')).toBeVisible();
  });
});
