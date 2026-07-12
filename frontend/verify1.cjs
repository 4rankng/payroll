const { chromium } = require('/Users/dev/Documents/projects/payroll/frontend/node_modules/.pnpm/playwright@1.61.1/node_modules/playwright/index.js');
const BASE='http://localhost:3000';
(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport:{width:375,height:812}, isMobile:true, hasTouch:true });
  const errs=[]; page.on('pageerror',e=>errs.push(e.message));
  // first hit to trigger vite rebundle
  await page.goto(BASE+'/login',{waitUntil:'networkidle'});
  await page.locator('input').first().fill('frankng');
  await page.locator('input[type="password"]').fill('Admin123');
  await page.locator('button[type="submit"]').click().catch(()=>{});
  await page.waitForTimeout(4000); // extra time for rebundle
  for(const [label,path] of [['settings','/admin/settings'],['advance','/admin/advance-payments']]){
    errs.length=0;
    await page.goto(BASE+path,{waitUntil:'networkidle',timeout:30000}).catch(()=>{});
    await page.waitForTimeout(2500);
    const info = await page.evaluate(() => {
      const body=document.body.innerText||'';
      const errFallback=/Lỗi khi tải|đã xảy ra|thử lại/i.test(body);
      const main=document.querySelector('main')?.innerText?.slice(0,90)||'';
      return {errFallback, main: main.replace(/\n/g,' ').slice(0,80)};
    });
    console.log(label.padEnd(10)+'| errs:'+String(errs.length).padStart(2)+' | fallback:'+(info.errFallback?'YES ❌':'no ✓')+' | "'+info.main+'"');
  }
  await browser.close();
})();
