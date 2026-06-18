const fs = require("fs");
const path = require("path");
const puppeteer = require("puppeteer-core");

const HOST = process.env.QA_HOST || "http://localhost:3000";
const API_HOST = process.env.API_HOST || "http://localhost:8080";
const CHROME_PATH = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const PASSWORD = "Admin123";
const OUT_DIR = path.resolve(__dirname, "../screenshots/role-visual-audit");

const roles = {
  admin: {
    username: "frankng",
    routes: [
      "/admin",
      "/admin/users",
      "/admin/projects",
      "/admin/employees",
      "/admin/timesheet",
      "/admin/ledger",
      "/admin/transactions",
      "/admin/loans",
      "/admin/loans/lenders",
      "/admin/advance-payments",
      "/admin/advance-payments/employees",
      "/admin/advance-payment-fees",
      "/admin/settings",
      "/admin/wallet",
      "/admin/system-health",
      "/admin/cron-health",
      "/admin/audit-log",
      "/admin/send-notification",
    ],
  },
  partner: {
    username: "ketoan",
    routes: [
      "/partner/dashboard",
      "/partner/projects",
      "/partner/employees",
      "/partner/timesheet",
      "/partner/timesheet/payment-history",
    ],
  },
  employee: {
    username: process.env.EMPLOYEE_USERNAME || null,
    routes: ["/employee"],
  },
};

const viewports = {
  desktop: { width: 1440, height: 900, isMobile: false },
  mobile: { width: 390, height: 844, isMobile: true, hasTouch: true, deviceScaleFactor: 3 },
};

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

