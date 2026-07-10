import { test, expect, type Page } from "@playwright/test";

const json = (data: unknown) => ({ status: "success", data });

function employeeToken(): string {
  const encode = (value: object) => Buffer.from(JSON.stringify(value)).toString("base64url");
  const now = Math.floor(Date.now() / 1000);
  return `${encode({ alg: "HS256", typ: "JWT" })}.${encode({
    exp: now + 3600,
    user_id: 77,
    username: "employee.mobile",
    role: "employee",
    sub: "77",
    nbf: now - 1,
    iat: now,
    jti: "employee-portal-e2e",
  })}.test-signature`;
}

async function mockEmployeePortal(page: Page) {
  await page.addInitScript((token) => {
    localStorage.setItem("auth_token", token);
    localStorage.setItem("userRole", "employee");
    localStorage.setItem("userName", "Nguyễn Thị Nhân Viên Có Tên Rất Dài");
  }, employeeToken());

  // Keep background portal queries isolated from the live backend. More
  // specific handlers registered below take precedence over this fallback.
  await page.route("**/api/v1/**", (route) => route.fulfill({ json: json({}) }));

  await page.route("**/api/v1/me", (route) => route.fulfill({
    json: json({
      id: 77,
      fullname: "Nguyễn Thị Nhân Viên Có Tên Rất Dài",
      username: "employee.mobile",
      payment_schedule: "flexible",
      check_in_enabled: false,
      bank: {
        id: 1,
        branch_name: "Ngân hàng Thương mại Cổ phần Ngoại thương Việt Nam",
        branch_code: "VCB",
      },
      bank_account_number: "123456789012345678901234567890",
      bank_account_name: "NGUYEN THI NHAN VIEN CO TEN RAT DAI",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-07-01T00:00:00Z",
    }),
  }));

  await page.route("**/api/v1/me/advance-payment", (route) => route.fulfill({
    json: json({
      forMonth: "2026-07",
      maxAdvanceAmount: 5_000_000,
      completedAmount: 1_000_000,
      pendingAmount: 0,
      remainingAmount: 4_000_000,
      canRequest: true,
      feePercentage: 2,
      minFee: 10_000,
      hasFlexible: true,
      quotas: [{
        forMonth: "2026-07",
        maxAdvanceAmount: 5_000_000,
        completedAmount: 1_000_000,
        pendingAmount: 0,
        remainingAmount: 4_000_000,
      }],
    }),
  }));

  await page.route("**/api/v1/me/advance-payment/history*", (route) => route.fulfill({
    json: {
      ...json([{
        id: 10,
        requestAmount: 1_000_000,
        fee: 20_000,
        netAmount: 980_000,
        status: "COMPLETED",
        forMonth: "2026-07",
        createdAt: "2026-07-03T08:00:00+07:00",
      }]),
      pagination: { page: 1, pageSize: 50, totalPages: 1, totalRecords: 1 },
    },
  }));

  await page.route("**/api/v1/me/advance-payment/calculate-fee", (route) => route.fulfill({
    json: json({ fee: 40_000, netAmount: 1_960_000 }),
  }));

  await page.route("**/api/v1/notifications/unread", (route) => route.fulfill({
    json: json({ notifications: [], count: 0 }),
  }));
}

test.describe("mobile employee payroll dashboard", () => {
  test.beforeEach(async ({ page }) => {
    await mockEmployeePortal(page);
    await page.goto("/employee");
    await expect(page.getByText("Có thể ứng")).toBeVisible();
  });

  test("preserves hierarchy and prevents overflow at supported widths", async ({ page }) => {
    for (const width of [360, 390, 430]) {
      await page.setViewportSize({ width, height: 844 });
      await expect(page.getByRole("heading", { name: "Nguyễn Thị Nhân Viên Có Tên Rất Dài" })).toBeVisible();
      await expect(page.getByText("4.000.000 ₫")).toBeVisible();
      await expect(page.getByText("123456789012345678901234567890")).toBeVisible();

      const metrics = await page.evaluate(() => {
        const visibleButtons = Array.from(document.querySelectorAll<HTMLElement>("button"))
          .filter((element) => {
            const style = getComputedStyle(element);
            const rect = element.getBoundingClientRect();
            return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
          })
          .map((element) => {
            const rect = element.getBoundingClientRect();
            return { label: element.getAttribute("aria-label") || element.textContent?.trim(), width: rect.width, height: rect.height };
          });
        return {
          viewportWidth: document.documentElement.clientWidth,
          scrollWidth: document.documentElement.scrollWidth,
          undersized: visibleButtons.filter((button) => button.width < 44 || button.height < 44),
        };
      });

      expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.viewportWidth);
      expect(metrics.undersized).toEqual([]);
    }
  });

  test("supports month navigation, transaction disclosure, account menu, and request confirmation", async ({ page }) => {
    await page.getByRole("button", { name: "Xem tháng trước" }).click();
    await expect(page).toHaveURL(/month=\d{4}-\d{2}/);

    await page.getByRole("button", { name: /Xem phí và chi tiết/ }).click();
    await expect(page.getByText("Phí giao dịch")).toBeVisible();

    await page.getByRole("button", { name: "Menu tài khoản" }).click();
    await expect(page.getByText("Đổi mật khẩu")).toBeVisible();
    await page.keyboard.press("Escape");

    await page.getByRole("button", { name: "50%" }).click();
    await expect(page.getByRole("button", { name: "Tiếp tục" })).toBeEnabled();
    await page.getByRole("button", { name: "Tiếp tục" }).click();
    await expect(page.getByText("Xác nhận yêu cầu ứng lương")).toBeVisible();
    await expect(page.getByText("123456789012345678901234567890")).toBeVisible();
  });
});
