const puppeteer = require('puppeteer-core');
const fs = require('fs');
const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const UPLOAD_FILE = process.argv[2];

(async () => {
  const browser = await puppeteer.launch({ executablePath: CHROME, headless: 'new', args: ['--no-sandbox'] });
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  const errors = [];
  const uploads = [];
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
  page.on('response', async (r) => {
    if (/bulk-transfer\/upload/.test(r.url()) && r.request().method() === 'POST') {
      try { uploads.push({ status: r.status(), body: (await r.text()).slice(0, 500) }); }
      catch (e) { uploads.push({ status: r.status(), error: e.message }); }
    }
  });

  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' });
  await page.type('input[type="text"]', 'frankng');
  await page.type('input[type="password"]', 'Admin123');
  await page.click('button[type="submit"]').catch(() => {});
  await page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {});
  await new Promise(r => setTimeout(r, 1500));
  await page.goto(`${BASE}/admin/wallet`, { waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 2500));

  // Open dialog
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const t = btns.find(b => (b.innerText || '').trim() === 'Tải File');
    if (t) t.click();
  });
  await new Promise(r => setTimeout(r, 1200));

  // Click the dropzone (visible label) which triggers the file input click
  const dropClicked = await page.evaluate(() => {
    // The label/div wrapping the input — try to click the visible dropzone
    const dz = document.querySelector('[data-slot="dropzone"], [class*="border-dashed"]') ||
               document.querySelector('label[for="bulk-transfer-upload-input"]');
    if (dz) { dz.click(); return true; }
    // Fallback: click on the input via parent
    const inp = document.getElementById('bulk-transfer-upload-input');
    if (inp) { inp.parentElement?.click(); return 'parent'; }
    return false;
  });
  console.log('Dropzone clicked:', dropClicked);

  // Set up file chooser acceptance BEFORE clicking
  const [fileChooser] = await Promise.all([
    page.waitForFileChooser({ timeout: 3000 }).catch(() => null),
    page.evaluate(() => {
      const inp = document.getElementById('bulk-transfer-upload-input');
      inp?.click();
    }),
  ]);
  if (fileChooser) {
    await fileChooser.accept([UPLOAD_FILE]);
    console.log('File accepted via chooser');
  } else {
    console.log('No file chooser dialog');
    // Fallback: uploadFile directly
    const inp = await page.$('input[type="file"]');
    await inp?.uploadFile(UPLOAD_FILE);
    console.log('Fallback: uploadFile used');
  }
  await new Promise(r => setTimeout(r, 2000));

  // Check size now
  const size = await page.evaluate(() => {
    const el = document.getElementById('bulk-transfer-upload-input');
    return { files: el?.files?.length ?? 0, size: el?.files?.[0]?.size ?? 0, name: el?.files?.[0]?.name };
  });
  console.log('After chooser:', JSON.stringify(size));

  // Click submit
  const submitted = await page.evaluate(() => {
    const dlg = document.querySelector('[role="dialog"]');
    if (!dlg) return 'no dialog';
    const btns = Array.from(dlg.querySelectorAll('button'));
    const submit = btns.find(b => (b.innerText || '').trim() === 'Tải File' && b.closest('footer, [data-slot="dialog-footer"]') === null);
    // Just take the last Tải File button (submit is at the bottom)
    const matches = btns.filter(b => (b.innerText || '').trim() === 'Tải File');
    if (matches.length >= 2) { matches[matches.length - 1].click(); return 'last-taifile'; }
    if (matches.length === 1) { matches[0].click(); return 'only-taifile'; }
    return 'no-submit';
  });
  console.log('Submit:', submitted);
  await new Promise(r => setTimeout(r, 5000));

  console.log('--- Upload responses:', uploads.length, '---');
  uploads.forEach(u => console.log(' ', JSON.stringify(u)));

  console.log('--- Console errors:', errors.length, '---');
  errors.slice(0, 5).forEach(e => console.log('  ', e.slice(0, 300)));

  await page.screenshot({ path: '/tmp/qa-10-after-upload.png' });
  const dlgText = await page.evaluate(() => document.querySelector('[role="dialog"]')?.innerText.slice(0, 800));
  console.log('--- Dialog after upload: ---', dlgText);

  await browser.close();
})();
