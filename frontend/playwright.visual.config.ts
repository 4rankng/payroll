import { defineConfig, devices } from '@playwright/test';

/**
 * Config for the visual-baseline suite ONLY.
 *
 * Serves the app from http://localhost:3000 — the origin the backend's CORS
 * whitelist actually allows for dev (vite.config.ts server.port = 3000, and
 * the QA tooling targets :3000). The main playwright.config.ts serves from
 * 127.0.0.1:5173, whose origin the login API rejects via CORS preflight (403),
 * so authenticated flows cannot run from there.
 *
 * Run: pnpm exec playwright test --config playwright.visual.config.ts
 */
export default defineConfig({
  testDir: './tests/e2e',
  // visual-baseline always; auth.spec rides this config because the main
  // config's 127.0.0.1:5173 origin is CORS-blocked by the backend (only
  // localhost:3000 is whitelisted).
  testMatch: '**/{visual-baseline,auth}.spec.ts',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10000,
    navigationTimeout: 30000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'pnpm exec vite --host localhost --port 3000',
    url: 'http://localhost:3000',
    reuseExistingServer: true,
    timeout: 120 * 1000,
  },
  timeout: 60 * 1000,
  expect: {
    timeout: 10000,
  },
  outputDir: 'test-results/',
  globalSetup: './tests/fixtures/global-setup.ts',
  globalTeardown: './tests/fixtures/global-teardown.ts',
});
