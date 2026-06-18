const puppeteer = require("puppeteer-core");

const HOST = process.env.HOST || "http://localhost:3000";
const CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";

(async () => {
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

  const username = await page.$('input[name="username"], input[type="text"]');
  const password = await page.$('input[name="password"], input[type="password"]');

  if (username && password) {
    await username.click({ clickCount: 3 });
    await username.type("frankng");
    await password.click({ clickCount: 3 });
    await password.type("Admin123");
    const submit = await page.$('button[type="submit"]');
    if (submit) {
      await Promise.allSettled([
        page.waitForNavigation({ waitUntil: "networkidle2", timeout: 8000 }),
        submit.click(),
      ]);
    }
  }

  await page.goto(`${HOST}/admin/advance-payments`, { waitUntil: "networkidle2" });
  await page.screenshot({
    path: "/tmp/payroll-mobile-advance-payments.png",
    fullPage: true,
  });

  const bodyText = await page.evaluate(() => document.body.innerText.slice(0, 1200));
  console.log(JSON.stringify({
    url: page.url(),
    screenshot: "/tmp/payroll-mobile-advance-payments.png",
    consoleErrors: errors.slice(0, 8),
    bodyText,
  }, null, 2));

  await browser.close();
})();
