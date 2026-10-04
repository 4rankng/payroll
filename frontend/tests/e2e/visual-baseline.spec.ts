import { expect, test, type Page } from "@playwright/test";

/**
 * Pre-migration visual baselines for the Untitled UI PRO migration.
 *
 * Captures the pre-migration look of the key admin/partner surfaces at
 * 1280x900 / 390x844 / 320x700 so every later migration wave produces a
 * reviewable visual diff against a committed baseline. Snapshots live in
 * `visual-baseline.spec.ts-snapshots/` (committable; only test-results/ and
 * playwright-report/ are gitignored).
 *
 * - Real logins against the live backend (no API mocks). The backend must be
 *   reachable at :8080 (the dev default of VITE_API_BASE_URL).
 * - Chromium project only — one rendering engine per baseline set:
 *     cd frontend && pnpm exec playwright test tests/e2e/visual-baseline.spec.ts \
 *       --project=chromium --update-snapshots
 * - Employee portal is intentionally absent: no existing spec performs a real
 *   employee login (employee suites use synthetic tokens + route mocks) and no
 *   employee QA account is documented. Add credentials before baselining
 *   /employee.
 *
 * Determinism: prefers-reduced-motion context, CSS animation/transition kill
 * stylesheet, `animations: "disabled"` screenshots, exact pixel match
 * (maxDiffPixelRatio: 0). If a surface proves unstable (live clocks, ticking
 * timestamps, canvas chart animation), register a selector for it in
 * VOLATILE_SELECTORS with a comment on what was unstable — do that instead of
 * loosening the diff tolerance.
 */

// QA credentials documented in frontend/CLAUDE.md (all QA accounts share one
// password). Post-login destinations mirror the redirects in pages/Login.tsx.
const LOGINS = {
  admin: { username: "frankng", password: "Admin123", destination: /\/admin$/ },
  partner: { username: "cuongnv", password: "Admin123", destination: /\/partner\/dashboard$/ },
} as const;

// Surfaces with nodes observed to flake between runs. Empty until proven:
// add entries only after seeing a real unstable diff, e.g.
//   "admin/dashboard": ["[data-testid='live-clock']"],
const VOLATILE_SELECTORS: Record<string, string[]> = {};

const SETTLE_MS = 1_000;

const ANIMATION_KILLER =
  "*, *::before, *::after { animation: none !important; transition: none !important; caret-color: transparent !important; }";

interface Surface {
  title: string;
  path: string;
}

// /admin renders the dashboard at its index route.
const ADMIN_SURFACES: Surface[] = [
  { title: "admin/dashboard", path: "/admin" },
  { title: "admin/users", path: "/admin/users" },
  { title: "admin/projects", path: "/admin/projects" },
  { title: "admin/employees", path: "/admin/employees" },
  { title: "admin/timesheet", path: "/admin/timesheet" },
];

const PARTNER_SURFACES: Surface[] = [
  { title: "partner/dashboard", path: "/partner/dashboard" },
  { title: "partner/timesheets", path: "/partner/timesheet" },
  { title: "partner/projects", path: "/partner/projects" },
];

const VIEWPORTS = [
  { label: "1280", width: 1280, height: 900 },
  { label: "390", width: 390, height: 844 },
  { label: "320", width: 320, height: 700 },
];

test.describe("visual baselines", () => {
  test.skip(({ browserName }) => browserName !== "chromium", "Baselines come from the chromium project only.");
  test.use({ contextOptions: { reducedMotion: "reduce" } });
  // Dev-server on-demand compile + real login + full-page capture per surface.
  test.setTimeout(90_000);

  async function login(page: Page, account: (typeof LOGINS)[keyof typeof LOGINS]) {
    await page.goto("/login");
    await page.locator("#emailOrUsername").fill(account.username);
    await page.locator("#password").fill(account.password);
    await page.locator('button[type="submit"]').click();
    await Promise.race([
      page.waitForURL(account.destination, { timeout: 20_000 }),
      // The server-side captcha lockout surfaces this field before the submit
      // can complete; report it instead of failing with a navigation timeout.
      page
        .locator("#captchaCode")
        .waitFor({ state: "visible", timeout: 20_000 })
        .then(() => {
          throw new Error(
            `CAPTCHA demanded for ${account.username} — clear the lockout on /login, then rerun.`
          );
        }),
      page
        .locator('[role="alert"]')
        .waitFor({ state: "visible", timeout: 20_000 })
        .then(() => {
          throw new Error(
            `Login rejected for ${account.username} — check the credentials and that the backend on :8080 is up.`
          );
        }),
    ]);
  }

  async function stabilize(page: Page, title: string) {
    // Reduced motion via context option may not cover every code path; state
    // it on the page as well.
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.addStyleTag({ content: ANIMATION_KILLER });
    // Pages with short polling intervals may never go fully idle; the settle
    // delay is the real determinism budget (lets canvas charts finish).
    await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
    await page.waitForTimeout(SETTLE_MS);
    for (const selector of VOLATILE_SELECTORS[title] ?? []) {
      await page.evaluate((sel) => {
        document.querySelectorAll(sel).forEach((node) => {
          (node as HTMLElement).style.visibility = "hidden";
        });
      }, selector);
    }
  }

  async function capture(page: Page, title: string, viewportLabel: string) {
    await expect(page).toHaveScreenshot(`${title.replace("/", "-")}@${viewportLabel}.png`, {
      fullPage: true,
      animations: "disabled",
      maxDiffPixelRatio: 0,
      timeout: 20_000,
    });
  }

  for (const viewport of VIEWPORTS) {
    test.describe(`viewport ${viewport.label}`, () => {
      test.use({ viewport: { width: viewport.width, height: viewport.height } });

      for (const surface of ADMIN_SURFACES) {
        test(`${surface.title}@${viewport.label}`, async ({ page }) => {
          await login(page, LOGINS.admin);
          await page.goto(surface.path);
          await stabilize(page, surface.title);
          await capture(page, surface.title, viewport.label);
        });
      }

      for (const surface of PARTNER_SURFACES) {
        test(`${surface.title}@${viewport.label}`, async ({ page }) => {
          await login(page, LOGINS.partner);
          await page.goto(surface.path);
          await stabilize(page, surface.title);
          await capture(page, surface.title, viewport.label);
        });
      }
    });
  }
});
