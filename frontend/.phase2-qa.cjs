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
    await page.waitForSelector('[role="dialog"]', { timeout: 3500 });
    const btn = await page.evaluateHandle(() =>
      Array.from(document.querySelectorAll('[role="dialog"] button')).find((b) => b.textContent.includes('Nhắc tôi sau')));
    const el = btn.asElement();
    if (el) { await el.click(); await new Promise((r) => setTimeout(r, 600)); }
  } catch (_) { /* none */ }
};

(async () => {
  const browser = await puppeteer.launch({ executablePath: CHROME, headless: 'new' });

  // ── Admin ──
  let page = await browser.newPage();
  const errors = [];
  page.on('pageerror', (e) => errors.push('ADMIN ' + String(e).slice(0, 160)));
  const { status, body } = await postJson('/api/v1/auth/login', { username: 'frankng', password: 'Admin123' });
  const adminToken = JSON.parse(body)?.data?.access_token;
  if (status !== 200 || !adminToken) { console.error('ADMIN LOGIN FAIL', status); process.exit(1); }
  await page.evaluateOnNewDocument((t) => {
    localStorage.setItem('auth_token', t); localStorage.setItem('userRole', 'admin');
  }, adminToken);

  for (const w of [1440, 1280, 1024, 768, 390, 320]) {
    const mobile = w < 768;
    await page.setViewport({ width: w, height: w < 1024 ? 900 : 900, isMobile: mobile, hasTouch: mobile });
    await page.goto(BASE + '/admin', { waitUntil: 'networkidle2', timeout: 60000 });
    await new Promise((r) => setTimeout(r, 3500));
    await dismissEmailModal(page);
    const state = await page.evaluate(() => ({
      sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth,
      url: location.pathname,
      kpiFigures: Array.from(document.querySelectorAll('[data-slot="admin-dashboard-bento"], section'))
        .length && Array.from(document.querySelectorAll('.console-value')).slice(0, 4).map((n) => n.textContent.trim()),
    }));
    const ok = state.sw === state.cw;
    console.log(`admin-${w}: overflow ${ok ? 'OK' : 'FAIL ' + state.sw + '>' + state.cw} kpi=[${state.kpiFigures.join(' | ')}]`);
    await page.screenshot({ path: `/tmp/console-shots/p2-admin-${w}.png` });
  }

  // ── Partner ──
  const p2 = await postJson('/api/v1/auth/login', { username: 'cuongnv', password: 'Admin123' });
  const partnerToken = JSON.parse(p2.body)?.data?.access_token;
  if (p2.status !== 200 || !partnerToken) {
    console.log('partner-login-failed ' + p2.status + ' — partner shots skipped');
  } else {
    const ppage = await browser.newPage();
    ppage.on('pageerror', (e) => errors.push('PARTNER ' + String(e).slice(0, 160)));
    await ppage.evaluateOnNewDocument((t) => {
      localStorage.setItem('auth_token', t); localStorage.setItem('userRole', 'partner');
    }, partnerToken);
    for (const w of [1440, 390]) {
      const mobile = w < 768;
      await ppage.setViewport({ width: w, height: 900, isMobile: mobile, hasTouch: mobile });
      await ppage.goto(BASE + '/partner/dashboard', { waitUntil: 'networkidle2', timeout: 60000 });
      await new Promise((r) => setTimeout(r, 3500));
      const st = await ppage.evaluate(() => ({
        sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth, url: location.pathname }));
      console.log(`partner-${w}: overflow ${st.sw === st.cw ? 'OK' : 'FAIL ' + st.sw + '>' + st.cw} route=${st.url}`);
      await ppage.screenshot({ path: `/tmp/console-shots/p2-partner-${w}.png` });
    }
  }
  console.log('pageerrors:', JSON.stringify(errors.slice(0, 8)));
  await browser.close();
})().catch((e) => { console.error('SCRIPT-FAIL:', e.message); process.exit(1); });
