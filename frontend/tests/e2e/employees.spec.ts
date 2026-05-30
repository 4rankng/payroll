import { test, expect } from '@playwright/test';
import { LoginPage } from '../page-objects/LoginPage';
import { EmployeesPage } from '../page-objects/EmployeesPage';
import { ApiHelpers } from '../utils/api-helpers';
import { CustomAssertions } from '../utils/assertions';
import { TestDataFactory } from '../fixtures/test-data';

test.describe('Employee Management', () => {
  let loginPage: LoginPage;
  let employeesPage: EmployeesPage;
  let apiHelpers: ApiHelpers;
  let assertions: CustomAssertions;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    employeesPage = new EmployeesPage(page);
    apiHelpers = new ApiHelpers(page);
    assertions = new CustomAssertions(page);

    // Mock authentication
    await apiHelpers.mockLoginSuccess({ email: 'admin@example.com', password: 'admin123' });
    await loginPage.goto();
    await loginPage.login('admin@example.com', 'admin123');

    // Mock employees list API
    await apiHelpers.mockEmployeesList([]);
    await employeesPage.goto();
  });

  test('should display employees page correctly', async () => {
    await employeesPage.expectToBeOnEmployeesPage();
    await employeesPage.expectEmployeeCount(0);
  });

  test('should add new employee successfully', async () => {
    const employeeData = TestDataFactory.createEmployee();
    await apiHelpers.mockCreateEmployee(employeeData);

    await employeesPage.addEmployee(employeeData);
    await assertions.expectSuccessToast('Employee added successfully');
    await employeesPage.expectEmployeeInTable(employeeData);
  });

  test('should validate required fields when adding employee', async () => {
    await employeesPage.openAddEmployeeModal();

    // Try to save without filling required fields
    await employeesPage.saveEmployee();

    await assertions.expectFormValidationError('firstName');
    await assertions.expectFormValidationError('lastName');
    await assertions.expectFormValidationError('email');
  });

  test('should validate email format when adding employee', async () => {
    const invalidEmployeeData = {
      ...TestDataFactory.createEmployee(),
      email: 'invalid-email-format'
    };

    await employeesPage.openAddEmployeeModal();
    await employeesPage.fillEmployeeForm(invalidEmployeeData);
    await employeesPage.saveEmployee();

    await assertions.expectFormValidationError('email');
  });

  test('should edit existing employee', async () => {
    const originalEmployee = TestDataFactory.createEmployee();
    const updatedEmployee = { ...originalEmployee, firstName: 'Updated Name' };

    // Mock initial employee list
    await apiHelpers.mockEmployeesList([originalEmployee]);
    await employeesPage.goto();

    // Mock update API
    await apiHelpers.mockUpdateEmployee('1', updatedEmployee);

    await employeesPage.editEmployee(0);
    await employeesPage.fillEmployeeForm(updatedEmployee);
    await employeesPage.saveEmployee();

    await assertions.expectSuccessToast('Employee updated successfully');
  });

  test('should delete employee with confirmation', async () => {
    const employeeData = TestDataFactory.createEmployee();

    // Mock initial employee list
    await apiHelpers.mockEmployeesList([employeeData]);
    await employeesPage.goto();

    // Mock delete API
    await apiHelpers.mockDeleteEmployee('1');

    await employeesPage.deleteEmployee(0);
    await assertions.expectSuccessToast('Employee deleted successfully');
    await employeesPage.expectEmployeeCount(0);
  });

  test('should cancel employee deletion', async () => {
    const employeeData = TestDataFactory.createEmployee();

    // Mock initial employee list
    await apiHelpers.mockEmployeesList([employeeData]);
    await employeesPage.goto();

    // Start deletion but cancel
    await employeesPage.page.click('button:has-text("Xóa"), button:has-text("Delete")');
    await employeesPage.cancelDelete();

    // Employee should still be in the table
    await employeesPage.expectEmployeeInTable(employeeData);
  });

  test('should search employees by name', async () => {
    const employees = [
      TestDataFactory.createEmployee(),
      TestDataFactory.createEmployee(),
      TestDataFactory.createEmployee()
    ];
    employees[0].firstName = 'John';
    employees[1].firstName = 'Jane';
    employees[2].firstName = 'Bob';

    // Mock initial list
    await apiHelpers.mockEmployeesList(employees);
    await employeesPage.goto();

    // Mock search results
    await apiHelpers.mockApiResponse('**/api/employees?search=John', {
      success: true,
      data: [employees[0]],
      total: 1
    });

    await employeesPage.searchEmployee('John');
    await employeesPage.expectSearchResults('John');
  });

  test('should filter employees by department', async () => {
    const employees = [
      { ...TestDataFactory.createEmployee(), department: 'Engineering' },
      { ...TestDataFactory.createEmployee(), department: 'Marketing' },
      { ...TestDataFactory.createEmployee(), department: 'Engineering' }
    ];

    // Mock initial list
    await apiHelpers.mockEmployeesList(employees);
    await employeesPage.goto();

    // Mock filtered results
    await apiHelpers.mockApiResponse('**/api/employees?department=Engineering', {
      success: true,
      data: [employees[0], employees[2]],
      total: 2
    });

    await employeesPage.filterEmployees('Engineering');
    await employeesPage.expectEmployeeCount(2);
  });

  test('should handle bulk operations', async () => {
    const employees = [
      TestDataFactory.createEmployee(),
      TestDataFactory.createEmployee(),
      TestDataFactory.createEmployee()
    ];

    // Mock initial list
    await apiHelpers.mockEmployeesList(employees);
    await employeesPage.goto();

    // Select all employees
    await employeesPage.selectAllEmployees();
    await employeesPage.expectBulkActionBarVisible();

    // Mock bulk delete
    await apiHelpers.mockApiResponse('**/api/employees/bulk-delete', {
      success: true,
      message: '3 employees deleted successfully'
    });

    await employeesPage.bulkDelete();
    await assertions.expectSuccessToast('3 employees deleted successfully');
  });

  test('should handle API errors gracefully', async () => {
    const employeeData = TestDataFactory.createEmployee();

    // Mock API error
    await apiHelpers.mockApiResponse('**/api/employees', {
      success: false,
      error: 'Internal server error'
    }, 500);

    await employeesPage.addEmployee(employeeData);
    await assertions.expectErrorToast('Failed to add employee');
  });

  test('should handle network errors', async () => {
    const employeeData = TestDataFactory.createEmployee();

    // Mock network error
    await apiHelpers.mockNetworkError('**/api/employees');

    await employeesPage.addEmployee(employeeData);
    await assertions.expectErrorToast('Network error occurred');
  });

  test('should be responsive on mobile devices', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await employeesPage.goto();

    await assertions.expectMobileLayout();
    await employeesPage.expectToBeOnEmployeesPage();
  });

  test('should support keyboard navigation', async ({ page }) => {
    await employeesPage.goto();

    // Tab through the page
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Enter'); // Should open add employee modal

    await employeesPage.expectModalOpen();
  });

  test('should export employee data', async ({ page }) => {
    const employees = [TestDataFactory.createEmployee()];

    // Mock employees list
    await apiHelpers.mockEmployeesList(employees);
    await employeesPage.goto();

    // Mock export API
    await apiHelpers.mockApiResponse('**/api/employees/export', {
      success: true,
      data: 'csv-data',
      filename: 'employees.csv'
    });

    await page.click('button:has-text("Xuất"), button:has-text("Export")');

    // Should trigger download
    const downloadPromise = page.waitForEvent('download');
    await downloadPromise;
  });

  test('should import employee data', async ({ page }) => {
    await employeesPage.goto();

    // Mock import API
    await apiHelpers.mockApiResponse('**/api/employees/import', {
      success: true,
      message: 'Employees imported successfully',
      imported: 5,
      errors: 0
    });

    // Create a test file
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles({
      name: 'employees.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from('name,email,phone\nJohn Doe,john@example.com,123-456-7890')
    });

    await assertions.expectSuccessToast('5 employees imported successfully');
  });
});

// Admin-specific employee tests
test.describe('Admin Employee Management', () => {
  test('should have full access to employee management features', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const employeesPage = new EmployeesPage(page);
    const apiHelpers = new ApiHelpers(page);

    const adminUser = {
      email: 'admin@example.com',
      password: 'admin123'
    };

    await apiHelpers.mockLoginSuccess(adminUser);
    await loginPage.login(adminUser.email, adminUser.password);

    await apiHelpers.mockEmployeesList([]);
    await employeesPage.goto();

    // Admin should see all employee management features
    await expect(page.locator('button:has-text("Thêm nhân viên")')).toBeVisible();
    await expect(page.locator('button:has-text("Xuất")')).toBeVisible();
    await expect(page.locator('button:has-text("Nhập")')).toBeVisible();
  });
});
