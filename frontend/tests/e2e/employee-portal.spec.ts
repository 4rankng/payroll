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

  // Keep the suite isolated from live services and fail loudly when the portal
  // gains an API dependency that is not represented by a synthetic fixture.
  await page.route("**/api/v1/**", (route) => {
    throw new Error(`Unexpected employee portal API request: ${route.request().url()}`);
  });

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
  await page.route("**/api/v1/notifications?*", (route) => route.fulfill({
    json: {
      ...json([]),
      pagination: { page: 1, pageSize: 20, totalPages: 0, totalRecords: 0 },
    },
  }));
}

async function clearPersistedEmployeeQueries(page: Page) {
  await page.evaluate(() => sessionStorage.removeItem("payroll-query-cache"));
}

test.describe("mobile employee payroll dashboard", () => {
  test.beforeEach(async ({ page }) => {
    await mockEmployeePortal(page);
    await page.goto("/employee");
    await expect(page.getByText("Số tiền có thể ứng")).toBeVisible();
  });

  test("preserves hierarchy and prevents overflow at supported widths", async ({ page }) => {
    const walletArtwork = page.locator('img[src="/employee-pay-wallet.png"]');
    await expect(walletArtwork).toBeVisible();
    expect(await walletArtwork.evaluate((image: HTMLImageElement) => image.naturalWidth)).toBeGreaterThan(0);

    for (const viewport of [
      { width: 360, height: 800 },
      { width: 390, height: 844 },
      { width: 393, height: 873 },
      { width: 430, height: 932 },
    ]) {
      await page.setViewportSize(viewport);
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

  test("keeps the approved salary-advance information order", async ({ page }) => {
    const sectionOrder = await page.evaluate(() => {
      const ids = ["employee-advance-request", "employee-history", "employee-bank"];
      return ids.map((id) => document.getElementById(id)?.getBoundingClientRect().top ?? -1);
    });

    expect(sectionOrder[0]).toBeLessThan(sectionOrder[1]);
    expect(sectionOrder[1]).toBeLessThan(sectionOrder[2]);
  });

  test("keeps illustration motion subtle and honors reduced-motion", async ({ page }) => {
    const artwork = page.locator(".employee-pay-art");
    await expect(artwork).toBeVisible();

    const activeAnimation = await artwork.evaluate((element) => getComputedStyle(element).animationName);
    expect(activeAnimation).toContain("employee-pay-float");

    await page.emulateMedia({ reducedMotion: "reduce" });
    const reducedMotion = await artwork.evaluate((element) => {
      const style = getComputedStyle(element);
      return {
        duration: style.animationDuration,
        iterations: style.animationIterationCount,
      };
    });
    expect(reducedMotion.iterations).toBe("1");
    expect(["0.01ms", "1e-05s"]).toContain(reducedMotion.duration);
  });

  test("renders exhausted and empty states without duplicating history navigation", async ({ page }) => {
    await page.unroute("**/api/v1/me/advance-payment");
    await page.unroute("**/api/v1/me/advance-payment/history*");
    await page.route("**/api/v1/me/advance-payment", (route) => route.fulfill({
      json: json({
        forMonth: "2026-07",
        maxAdvanceAmount: 5_000_000,
        completedAmount: 5_000_000,
        pendingAmount: 0,
        remainingAmount: 0,
        canRequest: false,
        feePercentage: 2,
        minFee: 10_000,
        hasFlexible: true,
        quotas: [{
          forMonth: "2026-07",
          maxAdvanceAmount: 5_000_000,
          completedAmount: 5_000_000,
          pendingAmount: 0,
          remainingAmount: 0,
        }],
      }),
    }));
    await page.route("**/api/v1/me/advance-payment/history*", (route) => route.fulfill({
      json: {
        ...json([]),
        pagination: { page: 1, pageSize: 100, totalPages: 0, totalRecords: 0 },
      },
    }));

    await clearPersistedEmployeeQueries(page);
    await page.reload();

    await expect(page.getByText("Đã dùng hết hạn mức")).toBeVisible();
    await expect(page.getByRole("button", { name: "Xem lịch sử yêu cầu" })).toHaveCount(0);
    await expect(page.getByText("Chưa có yêu cầu ứng lương")).toBeVisible();
    await expect(page.getByText("0 yêu cầu")).toBeVisible();
    const emptyArtwork = page.locator('img[src="/advance-payment-empty-state.png"]');
    await expect(emptyArtwork).toBeVisible();
    expect(await emptyArtwork.evaluate((image: HTMLImageElement) => image.naturalWidth)).toBeGreaterThan(0);
  });

  test("shows all-time requests newest first with five visible rows", async ({ page }) => {
    const requests = Array.from({ length: 7 }, (_, index) => ({
      id: index + 1,
      requestAmount: 1_000_000 + index * 10_000,
      fee: 20_000,
      netAmount: 980_000 + index * 10_000,
      status: "COMPLETED",
      forMonth: index < 3 ? "2026-07" : "2026-06",
      createdAt: `2026-07-${String(10 - index).padStart(2, "0")}T08:00:00+07:00`,
    }));
    await page.unroute("**/api/v1/me/advance-payment/history*");
    await page.route("**/api/v1/me/advance-payment/history*", (route) => route.fulfill({
      json: {
        ...json(requests),
        pagination: { page: 1, pageSize: 100, totalPages: 1, totalRecords: requests.length },
      },
    }));

    await clearPersistedEmployeeQueries(page);
    await page.reload();

    await expect(page.getByText("7 yêu cầu")).toBeVisible();
    const history = page.getByLabel("Lịch sử yêu cầu, cuộn để xem thêm");
    await expect(history).toBeVisible();
    const dimensions = await history.evaluate((element) => ({
      clientHeight: element.clientHeight,
      scrollHeight: element.scrollHeight,
    }));
    expect(dimensions.clientHeight).toBeLessThanOrEqual(390);
    expect(dimensions.scrollHeight).toBeGreaterThan(dimensions.clientHeight);
    await expect(history.getByRole("button").first()).toContainText("10/07/2026");
  });

  test("supports month navigation, transaction disclosure, account menu, and request confirmation", async ({ page }) => {
    await page.getByRole("button", { name: "Xem tháng trước" }).click();
    await expect(page).toHaveURL(/month=\d{4}-\d{2}/);

    await page.getByRole("button", { name: /Xem phí và chi tiết/ }).click();
    await expect(page.getByText("Phí giao dịch")).toBeVisible();

    await page.getByRole("button", { name: "Menu tài khoản" }).click();
    await expect(page.getByText("Đổi mật khẩu")).toBeVisible();
    await page.mouse.click(24, 220);
    await expect(page.getByText("Đổi mật khẩu")).toBeHidden();

    await page.getByRole("button", { name: "Xem tháng sau" }).click();

    await page.getByRole("button", { name: "50%" }).click();
    await expect(page.getByRole("button", { name: "Yêu cầu ứng lương" })).toBeEnabled();
    await page.getByRole("button", { name: "Yêu cầu ứng lương" }).click();
    await expect(page.getByRole("dialog").getByRole("heading", { name: "Xác nhận yêu cầu ứng lương" }).last()).toBeVisible();
    await expect(page.getByRole("dialog").getByText("123456789012345678901234567890")).toBeVisible();
  });

  test("keeps failed profile chrome non-interactive and retryable", async ({ page }) => {
    await page.unroute("**/api/v1/me");
    // A redacted synthetic null payload exercises the same unresolved-profile
    // chrome deterministically without coupling the test to transport retries.
    await page.route("**/api/v1/me", (route) => route.fulfill({ json: json(null) }));
    await clearPersistedEmployeeQueries(page);
    await page.reload();

    await expect(page.getByRole("heading", { name: "Chưa tải được hồ sơ" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Tải lại" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Thông báo" })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Menu tài khoản" })).toHaveCount(0);
  });

  test("grows the attendance toolbar and reserves content space at 200% text", async ({ page }) => {
    await page.unroute("**/api/v1/me");
    await page.route("**/api/v1/me", (route) => route.fulfill({
      json: json({
        id: 77,
        fullname: "Nguyễn An",
        username: "employee.mobile",
        payment_schedule: "flexible",
        check_in_enabled: true,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-07-01T00:00:00Z",
      }),
    }));
    await page.route("**/api/v1/mobile/attendance/today", (route) => route.fulfill({ json: json(null) }));
    await page.route("**/api/v1/mobile/attendance/history*", (route) => route.fulfill({ json: json([]) }));
    await clearPersistedEmployeeQueries(page);
    await page.reload();
    await page.evaluate(() => { document.documentElement.style.fontSize = "200%"; });

    const toolbar = page.getByRole("toolbar", { name: "Hành động nhân viên" });
    await expect(toolbar).toBeVisible();
    const layout = await page.evaluate(() => {
      const dock = document.querySelector<HTMLElement>(".employee-attendance-action-dock");
      const main = document.querySelector<HTMLElement>(".employee-portal-card-stack");
      const label = dock?.querySelectorAll("button")[1]?.querySelector("span");
      return {
        dockHeight: dock?.getBoundingClientRect().height ?? 0,
        reservedBottom: Number.parseFloat(main ? getComputedStyle(main).paddingBottom : "0"),
        labelOverflow: label ? label.scrollWidth > label.clientWidth : true,
      };
    });

    expect(layout.dockHeight).toBeGreaterThan(68);
    expect(layout.reservedBottom).toBeGreaterThanOrEqual(layout.dockHeight);
    expect(layout.labelOverflow).toBe(false);
  });
});
