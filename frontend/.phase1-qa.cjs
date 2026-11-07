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

const dismissEmailModal = async (page) => {
  try {
    await page.waitForSelector('[role="dialog"]', { timeout: 4000 });
    const btn = await page.evaluateHandle(() => {
      return Array.from(document.querySelectorAll('[role="dialog"] button'))
        .find((b) => b.textContent.includes('Nhắc tôi sau'));
    });
    const el = btn.asElement();
    if (el) { await el.click(); await new Promise((r) => setTimeout(r, 800)); console.log('  email modal dismissed'); }
  } catch (_) { /* no modal */ }
};

(async () => {
  const { status, body } = await postJson('/api/v1/auth/login', { username: 'frankng', password: 'Admin123' });
  if (status !== 200) { console.error('LOGIN-FAIL', status, body.slice(0, 300)); process.exit(1); }
  const token = JSON.parse(body)?.data?.access_token;
  if (!token) { console.error('NO-TOKEN', body.slice(0, 300)); process.exit(1); }

  const browser = await puppeteer.launch({ executablePath: CHROME, headless: 'new' });
  const page = await browser.newPage();
  const errors = [];
  page.on('pageerror', (e) => errors.push('PAGEERROR: ' + String(e).slice(0, 200)));
  await page.evaluateOnNewDocument((t) => {
    localStorage.setItem('auth_token', t);
    localStorage.setItem('userRole', 'admin');
  }, token);

  const shots = [
    { route: '/admin', w: 1280, h: 900, name: 'dash-1280' },
    { route: '/admin', w: 768, h: 1024, name: 'dash-768' },
    { route: '/admin/employees', w: 1280, h: 900, name: 'emp-1280' },
    { route: '/admin/employees', w: 390, h: 844, name: 'emp-390', mobile: true },
    { route: '/admin/employees', w: 320, h: 700, name: 'emp-320', mobile: true },
    { route: '/admin/ledger', w: 1280, h: 900, name: 'ledger-1280' },
  ];
  for (const s of shots) {
    await page.setViewport({ width: s.w, height: s.h, isMobile: !!s.mobile, hasTouch: !!s.mobile });
    await page.goto(BASE + s.route, { waitUntil: 'networkidle2', timeout: 60000 });
    await new Promise((r) => setTimeout(r, 3000));
    await dismissEmailModal(page);
    const state = await page.evaluate(() => ({
      url: location.pathname,
      sw: document.documentElement.scrollWidth,
      cw: document.documentElement.clientWidth,
    }));
    const ok = state.sw === state.cw;
    console.log(`${s.name}: overflow ${ok ? 'OK' : 'FAIL ' + state.sw + '>' + state.cw} route=${state.url}`);
    await page.screenshot({ path: '/tmp/console-shots/' + s.name + '.png' });
  }
  console.log('pageerrors:', JSON.stringify(errors.slice(0, 8)));
  await browser.close();
})().catch((e) => { console.error('SCRIPT-FAIL:', e.message); process.exit(1); });
