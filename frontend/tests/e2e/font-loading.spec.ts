import { expect, test, type Route } from '@playwright/test';

test.use({ serviceWorkers: 'block' });

test('login remains usable while remote font stylesheets are pending, then applies them', async ({ page }) => {
  const pendingFonts: Route[] = [];
  await page.route('https://fonts.googleapis.com/**', (route) => {
    pendingFonts.push(route);
  });

  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await expect(page.getByLabel('Tên đăng nhập', { exact: true })).toBeVisible();
  await page.getByLabel('Tên đăng nhập', { exact: true }).fill('kiem-thu');
  await expect(page.getByLabel('Tên đăng nhập', { exact: true })).toHaveValue('kiem-thu');
  await expect.poll(() => pendingFonts.length).toBe(3);
  await expect(page.locator('link[data-font-stylesheet][media="print"]')).toHaveCount(3);

  await Promise.all(pendingFonts.map((route) => route.fulfill({
    contentType: 'text/css',
    body: '/* Simulated successful font stylesheet response. */',
  })));
  await expect(page.locator('link[data-font-stylesheet][media="all"]')).toHaveCount(3);
  await expect(page.locator('html')).toHaveAttribute('lang', 'vi');
});

test('authenticated workspace renders while font stylesheets are unavailable', async ({ page }) => {
  const now = Math.floor(Date.now() / 1000);
  const encode = (value: object) => Buffer.from(JSON.stringify(value)).toString('base64url');
  const token = `${encode({ alg: 'HS256' })}.${encode({ exp: now + 3600, user_id: 41, username: 'qa.accountant', role: 'accountant', sub: '41', nbf: now - 1, iat: now, jti: 'font-qa' })}.fixture`;
  await page.addInitScript((value) => {
    localStorage.setItem('auth_token', value);
    localStorage.setItem('userRole', 'accountant');
    localStorage.setItem('userName', 'Kế toán kiểm thử');
    localStorage.setItem('userEmail', 'qa@example.test');
  }, token);
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === '/api/v1/notifications/unread') {
      await route.fulfill({ json: { status: 'success', data: { notifications: [], count: 0 } } });
    } else if (path === '/api/v1/timesheets' && route.request().method() === 'GET') {
      await route.fulfill({ json: { status: 'success', data: [], pagination: { page: 1, pageSize: 500, totalPages: 1, totalRecords: 0 } } });
    } else {
      throw new Error(`Unexpected font-loading fixture request: ${route.request().method()} ${path}`);
    }
  });
  // Leave the font requests pending for the duration of the interaction.
  await page.route('https://fonts.googleapis.com/**', () => {});
  await page.goto('/accountant', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Chấm công chờ duyệt' })).toBeVisible();
  await page.getByRole('tab', { name: 'Nhập KQ chuyển lô' }).click();
  await page.getByRole('button', { name: 'Tải lên file kết quả' }).click();
  await expect(page.getByRole('dialog', { name: 'Kết quả chuyển tiền' })).toBeVisible();
});
