const puppeteer = require('puppeteer-core');
const fs = require('fs');
const path = require('path');

const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const DL_DIR = '/tmp/qa-downloads';

(async () => {
  fs.mkdirSync(DL_DIR, { recursive: true });
  for (const f of fs.readdirSync(DL_DIR)) fs.unlinkSync(path.join(DL_DIR, f));

  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: 'new',
    args: ['--no-sandbox'],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  const client = await page.target().createCDPSession();
  await client.send('Page.setDownloadBehavior', { behavior: 'allow', downloadPath: DL_DIR });

  const errors = [];
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
  page.on('pageerror', (e) => errors.push('PAGE: ' + e.message));

  // Login
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' });
  await page.type('input[type="text"]', 'frankng', { delay: 20 });
  await page.type('input[type="password"]', 'Admin123', { delay: 20 });
  await page.click('button[type="submit"]').catch(() => {});
  await page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 1500));

  // Phase 1: go to Timesheet page
  console.log('--- Phase 1: Timesheet page ---');
  await page.goto(`${BASE}/admin/timesheet`, { waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 3500));
  await page.screenshot({ path: '/tmp/qa-04-timesheets.png' });
  console.log('Timesheet URL:', page.url());

  // Click "Chuyển OnePay" (header button — not in dropdown)
  console.log('--- Click Chuyển OnePay ---');
  const clicked = await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const t = btns.find(b => /chuyển.*onepay/i.test(b.innerText || ''));
    if (t) { t.click(); return t.innerText.trim(); }
    return null;
  });
  console.log('Clicked button:', clicked);
  await new Promise(r => setTimeout(r, 5000));
  await page.screenshot({ path: '/tmp/qa-06-after-chuyen.png' });

  // Check downloads
  const downloads = fs.readdirSync(DL_DIR);
  console.log('--- Downloads:', downloads.length, '---');
  for (const f of downloads) {
    const stat = fs.statSync(path.join(DL_DIR, f));
    console.log('  ', f, stat.size, 'bytes');
  }

  // Check toasts
  const bodyText = await page.evaluate(() => document.body.innerText);
  const last500 = bodyText.slice(-500);
  console.log('--- Body tail (last 500 chars): ---');
  console.log(last500);

  console.log('--- Console errors:', errors.length, '---');
  errors.slice(0, 10).forEach(e => console.log('  ERR:', e.slice(0, 200)));

  await browser.close();
})();
