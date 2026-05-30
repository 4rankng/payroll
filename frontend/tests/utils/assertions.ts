import { expect, Page } from '@playwright/test';

export class CustomAssertions {
  constructor(private page: Page) {}

  // Vietnamese text assertions
  async expectVietnameseText(selector: string, expectedText: string) {
    await expect(this.page.locator(selector)).toContainText(expectedText);
  }

  async expectVietnameseTitle(expectedTitle: string) {
    await expect(this.page).toHaveTitle(expectedTitle);
  }

  // Loading state assertions
  async expectLoadingState() {
    const loadingSelectors = [
      '[data-testid="loading"]',
      '.loading',
      '.spinner',
      '.loader',
      'text=Đang tải...',
      'text=Loading...'
    ];

    let loadingFound = false;
    for (const selector of loadingSelectors) {
      try {
        await expect(this.page.locator(selector)).toBeVisible({ timeout: 1000 });
        loadingFound = true;
        break;
      } catch {
        // Continue to next selector
      }
    }

    if (!loadingFound) {
      throw new Error('No loading indicator found');
    }
  }

  async expectNoLoadingState() {
    const loadingSelectors = [
      '[data-testid="loading"]',
      '.loading',
      '.spinner',
      '.loader'
    ];

    for (const selector of loadingSelectors) {
      await expect(this.page.locator(selector)).not.toBeVisible();
    }
  }

  // Form validation assertions
  async expectFormValidationError(fieldName: string) {
    const errorSelectors = [
      `[data-testid="error-${fieldName}"]`,
      `.error-${fieldName}`,
      `.field-error`,
      `[aria-invalid="true"]`
    ];

    let errorFound = false;
    for (const selector of errorSelectors) {
      try {
        await expect(this.page.locator(selector)).toBeVisible({ timeout: 1000 });
        errorFound = true;
        break;
      } catch {
        // Continue to next selector
      }
    }

    if (!errorFound) {
      throw new Error(`No validation error found for field: ${fieldName}`);
    }
  }

  async expectNoFormValidationErrors() {
    const errorSelectors = [
      '[data-testid*="error"]',
      '.field-error',
      '.error-message',
      '[aria-invalid="true"]'
    ];

    for (const selector of errorSelectors) {
      await expect(this.page.locator(selector)).not.toBeVisible();
    }
  }

  // Table assertions
  async expectTableRowCount(tableSelector: string, expectedCount: number) {
    const rows = this.page.locator(`${tableSelector} tbody tr`);
    await expect(rows).toHaveCount(expectedCount);
  }

  async expectTableContainsData(tableSelector: string, data: string) {
    const table = this.page.locator(tableSelector);
    await expect(table).toContainText(data);
  }

  async expectTableEmpty(tableSelector: string) {
    const rows = this.page.locator(`${tableSelector} tbody tr`);
    await expect(rows).toHaveCount(0);
  }

  // Modal assertions
  async expectModalOpen(modalSelector: string = '[data-testid="modal"], .modal') {
    await expect(this.page.locator(modalSelector)).toBeVisible();
  }

  async expectModalClosed(modalSelector: string = '[data-testid="modal"], .modal') {
    await expect(this.page.locator(modalSelector)).not.toBeVisible();
  }

  // Toast/notification assertions
  async expectSuccessToast(message?: string) {
    const toastSelectors = [
      '[data-testid="toast-success"]',
      '.toast-success',
      '.success-toast',
      '.notification-success'
    ];

    let toastFound = false;
    for (const selector of toastSelectors) {
      try {
        const toast = this.page.locator(selector);
        await expect(toast).toBeVisible({ timeout: 2000 });
        if (message) {
          await expect(toast).toContainText(message);
        }
        toastFound = true;
        break;
      } catch {
        // Continue to next selector
      }
    }

    if (!toastFound) {
      throw new Error('No success toast found');
    }
  }

  async expectErrorToast(message?: string) {
    const toastSelectors = [
      '[data-testid="toast-error"]',
      '.toast-error',
      '.error-toast',
      '.notification-error'
    ];

    let toastFound = false;
    for (const selector of toastSelectors) {
      try {
        const toast = this.page.locator(selector);
        await expect(toast).toBeVisible({ timeout: 2000 });
        if (message) {
          await expect(toast).toContainText(message);
        }
        toastFound = true;
        break;
      } catch {
        // Continue to next selector
      }
    }

    if (!toastFound) {
      throw new Error('No error toast found');
    }
  }

  // Accessibility assertions
  async expectAccessibleButton(selector: string) {
    const button = this.page.locator(selector);
    await expect(button).toBeVisible();
    await expect(button).toHaveAttribute('type', 'button');
    // Check for proper ARIA attributes
    const ariaLabel = await button.getAttribute('aria-label');
    const ariaDescribedBy = await button.getAttribute('aria-describedby');
    const hasText = await button.textContent();

    if (!ariaLabel && !hasText?.trim()) {
      throw new Error('Button lacks accessible text or aria-label');
    }
  }

  async expectAccessibleFormField(selector: string, labelText?: string) {
    const field = this.page.locator(selector);
    await expect(field).toBeVisible();

    if (labelText) {
      const label = this.page.locator(`label:has-text("${labelText}")`);
      await expect(label).toBeVisible();
    }
  }

  // Performance assertions
  async expectPageLoadTime(maxTime: number = 3000) {
    const startTime = Date.now();
    await this.page.waitForLoadState('networkidle');
    const loadTime = Date.now() - startTime;

    if (loadTime > maxTime) {
      throw new Error(`Page load time ${loadTime}ms exceeds maximum ${maxTime}ms`);
    }
  }

  // Responsive design assertions
  async expectMobileLayout() {
    await this.page.setViewportSize({ width: 375, height: 667 });
    await this.page.waitForTimeout(100); // Allow for layout adjustments

    // Check for mobile-specific elements
    const mobileMenu = this.page.locator('[data-testid="mobile-menu"], .mobile-menu');
    await expect(mobileMenu).toBeVisible();
  }

  async expectDesktopLayout() {
    await this.page.setViewportSize({ width: 1920, height: 1080 });
    await this.page.waitForTimeout(100); // Allow for layout adjustments

    // Check for desktop-specific elements
    const sidebar = this.page.locator('[data-testid="sidebar"], .sidebar');
    await expect(sidebar).toBeVisible();
  }
}
