const puppeteer = require('puppeteer-core');
const fs = require('fs');

const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

(async () => {
  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: 'new',
    args: ['--no-sandbox'],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  // Capture console + network for diagnostics
  const consoleErrors = [];
  const networkErrors = [];
  page.on('console', (m) => { if (m.type() === 'error') consoleErrors.push(m.text()); });
  page.on('requestfailed', (req) => { networkErrors.push(`${req.url()} :: ${req.failure()?.errorText}`); });

  // Step 1: go to /admin → expect login redirect
  console.log('--- Step 1: navigate to /admin ---');
  await page.goto(`${BASE}/admin`, { waitUntil: 'networkidle2', timeout: 15000 }).catch(e => console.log('goto done:', e.message));
  await new Promise(r => setTimeout(r, 1500));
  await page.screenshot({ path: '/tmp/qa-01-landing.png' });
  console.log('URL after /admin:', page.url());

  // Look for login form
  const hasUsernameField = await page.$('input[name="username"], input#username, input[type="text"]') !== null;
  const hasPasswordField = await page.$('input[type="password"]') !== null;
  console.log('Has username field:', hasUsernameField, 'Has password field:', hasPasswordField);

  // Fill login
  if (hasUsernameField && hasPasswordField) {
    console.log('--- Step 2: login as frankng/Admin123 ---');
    await page.type('input[type="text"]', 'frankng', { delay: 20 });
    await page.type('input[type="password"]', 'Admin123', { delay: 20 });
    await page.screenshot({ path: '/tmp/qa-02-login-filled.png' });
    await page.click('button[type="submit"]').catch(() => {});
    await page.waitForNavigation({ waitUntil: 'networkidle2', timeout: 10000 }).catch(() => {});
    await new Promise(r => setTimeout(r, 2000));
    await page.screenshot({ path: '/tmp/qa-03-after-login.png' });
    console.log('URL after login:', page.url());
  }

  console.log('--- Console errors:', consoleErrors.length, '---');
  consoleErrors.slice(0, 10).forEach(e => console.log('  ERR:', e.slice(0, 200)));
  console.log('--- Network errors:', networkErrors.length, '---');
  networkErrors.slice(0, 10).forEach(e => console.log('  NET:', e));

  await browser.close();
})();
