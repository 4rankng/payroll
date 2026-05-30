import { FullConfig } from '@playwright/test';

async function globalTeardown(config: FullConfig) {
  console.log('🧹 Starting global teardown...');

  // Clean up any global resources
  // Remove temporary files, close connections, etc.

  console.log('✅ Global teardown completed');
}

export default globalTeardown;
