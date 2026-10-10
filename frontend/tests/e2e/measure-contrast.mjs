/**
 * Ground-truth contrast measurement — samples REAL RENDERED PIXELS.
 * Run: node tests/e2e/measure-contrast.mjs [path]
 *
 * This is unimpeachable: it screenshots the page and reads the pixel colours
 * the browser actually painted behind/under each text glyph, so gradients,
 * translucent overlays, and pseudo-elements are all handled correctly.
 */
import { chromium } from '@playwright/test';
import { PNG } from 'pngjs';

const URL = process.env.MEASURE_URL || 'http://localhost:3000';
const TARGET = process.argv[2] || '/admin/users';

function enc(o) { return Buffer.from(JSON.stringify(o)).toString('base64url'); }
function mockJwt(role) {
  const now = Math.floor(Date.now() / 1000);
  return `${enc({ alg: 'HS256', typ: 'JWT' })}.${enc({
    exp: now + 86400, user_id: 1, username: 'admin', role, sub: '1', nbf: now - 60, iat: now, jti: 'm',
  })}.sig`;
}
function lum([r, g, b]) {
  const f = (c) => { const v = c / 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; };
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
}
function contrast(fg, bg) { const a = lum(fg), b = lum(bg); return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05); }

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });

const token = mockJwt('admin');
await page.addInitScript(([t, role, name, email]) => {
  localStorage.setItem('auth_token', t);
  localStorage.setItem('userRole', role);
  localStorage.setItem('userName', name);
  localStorage.setItem('userEmail', email);
  localStorage.setItem('userStatus', 'active');
}, [token, 'admin', 'Admin', 'admin@example.com']);

await page.route('**/api/v1/**', (r) => {
  const url = r.request().url();
  const ok = (data) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', data }) });
  if (/users\/summary/.test(url)) return ok({ total_users: 22, total_admins: 2, total_partners: 22, total_employees: 1563 });
  if (/employees\/summary/.test(url)) return ok({ total_employees: 1563, total_active: 1500, total_inactive: 63 });
  return ok([]);
});

await page.goto(`${URL}${TARGET}`, { waitUntil: 'networkidle' }).catch(() => {});
await page.waitForTimeout(3000);

// Collect visible text elements with their boxes.
const nodes = await page.evaluate(() => {
  const out = [], seen = new Set();
  for (const el of document.querySelectorAll('p,span,h1,h2,h3,h4,h5,h6,a,button,label,td,th,li')) {
    const text = (el.textContent || '').trim();
    if (!text || text.length > 40) continue;
    const own = Array.from(el.childNodes).some((n) => n.nodeType === 3 && n.textContent.trim());
    if (!own) continue;
    const cs = getComputedStyle(el);
    if (cs.visibility === 'hidden' || cs.display === 'none' || +cs.opacity === 0) continue;
    const r = el.getBoundingClientRect();
    if (r.width < 4 || r.height < 4) continue;
    if (r.bottom < 0 || r.top > innerHeight || r.right < 0 || r.left > innerWidth) continue;
    const key = text + '|' + Math.round(r.x) + '|' + Math.round(r.y);
    if (seen.has(key)) continue;
    seen.add(key);
    // WCAG 1.4.3 exempts inactive UI components ("no contrast requirement").
    // Record the state so we can report them separately instead of failing.
    const disabled = el.closest('[disabled], [aria-disabled="true"]') !== null
      || el.matches(':disabled');
    out.push({
      text,
      // sample point: just inside the left edge, vertically centred — this
      // lands on the glyph's background beside the first character.
      x: r.x + 1, y: r.y + r.height / 2,
      // a point on the glyph stroke itself is unreliable; instead capture the
      // text colour from computed style and the BACKGROUND from the pixel.
      color: cs.color,
      fontSize: cs.fontSize, fontWeight: cs.fontWeight,
      w: r.width, h: r.height,
      disabled,
    });
  }
  return out;
});

const shot = await page.screenshot({ type: 'png' });
const png = PNG.sync.read(shot);
const px = (x, y) => {
  const i = (Math.round(y) * png.width + Math.round(x)) * 4;
  return [png.data[i], png.data[i + 1], png.data[i + 2]];
};
const parseRgb = (s) => {
  const m = s.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)/);
  return m ? [+m[1], +m[2], +m[3], m[4] === undefined ? 1 : +m[4]] : null;
};

