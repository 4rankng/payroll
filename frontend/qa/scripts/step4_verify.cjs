const puppeteer = require('puppeteer-core');
const fs = require('fs');
const path = require('path');

const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const BATCH_ID = parseInt(process.argv[2] || '5', 10);
const DL_DIR = '/tmp/qa-downloads';

(async () => {
  fs.mkdirSync(DL_DIR, { recursive: true });
  for (const f of fs.readdirSync(DL_DIR)) fs.unlinkSync(path.join(DL_DIR, f));

  const browser = await puppeteer.launch({
    executablePath: CHROME, headless: 'new', args: ['--no-sandbox'],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });
  const client = await page.target().createCDPSession();
  await client.send('Page.setDownloadBehavior', { behavior: 'allow', downloadPath: DL_DIR });

  const errors = [];
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
  page.on('pageerror', (e) => errors.push('PAGE: ' + e.message));

  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' });
  await page.type('input[type="text"]', 'frankng');
  await page.type('input[type="password"]', 'Admin123');
  await page.click('button[type="submit"]').catch(() => {});
  await page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 1500));

  console.log('--- Phase 4: wallet page → open batch', BATCH_ID, '---');
  await page.goto(`${BASE}/admin/wallet`, { waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 3000));

  // Scroll down to the batch list
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await new Promise(r => setTimeout(r, 1500));
  await page.screenshot({ path: '/tmp/qa-11-wallet-batchlist.png', fullPage: true });

  // Click the BatchCard (Card with role="button"). The title shows filename,
  // not batch ID — we pick the first one (most recent, sorted desc).
  const clicked = await page.evaluate(() => {
    const cards = Array.from(document.querySelectorAll('[role="button"]'));
    // Match BatchCard: a Card with role="button" containing a status badge
    const target = cards.find(el => {
      const t = el.innerText || '';
      return /Đang xử lý|Hoàn tất|Thất bại|Chờ xử lý/i.test(t) &&
             /(xlsx|Lô #)/i.test(t) &&
             t.length < 500;
    });
    if (target) {
      target.click();
      return (target.innerText || '').replace(/\s+/g, ' ').slice(0, 80);
    }
    return null;
  });
  console.log('Batch card click target (first card):', clicked);
  await new Promise(r => setTimeout(r, 2500));
  await page.screenshot({ path: '/tmp/qa-12-progress-sheet.png' });

  // The sheet should now show BulkTransferProgress.
  // Wait for KQ auto-download via the new useEffect (status === 'completed')
  console.log('--- Waiting up to 20s for KQ auto-download ---');
  let kqFiles = null;
  for (let i = 0; i < 20; i++) {
    await new Promise(r => setTimeout(r, 1000));
    const files = fs.readdirSync(DL_DIR).filter(f => /\.xlsx$/i.test(f) && !/Yeu_cau|export/i.test(f));
    if (files.length > 0) { kqFiles = files; break; }
  }
  console.log('--- KQ downloads:', kqFiles, '---');
  if (kqFiles) {
    for (const f of kqFiles) {
      const stat = fs.statSync(path.join(DL_DIR, f));
      console.log('  ', f, stat.size, 'bytes');
    }
  }

  const pageText = await page.evaluate(() => document.body.innerText);
  console.log('--- Sheet state: ---');
  // Print last 300 chars (where the sheet content lives)
  console.log(pageText.slice(-400).replace(/\n+/g, ' | '));

  console.log('--- Errors:', errors.length, '---');
  errors.slice(0, 5).forEach(e => console.log('  ', e.slice(0, 300)));

  await browser.close();
})();
