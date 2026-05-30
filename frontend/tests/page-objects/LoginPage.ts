import { Page, expect } from '@playwright/test';

export class LoginPage {
  constructor(private page: Page) {}

  // Selectors
  private readonly emailInput = 'input[type="email"], input[name="email"]';
  private readonly passwordInput = 'input[type="password"], input[name="password"]';
  private readonly loginButton = 'button[type="submit"], button:has-text("Đăng nhập"), button:has-text("Login")';
  private readonly errorMessage = '[data-testid="error-message"], .error-message, .alert-error';
  private readonly forgotPasswordLink = 'a:has-text("Quên mật khẩu"), a:has-text("Forgot password")';
  private readonly loadingSpinner = '[data-testid="loading"], .loading, .spinner';

  // Actions
  async goto() {
    await this.page.goto('/login');
    await this.page.waitForLoadState('networkidle');
  }

  async login(email: string, password: string) {
    await this.page.fill(this.emailInput, email);
    await this.page.fill(this.passwordInput, password);
    await this.page.click(this.loginButton);

    // Wait for navigation or error message
    await Promise.race([
      this.page.waitForURL('**/dashboard', { timeout: 10000 }),
      this.page.waitForSelector(this.errorMessage, { timeout: 5000 })
    ]);
  }

  async loginWithInvalidCredentials(email: string, password: string) {
    await this.page.fill(this.emailInput, email);
    await this.page.fill(this.passwordInput, password);
    await this.page.click(this.loginButton);

    // Wait for error message
    await this.page.waitForSelector(this.errorMessage, { timeout: 5000 });
  }

  async clickForgotPassword() {
    await this.page.click(this.forgotPasswordLink);
  }

  async waitForLoadingToFinish() {
    await this.page.waitForSelector(this.loadingSpinner, { state: 'hidden', timeout: 10000 });
  }

  // Assertions
  async expectToBeOnLoginPage() {
    await expect(this.page).toHaveURL(/.*login/);
    await expect(this.page.locator(this.emailInput)).toBeVisible();
    await expect(this.page.locator(this.passwordInput)).toBeVisible();
    await expect(this.page.locator(this.loginButton)).toBeVisible();
  }

  async expectErrorMessage(message: string) {
    await expect(this.page.locator(this.errorMessage)).toContainText(message);
  }

  async expectLoginButtonToBeDisabled() {
    await expect(this.page.locator(this.loginButton)).toBeDisabled();
  }

  async expectLoginButtonToBeEnabled() {
    await expect(this.page.locator(this.loginButton)).toBeEnabled();
  }

  async expectToBeLoggedIn() {
    // Check for dashboard elements or user menu
    await expect(this.page).toHaveURL(/.*dashboard/);
    await expect(this.page.locator('[data-testid="user-menu"], .user-menu, .profile-menu')).toBeVisible();
  }

  // Rate limiting tests
  async attemptMultipleLogins(email: string, password: string, attempts: number = 5) {
    for (let i = 0; i < attempts; i++) {
      await this.page.fill(this.emailInput, email);
      await this.page.fill(this.passwordInput, password);
      await this.page.click(this.loginButton);

      // Wait a bit between attempts
      await this.page.waitForTimeout(1000);
    }
  }

  async expectRateLimitMessage() {
    await expect(this.page.locator(this.errorMessage)).toContainText(/rate limit|too many attempts|quá nhiều lần thử/);
  }
}