async function api(method, apiPath, body, token) {
  const res = await fetch(`${API_HOST}${apiPath}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let data = text;
  try {
    data = JSON.parse(text);
  } catch {}
  return { ok: res.ok, status: res.status, data };
}

function extract(payload) {
  return payload?.data?.data ?? payload?.data ?? payload;
}

async function loginToken(username) {
  const res = await api("POST", "/api/v1/auth/login", { username, password: PASSWORD });
  if (!res.ok) {
    throw new Error(`API login failed for ${username}: ${res.status} ${JSON.stringify(res.data).slice(0, 220)}`);
  }
  const data = extract(res.data);
  return data?.access_token || data?.token || res.data?.access_token || res.data?.token;
}

async function canLogin(username) {
  try {
    await loginToken(username);
    return true;
  } catch {
    return false;
  }
}

async function discoverUsername(adminToken, role, fallbacks = []) {
  for (const username of fallbacks) {
    if (username && await canLogin(username)) return username;
  }

  const candidates = [
    `/api/v1/users?role=${role}&pageSize=100`,
    "/api/v1/users?pageSize=100",
  ];

  for (const candidate of candidates) {
    const res = await api("GET", candidate, null, adminToken);
    if (!res.ok) continue;

    const data = extract(res.data);
    const rows = Array.isArray(data) ? data : data?.items || data?.users || [];
    for (const row of rows) {
      if (row?.role === role && row?.username && row?.status !== "inactive" && await canLogin(row.username)) {
        return row.username;
      }
    }
  }

  return null;
}

async function discoverEmployeeUsername(adminToken) {
  const candidates = [
    "/api/v1/users?pageSize=100",
    "/api/v1/users?role=employee&pageSize=100",
    "/api/v1/employees?pageSize=100",
  ];

  for (const candidate of candidates) {
    const res = await api("GET", candidate, null, adminToken);
    if (!res.ok) continue;

    const data = extract(res.data);
    const rows = Array.isArray(data) ? data : data?.items || data?.users || data?.employees || [];
    const employee = rows.find((row) =>
      row?.role === "employee" &&
      row?.username &&
      row?.status !== "inactive"
    ) || rows.find((row) => row?.username && row?.employee_code);

    if (employee?.username) return employee.username;
  }

  return null;
}

async function seedAuth(page, username) {
  const token = await loginToken(username);
  const payload = JSON.parse(Buffer.from(token.split(".")[1], "base64url").toString("utf8"));

  await page.goto(`${HOST}/login`, { waitUntil: "domcontentloaded" });
  await page.evaluate(({ tokenValue, payloadValue }) => {
    localStorage.setItem("auth_token", tokenValue);
    localStorage.setItem("userName", payloadValue.username);
    localStorage.setItem("userEmail", "");
    localStorage.setItem("userStatus", "active");
  }, { tokenValue: token, payloadValue: payload });

  return payload;
}

async function waitForSettledPage(page) {
  await page.waitForFunction(
    () => document.readyState === "complete" || document.readyState === "interactive",
    { timeout: 8000 },
  ).catch(() => {});
  await new Promise((resolve) => setTimeout(resolve, 900));
}

async function tryKeyboardSearch(page) {
  const searchSelectors = [
    "input[type='search']",
    "input[placeholder*='Tìm']",
    "input[placeholder*='tìm']",
    "input[aria-label*='Tìm']",
  ];
  const actions = [];

  for (const selector of searchSelectors) {
    const input = await page.$(selector);
    if (!input) continue;

    const before = page.url();
    await input.click({ clickCount: 3 }).catch(() => {});
    await page.keyboard.type("qa", { delay: 10 }).catch(() => {});
    await new Promise((resolve) => setTimeout(resolve, 450));
    await page.keyboard.down("Meta").catch(() => {});
    await page.keyboard.press("KeyA").catch(() => {});
    await page.keyboard.up("Meta").catch(() => {});
    await page.keyboard.press("Backspace").catch(() => {});
    actions.push({ type: "search", selector, urlChanged: page.url() !== before });
    break;
  }

  return actions;
}

async function clickSafeControls(page) {
  const labels = [
    "Bộ lọc",
    "Lọc",
    "Thêm",
    "Xuất",
    "Lịch sử",
    "Tải lên",
    "Nhập công",
    "Tài khoản",
    "Xóa lọc",
    "Trang trước",
    "Trang sau",
    "Tháng trước",
    "Tháng sau",
  ];
  const actions = [];

  for (const label of labels) {
    const handles = await page.$$("button, a, [role='button']");
    let clicked = false;
    for (const handle of handles) {
      const meta = await handle.evaluate((el) => {
        const text = (el.innerText || el.getAttribute("aria-label") || el.getAttribute("title") || "").trim();
        const rect = el.getBoundingClientRect();
        const disabled = Boolean(el.disabled || el.getAttribute("aria-disabled") === "true");
        return {
          text,
          visible: rect.width > 0 && rect.height > 0,
          disabled,
        };
      }).catch(() => null);

      if (!meta?.visible || meta.disabled || !meta.text.includes(label)) continue;

      const beforeUrl = page.url();
      await handle.click().catch(() => {});
      await new Promise((resolve) => setTimeout(resolve, 550));
      actions.push({
        type: "click",
        label,
        text: meta.text.slice(0, 80),
        urlChanged: page.url() !== beforeUrl,
        dialogOpen: Boolean(await page.$("[role='dialog'], [data-radix-popper-content-wrapper]")),
      });

      await page.keyboard.press("Escape").catch(() => {});
      await new Promise((resolve) => setTimeout(resolve, 200));
      clicked = true;
      break;
    }
    if (!clicked) actions.push({ type: "missing-control", label });
  }

  return actions;
}

async function inspectOpenOverlays(page) {
  return page.evaluate(() => {
    return Array.from(document.querySelectorAll("[role='dialog'], [data-radix-popper-content-wrapper]"))
      .map((el) => {
        const rect = el.getBoundingClientRect();
        return {
          text: (el.textContent || "").trim().slice(0, 160),
          width: Math.round(rect.width),
          height: Math.round(rect.height),
          left: Math.round(rect.left),
          top: Math.round(rect.top),
          clipped: rect.left < -1 || rect.top < -1 || rect.right > window.innerWidth + 1 || rect.bottom > window.innerHeight + 1,
        };
      });
  });
}

async function inspectPage(page) {
  return page.evaluate(() => {
    const doc = document.documentElement;
    const body = document.body;
    const text = body.innerText || "";
    const clickable = Array.from(document.querySelectorAll("button, a, input, select, textarea, [role='button']"));
    const smallTargets = clickable
      .map((el) => {
        const rect = el.getBoundingClientRect();
        const label = (el.innerText || el.getAttribute("aria-label") || el.getAttribute("placeholder") || el.tagName).trim();
        return { label: label.slice(0, 48), width: Math.round(rect.width), height: Math.round(rect.height) };
      })
      .filter((item) => item.width > 0 && item.height > 0 && (item.width < 36 || item.height < 36))
      .slice(0, 8);

    return {
      title: document.title,
      url: location.pathname,
      bodyChars: text.length,
      hasSpinner: Boolean(document.querySelector(".animate-spin")),
      horizontalOverflow: Math.max(doc.scrollWidth, body.scrollWidth) > window.innerWidth + 2,
      scrollWidth: Math.max(doc.scrollWidth, body.scrollWidth),
      viewportWidth: window.innerWidth,
      heading: document.querySelector("h1,h2")?.textContent?.trim() || "",
      buttonCount: document.querySelectorAll("button").length,
      inputCount: document.querySelectorAll("input, textarea, [role='combobox']").length,
      emptyStateText: Array.from(document.querySelectorAll("h2,h3,p"))
        .map((el) => el.textContent?.trim() || "")
        .find((text) => /không có|chưa có|trống|empty/i.test(text)) || "",
      smallTargets,
    };
  });
}

async function run() {
  ensureDir(OUT_DIR);

  const adminToken = await loginToken(roles.admin.username);
  roles.partner.username = await discoverUsername(adminToken, "partner", [roles.partner.username, "thanhmai"]);
  if (!roles.partner.username) {
    throw new Error("Could not discover a partner username.");
  }
  if (!roles.employee.username) {
    roles.employee.username = await discoverUsername(adminToken, "employee") || await discoverEmployeeUsername(adminToken);
  }
  if (!roles.employee.username) {
    throw new Error("Could not discover an employee username. Set EMPLOYEE_USERNAME and re-run.");
  }

  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: "new",
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  });

  const results = [];

  for (const [role, config] of Object.entries(roles)) {
    for (const [viewportName, viewport] of Object.entries(viewports)) {
      const page = await browser.newPage();
      const consoleEvents = [];
      page.on("console", (msg) => {
        if (["error", "warning"].includes(msg.type())) {
          consoleEvents.push(`${msg.type()}: ${msg.text()}`.slice(0, 300));
        }
      });
      page.on("pageerror", (error) => consoleEvents.push(`pageerror: ${error.message}`));

      await page.setViewport(viewport);
      const payload = await seedAuth(page, config.username);

      for (const route of config.routes) {
        const slug = `${role}-${viewportName}-${route.replace(/^\//, "").replace(/[/?=&]+/g, "-") || "root"}`;
        const shot = path.join(OUT_DIR, `${slug}.png`);
        const startedConsoleCount = consoleEvents.length;
        let inspection;
        let status = "ok";

        try {
          await page.goto(`${HOST}${route}`, { waitUntil: "networkidle0", timeout: 20000 });
          await waitForSettledPage(page);
          const searchActions = await tryKeyboardSearch(page);
          const clickActions = await clickSafeControls(page);
          const overlays = await inspectOpenOverlays(page);
          inspection = await inspectPage(page);
          await page.screenshot({ path: shot, fullPage: false });
          inspection.interactions = [...searchActions, ...clickActions];
          inspection.overlays = overlays;
        } catch (error) {
          status = "error";
          inspection = { error: error.message };
        }

        const newConsole = consoleEvents.slice(startedConsoleCount);
        results.push({
          role,
          username: config.username,
          authRole: payload.role,
          viewport: viewportName,
          route,
          status,
          screenshot: shot,
          console: newConsole.slice(0, 5),
          ...inspection,
        });
      }

      await page.close();
    }
  }

  await browser.close();

  const reportPath = path.join(OUT_DIR, "report.json");
  fs.writeFileSync(reportPath, JSON.stringify(results, null, 2));
  console.log(JSON.stringify({
    reportPath,
    employeeUsername: roles.employee.username,
    issueSummary: results
      .filter((item) => item.status !== "ok" || item.horizontalOverflow || item.console?.length || item.hasSpinner || item.bodyChars < 80 || item.smallTargets?.length)
      .map((item) => ({
        role: item.role,
        viewport: item.viewport,
        route: item.route,
        status: item.status,
        horizontalOverflow: item.horizontalOverflow,
        bodyChars: item.bodyChars,
        hasSpinner: item.hasSpinner,
        smallTargets: item.smallTargets?.length || 0,
        buttons: item.buttonCount || 0,
        inputs: item.inputCount || 0,
        console: item.console?.length || 0,
      })),
  }, null, 2));
}

run().catch((error) => {
  console.error(error);
  process.exit(1);
});
