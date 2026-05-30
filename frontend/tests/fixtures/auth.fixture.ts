/* eslint-disable react-hooks/rules-of-hooks */
import { test as base, expect } from '@playwright/test';
import { LoginPage } from '../page-objects/LoginPage';

// Extend basic test by providing a "loginPage" fixture
export const test = base.extend<{ loginPage: LoginPage }>({
  loginPage: async ({ page }, use) => {
    const loginPage = new LoginPage(page);
    await use(loginPage);
  },
});

// Admin user fixture
export const testAsAdmin = test.extend<{ adminUser: { email: string; password: string } }>({
  adminUser: async (_, use) => {
    const adminUser = {
      email: 'admin@example.com',
      password: 'admin123'
    };
    await use(adminUser);
  },
});

// Partner user fixture
export const testAsPartner = test.extend<{ partnerUser: { email: string; password: string } }>({
  partnerUser: async (_, use) => {
    const partnerUser = {
      email: 'partner@example.com',
      password: 'partner123'
    };
    await use(partnerUser);
  },
});

export { expect };
