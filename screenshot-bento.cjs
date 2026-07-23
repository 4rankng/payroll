const { chromium } = require('/Users/dev/Documents/projects/payroll/frontend/node_modules/playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1680, height: 1100 }, deviceScaleFactor: 1 });
  await page.goto('file:///Users/dev/Documents/projects/payroll/bento-overview.html');
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(2500);
  await page.screenshot({ path: '/Users/dev/Documents/projects/payroll/bento-preview.png', fullPage: true });
  await browser.close();
  console.log('done');
})();
