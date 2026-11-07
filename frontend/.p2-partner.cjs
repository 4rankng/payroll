const puppeteer = require('puppeteer-core');
const http = require('http');
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const BASE = 'http://localhost:3000';
const postJson = (path, payload) => new Promise((resolve, reject) => {
  const data = JSON.stringify(payload);
  const req = http.request({ host: 'localhost', port: 8081, path, method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(data) } },
    (res) => { let buf = ''; res.on('data', (c) => (buf += c));
      res.on('end', () => resolve({ status: res.statusCode, body: buf })); });
  req.on('error', reject);
  req.write(data);
  req.end();
});
(async () => {
  const { body } = await postJson('/api/v1/auth/login', { username: 'cuongnv', password: 'Admin123' });
  const token = JSON.parse(body)?.data?.access_token;
  const browser = await puppeteer.launch({ executablePath: CHROME, headless: 'new' });
  const page = await browser.newPage();
  await page.evaluateOnNewDocument((t) => {
    localStorage.setItem('auth_token', t); localStorage.setItem('userRole', 'partner');
  }, token);
  for (const [w, name] of [[1440, 'p2-partner-1440'], [390, 'p2-partner-390']]) {
    const mobile = w < 768;
    await page.setViewport({ width: w, height: 900, isMobile: mobile, hasTouch: mobile });
    await page.goto(BASE + '/partner/dashboard', { waitUntil: 'networkidle2', timeout: 60000 });
    await new Promise((r) => setTimeout(r, 3000));
    try {
      await page.waitForSelector('[role="dialog"]', { timeout: 3500 });
      const btn = await page.evaluateHandle(() =>
        Array.from(document.querySelectorAll('[role="dialog"] button')).find((b) => b.textContent.includes('Nhắc tôi sau')));
      const el = btn.asElement();
      if (el) { await el.click(); await new Promise((r) => setTimeout(r, 800)); }
    } catch (_) {}
    await page.screenshot({ path: `/tmp/console-shots/${name}.png` });
    console.log(name + ' saved');
  }
  await browser.close();
})().catch((e) => { console.error('FAIL:', e.message); process.exit(1); });
