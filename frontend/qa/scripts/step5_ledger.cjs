const puppeteer = require('puppeteer-core');
const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

(async () => {
  const browser = await puppeteer.launch({ executablePath: CHROME, headless: 'new', args: ['--no-sandbox'] });
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });
  const errors = [];
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });

  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' });
  await page.type('input[type="text"]', 'frankng');
  await page.type('input[type="password"]', 'Admin123');
  await page.click('button[type="submit"]').catch(() => {});
  await page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 1500));

  console.log('--- Phase 5: Ledger page ---');
  await page.goto(`${BASE}/admin/ledger`, { waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 3000));
  await page.screenshot({ path: '/tmp/qa-13-ledger.png' });

  // Find the OnePay fee entry
  const feeRow = await page.evaluate(() => {
    const rows = Array.from(document.querySelectorAll('tr, [role="row"], div'));
    const t = rows.find(el => /onepay/i.test(el.innerText || '') && /ph|chi.*ph|fee/i.test(el.innerText || ''));
    if (!t) return null;
    return (t.innerText || '').replace(/\s+/g, ' ').slice(0, 300);
  });
  console.log('OnePay fee entry in ledger:', feeRow);

  console.log('--- Errors:', errors.length, '---');
  errors.slice(0, 5).forEach(e => console.log('  ', e.slice(0, 200)));

  await browser.close();
})();
