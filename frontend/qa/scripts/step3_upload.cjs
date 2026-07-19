const puppeteer = require('puppeteer-core');
const fs = require('fs');
const path = require('path');

const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const DL_DIR = '/tmp/qa-downloads';
const UPLOAD_FILE = process.argv[2];

if (!UPLOAD_FILE || !fs.existsSync(UPLOAD_FILE)) {
  console.error('Usage: node step3_upload.cjs <xlsx-to-upload>');
  process.exit(1);
}

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
  const apiResponses = [];
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
  page.on('pageerror', (e) => errors.push('PAGE: ' + e.message));
  page.on('response', async (r) => {
    const url = r.url();
    if (/bulk-transfer\/upload/.test(url) && r.request().method() === 'POST') {
      try {
        const t = await r.text();
        apiResponses.push({ status: r.status(), body: t.slice(0, 800) });
      } catch (e) {
        apiResponses.push({ status: r.status(), body: '<unreadable: ' + e.message + '>' });
      }
    }
  });

  // Login
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' });
  await page.type('input[type="text"]', 'frankng');
  await page.type('input[type="password"]', 'Admin123');
  await page.click('button[type="submit"]').catch(() => {});
  await page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 1500));

  console.log('--- Phase 3: Wallet page ---');
  await page.goto(`${BASE}/admin/wallet`, { waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 3000));
  await page.screenshot({ path: '/tmp/qa-07-wallet.png' });
  console.log('Wallet URL:', page.url());

  // Click "Tải File" header button (was "Tải lên chuyển tiền")
  console.log('--- Open Tải File dialog ---');
  const opened = await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const t = btns.find(b => (b.innerText || '').trim() === 'Tải File');
    if (t) { t.click(); return true; }
    return false;
  });
  console.log('Opened dialog:', opened);
  await new Promise(r => setTimeout(r, 1200));
  await page.screenshot({ path: '/tmp/qa-08-dialog.png' });

  // Drop file into the FileDropZone via input[type=file]
  console.log('--- Upload file:', UPLOAD_FILE, '---');
  const fileInput = await page.$('input[type="file"]');
  if (!fileInput) {
    console.log('!!! file input NOT FOUND');
  } else {
    await fileInput.uploadFile(UPLOAD_FILE);
    await new Promise(r => setTimeout(r, 1500));
    await page.screenshot({ path: '/tmp/qa-09-file-selected.png' });

    // Click "Tải File" submit button (inside dialog, has Upload icon)
    const submitted = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button'));
      // Submit button is the one with "Tải File" text inside the dialog footer
      const t = btns.find(b => {
        const txt = (b.innerText || '').trim();
        if (txt !== 'Tải File') return false;
        // Submit button is inside DialogFooter
        return !!b.closest('[role="dialog"], [data-slot="dialog-footer"]') ||
               b.closest('.DialogContent') != null;
      });
      if (t) { t.click(); return true; }
      return false;
    });
    console.log('Submit clicked:', submitted);
    await new Promise(r => setTimeout(r, 6000));
    await page.screenshot({ path: '/tmp/qa-10-after-upload.png' });
  }

  console.log('--- Upload API responses:', apiResponses.length, '---');
  apiResponses.forEach(r => console.log('  ', r.status, r.body));

  // Check the result toast / dialog state
  const dialogText = await page.evaluate(() => {
    const dlg = document.querySelector('[role="dialog"], .DialogContent');
    return dlg ? dlg.innerText.slice(0, 500) : null;
  });
  console.log('--- Dialog text after upload: ---');
  console.log(dialogText);

  console.log('--- Console/page errors:', errors.length, '---');
  errors.slice(0, 10).forEach(e => console.log('  ERR:', e.slice(0, 300)));

  await browser.close();
})();
