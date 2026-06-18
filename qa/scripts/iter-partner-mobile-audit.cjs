const puppeteer = require("puppeteer-core");

const HOST = process.env.HOST || "http://localhost:3000";
const CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const OUT_DIR = "/tmp/payroll-partner-mobile";

const routes = [
  { name: "dashboard", path: "/partner/dashboard" },
  { name: "projects", path: "/partner/projects" },
  { name: "employees", path: "/partner/employees" },
  { name: "timesheet", path: "/partner/timesheet" },
  { name: "payment-history", path: "/partner/timesheet/payment-history" },
];

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function safeClick(page, selector) {
  const el = await page.$(selector);
  if (!el) return false;
  await el.click();
  return true;
}

(async () => {
  const fs = require("fs");
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: "new",
    args: ["--no-sandbox"],
  });

  const page = await browser.newPage();
  const errors = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(msg.text());
  });

  await page.setViewport({
    width: 390,
    height: 844,
    isMobile: true,
    hasTouch: true,
    deviceScaleFactor: 3,
  });

  await page.goto(`${HOST}/login`, { waitUntil: "networkidle2" });
  await page.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  });
  await page.goto(`${HOST}/login`, { waitUntil: "networkidle2" });
  const username = await page.$('input[name="username"], input[type="text"]');
  const password = await page.$('input[name="password"], input[type="password"]');
  if (username && password) {
    await username.click({ clickCount: 3 });
    await username.type("ketoan");
    await password.click({ clickCount: 3 });
    await password.type("Admin123");
    await Promise.allSettled([
      page.waitForNavigation({ waitUntil: "networkidle2", timeout: 8000 }),
      safeClick(page, 'button[type="submit"]'),
    ]);
  }

  await page.goto(`${HOST}/partner/dashboard`, { waitUntil: "networkidle2" });
  const afterLoginText = await page.evaluate(() => document.body.innerText);
  if (/Phiên làm việc|Chào mừng trở lại|TÊN ĐĂNG NHẬP/.test(afterLoginText)) {
    console.log(JSON.stringify({
      blocked: true,
      reason: "Partner login did not authenticate; local backend/auth data may be invalid.",
      url: page.url(),
      errors,
      text: afterLoginText.slice(0, 600),
    }, null, 2));
    await browser.close();
    process.exit(0);
  }

  const results = [];
  for (const route of routes) {
    await page.goto(`${HOST}${route.path}`, { waitUntil: "networkidle2" });
    await sleep(600);

    const metrics = await page.evaluate(() => {
      const doc = document.documentElement;
      const body = document.body;
      const buttons = [...document.querySelectorAll("button")].map((button) => {
        const rect = button.getBoundingClientRect();
        return {
          text: button.innerText.trim().slice(0, 40),
          width: Math.round(rect.width),
          height: Math.round(rect.height),
        };
      });
      const smallButtons = buttons.filter((b) => b.width > 0 && b.height > 0 && (b.width < 36 || b.height < 36));
      return {
        title: body.innerText.split("\n").filter(Boolean).slice(0, 8),
        scrollWidth: doc.scrollWidth,
        clientWidth: doc.clientWidth,
        horizontalOverflow: doc.scrollWidth > doc.clientWidth + 1,
        smallButtons: smallButtons.slice(0, 8),
        buttonCount: buttons.length,
      };
    });

    const screenshot = `${OUT_DIR}/${route.name}.png`;
    await page.screenshot({ path: screenshot, fullPage: true });
    results.push({ ...route, screenshot, ...metrics });
  }

  console.log(JSON.stringify({ errors, results }, null, 2));
  await browser.close();
})();
