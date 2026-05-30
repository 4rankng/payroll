import { Page, expect } from '@playwright/test';
import { TestDataFactory } from '../fixtures/test-data';

export class EmployeesPage {
  constructor(private page: Page) {}

  // Selectors
  private readonly addEmployeeButton = 'button:has-text("Thêm nhân viên"), button:has-text("Add Employee")';
  private readonly searchInput = 'input[placeholder*="Tìm kiếm"], input[placeholder*="Search"]';
  private readonly filterDropdown = '[data-testid="filter-dropdown"], .filter-dropdown';
  private readonly employeeTable = '[data-testid="employee-table"], .employee-table, table';
  private readonly employeeRows = '[data-testid="employee-row"], .employee-row, tbody tr';
  private readonly editButton = 'button:has-text("Chỉnh sửa"), button:has-text("Edit")';
  private readonly deleteButton = 'button:has-text("Xóa"), button:has-text("Delete")';
  private readonly confirmDeleteButton = 'button:has-text("Xác nhận"), button:has-text("Confirm")';
  private readonly cancelButton = 'button:has-text("Hủy"), button:has-text("Cancel")';

  // Modal selectors
  private readonly employeeModal = '[data-testid="employee-modal"], .employee-modal, .modal';
  private readonly firstNameInput = 'input[name="firstName"], input[placeholder*="Tên"]';
  private readonly lastNameInput = 'input[name="lastName"], input[placeholder*="Họ"]';
  private readonly emailInput = 'input[name="email"], input[type="email"]';
  private readonly phoneInput = 'input[name="phone"], input[placeholder*="Số điện thoại"]';
  private readonly salaryInput = 'input[name="salary"], input[placeholder*="Lương"]';
  private readonly positionInput = 'input[name="position"], input[placeholder*="Chức vụ"]';
  private readonly saveButton = 'button:has-text("Lưu"), button:has-text("Save")';

  // Actions
  async goto() {
    await this.page.goto('/employees');
    await this.page.waitForLoadState('networkidle');
  }

  async openAddEmployeeModal() {
    await this.page.click(this.addEmployeeButton);
    await this.page.waitForSelector(this.employeeModal);
  }

  async fillEmployeeForm(employeeData: ReturnType<typeof TestDataFactory.createEmployee>) {
    await this.page.fill(this.firstNameInput, employeeData.firstName);
    await this.page.fill(this.lastNameInput, employeeData.lastName);
    await this.page.fill(this.emailInput, employeeData.email);
    await this.page.fill(this.phoneInput, employeeData.phone);
    await this.page.fill(this.salaryInput, employeeData.salary.toString());
    await this.page.fill(this.positionInput, employeeData.position);
  }

  async saveEmployee() {
    await this.page.click(this.saveButton);
    await this.page.waitForSelector(this.employeeModal, { state: 'hidden' });
  }

  async addEmployee(employeeData?: ReturnType<typeof TestDataFactory.createEmployee>) {
    const data = employeeData || TestDataFactory.createEmployee();
    await this.openAddEmployeeModal();
    await this.fillEmployeeForm(data);
    await this.saveEmployee();
    return data;
  }

  async searchEmployee(searchTerm: string) {
    await this.page.fill(this.searchInput, searchTerm);
    await this.page.waitForTimeout(500); // Wait for debounced search
  }

  async filterEmployees(filterOption: string) {
    await this.page.click(this.filterDropdown);
    await this.page.click(`option:has-text("${filterOption}")`);
  }

  async editEmployee(index: number = 0) {
    const editButtons = this.page.locator(this.editButton);
    await editButtons.nth(index).click();
    await this.page.waitForSelector(this.employeeModal);
  }

  async deleteEmployee(index: number = 0) {
    const deleteButtons = this.page.locator(this.deleteButton);
    await deleteButtons.nth(index).click();
    await this.page.click(this.confirmDeleteButton);
    await this.page.waitForTimeout(1000); // Wait for deletion
  }

  async cancelDelete() {
    await this.page.click(this.cancelButton);
  }

  // Assertions
  async expectToBeOnEmployeesPage() {
    await expect(this.page).toHaveURL(/.*employees/);
    await expect(this.page.locator(this.addEmployeeButton)).toBeVisible();
    await expect(this.page.locator(this.employeeTable)).toBeVisible();
  }

  async expectEmployeeInTable(employeeData: ReturnType<typeof TestDataFactory.createEmployee>) {
    const table = this.page.locator(this.employeeTable);
    await expect(table).toContainText(employeeData.firstName);
    await expect(table).toContainText(employeeData.lastName);
    await expect(table).toContainText(employeeData.email);
  }

  async expectEmployeeCount(expectedCount: number) {
    const rows = this.page.locator(this.employeeRows);
    await expect(rows).toHaveCount(expectedCount);
  }

  async expectSearchResults(searchTerm: string) {
    const rows = this.page.locator(this.employeeRows);
    const count = await rows.count();

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i);
      await expect(row).toContainText(searchTerm);
    }
  }

  async expectNoSearchResults() {
    const rows = this.page.locator(this.employeeRows);
    await expect(rows).toHaveCount(0);
  }

  async expectModalOpen() {
    await expect(this.page.locator(this.employeeModal)).toBeVisible();
  }

  async expectModalClosed() {
    await expect(this.page.locator(this.employeeModal)).not.toBeVisible();
  }

  async expectFormValidationError(fieldName: string) {
    const errorSelector = `[data-testid="error-${fieldName}"], .error-${fieldName}, .field-error`;
    await expect(this.page.locator(errorSelector)).toBeVisible();
  }

  // Bulk operations
  async selectAllEmployees() {
    const selectAllCheckbox = this.page.locator('[data-testid="select-all"], .select-all input[type="checkbox"]');
    await selectAllCheckbox.check();
  }

  async selectEmployee(index: number) {
    const checkboxes = this.page.locator('input[type="checkbox"]');
    await checkboxes.nth(index + 1).check(); // +1 to skip select-all checkbox
  }

  async bulkDelete() {
    await this.page.click('button:has-text("Xóa đã chọn"), button:has-text("Delete Selected")');
    await this.page.click(this.confirmDeleteButton);
  }

  async expectBulkActionBarVisible() {
    await expect(this.page.locator('[data-testid="bulk-action-bar"], .bulk-action-bar')).toBeVisible();
  }
}
