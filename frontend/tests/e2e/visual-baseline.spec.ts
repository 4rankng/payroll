import { expect, test, type Page, type Route } from "@playwright/test";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Pre-migration visual baselines for the Untitled UI PRO migration.
 *
 * Captures the pre-migration look of the key admin/partner surfaces at
 * 1280x900 / 390x844 / 320x700 so every later migration wave produces a
 * reviewable visual diff against a committed baseline. Snapshots live in
 * `visual-baseline.spec.ts-snapshots/` (committable; only test-results/ and
 * playwright-report/ are gitignored).
 *
 * Run (needs the backend on :8080; the app must be served from
 * localhost:3000 — the CORS-whitelisted dev origin, hence the dedicated
 * config playwright.visual.config.ts):
 *     cd frontend && pnpm exec playwright test \
 *       --config playwright.visual.config.ts [--update-snapshots]
 *
 * Determinism strategy, in order:
 *   1. Real logins against the live backend (no business-data mocks).
 *   2. Volatile data endpoints are pinned to CAPTURED real responses stored
 *      in tests/fixtures/visual-baseline/ (see PINNED_DATA below).
 *      RULING 2026-10-04 (team-lead, option 2): business-data pins apply to
 *      visual-baseline.spec.ts ONLY, for determinism of the styling
 *      reference — the diff target is styling chrome, not business numbers.
 *      Payloads are captured live responses (never hand-invented) so data
 *      density and Vietnamese realism stay authentic. Sole exception:
 *      admin--dashboard-bank-usage-projects.json is a deterministic empty
 *      payload because that endpoint stalls >90s (known perf bug) and never
 *      returned a capturable 200 — swap in a captured file if one is ever
 *      obtained. `notifications/unread` stays pinned empty as chrome
 *      suppression (roster banner presence flip), not business data.
 *   3. Capture waits for the page to finish LOADING, not just for network
 *      idle: a 2.5s network-quiet window must elapse (all slow aggregate
 *      endpoints drained), then visible skeletons (`.animate-pulse`,
 *      `.ct-loading`, `[data-slot="skeleton"]`) and "Đang tải…" text markers
 *      must clear (bounded waits — a stuck marker is captured as-is). The
 *      first baseline pass committed skeleton states on partner surfaces
 *      because it only waited for network idle + a fixed delay.
 *   4. Chromium-only, reduced motion, CSS animation kill stylesheet,
 *      `animations: "disabled"` screenshots, exact pixel match
 *      (maxDiffPixelRatio: 0).
 *   5. Live-count text that legitimately changes with QA data (pending
 *      approval counts, the "today" date) is masked via VOLATILE_SELECTORS
 *      below — each entry names what it hides. Masking keeps layout
 *      (visibility: hidden). Add entries only after seeing a real unstable
 *      diff; do not loosen the tolerance instead.
 *
 * Employee portal is intentionally absent: no existing spec performs a real
 * employee login (employee suites use synthetic tokens + route mocks) and no
 * employee QA account is documented.
 */

// QA credentials documented in frontend/CLAUDE.md (all QA accounts share one
// password). Post-login destinations mirror the redirects in pages/Login.tsx.
const LOGINS = {
  admin: { username: "frankng", password: "Admin123", destination: /\/admin$/ },
  partner: { username: "cuongnv", password: "Admin123", destination: /\/partner\/dashboard$/ },
} as const;

// Surfaces with nodes observed to flake between runs. Every entry names what
// it masks and why. `visibility: hidden` keeps layout, so masking never
// shifts the page.
const VOLATILE_SELECTORS: Record<string, string[]> = {
  // admin mobile dashboard ("Tổng quan"):
  // - h1 + p: MobilePageHeader subtitle renders `format(new Date(),
  //   "EEEE, dd/MM")` — the literal today date, changes at midnight.
  // - button > span.rounded-full.bg-warning: the "Duyệt công" quick-action
  //   badge (pending approvals count) — a live number.
  // - section:nth-of-type(2) is the "Cần xử lý" card (section 1 is the
  //   operations panel; the roster banner above them never renders while
  //   notifications are pinned empty): span.tabular-nums are the live row
  //   values (pending count / ví / bank-project count), span.mt-1 are the
  //   row descriptions (they change with queue state: loading text / queue
  //   text / empty text).
  "admin/dashboard": [
    ".admin-dashboard-page-mobile h1 + p",
    ".admin-dashboard-page-mobile button > span.rounded-full.bg-warning",
    ".admin-dashboard-page-mobile > section:nth-of-type(2) span.tabular-nums",
    ".admin-dashboard-page-mobile > section:nth-of-type(2) span.mt-1",
  ],
};

