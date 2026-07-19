const puppeteer = require('puppeteer-core');
const fs = require('fs');
const BASE = 'http://localhost:3000';
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const UPLOAD_FILE = process.argv[2];

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
  await new Promise(r => setTimeout(r, 2500));

  // Open dialog
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const t = btns.find(b => (b.innerText || '').trim() === 'Tải File');
    if (t) t.click();
  });
  await new Promise(r => setTimeout(r, 1200));

  // Find ALL input[type=file]
  const inputs = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('input[type="file"]')).map((el, i) => ({
      i,
      id: el.id,
      accept: el.accept,
      cls: el.className,
      visible: el.offsetParent !== null,
      value: el.value,
    }));
  });
  console.log('--- file inputs in DOM:', JSON.stringify(inputs, null, 2));

  const fileInput = await page.$('input[type="file"]');
  if (!fileInput) {
    console.log('NONE');
  } else {
    // Upload using puppeteer uploadFile
    await fileInput.uploadFile(UPLOAD_FILE);
    await new Promise(r => setTimeout(r, 1500));

    // Check what value got set
    const postState = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input[type="file"]')).map(el => ({
        id: el.id,
        files: el.files?.length,
        value: el.value,
        size: el.files?.[0]?.size,
      }));
    });
    console.log('--- after uploadFile:', JSON.stringify(postState, null, 2));

    // Inspect what the dialog shows
    const dlgState = await page.evaluate(() => {
      const dlg = document.querySelector('[role="dialog"]');
      return dlg ? dlg.innerText.slice(0, 800) : null;
    });
    console.log('--- dialog state: ---');
    console.log(dlgState);
  }

  await browser.close();
})();
