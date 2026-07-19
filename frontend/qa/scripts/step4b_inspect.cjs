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
  await page.goto(`${BASE}/admin/wallet`, { waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 3000));

  // Look for anything with "Lô #" or batch-related
  const batches = await page.evaluate(() => {
    const result = [];
    document.querySelectorAll('*').forEach(el => {
      const t = (el.innerText || '').trim();
      if (t.startsWith('Lô #') && t.length < 30) {
        result.push({ text: t, tag: el.tagName, clickable: !!el.closest('button, a, [role="button"]') });
      }
    });
    return result;
  });
  console.log('Lô # references:', JSON.stringify(batches, null, 2));

  // Find all clickable items in the bulk transfer section
  const clickables = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('button, a, [role="button"]')).map(b => ({
      text: (b.innerText || '').trim().slice(0, 40),
      ariaLabel: b.getAttribute('aria-label') || '',
      href: b.getAttribute('href') || '',
    })).filter(b => b.text || b.ariaLabel || b.href);
  });
  console.log('--- Clickables:', clickables.length, '---');
  console.log(JSON.stringify(clickables.slice(0, 30), null, 2));

  await browser.close();
})();
