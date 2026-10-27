// Rung-3 QA driver — drives the payment-history ledger in a real browser.
// Artifacts: screenshots + DOM assertions + console log + this driver log.
// Screenshots land in payment-history-ledger/ (gitignored: they capture the
// real dataset incl. employee PII).
const { chromium } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const OUT = path.join(__dirname, 'payment-history-ledger');
fs.mkdirSync(OUT, { recursive: true });

const logLines = [];
function log(msg) {
  const line = `[${new Date().toISOString()}] ${msg}`;
  logLines.push(line);
  console.log(line);
}

async function shot(page, name) {
  const p = path.join(OUT, `${name}.png`);
  await page.screenshot({ path: p, fullPage: true });
  log(`screenshot: ${p}`);
}

async function login(page, username, password) {
  await page.goto('http://localhost:3000/login', { waitUntil: 'networkidle' });
  await page.fill('#emailOrUsername', username);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(admin|partner)/, { timeout: 20000 });
  log(`login ok: ${username} -> ${page.url()}`);
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  const results = [];
  const assert = (name, cond, detail) => {
    results.push({ name, pass: !!cond, detail });
    log(`${cond ? 'PASS' : 'FAIL'} — ${name}${detail ? ` (${detail})` : ''}`);
  };

  try {
    for (const role of [
      { user: 'frankng', route: '/admin/payment-history', label: 'admin' },
      { user: 'cuongnv', route: '/partner/timesheet/payment-history', label: 'partner' },
    ]) {
      for (const vp of [
        { width: 1280, height: 900, label: '1280' },
        { width: 390, height: 844, label: '390' },
        { width: 320, height: 700, label: '320' },
      ]) {
        const context = await browser.newContext({ viewport: { width: vp.width, height: vp.height } });
        const page = await context.newPage();
        const errors = [];
        page.on('console', (m) => m.type() === 'error' && errors.push(m.text()));
        page.on('pageerror', (e) => errors.push(String(e)));

        await login(page, role.user, 'Admin123');
        await page.goto(`http://localhost:3000${role.route}`, { waitUntil: 'networkidle' });
        await page.waitForTimeout(800);
        const tag = `${role.label}-${vp.label}`;
        log(`--- ${tag} ${page.url()}`);

        // The dev DB has no data for the current month (2026-10) — switch to
        // 2026-08 first so row assertions run against a populated ledger.
        await page.getByRole('button', { name: /Chọn tháng kỳ lương/ }).click();
        await page.waitForTimeout(400);
        await page.getByRole('button', { name: /Chọn tháng 08/ }).click();
        await page.waitForTimeout(1500);
        await shot(page, `${tag}-monthpicker-after-switch`);

        // Data assertions from the rendered DOM.
        const bodyText = await page.textContent('body');
        assert(`${tag}: summary band present`, (await page.locator('[data-slot="payment-history-summary"]').count()) > 0);
        assert(`${tag}: summary shows Tổng đã chuyển`, bodyText.includes('Tổng đã chuyển'));
        // Summary must carry the API aggregate (regression: handler once dropped it).
        const summaryText = await page.locator('[data-slot="payment-history-summary"]').innerText();
        assert(
          `${tag}: summary has non-zero total`,
          !/TỔNG ĐÃ CHUYỂN\s*0\s*₫/i.test(summaryText),
          summaryText.replace(/\n/g, ' '),
        );

        const tableCount = await page.locator('[data-slot="payment-history-records"] table').count();
        const mobileCount = await page.locator('[data-slot="payment-history-mobile-records"]').count();
        assert(`${tag}: desktop table rendered`, tableCount > 0 || vp.width < 1280, `tables=${tableCount}`);
        assert(`${tag}: mobile blocks rendered`, mobileCount === 1);
        assert(`${tag}: table hidden below 1280`, (await page.locator('[data-slot="payment-history-records"]').isVisible()) === (vp.width >= 1280));

        if (vp.width >= 1280) {
          const refs = await page.locator('[data-slot="payment-history-records"] table tbody td').count();
          assert(`${tag}: ledger rows present`, refs > 0, `cells=${refs}`);
          const headers = await page.locator('[data-slot="payment-history-records"] thead th').allTextContents();
          assert(`${tag}: 9 column headers`, headers.length === 9, headers.join('|'));
          const rowsWithRowspan = await page.locator('[data-slot="payment-history-records"] td[rowspan]').count();
          log(`${tag}: rowSpan identity cells = ${rowsWithRowspan}`);
        }

        // No horizontal overflow.
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
        assert(`${tag}: no horizontal overflow`, overflow <= 1, `overflow=${overflow}px`);

        // Touch targets for interactive controls at mobile widths.
        if (vp.width < 1280) {
          const btn = page.getByRole('button', { name: /Chọn tháng kỳ lương/ });
          const box = await btn.boundingBox();
          assert(`${tag}: month picker >=44px tall`, box && box.height >= 44, `h=${box && box.height}`);
        }

        await shot(page, tag);

        // Interaction: open the month picker and pick a different month (real click).
        await page.getByRole('button', { name: /Chọn tháng kỳ lương/ }).click();
        await page.waitForTimeout(400);
        await shot(page, `${tag}-monthpicker`);
        const sep = page.getByRole('button', { name: /Chọn tháng 09/ });
        if (await sep.count()) {
          await sep.click();
          await page.waitForTimeout(1200);
          const after = await page.textContent('body');
          assert(`${tag}: month switch updates label`, after.includes('09/2026'));
          await shot(page, `${tag}-after-month-switch`);
          // Switch back so the run is idempotent.
          await page.getByRole('button', { name: /Chọn tháng kỳ lương/ }).click();
          await page.getByRole('button', { name: /Chọn tháng 08/ }).click();
          await page.waitForTimeout(800);
        }

        assert(`${tag}: no console errors`, errors.length === 0, errors.slice(0, 3).join(' ;; '));
        await context.close();
      }
    }
  } finally {
    fs.writeFileSync(path.join(OUT, 'driver.log'), logLines.join('\n'));
    const fails = results.filter((r) => !r.pass);
    fs.writeFileSync(
      path.join(OUT, 'assertions.json'),
      JSON.stringify({ total: results.length, failed: fails.length, results }, null, 2),
    );
    console.log(`\n== ${results.length - fails.length}/${results.length} assertions passed ==`);
    await browser.close();
    process.exit(fails.length ? 1 : 0);
  }
})();
