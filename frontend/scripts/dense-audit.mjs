// Data-density + layout audit harness.
// Walks every admin/partner route at 4 device widths and measures:
//   - horizontal overflow
//   - interactive targets under the touch floor: 40px on touch widths
//     (<1200px), 32px on desktop (>=1200px, mouse-dense by design)
//   - oversized list rows (>88px) — genuinely sparse rows; 2–3-line identity
//     rows (~76-92px) are data-rich by design
// Known-accepted flags: the compact DateRangePicker inner inputs measure 28px
// but sit inside a 32px clickable container (the container is the target);
// /admin/advance-payments identity rows carry name+CCCD+project by design.
// Usage: node scripts/dense-audit.mjs [--base http://localhost:5174]
// Requires: backend on :8080, payroll-redis container for the login captcha.
import { chromium } from "@playwright/test";
import { execSync } from "node:child_process";

const BASE = process.argv.includes("--base")
  ? process.argv[process.argv.indexOf("--base") + 1]
  : "http://localhost:5174";

const ROUTES = [
  "/admin",
  "/admin/users",
  "/admin/projects",
  "/admin/employees",
  "/admin/timesheet",
  "/admin/payment-history",
  "/PROBE_SKIP", // placeholder to keep numbering readable
  "/admin/ledger",
  "/admin/loans",
  "/admin/advance-payments",
  "/admin/advance-payments/check-in-settings",
  "/admin/advance-payments/employees",
  "/admin/approvals",
  "/admin/settings",
  "/admin/wallet",
  "/admin/audit-log",
  "/admin/send-notification",
  "/admin/email",
  "/partner",
  "/partner/projects",
  "/partner/employees",
  "/partner/timesheet",
  "/partner/timesheet/payment-history",
].filter((r) => r !== "/PROBE_SKIP");

const WIDTHS = [390, 834, 1194, 1366];

const PROBE = `(() => {
  const vis = (el) => {
    const r = el.getBoundingClientRect();
    const s = getComputedStyle(el);
    return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none';
  };
  const targets = [...document.querySelectorAll('button, a, input, select, [role="button"], [role="tab"]')].filter(vis);
  const isTouch = window.innerWidth < 1200; // touch widths enforce the 40px floor
  const floor = isTouch ? 40 : 32; // desktop-dense controls are 32-36px by design
  const tiny = targets
    .filter((el) => {
      if ((el.getAttribute('role') || '') === 'switch') return false;
      const t = el.getBoundingClientRect();
      return t.height < floor || t.width < 24;
    })
    .slice(0, 5)
    .map((el) => {
      const b = el.getBoundingClientRect();
      return Math.round(b.height) + 'x' + Math.round(b.width) + ' ' + (el.textContent || el.placeholder || el.ariaLabel || '').trim().slice(0, 22);
    });
  const rows = [...document.querySelectorAll('tbody tr, [class*="list"] > div')].filter(vis);
  // Three-line identity rows (name + CCCD + project, ~84px) are data-rich by
  // design; flag only genuinely oversized rows.
  const bigRows = rows.filter((r) => r.getBoundingClientRect().height > 88).length;
  const overflow = document.documentElement.scrollWidth - document.documentElement.clientWidth;
  return {
    overflowX: overflow,
    tinyCount: tiny.length,
    tiny,
    rowCount: rows.length,
    bigRows,
  };
})()`;

async function login(page) {
  await page.goto(BASE + "/login");
  // The captcha field only renders when the server requires it (per-username
  // failure counter, TTL'd). Fill credentials, submit, then handle captcha
  // if the form grew a third input.
  const solveCaptcha = async () => {
    const cap = await (await fetch(BASE.replace("5174", "8080") + "/api/v1/auth/captcha")).json();
    const id = cap.data.captcha_id;
    return execSync(`docker exec payroll-redis redis-cli get captcha:${id}`).toString().trim();
  };
  await page.locator("input").nth(0).fill("frankng");
  await page.locator("input[type=password]").fill("TempAudit2026");
  await page.getByRole("button", { name: "Đăng nhập" }).click();
  try {
    await page.waitForURL("**/admin", { timeout: 4000 });
    return;
  } catch {
    /* captcha likely required */
  }
  await page.locator("input").nth(2).fill(await solveCaptcha(), { timeout: 8000 });
  await page.getByRole("button", { name: "Đăng nhập" }).click();
  await page.waitForURL("**/admin", { timeout: 15000 });
}

async function probe(page) {
  await page.waitForTimeout(1600);
  return page.evaluate(PROBE);
}

const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: 1366, height: 900 } });
const page = await ctx.newPage();
await login(page);

const results = [];
for (const width of WIDTHS) {
  await page.setViewportSize({ width, height: 900 });
  for (const route of ROUTES) {
    await page.goto(BASE + route, { waitUntil: "domcontentloaded" });
    const r = await probe(page);
    results.push({ width, route, ...r });
    const bad =
      r.overflowX > 0 || r.tinyCount > 0 || r.bigRows > 0 ? " !! " : " ok ";
    console.log(`${width}\t${route}\t${bad} overflow=${r.overflowX} tiny=${r.tinyCount} bigRows=${r.bigRows}`);
  }
}

await browser.close();
const json = JSON.stringify(results, null, 1);
await import("node:fs").then((fs) => fs.writeFileSync("/tmp/kanban-docx/dense-audit.json", json));
const bad = results.filter((r) => r.overflowX > 0 || r.tinyCount > 0 || r.bigRows > 0);
console.log(`\nTOTAL ${results.length} probes, ${bad.length} flagged`);
