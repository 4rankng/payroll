import { Page, expect } from '@playwright/test';

export class LoginPage {
  constructor(private page: Page) {}

  // Selectors — retargeted to the rebuilt (UU PRO) login markup.
  // #emailOrUsername is the CCCD/phone/username identity field (type="text"
  // by design — it is NOT an email input); #password is the password field;
  // the submit button is the native type=submit "Đăng nhập"; the login error
  // alert carries role="alert" + data-testid="error-message".
  private readonly emailInput = '#emailOrUsername';
  private readonly passwordInput = '#password';
  private readonly loginButton = 'button[type="submit"]';
  private readonly errorMessage = '[data-testid="error-message"]';
  private readonly forgotPasswordLink = 'a:has-text("Quên mật khẩu")';
  private readonly loadingSpinner = '[data-testid="loading"], .loading, .spinner';

  // Actions
  async goto() {
    await this.page.goto('/login');
    await this.page.waitForLoadState('networkidle');
  }

  async fillCredentials(username: string, password: string) {
    await this.page.fill(this.emailInput, username);
    await this.page.fill(this.passwordInput, password);
  }

  async submit() {
    await this.page.click(this.loginButton);
  }

  async login(username: string, password: string, destination: string = '**/dashboard') {
    await this.fillCredentials(username, password);
    await this.submit();

    // Wait for post-login navigation (destination varies by role: admin →
    // /admin, partner → /partner/dashboard) or the inline error alert.
    await Promise.race([
      this.page.waitForURL(destination, { timeout: 15000 }),
      this.page.waitForSelector(this.errorMessage, { timeout: 15000 })
    ]);
  }

  async loginWithInvalidCredentials(username: string, password: string) {
    await this.fillCredentials(username, password);
    await this.submit();

    // Wait for error message
    await this.page.waitForSelector(this.errorMessage, { timeout: 15000 });
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
    // Away from /login and inside an authenticated app shell.
    await expect(this.page).not.toHaveURL(/.*login/);
    await expect(this.page.locator('main').first()).toBeVisible();
  }

  async expectMobileLoginForm() {
    // The rebuilt login renders a usable single-column form at mobile widths
    // (hero strip on top, form below); assert the form contract is present.
    await expect(this.page.locator(this.emailInput)).toBeVisible();
    await expect(this.page.locator(this.passwordInput)).toBeVisible();
    await expect(this.page.locator(this.loginButton)).toBeVisible();
  }

  // Rate limiting tests
  async attemptMultipleLogins(username: string, password: string, attempts: number = 5) {
    for (let i = 0; i < attempts; i++) {
      await this.fillCredentials(username, password);
      await this.submit();

      // Wait a bit between attempts
      await this.page.waitForTimeout(1000);
    }
  }

  async expectRateLimitMessage() {
    // The axios interceptor owns the 429 copy ("Quá nhiều yêu cầu…"); the
    // fallback alert copy is "Quá nhiều lần đăng nhập…".
    await expect(this.page.locator(this.errorMessage)).toContainText(/Quá nhiều yêu cầu|Quá nhiều lần đăng nhập/);
  }
}
