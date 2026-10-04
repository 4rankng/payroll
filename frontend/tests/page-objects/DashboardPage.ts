import { Page, expect } from '@playwright/test';

export class DashboardPage {
  constructor(private page: Page) {}

  // Selectors — retargeted to the real admin shell (AdminSidebar): the nav
  // landmark carries aria-label "Điều hướng chính", the account menu trigger
  // is .admin-sidebar-account (aria-label "Tài khoản: …"), and the logout
  // action is the "Đăng xuất" item inside its Radix dropdown.
  private readonly sidebar = '[role="navigation"], nav';
  private readonly userMenu = 'button.admin-sidebar-account, button[aria-label^="Tài khoản"], button:has-text("Xin chào")';
  private readonly logoutButton = '[role="menuitem"]:has-text("Đăng xuất"), button:has-text("Đăng xuất")';
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
    // Role landing pages: admin → /admin, partner → /partner/dashboard.
    await expect(this.page).toHaveURL(/\/admin(\/|$)|\/dashboard(\/|$)/);
    await expect(this.page.locator(this.sidebar).first()).toBeVisible();
    await expect(this.page.locator(this.userMenu).first()).toBeVisible();
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
  /** Expand a collapsible sidebar nav group if the sidebar collapsed it. */
  private async openNavGroup(label: string) {
    const toggle = this.page.getByRole('button', { name: label, exact: true });
    try {
      if ((await toggle.getAttribute('aria-expanded', { timeout: 2000 })) === 'false') {
        await toggle.click();
      }
    } catch {
      // Group toggle not present (collapsed rail / flat nav) — links may
      // still be directly visible.
    }
  }

  async expectAdminFeatures() {
    // Admin should see the full management navigation. The Quản lý group can
    // auto-collapse on overflow, so expand it defensively first. The admin
    // sidebar has no "Phê duyệt" entry — "Người dùng" (/admin/users,
    // partner-hidden) is the admin-only nav item this assertion targets.
    await this.openNavGroup('Quản lý');
    await expect(this.page.locator('a:has-text("Nhân viên"), a:has-text("Employees")').first()).toBeVisible();
    await expect(this.page.locator('a:has-text("Dự án"), a:has-text("Projects")').first()).toBeVisible();
    await expect(this.page.locator('a:has-text("Người dùng"), a:has-text("Users")').first()).toBeVisible();
  }

  async expectPartnerFeatures() {
    // Partner should see limited navigation
    await this.openNavGroup('Quản lý');
    await expect(this.page.locator('a:has-text("Nhân viên"), a:has-text("Employees")').first()).toBeVisible();
    await expect(this.page.locator('a:has-text("Dự án"), a:has-text("Projects")').first()).toBeVisible();
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