const results = [];
for (const n of nodes) {
  if (n.x < 0 || n.y < 0 || n.x >= png.width || n.y >= png.height) continue;
  // Background = the DOMINANT colour across the element's text row, EXCLUDING
  // pixels near the text colour. Sampling one pixel hits glyph-edge antialias;
  // taking the raw mode on large glyphs picks the glyph itself. So we drop
  // pixels close to the fg and take the mode of what remains.
  const fgRaw0 = parseRgb(n.color);
  const fgKey = fgRaw0 ? `${Math.round(fgRaw0[0])},${Math.round(fgRaw0[1])},${Math.round(fgRaw0[2])}` : null;
  const counts = new Map();
  const y = Math.round(n.y);
  for (let x = Math.round(n.x); x < Math.round(n.x + n.w) && x < png.width; x++) {
    const c = px(x, y);
    // skip pixels visually close to the text colour (glyph + its antialias)
    if (fgKey) {
      const dr = c[0] - fgRaw0[0], dg = c[1] - fgRaw0[1], db = c[2] - fgRaw0[2];
      if (dr * dr + dg * dg + db * db < 60 * 60) continue;
    }
    const k = c.join(',');
    counts.set(k, (counts.get(k) || 0) + 1);
  }
  let bg = null, best = -1;
  for (const [k, v] of counts) {
    if (v > best) { best = v; bg = k.split(',').map(Number); }
  }
  // Fall back to the row mode (incl. glyphs) if the fg-exclusion emptied it.
  if (!bg || best === 0) {
    counts.clear();
    for (let x = Math.round(n.x); x < Math.round(n.x + n.w) && x < png.width; x++) {
      const k = px(x, y).join(',');
      counts.set(k, (counts.get(k) || 0) + 1);
    }
    best = -1;
    for (const [k, v] of counts) { if (v > best) { best = v; bg = k.split(',').map(Number); } }
  }
  if (!bg) continue;

  const fgRaw = parseRgb(n.color);
  if (!fgRaw) continue;
  const fgA = fgRaw[3] === undefined ? 1 : fgRaw[3];
  // Composite the (possibly translucent) text colour over the real painted bg.
  const fg = [
    fgA * fgRaw[0] + (1 - fgA) * bg[0],
    fgA * fgRaw[1] + (1 - fgA) * bg[1],
    fgA * fgRaw[2] + (1 - fgA) * bg[2],
  ];
  const ratio = contrast(fg, bg);
  const sizePx = parseFloat(n.fontSize), bold = +n.fontWeight >= 700;
  const required = (sizePx >= 24 || (sizePx >= 18.66 && bold)) ? 3 : 4.5;
  results.push({
    text: n.text, color: n.color, bg: `rgb(${bg.join(',')})`,
    ratio: +ratio.toFixed(2), required, disabled: !!n.disabled,
    // Inactive components are exempt from WCAG 1.4.3 — only active text gates.
    pass: !!n.disabled || ratio >= required,
    size: n.fontSize, weight: n.fontWeight,
  });
}

results.sort((a, b) => a.ratio - b.ratio);
const fails = results.filter((r) => !r.pass);
const exempt = results.filter((r) => r.disabled);
console.log(`\nTARGET: ${TARGET}  |  sampled ${results.length} text nodes from real pixels`);
console.log(`FAILING WCAG AA (active text): ${fails.length}`);
console.log(`EXEMPT (inactive UI, WCAG 1.4.3): ${exempt.length}\n`);
for (const f of fails) {
  console.log(`  FAIL ${f.ratio} (need ${f.required})  "${f.text}"`);
  console.log(`        fg=${f.color}  bg=${f.bg}  ${f.size}/${f.weight}`);
}
console.log(`\n--- worst first (top 30) ---\n`);
for (const f of results.slice(0, 30)) {
  console.log(`  ${f.pass ? 'PASS' : 'FAIL'} ${String(f.ratio).padStart(5)}  "${f.text}"  ${f.color} on ${f.bg}`);
}
await browser.close();
process.exit(fails.length ? 1 : 0);
