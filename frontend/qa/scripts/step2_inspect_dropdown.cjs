const puppeteer = require('puppeteer-core');
const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

(async () => {
  const browser = await puppeteer.launch({ executablePath: CHROME, headless: 'new', args: ['--no-sandbox'] });
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' });
  await page.type('input[type="text"]', 'frankng');
  await page.type('input[type="password"]', 'Admin123');
  await page.click('button[type="submit"]').catch(() => {});
  await page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 1500));
  await page.goto(`${BASE}/admin/timesheet`, { waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 3000));

  // Hover and try clicking via puppeteer click (forces event sequence)
  const btn = await page.$('button[aria-label="Thêm tùy chọn"]');
  if (!btn) {
    console.log('NOT FOUND');
    await browser.close();
    return;
  }
  await btn.click();
  await new Promise(r => setTimeout(r, 1000));

  // Find ALL text nodes with "Chuyển OnePay"
  const found = await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('*'));
    return all
      .map(el => ({ tag: el.tagName, role: el.getAttribute('role') || '', text: (el.innerText || '').slice(0, 30) }))
      .filter(x => /chuyển/i.test(x.text));
  });
  console.log('Found "chuyển" elements:', JSON.stringify(found, null, 2));
  await page.screenshot({ path: '/tmp/qa-ellipsis-open-FRESH.png' });

  // Look for radix-ui portal menus
  const portalMenus = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('[role="menu"], [data-radix-popper-content-wrapper], [data-side]'))
      .map(el => ({ tag: el.tagName, role: el.getAttribute('role') || '', text: (el.innerText || '').slice(0, 200) }));
  });
  console.log('Portal menus:', JSON.stringify(portalMenus, null, 2));

  await browser.close();
})();
