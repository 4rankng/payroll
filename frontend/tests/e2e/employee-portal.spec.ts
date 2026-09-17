import { test, expect, type Page } from "@playwright/test";

// Service-worker fetches bypass Playwright's page routes, particularly in
// WebKit. This synthetic suite must never send fixture tokens to the live API.
test.use({ serviceWorkers: "block" });

const json = (data: unknown) => ({ status: "success", data });
const EMPLOYEE_TEST_DATE = new Date("2026-07-10T12:00:00+07:00");

function employeeToken(): string {
  const encode = (value: object) => Buffer.from(JSON.stringify(value)).toString("base64url");
  const now = Math.floor(EMPLOYEE_TEST_DATE.getTime() / 1000);
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
  // Keep July payroll fixtures deterministic while allowing time-based UI to run.
  await page.clock.install({ time: EMPLOYEE_TEST_DATE });
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

  await page.route("**/api/v1/auth/me", (route) => route.fulfill({
    json: json({ id: 77, username: "employee.mobile", fullname: "Nguyễn Thị Nhân Viên Có Tên Rất Dài", role: "employee", must_change_password: false }),
  }));
  await page.route("**/api/v1/me/ad-banner", (route) => route.fulfill({ json: json(null) }));

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
  await page.route("**/api/v1/me/check-in-advance", (route) => route.fulfill({
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
  await page.addInitScript(() => sessionStorage.removeItem("payroll-query-cache"));
  await page.evaluate(() => sessionStorage.removeItem("payroll-query-cache"));
}

test.describe("mobile employee payroll dashboard", () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await mockEmployeePortal(page);
    await page.goto("/employee", { waitUntil: "domcontentloaded" });
    await expect(page.getByText("Có thể ứng", { exact: true })).toBeVisible();
  });

  test("preserves hierarchy and prevents overflow at supported widths", async ({ page }) => {
    for (const viewport of [
      { width: 320, height: 800 },
      { width: 360, height: 800 },
      { width: 390, height: 844 },
      { width: 393, height: 873 },
      { width: 430, height: 932 },
      { width: 1280, height: 900 },
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

  test("shows a completed shift as a compact wage and advance pair", async ({ page }) => {
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
    await page.unroute("**/api/v1/me/check-in-advance");
    await page.route("**/api/v1/me/check-in-advance", (route) => route.fulfill({
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
        advancePercentage: 70,
      }),
    }));
    await page.route("**/api/v1/mobile/attendance/today", (route) => route.fulfill({ json: json(null) }));
    await page.route("**/api/v1/mobile/attendance/history*", (route) => route.fulfill({
      json: json([{
        id: 42,
        project_id: 10,
        employee_id: 77,
        date: "2026-07-09",
        check_in_time: "2026-07-09T08:36:00+07:00",
        check_out_time: "2026-07-09T19:00:00+07:00",
        earning_amount: 252_000,
        salary_status: "recorded",
        status: "completed",
      }]),
    }));
    await clearPersistedEmployeeQueries(page);
    await page.reload();

    const summary = page.getByTestId("attendance-earnings-summary");
    await expect(summary).toBeVisible();
    await expect(summary).toContainText("Tiền công");
    await expect(summary).toContainText("252.000₫");
    await expect(summary).toContainText("Được ứng (70%)");
    await expect(summary).toContainText("+176.400₫");
    await expect(summary).not.toContainText("+252.000₫");

    for (const viewport of [
      { width: 390, height: 844, columns: 2 },
      { width: 320, height: 844, columns: 2 },
    ]) {
      await page.setViewportSize(viewport);
      const layout = await summary.evaluate((element) => ({
        columns: getComputedStyle(element).gridTemplateColumns.trim().split(/\s+/).length,
        overflow: element.scrollWidth > element.clientWidth,
      }));

      expect(layout.columns).toBe(viewport.columns);
      expect(layout.overflow).toBe(false);
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(viewport.width);
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

  test("shows the final available amount with reduced-motion enabled", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.reload();
    await expect(page.getByText("4.000.000 ₫", { exact: true })).toBeVisible();
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

  test("recovers a failed fee preview and disables stale quotes when the amount changes", async ({ page }) => {
    let failFee = true;
    let holdFee = false;
    let releaseFee: (() => void) | undefined;
    await page.route("**/api/v1/me/advance-payment/calculate-fee", async (route) => {
      if (failFee) {
        await route.fulfill({ status: 503, json: { status: "error", message: "Fee unavailable" } });
        return;
      }
      if (holdFee) await new Promise<void>((resolve) => { releaseFee = resolve; });
      const { amount } = route.request().postDataJSON() as { amount: number };
      await route.fulfill({ json: json({ fee: 40_000, netAmount: amount - 40_000 }) });
    });
    await page.getByRole("button", { name: "50%" }).click();
    const request = page.getByRole("button", { name: "Yêu cầu ứng lương" });
    await expect(page.getByRole("alert")).toContainText("Không thể tính phí chuyển tiền");
    await expect(request).toBeDisabled();
    failFee = false;
    await page.getByRole("button", { name: "Tính lại phí" }).click();
    await expect(request).toBeEnabled();
    holdFee = true;
    await page.getByRole("button", { name: "25%" }).click();
    await expect(request).toBeDisabled();
    await expect(page.locator("#employee-advance-request").getByText("1.960.000 ₫", { exact: true })).toHaveCount(0);
    await expect.poll(() => Boolean(releaseFee)).toBe(true);
    releaseFee?.();
    await expect(request).toBeEnabled();
    await request.click();
    await expect(page.getByRole("dialog").getByText("960.000 ₫", { exact: true })).toBeVisible();
  });

  test("clears the fee preview and amount after a synthetic successful request", async ({ page }) => {
    let submitted = 0;
    await page.route("**/api/v1/me/advance-payment/request", async (route) => {
      expect(route.request().method()).toBe("POST");
      expect(route.request().postDataJSON()).toEqual({ amount: 2_000_000, forMonth: "2026-07" });
      submitted += 1;
      await route.fulfill({ json: json({ id: 11, status: "PENDING", requestAmount: 2_000_000, forMonth: "2026-07" }) });
    });
    await page.getByRole("button", { name: "50%" }).click();
    const request = page.getByRole("button", { name: "Yêu cầu ứng lương" });
    await expect(request).toBeEnabled();
    await request.click();
    await page.getByRole("button", { name: "Xác nhận giao dịch", exact: true }).click();
    await expect(page.getByRole("dialog")).toBeHidden();
    await expect(page.getByRole("textbox", { name: "Số tiền muốn ứng" })).toHaveValue("");
    await expect(request).toBeDisabled();
    await expect(page.locator("#employee-advance-request").getByText("1.960.000 ₫", { exact: true })).toHaveCount(0);
    expect(submitted).toBe(1);
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

const regularTimesheets = {
  ...json([{
    id: 81,
    date: '2026-07-09',
    project: { id: 10, name: 'Dự án kiểm thử', code: 'QA', client_name: 'Khách hàng kiểm thử' },
    hours_worked: 8,
    amount: 12_345_678,
    paid_amount: 2_345_678,
    timesheet_status: 'approved',
    payment_status: 'pending',
    payment_date: null,
    approved_at: null,
    approved_by: null,
    created_at: '2026-07-09T00:00:00+07:00',
  }]),
  pagination: { page: 1, pageSize: 50, totalPages: 1, totalRecords: 1 },
};
const paymentSetting = json({ id: 1, key: 'bulk_transfer_payment_percentage', value: '0.70' });

async function mockRegularEmployee(page: Page) {
  await mockEmployeePortal(page);
  await page.route('**/api/v1/me', (route) => route.fulfill({ json: json({
    id: 77,
    fullname: 'Nguyễn Thị Nhân Viên Có Tên Rất Dài',
    username: 'employee.mobile',
    payment_schedule: 'weekly',
    check_in_enabled: false,
    bank: { id: 1, branch_name: 'Ngân hàng kiểm thử' },
    bank_account_number: '123456789012345678901234567890',
    bank_account_name: 'NGUYEN THI NHAN VIEN',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
  }) }));
  await page.route('**/api/v1/me/timesheet?*', (route) => route.fulfill({ json: regularTimesheets }));
  await page.route('**/api/v1/settings/key/bulk_transfer_payment_percentage', (route) => route.fulfill({ json: paymentSetting }));
}

test.describe('regular employee payroll recovery', () => {
  test('keeps full wage values and accessible account controls at desktop and narrow mobile sizes', async ({ page }) => {
    await mockRegularEmployee(page);
    await page.goto('/employee');
    await expect(page.getByRole('button', { name: 'Ẩn số tiền' })).toBeVisible();
    for (const width of [1280, 390, 320]) {
      await page.setViewportSize({ width, height: 900 });
      await expect(page.getByText('12.345.678 ₫', { exact: true }).first()).toBeVisible();
      const metrics = await page.evaluate(() => ({
        width: innerWidth,
        scrollWidth: document.documentElement.scrollWidth,
        clipped: Array.from(document.querySelectorAll('h1, [aria-label="Thu nhập tháng"] p, main p, main dd'))
          .filter((element) => element.clientWidth > 0 && element.scrollWidth > element.clientWidth + 1)
          .map((element) => element.textContent),
      }));
      expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.width);
      expect(metrics.clipped).toEqual([]);
      await page.getByRole('button', { name: 'Menu tài khoản' }).click();
      await page.getByRole('menuitem', { name: 'Đổi mật khẩu' }).click();
      await page.getByRole('button', { name: 'Đổi mật khẩu', exact: true }).click();
      await expect(page.getByLabel('Mật khẩu hiện tại', { exact: true })).toHaveAttribute('aria-invalid', 'true');
      await expect(page.getByText('Nhập mật khẩu hiện tại.', { exact: true })).toBeVisible();
      const buttons = await page.getByRole('dialog').locator('button:visible').evaluateAll((elements) => elements.map((element) => ({ name: element.getAttribute('aria-label') || element.textContent, width: element.getBoundingClientRect().width, height: element.getBoundingClientRect().height })));
      expect(buttons.filter((button) => button.width < 44 || button.height < 44)).toEqual([]);
      await page.keyboard.press('Escape');
    }
  });

  for (const failedDependency of ['timesheet', 'payment-setting']) {
    test(`shows a recoverable error instead of empty or paid wages when ${failedDependency} fails`, async ({ page }) => {
      await mockRegularEmployee(page);
      const endpoint = failedDependency === 'timesheet' ? '**/api/v1/me/timesheet?*' : '**/api/v1/settings/key/bulk_transfer_payment_percentage';
      await page.route(endpoint, (route) => route.fulfill({ status: 400, json: { status: 'error', message: 'Dữ liệu tạm thời chưa sẵn sàng' } }));
      await page.goto('/employee');
      const panel = page.locator('#employee-timesheets');
      await expect(panel.getByRole('alert')).toContainText('Chưa tải được bảng công', { timeout: 15_000 });
      await expect(page.getByRole('button', { name: 'Ẩn số tiền' })).toHaveCount(0);
      await expect(page.getByText('Chưa có bảng công')).toHaveCount(0);
      await expect(page.getByText('Đã trả đủ')).toHaveCount(0);
      await page.route(endpoint, (route) => route.fulfill({ json: failedDependency === 'timesheet' ? regularTimesheets : paymentSetting }));
      await panel.getByRole('button', { name: 'Tải lại', exact: true }).click();
      await expect(page.getByRole('button', { name: 'Ẩn số tiền' })).toBeVisible();
      await expect(page.getByText('12.345.678 ₫', { exact: true }).first()).toBeVisible();
      await expect(panel.getByRole('alert')).toHaveCount(0);
    });
  }
});
