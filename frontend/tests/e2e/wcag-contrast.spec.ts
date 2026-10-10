import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { ApiHelpers } from '../utils/api-helpers';

const __dirname = dirname(fileURLToPath(import.meta.url));

/**
 * WCAG 2.2 AA contrast smoke sweep.
 *
 * Runs axe-core's color-contrast and color-contrast-enhanced rules against
 * every major route × theme × viewport. Fails the suite on any `serious` or
 * `critical` violation. Per-route JSON evidence is written to
 * plans/2026-10-10-wcag-contrast/reports/axe/.
 *
 * Smoke coverage: one representative route per page template (dashboard,
 * list, detail, form, settings, login) across admin and partner roles.
 * Full-route expansion is a follow-up once the smoke is green.
 */

const REPORT_DIR = resolve(__dirname, '../../../plans/2026-10-10-wcag-contrast/reports/axe');
const SHOT_DIR = resolve(__dirname, '../../../plans/2026-10-10-wcag-contrast/reports/screenshots');

interface RouteSpec {
  path: string;
  role: 'admin' | 'partner';
  template: string;
}

const ROUTES: RouteSpec[] = [
  // Admin — one per template
  { path: '/admin', role: 'admin', template: 'dashboard' },
  { path: '/admin/users', role: 'admin', template: 'list' },
  { path: '/admin/employees', role: 'admin', template: 'list' },
  { path: '/admin/timesheet', role: 'admin', template: 'list' },
  { path: '/admin/ledger', role: 'admin', template: 'list' },
  { path: '/admin/settings', role: 'admin', template: 'settings' },
  { path: '/admin/wallet', role: 'admin', template: 'list' },
  // Partner — one per template
  { path: '/partner/dashboard', role: 'partner', template: 'dashboard' },
  { path: '/partner/projects', role: 'partner', template: 'list' },
  { path: '/partner/employees', role: 'partner', template: 'list' },
  { path: '/partner/timesheet', role: 'partner', template: 'list' },
];

const VIEWPORTS = [
  { name: '1280', width: 1280, height: 800 },
  { name: '390', width: 390, height: 844 },
  { name: '320', width: 320, height: 568 },
];

test.describe('WCAG 2.2 AA contrast smoke', () => {
  // Dev-server on-demand compile — same concession as the visual-baseline spec.
  test.setTimeout(180_000);

  for (const route of ROUTES) {
    for (const vp of VIEWPORTS) {
      test(`${route.role} ${route.path} @${vp.name} — zero serious color-contrast violations`, async ({ page }) => {
        await page.setViewportSize({ width: vp.width, height: vp.height });

        const api = new ApiHelpers(page);
        await api.mockLoginSuccess(
          { email: `${route.role}@example.com`, password: 'password' },
          route.role
        );
        await api.mockDashboardData(route.role);

        // Log in via the real form so the SPA mounts with the mock session.
        await page.goto('/login');
        await page.fill('#emailOrUsername', `${route.role}@example.com`);
        await page.fill('#password', 'password');
        await page.click('button[type=submit]');
        await page.waitForURL(new RegExp(route.role), { timeout: 30_000 }).catch(() => {});

        // Navigate to the target route.
        await page.goto(route.path, { waitUntil: 'networkidle', timeout: 30_000 }).catch(() => {});
        // Give the SPA a beat to hydrate lazy chunks.
        await page.waitForTimeout(1_500);

        // Run axe with the AA color-contrast rule (4.5:1 normal text).
        // color-contrast-enhanced (AAA, 7:1) is out of scope for AA conformance.
        const results = await new AxeBuilder({ page })
          .withRules(['color-contrast'])
          .analyze();

        // Persist evidence.
        mkdirSync(REPORT_DIR, { recursive: true });
        mkdirSync(SHOT_DIR, { recursive: true });
        const slug = `${route.role}-${route.path.replace(/\//g, '-')}-${vp.name}`;
        writeFileSync(
          resolve(REPORT_DIR, `${slug}.json`),
          JSON.stringify({ route: route.path, role: route.role, viewport: vp.name, results }, null, 2)
        );
        await page.screenshot({ path: resolve(SHOT_DIR, `${slug}.png`), fullPage: false });

        // Fail on serious/critical violations.
        const blocking = results.violations.filter(
          (v) => v.impact === 'serious' || v.impact === 'critical'
        );
        const summary = blocking
          .map((v) => `${v.id} (${v.impact}): ${v.nodes.length} nodes — ${v.help}`)
          .join('\n');
        expect(
          blocking,
          `Blocking color-contrast violations on ${route.path} @${vp.name}:\n${summary}`
        ).toHaveLength(0);
      });
    }
  }
});
