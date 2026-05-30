import { Page, expect } from '@playwright/test';

export class DashboardPage {
  constructor(private page: Page) {}

  // Selectors
  private readonly sidebar = '[data-testid="sidebar"], .sidebar, nav';
  private readonly userMenu = '[data-testid="user-menu"], .user-menu, .profile-menu';
  private readonly logoutButton = 'button:has-text("Đăng xuất"), button:has-text("Logout")';
  private readonly statsCards = '[data-testid="stats-cards"], .stats-cards, .dashboard-stats';
  private readonly recentActivities = '[data-testid="recent-activities"], .recent-activities';
  private readonly urgentTasks = '[data-testid="urgent-tasks"], .urgent-tasks';
  private readonly navigationMenu = '[data-testid="navigation"], .navigation, .nav-menu';

  // Actions
  async goto() {
    await this.page.goto('/dashboard');
    await this.page.waitForLoadState('networkidle');
  }

  async logout() {
    await this.page.click(this.userMenu);
    await this.page.click(this.logoutButton);
    await this.page.waitForURL('**/login');
  }

  async navigateToEmployees() {
    await this.page.click('a:has-text("Nhân viên"), a:has-text("Employees")');
    await this.page.waitForLoadState('networkidle');
  }

  async navigateToProjects() {
    await this.page.click('a:has-text("Dự án"), a:has-text("Projects")');
    await this.page.waitForLoadState('networkidle');
  }

  async navigateToTimesheet() {
    await this.page.click('a:has-text("Bảng chấm công"), a:has-text("Timesheet")');
    await this.page.waitForLoadState('networkidle');
  }

  async navigateToApprovals() {
    await this.page.click('a:has-text("Phê duyệt"), a:has-text("Approvals")');
    await this.page.waitForLoadState('networkidle');
  }

  // Assertions
  async expectToBeOnDashboard() {
    await expect(this.page).toHaveURL(/.*dashboard/);
    await expect(this.page.locator(this.sidebar)).toBeVisible();
    await expect(this.page.locator(this.userMenu)).toBeVisible();
  }

  async expectStatsCardsVisible() {
    await expect(this.page.locator(this.statsCards)).toBeVisible();
  }

  async expectRecentActivitiesVisible() {
    await expect(this.page.locator(this.recentActivities)).toBeVisible();
  }

  async expectUrgentTasksVisible() {
    await expect(this.page.locator(this.urgentTasks)).toBeVisible();
  }

  async expectNavigationMenuVisible() {
    await expect(this.page.locator(this.navigationMenu)).toBeVisible();
  }

  // Role-based assertions
  async expectAdminFeatures() {
    // Admin should see all navigation items
    await expect(this.page.locator('a:has-text("Nhân viên"), a:has-text("Employees")')).toBeVisible();
    await expect(this.page.locator('a:has-text("Dự án"), a:has-text("Projects")')).toBeVisible();
    await expect(this.page.locator('a:has-text("Phê duyệt"), a:has-text("Approvals")')).toBeVisible();
  }

  async expectPartnerFeatures() {
    // Partner should see limited navigation
    await expect(this.page.locator('a:has-text("Nhân viên"), a:has-text("Employees")')).toBeVisible();
    await expect(this.page.locator('a:has-text("Dự án"), a:has-text("Projects")')).toBeVisible();
    // Partner might not see admin-only features
  }

  // Data validation
  async expectStatsDataLoaded() {
    // Check that stats cards have data (not just loading states)
    const statsCards = this.page.locator(this.statsCards);
    await expect(statsCards).toBeVisible();

    // Check for actual numbers in stats (not just "0" or loading text)
    const statValues = statsCards.locator('[data-testid="stat-value"], .stat-value');
    await expect(statValues.first()).not.toHaveText(/loading|đang tải|0/);
  }
}
