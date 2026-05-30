import { chromium, FullConfig } from '@playwright/test';

async function globalSetup(config: FullConfig) {
  const { baseURL } = config.projects[0].use;

  console.log('🚀 Starting global setup...');

  // Start browser for setup tasks
  const browser = await chromium.launch();
  const page = await browser.newPage();

  try {
    // Wait for the application to be ready
    await page.goto(baseURL!);
    await page.waitForLoadState('networkidle');

    // Check if the app is running
    const title = await page.title();
    console.log(`✅ Application is running with title: ${title}`);

    // Store any global state if needed
    // await page.context().storageState({ path: 'tests/fixtures/auth-state.json' });

  } catch (error) {
    console.error('❌ Global setup failed:', error);
    throw error;
  } finally {
    await browser.close();
  }

  console.log('✅ Global setup completed');
}

export default globalSetup;
