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
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await new Promise(r => setTimeout(r, 1500));

  // Inspect BatchCard structure
  const cards = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('[role="button"], .cursor-pointer, button'))
      .filter(el => /Lô #\d+/.test(el.innerText || ''))
      .map(el => ({
        tag: el.tagName,
        text: (el.innerText || '').replace(/\s+/g, ' ').slice(0, 100),
        cls: (el.className?.baseVal || el.className || '').slice(0, 100),
      }));
  });
  console.log(JSON.stringify(cards, null, 2));

  await browser.close();
})();
