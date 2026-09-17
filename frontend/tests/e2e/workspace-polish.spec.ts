import { test, expect, type Page } from '@playwright/test';

test.use({ serviceWorkers: 'block' });
test.setTimeout(60_000);

const fixtureRow = {
  id: 41, employeeName: 'Nguyễn Thị Nhân Viên Có Tên Rất Dài', employeeCCCD: '012345678901',
  projectName: 'Dự án kiểm thử với tên dài để xác minh bảng trên điện thoại',
  date: '2026-09-17', hours_worked: 8, amount: 123456789,
};

async function mockAccountant(page: Page) {
  const now = Math.floor(Date.now() / 1000);
  const encode = (value: object) => Buffer.from(JSON.stringify(value)).toString('base64url');
  const token = `${encode({ alg: 'HS256' })}.${encode({ exp: now + 3600, user_id: 41, username: 'qa.accountant', role: 'accountant', sub: '41', nbf: now - 1, iat: now, jti: 'ui-qa' })}.fixture`;
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
      return;
    }
    if (path === '/api/v1/timesheets' && route.request().method() === 'GET') {
      await route.fulfill({ json: { status: 'success', data: [fixtureRow], pagination: { page: 1, pageSize: 500, totalPages: 1, totalRecords: 1 } } });
      return;
    }
    throw new Error(`Unexpected accountant fixture request: ${route.request().method()} ${path}`);
  });
}

async function expectNoPageOverflow(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
}

for (const width of [320, 390, 768, 1023, 1024, 1280]) {
  test(`accountant controls and payment dialogs remain usable at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await mockAccountant(page);
    await page.route(/https:\/\/fonts\.(googleapis|gstatic)\.com\//, route => route.abort());
    await page.goto('/accountant', { waitUntil: 'domcontentloaded' });
    await expect(page.getByText(fixtureRow.employeeName)).toBeVisible({ timeout: 15_000 });
    await expectNoPageOverflow(page);
    const tableRegion = page.getByRole('region', { name: 'Chấm công chờ duyệt' });
    await expect(tableRegion).toBeVisible();
    if (width < 640) {
      for (const control of [page.getByLabel('Từ ngày'), page.getByLabel('Đến ngày'), page.getByRole('checkbox', { name: 'Chọn tất cả' }).locator('..')]) {
        const box = await control.boundingBox();
        expect(box?.height).toBeGreaterThanOrEqual(44);
        expect(box?.width).toBeGreaterThanOrEqual(44);
      }
    }
    const approve = page.getByRole('button', { name: 'Duyệt (0)' });
    await expect(approve).toBeDisabled();
    if (width < 640) {
      // The surrounding label supplies the touch target while retaining the
      // familiar compact checkbox. Its outer edge must also select the row.
      const selectAll = page.getByRole('checkbox', { name: 'Chọn tất cả' });
      await selectAll.locator('..').click({ position: { x: 2, y: 2 } });
      await expect(page.getByRole('button', { name: 'Duyệt (1)' })).toBeEnabled();
      await selectAll.uncheck();
    }
    await page.getByRole('checkbox', { name: `Chọn ${fixtureRow.employeeName}` }).check();
    await expect(page.getByRole('button', { name: 'Duyệt (1)' })).toBeEnabled();
    await page.getByLabel('Từ ngày').fill('2026-09-16');
    await expect(page.getByRole('button', { name: 'Duyệt (0)' })).toBeDisabled();

    await page.getByRole('tab', { name: 'Nhập KQ chuyển lô' }).click();
    const uploadTrigger = page.getByRole('button', { name: 'Tải lên file kết quả' });
    // Safari does not focus buttons on pointer click. Exercise the keyboard
    // contract explicitly so Escape returns to the user's actual opener.
    await uploadTrigger.focus();
    await page.keyboard.press('Enter');
    const dialog = page.getByRole('dialog', { name: 'Kết quả chuyển tiền' });
    await expect(dialog).toBeVisible();
    await dialog.evaluate(element => Promise.all(element.getAnimations().map(animation => animation.finished)));
    const box = await dialog.boundingBox();
    expect(box!.x).toBeGreaterThanOrEqual(0);
    expect(box!.x + box!.width).toBeLessThanOrEqual(width);
    if (width < 1024) {
      expect(box!.x).toBeCloseTo(0, 0);
      expect(box!.x + box!.width).toBeCloseTo(width, 0);
    }
    await expectNoPageOverflow(page);
    await page.keyboard.press('Escape');
    await expect(dialog).not.toBeVisible();
    await expect(uploadTrigger).toBeFocused();

    await page.getByRole('tab', { name: 'Xuất sao kê' }).click();
    await page.getByRole('button', { name: 'Mở hộp thoại xuất sao kê' }).click();
    const reportDialog = page.getByRole('dialog', { name: 'Xuất sao kê thanh toán' });
    await expect(reportDialog).toBeVisible();
    await reportDialog.evaluate(element => Promise.all(element.getAnimations().map(animation => animation.finished)));
    if (width < 1024) {
      const reportBox = await reportDialog.boundingBox();
      expect(reportBox!.x).toBeCloseTo(0, 0);
      expect(reportBox!.x + reportBox!.width).toBeCloseTo(width, 0);
    }
    const visibleTitle = reportDialog.getByRole('heading', { name: 'Xuất sao kê thanh toán' }).filter({ visible: true });
    // The final close control is the header icon; the footer also has Đóng.
    const closeBox = await reportDialog.getByRole('button', { name: 'Đóng', exact: true }).last().boundingBox();
    const titleBox = await visibleTitle.boundingBox();
    const headerBox = await visibleTitle.locator('..').boundingBox();
    expect(closeBox!.y + closeBox!.height).toBeLessThanOrEqual(headerBox!.y + headerBox!.height);
    expect(titleBox!.x + titleBox!.width).toBeLessThanOrEqual(closeBox!.x);
    expect(await reportDialog.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);
    await expectNoPageOverflow(page);
    await page.keyboard.press('Escape');
  });
}
