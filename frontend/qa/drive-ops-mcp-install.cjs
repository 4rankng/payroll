// Rung-3 QA driver — drives the payroll-ops-mcp install section in the admin
// Settings → API tab in a real browser: renders (desktop 1280 + mobile 390),
// creates a pk_ops_* key through the UI, toggles scope, revokes, and uses
// the copy buttons. DB side effects verified via the ops_api_keys table.
// Artifacts: screenshots + this log in frontend/qa/ops-mcp-install/
// (gitignored). Driver: node frontend/qa/drive-ops-mcp-install.cjs
const { chromium } = require('@playwright/test');
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const OUT = path.join(__dirname, 'ops-mcp-install');
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

function dbScalar(sql) {
  return execSync(
    `docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -N -B -e "${sql.replace(/"/g, '\\"')}" 2>/dev/null`,
  )
    .toString()
    .trim();
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  const results = [];
  const assert = (name, cond, detail) => {
    results.push({ name, pass: !!cond, detail });
    log(`${cond ? 'PASS' : 'FAIL'} — ${name}${detail ? ` (${detail})` : ''}`);
  };

  try {
    const context = await browser.newContext({
      viewport: { width: 1280, height: 900 },
      permissions: ['clipboard-read', 'clipboard-write'],
    });
    const page = await context.newPage();

    // ── login ────────────────────────────────────────────────────────────
    await page.goto('http://localhost:3000/login', { waitUntil: 'networkidle' });
    await page.fill('#emailOrUsername', 'frankng');
    await page.fill('#password', 'Admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/admin/, { timeout: 20000 });
    log(`login ok -> ${page.url()}`);

    // ── open Settings → API tab ──────────────────────────────────────────
    await page.goto('http://localhost:3000/admin/settings?tab=api', { waitUntil: 'networkidle' });
    await page.waitForSelector('#ops-mcp-install', { timeout: 15000 });
    log('settings API tab loaded, ops section present');

    // ── 1. the section renders with all three cards + five steps ─────────
    assert('create card renders', await page.locator('#ops-keys-create-title').isVisible());
    assert('list card renders', await page.locator('#ops-keys-list-title').isVisible());
    assert('install guide renders', await page.locator('#ops-mcp-install-title').isVisible());
    for (const step of [
      'ở khung phía trên',
      'Lưu khoá vào file bí mật (chế độ 600)',
      'Cấu hình MCP client',
      'Phạm vi khoá: chỉ đọc theo mặc định',
      'Kill switch (tắt toàn bộ kênh khi cần)',
    ]) {
      assert(`step "${step.slice(0, 28)}…"`, await page.getByText(step, { exact: false }).first().isVisible());
    }
    assert(
      'config snippet references PAYROLL_OPS_KEY_FILE (never the key)',
      (await page.locator('#ops-mcp-install pre').allTextContents()).join('\n').includes('PAYROLL_OPS_KEY_FILE'),
    );
    await shot(page, '01-install-guide-1280');

    // ── 2. create a read_only key through the UI ─────────────────────────
    const agentName = `ui-qa-${Date.now()}`;
    await page.fill('#ops-key-name', agentName);
    await page.check('#ops-key-scope-read_only');
    await page.getByRole('button', { name: 'Tạo khoá ops' }).click();
    await page.waitForSelector('input[aria-label^="Khoá ops cho"]', { timeout: 15000 });
    const plaintext = await page.locator('input[aria-label^="Khoá ops cho"]').inputValue();
    assert('plaintext revealed once, pk_ops_* namespace', plaintext.startsWith('pk_ops_'));
    await shot(page, '02-key-revealed-1280');
    await page.getByRole('button', { name: 'Đã lưu khoá (Đóng)' }).click();

    // DB side effect: the row exists with read_only + no window.
    await page.waitForTimeout(700); // list refetch
    const row = dbScalar(
      `SELECT CONCAT(id,'|',scope,'|',IFNULL(write_window_expires_at,'-'),'|',IFNULL(revoked_at,'-')) FROM ops_api_keys WHERE name='${agentName}'`,
    );
    log(`db row: ${row}`);
    assert('db row created read_only, no window, live', /^\d+\|read_only\|-\|-$/.test(row), row);

    // ── 3. scope toggle through the UI (dev: OTP off → otp_verified=true) ─
    page.once('dialog', (d) => d.accept());
    const rowEl = page.locator('li', { hasText: agentName }).first();
    await rowEl.getByRole('button', { name: 'Cấp ghi (8h)' }).click();
    await page.waitForFunction(
      () => document.body.innerText.includes('Đã chuyển khoá sang read_write'),
      { timeout: 15000 },
    );
    await page.waitForTimeout(700);
    const rowAfterToggle = dbScalar(
      `SELECT CONCAT(scope,'|',write_window_expires_at IS NOT NULL) FROM ops_api_keys WHERE name='${agentName}'`,
    );
    log(`db row after toggle: ${rowAfterToggle}`);
    assert('db row flipped to read_write with an 8h window', rowAfterToggle === 'read_write|1', rowAfterToggle);
    await shot(page, '03-after-scope-toggle-1280');

    // ── 4. copy button (secrets-file snippet) ────────────────────────────
    await page.locator('#ops-mcp-install').getByRole('button', { name: 'Sao chép' }).first().click();
    await page.waitForFunction(() => document.body.innerText.includes('Đã sao chép'), { timeout: 10000 });
    const clip = await page.evaluate(() => navigator.clipboard.readText());
    assert(
      'clipboard received the secrets-file commands (chmod 600)',
      clip.includes('chmod 600 ~/.secrets/payroll-ops-agent.key'),
      clip.split('\n')[0],
    );

    // ── 5. revoke through the UI ─────────────────────────────────────────
    page.once('dialog', (d) => d.accept());
    await rowEl.getByRole('button', { name: 'Thu hồi' }).click();
    await page.waitForFunction(
      () => document.body.innerText.includes('Đã thu hồi khoá ops'),
      { timeout: 15000 },
    );
    await page.waitForTimeout(700);
    const revoked = dbScalar(`SELECT revoked_at IS NOT NULL FROM ops_api_keys WHERE name='${agentName}'`);
    assert('db row revoked', revoked === '1', revoked);
    await shot(page, '04-after-revoke-1280');

    // ── 6. mobile 390px: section usable, no horizontal overflow ──────────
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('http://localhost:3000/admin/settings?tab=api', { waitUntil: 'networkidle' });
    await page.waitForSelector('#ops-mcp-install', { timeout: 15000 });
    assert('install guide visible at 390px', await page.locator('#ops-mcp-install-title').isVisible());
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    assert('no horizontal overflow at 390px', overflow <= 0, `delta=${overflow}px`);
    // Touch targets: the snippet copy buttons are min-h-11 (44px) on mobile.
    const btnBox = await page
      .locator('#ops-mcp-install')
      .getByRole('button', { name: 'Sao chép' })
      .first()
      .boundingBox();
    assert('copy button >= 44px tall at 390px', btnBox && btnBox.height >= 44, JSON.stringify(btnBox));
    await shot(page, '05-install-guide-390');

    await context.close();
  } catch (err) {
    log(`DRIVER ERROR: ${err.message}`);
    results.push({ name: 'driver completed', pass: false, detail: err.message });
  } finally {
    fs.writeFileSync(path.join(OUT, 'drive.log'), logLines.join('\n') + '\n');
    await browser.close();
  }

  const failed = results.filter((r) => !r.pass);
  console.log(`\n${failed.length === 0 ? '✅ ALL PASS' : '❌ FAILURES'} — ${results.length - failed.length}/${results.length}`);
  process.exit(failed.length === 0 ? 0 : 1);
})();
