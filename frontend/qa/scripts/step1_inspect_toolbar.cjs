const puppeteer = require('puppeteer-core');

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

  // Login
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' });
  await page.type('input[type="text"]', 'frankng');
  await page.type('input[type="password"]', 'Admin123');
  await page.click('button[type="submit"]').catch(() => {});
  await page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 1500));

  await page.goto(`${BASE}/admin/timesheet`, { waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 3000));

  // Inspect toolbar - find all buttons in the page header area
  const toolbar = await page.evaluate(() => {
    // Look at the top ~200px of the page (header + KPI cards + toolbar)
    const allBtns = Array.from(document.querySelectorAll('button'));
    return allBtns.map(b => {
      const r = b.getBoundingClientRect();
      return {
        text: (b.innerText || '').trim().slice(0, 50),
        ariaLabel: b.getAttribute('aria-label') || '',
        title: b.getAttribute('title') || '',
        svgClasses: Array.from(b.querySelectorAll('svg')).map(s => s.className?.baseVal || s.getAttribute('class') || ''),
        rect: { top: Math.round(r.top), left: Math.round(r.left), w: Math.round(r.width) },
      };
    }).filter(b => b.rect.top < 350); // top region only
  });
  console.log('--- Top-region buttons ---');
  console.log(JSON.stringify(toolbar, null, 2));

  await browser.close();
})();