const SETTLE_MS = 400;

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
  // Dev-server on-demand compile + real login + slow aggregate endpoints +
  // full-page capture per surface.
  test.setTimeout(180_000);

  function emptyNotifications(route: Route) {
    return route.fulfill({ json: { status: "success", data: { notifications: [], count: 0 } } });
  }

  // Captured live responses, one file per volatile endpoint, role-scoped
  // (admin and partner share several paths with different query params).
  // Fixture names map 1:1 to the files in tests/fixtures/visual-baseline/.
  const FIXTURE_DIR = path.join(
    path.dirname(fileURLToPath(import.meta.url)),
    "..",
    "fixtures",
    "visual-baseline"
  );

  const PINNED_DATA: Record<"admin" | "partner", Array<[pattern: string, fixture: string]>> = {
    admin: [
      ["**/api/v1/dashboard/summary*", "admin--dashboard-summary.json"],
      ["**/api/v1/dashboard/bank-usage/projects*", "admin--dashboard-bank-usage-projects.json"],
      ["**/api/v1/dashboard/monthly-financials*", "admin--dashboard-monthly-financials.json"],
      ["**/api/v1/dashboard/employee-activity*", "admin--dashboard-employee-activity.json"],
      ["**/api/v1/dashboard/new-employees*", "admin--dashboard-new-employees.json"],
      ["**/api/v1/dashboard/recent-activities*", "admin--dashboard-recent-activities.json"],
      ["**/api/v1/dashboard/top-paid-employees*", "admin--dashboard-top-paid-employees.json"],
      ["**/api/v1/dashboard/check-in-health*", "admin--dashboard-check-in-health.json"],
      ["**/api/v1/dashboard/salary-distribution*", "admin--dashboard-salary-distribution.json"],
      ["**/api/v1/dashboard/project-weekly-profit*", "admin--dashboard-project-weekly-profit.json"],
      ["**/api/v1/dashboard/project-profitability*", "admin--dashboard-project-profitability.json"],
      ["**/api/v1/timesheets/grouped*", "admin--timesheets-grouped.json"],
      ["**/api/v1/timesheets/summary*", "admin--timesheets-summary.json"],
      ["**/api/v1/timesheets/edit-requests*", "admin--timesheets-edit-requests.json"],
      ["**/api/v1/timesheets/cash-readiness*", "admin--timesheets-cash-readiness.json"],
      ["**/api/v1/timesheets*", "admin--timesheets-pending.json"],
      ["**/api/v1/users/summary", "admin--users-summary.json"],
      ["**/api/v1/users*", "admin--users.json"],
      ["**/api/v1/employees/summary", "admin--employees-summary.json"],
      ["**/api/v1/employees/missing-bank-details", "admin--employees-missing-bank-details.json"],
      ["**/api/v1/employees*", "admin--employees.json"],
      ["**/api/v1/projects*", "admin--projects.json"],
    ],
    partner: [
      ["**/api/v1/dashboard/partner*", "partner--dashboard-partner.json"],
      ["**/api/v1/projects/partner-summary", "partner--projects-partner-summary.json"],
      ["**/api/v1/projects*", "partner--projects.json"],
      ["**/api/v1/timesheets/grouped*", "partner--timesheets-grouped.json"],
      ["**/api/v1/timesheets/summary*", "partner--timesheets-summary.json"],
      ["**/api/v1/timesheets/edit-requests*", "partner--timesheets-edit-requests.json"],
      ["**/api/v1/timesheets*", "partner--timesheets.json"],
      ["**/api/v1/employees?*", "partner--employees.json"],
      ["**/api/v1/employees/missing-bank-details", "partner--employees-missing-bank-details.json"],
    ],
  };

  async function pinVolatileData(page: Page, role: "admin" | "partner") {
    for (const [pattern, fixture] of PINNED_DATA[role]) {
      await page.route(pattern, (route) => route.fulfill({ path: path.join(FIXTURE_DIR, fixture) }));
    }
  }

  function pinChromeData(page: Page) {
    // Registered before navigation so the first unread fetch is intercepted.
    return page.route("**/api/v1/notifications/unread*", emptyNotifications);
  }

  async function login(page: Page, account: (typeof LOGINS)[keyof typeof LOGINS]) {
    // One retry: a single login timeout was observed (admin/users@1280) with
    // neither captcha nor alert visible — a transient backend stall. Captcha
    // and bad-credential outcomes are not retried; retrying cannot fix them.
    for (let attempt = 1; attempt <= 2; attempt++) {
      await page.goto("/login");
      await page.locator("#emailOrUsername").fill(account.username);
      await page.locator("#password").fill(account.password);
      await page.locator('button[type="submit"]').click();
      const outcome = await Promise.race([
        page.waitForURL(account.destination, { timeout: 30_000 }).then(() => "navigated" as const),
        page
          .locator("#captchaCode")
          .waitFor({ state: "visible", timeout: 30_000 })
          .then(() => "captcha" as const),
        page
          .locator('[role="alert"]')
          .waitFor({ state: "visible", timeout: 30_000 })
          .then(() => "alert" as const),
      ]);
      if (outcome === "navigated") return;
      if (outcome === "captcha") {
        throw new Error(
          `CAPTCHA demanded for ${account.username} — clear the lockout on /login, then rerun.`
        );
      }
      // alert or timeout: back off briefly and retry the whole flow once.
      await page.waitForTimeout(3_000);
    }
    throw new Error(
      `Login did not complete for ${account.username} after one retry — check the backend on :8080, ` +
        "the QA credentials, and test-results/ for the alert text."
    );
  }

  // Dashboard aggregates (bank usage, payout forecast, ops metrics) are slow
  // (>10s endpoints observed) and their widgets render intermediate
  // skeleton/zero states until the data lands — at varying times per run.
  // Capture only after the page has made NO request for `quietMs`
  // continuously, so every endpoint has drained before the shot.
  async function waitForNetworkQuiet(page: Page, quietMs: number, budgetMs: number) {
    await page.evaluate(() => {
      const w = window as unknown as { __lastNetActivity?: number };
      w.__lastNetActivity = Date.now();
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.entryType === "resource") w.__lastNetActivity = Date.now();
        }
      }).observe({ type: "resource", buffered: false });
    });
    await page
      .waitForFunction(
        ({ quietWindow }) => {
          const w = window as unknown as { __lastNetActivity?: number };
          return w.__lastNetActivity !== undefined && Date.now() - w.__lastNetActivity >= quietWindow;
        },
        { quietWindow: quietMs },
        { timeout: budgetMs }
      )
      .catch(() => {
        // Continuous polling starves the quiet window; capture whatever state
        // was reached instead of failing the surface.
      });
  }

  async function stabilize(page: Page, title: string) {
    // Reduced motion via context option may not cover every code path; state
    // it on the page as the second layer.
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.addStyleTag({ content: ANIMATION_KILLER });
    // Remote font stylesheets apply via an async media="print" -> "all"
    // swap (index.html data-font-stylesheet links: Manrope, Inter, JetBrains
    // Mono). If a capture lands before a stylesheet applies or its woff2
    // files finish, text renders with fallback metrics — subpixel glyph
    // diffs between runs (observed on partner/projects@320: 257 differing
    // antialiasing pixels in the search placeholder, content identical).
    // Wait for every stylesheet to apply, then for document.fonts.ready and
    // an explicit load() of all registered faces, so captures always use
    // final font metrics.
    await page
      .waitForFunction(
        () =>
          Array.from(document.querySelectorAll("link[data-font-stylesheet]")).every(
            (link) => link.getAttribute("media") === "all"
          ),
        undefined,
        { timeout: 15_000 }
      )
      .catch(() => {
        // Offline environment: the links never apply and fallback fonts are
        // the consistent state — capture proceeds.
      });
    await page
      .evaluate(() => {
        const faces = Array.from(document.fonts);
        return Promise.all([
          document.fonts.ready,
          ...faces.map((face) => face.load().catch(() => null)),
        ]).then(() => true);
      })
      .catch(() => {});
    // Pages with short polling intervals may never go fully idle; the quiet
    // window and loading waits below are the real determinism budget.
    await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
    await waitForNetworkQuiet(page, 2_500, 45_000);
    // The committed-baseline incident: partner surfaces captured their
    // skeleton state because the first harness only waited network idle + a
    // fixed delay. Runs AFTER the quiet window: a refetch that completed
    // during it has already left its skeleton phase. Wait until no skeleton
    // placeholder is visibly occupying space before capturing.
    await page
      .waitForFunction(
        () => {
          const placeholders = document.querySelectorAll(
            '.animate-pulse, .ct-loading, [data-slot="skeleton"]'
          );
          return Array.from(placeholders).every((el) => {
            const rect = el.getBoundingClientRect();
            return rect.width === 0 || rect.height === 0;
          });
        },
        undefined,
        { timeout: 15_000 }
      )
      .catch(() => {
        // A permanently pulsing element (live indicator) stalls forever;
        // capture whatever state was reached instead of failing the surface.
      });
    // Same for explicit "Đang tải…" text markers (e.g. the timesheet forecast
    // widget frozen at "Đang tải dự báo tiền trả" in a committed baseline).
    await page
      .waitForFunction(
        () =>
          !Array.from(document.querySelectorAll<HTMLElement>("body *")).some(
            (el) =>
              el.childElementCount === 0 &&
              el.offsetParent !== null &&
              /^Đang tải/.test((el.textContent || "").trim())
          ),
        undefined,
        { timeout: 15_000 }
      )
      .catch(() => {
        // Bounded: a never-resolving marker stays captured as-is.
      });
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

      for (const surface of [...ADMIN_SURFACES, ...PARTNER_SURFACES]) {
        test(`${surface.title}@${viewport.label}`, async ({ page }) => {
          const role = surface.title.startsWith("admin") ? ("admin" as const) : ("partner" as const);
          await login(page, LOGINS[role]);
          await pinChromeData(page);
          await pinVolatileData(page, role);
          await page.goto(surface.path);
          await stabilize(page, surface.title);
          await capture(page, surface.title, viewport.label);
        });
      }
    });
  }
});
